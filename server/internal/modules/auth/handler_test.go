package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/config"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/errs"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/oidc"
	"git.stuhelper.com/StuHelper/StuHelper/internal/testutil/redisfixture"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func newTestHandler(t *testing.T) (*Handler, *redisfixture.Fixture) {
	t.Helper()
	fixture := redisfixture.Start(t)

	h := &Handler{
		oidcClient:           oidc.NewStubClient("https://sso.example.com/authorize"),
		redisClient:          fixture.Client,
		tokenConfig:          config.TokenConfig{},
		defaultRedirectURL:   "https://web.example.com",
		allowedRedirectHosts: map[string]struct{}{"web.example.com": {}},
	}
	return h, fixture
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

func TestRegisteredRouteSurfaceDoesNotExposeLocalPhoneOTPLogin(t *testing.T) {
	h, _ := newTestHandler(t)
	r := gin.New()
	api := r.Group("/api/v1")

	h.RegisterPublicRoutes(api)
	h.RegisterRoutesWithAuthMiddleware(api, func(c *gin.Context) { c.Next() })

	routes := make(map[string]struct{}, len(r.Routes()))
	for _, route := range r.Routes() {
		routes[route.Method+" "+route.Path] = struct{}{}
	}

	assert.Contains(t, routes, "POST /api/v1/auth/exchange-native")
	for _, route := range []string{
		"POST /api/v1/auth/otp/request",
		"POST /api/v1/auth/otp/verify",
		"POST /api/v1/auth/phone/request-otp",
		"POST /api/v1/auth/phone/verify-otp",
	} {
		assert.NotContains(t, routes, route)
	}
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

func TestGetStepUpURL_ReturnsReauthURL(t *testing.T) {
	h, _ := newTestHandler(t)

	r := gin.New()
	r.GET("/auth/step-up", h.GetStepUpURL)

	req := httptest.NewRequest(http.MethodGet, "/auth/step-up", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"url"`)
	assert.Contains(t, w.Body.String(), `"state"`)
	assert.Contains(t, w.Body.String(), "prompt=login")
	assert.Contains(t, w.Body.String(), "max_age=0")
	assert.Contains(t, w.Body.String(), "acr_values=mfa")
}

func TestGetLoginURL_WithPromptLoginReturnsReauthURL(t *testing.T) {
	h, _ := newTestHandler(t)

	r := gin.New()
	r.GET("/auth/login", h.GetLoginURL)

	req := httptest.NewRequest(http.MethodGet, "/auth/login?prompt=login&max_age=0", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "prompt=login")
	assert.Contains(t, w.Body.String(), "max_age=0")
	assert.Contains(t, w.Body.String(), "acr_values=mfa")
}

func TestAuthURLHandlersRejectRepeatedSingleValueQueryParameters(t *testing.T) {
	tests := []struct {
		name   string
		target string
		mount  func(*gin.Engine, *Handler)
	}{
		{
			name:   "login repeated prompt",
			target: "/auth/login?prompt=login&prompt=none",
			mount:  func(r *gin.Engine, h *Handler) { r.GET("/auth/login", h.GetLoginURL) },
		},
		{
			name:   "login repeated redirect",
			target: "/auth/login?redirect=/courses/1&redirect=/admin",
			mount:  func(r *gin.Engine, h *Handler) { r.GET("/auth/login", h.GetLoginURL) },
		},
		{
			name:   "signup repeated app",
			target: "/auth/signup?app=web&app=admin",
			mount:  func(r *gin.Engine, h *Handler) { r.GET("/auth/signup", h.GetSignupURL) },
		},
		{
			name:   "step up repeated redirect",
			target: "/auth/step-up?redirect=/settings&redirect=/admin",
			mount:  func(r *gin.Engine, h *Handler) { r.GET("/auth/step-up", h.GetStepUpURL) },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, _ := newTestHandler(t)
			r := gin.New()
			tt.mount(r, h)

			req := httptest.NewRequest(http.MethodGet, tt.target, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
			assert.Contains(t, w.Body.String(), string(errs.ErrInvalidParam))
			assert.NotContains(t, w.Body.String(), `"url"`)
			assert.NotContains(t, w.Body.String(), `"state"`)
		})
	}
}

func TestGetLoginURL_StoresStateInRedis(t *testing.T) {
	h, fixture := newTestHandler(t)

	r := gin.New()
	r.GET("/auth/login", h.GetLoginURL)

	req := httptest.NewRequest(http.MethodGet, "/auth/login?redirect=/courses/1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	found := false
	for _, k := range fixture.Server.Keys() {
		if len(k) > len(oidcStateRedisPrefix) && k[:len(oidcStateRedisPrefix)] == oidcStateRedisPrefix {
			found = true
			break
		}
	}
	assert.True(t, found, "OIDC state should be stored in Redis")
}

func TestGetLoginURL_StoresRequestedApplicationInState(t *testing.T) {
	h, fixture := newTestHandler(t)

	r := gin.New()
	r.GET("/auth/login", h.GetLoginURL)

	req := httptest.NewRequest(http.MethodGet, "/auth/login?app=admin", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var envelope struct {
		Data struct {
			State string `json:"state"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &envelope))

	raw, err := fixture.Client.Get(context.Background(), oidcStateRedisPrefix+envelope.Data.State).Result()
	require.NoError(t, err)

	var payload oidcStatePayload
	require.NoError(t, json.Unmarshal([]byte(raw), &payload))
	assert.Equal(t, "admin", payload.Application)
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

func TestGetLoginURL_NativeSkipsStateCookie(t *testing.T) {
	h, _ := newTestHandler(t)

	r := gin.New()
	r.GET("/auth/login", h.GetLoginURL)

	req := httptest.NewRequest(http.MethodGet, "/auth/login?platform=native", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	for _, c := range w.Result().Cookies() {
		assert.NotEqual(t, oidcStateCookieName, c.Name)
	}
}

func TestConsumeOIDCState_OneTimeAndCookieBound(t *testing.T) {
	fixture := redisfixture.Start(t)

	h := &Handler{
		redisClient:        fixture.Client,
		tokenConfig:        config.TokenConfig{},
		defaultRedirectURL: "https://web.example.com",
	}

	const state = "test-state"
	require.NoError(t, h.storeOIDCState(context.Background(), oidcStateInput{
		state:        state,
		redirect:     "/courses/1",
		codeVerifier: "test-verifier",
		application:  oidc.ApplicationWeb,
	}))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/callback?state="+state, nil)
	req.AddCookie(&http.Cookie{Name: oidcStateCookieName, Value: state})
	c.Request = req

	redirect, codeVerifier, appKey, isNative, err := h.consumeOIDCState(c, state)
	require.NoError(t, err)
	assert.Equal(t, "/courses/1", redirect)
	assert.Equal(t, "test-verifier", codeVerifier)
	assert.Equal(t, oidc.ApplicationWeb, appKey)
	assert.False(t, isNative)

	w2 := httptest.NewRecorder()
	c2, _ := gin.CreateTestContext(w2)
	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/auth/callback?state="+state, nil)
	req2.AddCookie(&http.Cookie{Name: oidcStateCookieName, Value: state})
	c2.Request = req2

	_, _, _, _, err = h.consumeOIDCState(c2, state)
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

func TestResolveRedirectTarget_AllowedAbsoluteAndEmpty(t *testing.T) {
	h := &Handler{
		defaultRedirectURL:   "https://web.example.com",
		allowedRedirectHosts: map[string]struct{}{"web.example.com": {}, "admin.example.com": {}},
	}
	assert.Equal(t, "https://admin.example.com/reviews", h.resolveRedirectTarget("https://admin.example.com/reviews"))
	assert.Equal(t, "https://web.example.com", h.resolveRedirectTarget("   "))
}
