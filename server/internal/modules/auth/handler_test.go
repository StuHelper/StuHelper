package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/config"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/oidc"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func newTestHandler(t *testing.T) (*Handler, *miniredis.Miniredis) {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	h := &Handler{
		oidcClient:           oidc.NewStubClient("https://sso.example.com/authorize"),
		redisClient:          rdb,
		tokenConfig:          config.TokenConfig{},
		defaultRedirectURL:   "https://web.example.com",
		allowedRedirectHosts: map[string]struct{}{"web.example.com": {}},
	}
	return h, mr
}

func TestGetLoginURL_ReturnsAuthURL(t *testing.T) {
	h, _ := newTestHandler(t)

	r := gin.New()
	r.GET("/auth/login", h.GetLoginURL)

	req := httptest.NewRequest(http.MethodGet, "/auth/login", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"url"`)
	assert.Contains(t, w.Body.String(), `"state"`)
	assert.Contains(t, w.Body.String(), `"success":true`)
}

func TestGetSignupURL_ReturnsAuthURL(t *testing.T) {
	h, _ := newTestHandler(t)

	r := gin.New()
	r.GET("/auth/signup", h.GetSignupURL)

	req := httptest.NewRequest(http.MethodGet, "/auth/signup", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"url"`)
	assert.Contains(t, w.Body.String(), `"state"`)
}

func TestGetLoginURL_StoresStateInRedis(t *testing.T) {
	h, mr := newTestHandler(t)

	r := gin.New()
	r.GET("/auth/login", h.GetLoginURL)

	req := httptest.NewRequest(http.MethodGet, "/auth/login?redirect=/courses/1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	found := false
	for _, k := range mr.Keys() {
		if len(k) > len(oidcStateRedisPrefix) && k[:len(oidcStateRedisPrefix)] == oidcStateRedisPrefix {
			found = true
			break
		}
	}
	assert.True(t, found, "OIDC state should be stored in Redis")
}

func TestGetLoginURL_SetsCookie(t *testing.T) {
	h, _ := newTestHandler(t)

	r := gin.New()
	r.GET("/auth/login", h.GetLoginURL)

	req := httptest.NewRequest(http.MethodGet, "/auth/login", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var stateCookie *http.Cookie
	for _, c := range w.Result().Cookies() {
		if c.Name == oidcStateCookieName {
			stateCookie = c
			break
		}
	}
	require.NotNil(t, stateCookie, "oidc_state cookie should be set")
	assert.True(t, stateCookie.HttpOnly)
}

func TestConsumeOIDCState_OneTimeAndCookieBound(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer rdb.Close()

	h := &Handler{
		redisClient:        rdb,
		tokenConfig:        config.TokenConfig{},
		defaultRedirectURL: "https://web.example.com",
	}

	const state = "test-state"
	require.NoError(t, h.storeOIDCState(context.Background(), state, "/courses/1", "test-verifier"))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/callback?state="+state, nil)
	req.AddCookie(&http.Cookie{Name: oidcStateCookieName, Value: state})
	c.Request = req

	redirect, codeVerifier, err := h.consumeOIDCState(c, state)
	require.NoError(t, err)
	assert.Equal(t, "/courses/1", redirect)
	assert.Equal(t, "test-verifier", codeVerifier)

	w2 := httptest.NewRecorder()
	c2, _ := gin.CreateTestContext(w2)
	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/auth/callback?state="+state, nil)
	req2.AddCookie(&http.Cookie{Name: oidcStateCookieName, Value: state})
	c2.Request = req2

	_, _, err = h.consumeOIDCState(c2, state)
	require.Error(t, err)
}

func TestResolveRedirectTarget_SchemeRelativeFallsBack(t *testing.T) {
	h := &Handler{defaultRedirectURL: "https://web.example.com"}
	assert.Equal(t, "https://web.example.com", h.resolveRedirectTarget("//evil.example.com"))
}

func TestResolveRedirectTarget_RelativePath(t *testing.T) {
	h := &Handler{
		defaultRedirectURL:   "https://web.example.com",
		allowedRedirectHosts: map[string]struct{}{"web.example.com": {}},
	}
	assert.Equal(t, "https://web.example.com/courses/1", h.resolveRedirectTarget("/courses/1"))
}

func TestResolveRedirectTarget_DisallowedHost(t *testing.T) {
	h := &Handler{
		defaultRedirectURL:   "https://web.example.com",
		allowedRedirectHosts: map[string]struct{}{"web.example.com": {}},
	}
	assert.Equal(t, "https://web.example.com", h.resolveRedirectTarget("https://evil.example.com/phish"))
}
