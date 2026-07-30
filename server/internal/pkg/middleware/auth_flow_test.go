package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/StuHelper/StuHelper/server/internal/pkg/config"
	"github.com/StuHelper/StuHelper/server/internal/pkg/crypto"
	"github.com/StuHelper/StuHelper/server/internal/pkg/errs"
	"github.com/StuHelper/StuHelper/server/internal/pkg/token"
	"github.com/StuHelper/StuHelper/server/internal/testutil/redisfixture"
)

func newTokenServiceForMiddlewareTest(t *testing.T) *token.Service {
	t.Helper()
	require.NoError(t, crypto.InitHMACKey("test-middleware-hmac-secret-32char", false))

	fixture := redisfixture.Start(t)

	svc, err := token.NewService(token.ServiceConfig{RedisClient: fixture.Client, AccessTTL: 300, RefreshTTL: 600})
	require.NoError(t, err)
	t.Cleanup(svc.Close)
	return svc
}

func mustSignAccessToken(t *testing.T, typ token.JWTTokenType) string {
	t.Helper()
	tok, err := token.SignJWT(crypto.GetHMACKey(), token.JWTClaims{
		Sub:         "user-1",
		Name:        "tester",
		DisplayName: "Tester",
		Roles:       []string{"user"},
		Typ:         typ,
		Sid:         "sid-1",
	}, time.Minute)
	require.NoError(t, err)
	return tok
}

func TestAuthMiddleware_MissingToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tokenSvc := newTokenServiceForMiddlewareTest(t)

	r := gin.New()
	r.Use(AuthMiddleware(nil, tokenSvc))
	r.GET("/me", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthMiddleware_RejectsSelfSignedCookieToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tokenSvc := newTokenServiceForMiddlewareTest(t)
	accessToken := mustSignAccessToken(t, token.JWTTokenTypeAccess)

	r := gin.New()
	r.Use(AuthMiddleware(nil, tokenSvc))
	r.GET("/me", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"userID": GetUserID(c), "name": GetUsername(c), "authed": IsAuthenticated(c)})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	req.AddCookie(&http.Cookie{Name: CookieAccessToken, Value: accessToken})
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), string(errs.ErrTokenInvalid))
	require.NotEmpty(t, w.Result().Cookies())
}

func TestOptionalAuthMiddleware_InvalidCookieClearsAndContinues(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tokenSvc := newTokenServiceForMiddlewareTest(t)
	refreshToken := mustSignAccessToken(t, token.JWTTokenTypeRefresh)

	r := gin.New()
	r.Use(OptionalAuthMiddleware(nil, tokenSvc, OptionalAuthConfig{CookieDomain: "example.com", CookieSecure: true}))
	r.GET("/public", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"userID": GetUserID(c), "authed": IsAuthenticated(c)})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/public", nil)
	req.AddCookie(&http.Cookie{Name: CookieAccessToken, Value: refreshToken})
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"userID":""`)
	assert.Contains(t, w.Body.String(), `"authed":false`)

	cookies := w.Result().Cookies()
	require.Len(t, cookies, 4)
	for _, cookie := range cookies {
		assert.Empty(t, cookie.Value)
		assert.Less(t, cookie.MaxAge, 0)
	}
}

func TestOptionalAuthMiddleware_RevokedCookieClearsAndContinues(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tokenSvc := newTokenServiceForMiddlewareTest(t)
	accessToken := mustSignAccessToken(t, token.JWTTokenTypeAccess)
	require.NoError(t, tokenSvc.GetBlacklist().Add(reqContext(), accessToken, tokenSvc.GetAccessTokenTTL()))

	r := gin.New()
	r.Use(OptionalAuthMiddleware(nil, tokenSvc, OptionalAuthConfig{CookieDomain: "example.com", CookieSecure: true}))
	r.GET("/public", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/public", nil)
	req.AddCookie(&http.Cookie{Name: CookieAccessToken, Value: accessToken})
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	cookies := w.Result().Cookies()
	require.Len(t, cookies, 4)
}

func TestAuthMiddleware_RevokedToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tokenSvc := newTokenServiceForMiddlewareTest(t)
	accessToken := mustSignAccessToken(t, token.JWTTokenTypeAccess)
	require.NoError(t, tokenSvc.GetBlacklist().Add(reqContext(), accessToken, tokenSvc.GetAccessTokenTTL()))

	r := gin.New()
	r.Use(AuthMiddleware(nil, tokenSvc))
	r.GET("/me", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	req.AddCookie(&http.Cookie{Name: CookieAccessToken, Value: accessToken})
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), string(errs.ErrTokenRevoked))
}

func TestOptionalAuthMiddleware_CanceledRequestDoesNotPoisonBlacklistBreaker(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tokenSvc := newTokenServiceForMiddlewareTest(t)
	accessToken := mustSignAccessToken(t, token.JWTTokenTypeAccess)

	r := gin.New()
	r.Use(OptionalAuthMiddleware(nil, tokenSvc, OptionalAuthConfig{}))
	r.GET("/public", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"backendFailure": c.GetBool(CtxKeyAuthBackendFailure),
			"userID":         GetUserID(c),
		})
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	req := httptest.NewRequest(http.MethodGet, "/public", nil).WithContext(ctx)
	req.AddCookie(&http.Cookie{Name: CookieAccessToken, Value: accessToken})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"backendFailure":false`)
	assert.Contains(t, w.Body.String(), `"userID":""`)
	assert.Equal(t, 0, tokenSvc.GetBlacklist().CircuitBreakerMetrics()["failures"])
}

func TestRequireHealthyOptionalAuth_RejectsBackendFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(CtxKeyAuthBackendFailure, true)
		c.Next()
	})
	r.GET("/public", RequireHealthyOptionalAuth(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/public", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Contains(t, w.Body.String(), string(errs.ErrServiceUnavailable))
	assert.NotContains(t, w.Body.String(), `"ok":true`)
}

func TestClearAuthCookies_PathContract(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	clearAuthCookies(c, OptionalAuthConfig{CookieDomain: "example.com", CookieSecure: true})

	cookies := w.Result().Cookies()
	require.Len(t, cookies, 4)
	paths := map[string]string{}
	for _, cookie := range cookies {
		paths[cookie.Name] = cookie.Path
	}
	assert.Equal(t, "/", paths[CookieAccessToken])
	assert.Equal(t, "/api/v1/auth", paths[CookieRefreshToken])
	assert.Equal(t, "/", paths[CookieSessionID])
	assert.Equal(t, "/", paths[CSRFCookieName])
}

func TestOptionalAuthConfigMatchesTokenConfig(t *testing.T) {
	cfg := config.TokenConfig{CookieDomain: "example.com", CookieSecure: true}
	opt := OptionalAuthConfig{CookieDomain: cfg.CookieDomain, CookieSecure: cfg.CookieSecure}
	assert.Equal(t, cfg.CookieDomain, opt.CookieDomain)
	assert.Equal(t, cfg.CookieSecure, opt.CookieSecure)
}

func reqContext() context.Context {
	return context.Background()
}
