package auth

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/config"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/crypto"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/errs"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/middleware"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/token"
)

func TestResolveRefreshToken_UsesSingleCredentialSource(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/refresh", bytes.NewBufferString(`{"refreshToken":"body-token"}`))
	c.Request.Header.Set("Content-Type", "application/json")

	refreshToken, fromBody, ok := resolveRefreshToken(c)
	require.True(t, ok)
	assert.Equal(t, "body-token", refreshToken)
	assert.True(t, fromBody)

	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/refresh", nil)
	c.Request.AddCookie(&http.Cookie{Name: middleware.CookieRefreshToken, Value: "cookie-token"})

	refreshToken, fromBody, ok = resolveRefreshToken(c)
	require.True(t, ok)
	assert.Equal(t, "cookie-token", refreshToken)
	assert.False(t, fromBody)
}

func TestResolveRefreshToken_RejectsAmbiguousOrMalformedBody(t *testing.T) {
	tests := []struct {
		name   string
		body   string
		cookie *http.Cookie
	}{
		{
			name:   "body and refresh cookie",
			body:   `{"refreshToken":"body-token"}`,
			cookie: &http.Cookie{Name: middleware.CookieRefreshToken, Value: "cookie-token"},
		},
		{
			name:   "body and session cookie",
			body:   `{"refreshToken":"body-token"}`,
			cookie: &http.Cookie{Name: sessionCookieName, Value: "sid-cookie"},
		},
		{
			name: "malformed body",
			body: `{`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodPost, "/refresh", bytes.NewBufferString(tt.body))
			c.Request.Header.Set("Content-Type", "application/json")
			if tt.cookie != nil {
				c.Request.AddCookie(tt.cookie)
			}

			refreshToken, fromBody, ok := resolveRefreshToken(c)

			require.False(t, ok)
			assert.Empty(t, refreshToken)
			assert.False(t, fromBody)
			assert.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
			assert.Contains(t, w.Body.String(), string(errs.ErrInvalidParam))
		})
	}
}

func TestBuildRefreshResponse_NativeAndWeb(t *testing.T) {
	h := &Handler{}
	h.tokenConfig.AccessTokenTTL = 3600

	resp := h.buildRefreshResponse("access", "refresh", false)
	assert.NotContains(t, resp, "accessToken")
	assert.NotContains(t, resp, "refreshToken")

	resp = h.buildRefreshResponse("access", "refresh", true)
	assert.Equal(t, "access", resp["accessToken"])
	assert.Equal(t, "refresh", resp["refreshToken"])
}

func TestRefreshToken_ValidationBranches(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := &Handler{}

	r := gin.New()
	r.POST("/refresh", h.RefreshToken)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/refresh", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "missing refresh token")

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/refresh", nil)
	req.AddCookie(&http.Cookie{Name: middleware.CookieRefreshToken, Value: "cookie-refresh"})
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), "csrf token missing")
}

func TestRefreshToken_RejectsRevokedCookieRefresh(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, tokenSvc := newRefreshTestHandler(t, &fakeUserSyncRepo{})

	require.NoError(t, tokenSvc.GetBlacklist().Add(t.Context(), "revoked-cookie-refresh", tokenSvc.GetRefreshTokenTTL()))

	r := gin.New()
	r.POST("/refresh", h.RefreshToken)

	req := httptest.NewRequest(http.MethodPost, "/refresh", nil)
	req.AddCookie(&http.Cookie{Name: middleware.CookieRefreshToken, Value: "revoked-cookie-refresh"})
	req.AddCookie(&http.Cookie{Name: middleware.CSRFCookieName, Value: "csrf-123"})
	req.Header.Set(middleware.CSRFHeaderName, "csrf-123")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), string(errs.ErrTokenRevoked))
}

func TestRefreshToken_RejectsInvalidCSRFMismatch(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := &Handler{}

	r := gin.New()
	r.POST("/refresh", h.RefreshToken)

	req := httptest.NewRequest(http.MethodPost, "/refresh", nil)
	req.AddCookie(&http.Cookie{Name: middleware.CookieRefreshToken, Value: "cookie-refresh"})
	req.AddCookie(&http.Cookie{Name: middleware.CSRFCookieName, Value: "csrf-cookie"})
	req.Header.Set(middleware.CSRFHeaderName, "csrf-header")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), string(errs.ErrCSRFTokenInvalid))
}

func TestRefreshToken_RejectsAmbiguousOrRepeatedSessionIDSource(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, tokenSvc := newRefreshTestHandler(t, &fakeUserSyncRepo{})

	r := gin.New()
	r.POST("/refresh", h.RefreshToken)

	t.Run("repeated native session header", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/refresh", bytes.NewBufferString(`{"refreshToken":"refresh-repeated-header"}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Add(nativeSessionIDHeader, "sid-a")
		req.Header.Add(nativeSessionIDHeader, "sid-b")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
		assert.Contains(t, w.Body.String(), string(errs.ErrInvalidParam))

		blacklisted, err := tokenSvc.GetBlacklist().IsBlacklisted(t.Context(), "refresh-repeated-header")
		require.NoError(t, err)
		assert.False(t, blacklisted)
	})

	t.Run("native session header and browser session cookie", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/refresh", nil)
		req.AddCookie(&http.Cookie{Name: middleware.CookieRefreshToken, Value: "refresh-ambiguous-session"})
		req.AddCookie(&http.Cookie{Name: middleware.CSRFCookieName, Value: "csrf-123"})
		req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "sid-cookie"})
		req.Header.Set(middleware.CSRFHeaderName, "csrf-123")
		req.Header.Set(nativeSessionIDHeader, "sid-header")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
		assert.Contains(t, w.Body.String(), string(errs.ErrInvalidParam))

		blacklisted, err := tokenSvc.GetBlacklist().IsBlacklisted(t.Context(), "refresh-ambiguous-session")
		require.NoError(t, err)
		assert.False(t, blacklisted)
	})
}

func TestLogout_SuccessAndFailureBranches(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, tokenSvc := newRefreshTestHandler(t, &fakeUserSyncRepo{})
	h.tokenConfig = config.TokenConfig{AccessTokenTTL: 300, RefreshTokenTTL: 600}

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(middleware.CtxKeyUserID, "user-logout")
		c.Set(middleware.CtxKeyUsername, "logout-user")
		c.Set(middleware.CtxKeyRequestID, "req-logout")
		c.Next()
	})
	r.POST("/logout", h.Logout)

	accessToken := mustSignAccessTokenForHandler(t, "user-logout", "sid-logout")
	_, err := h.svc.CreateSession(t.Context(), "sid-logout", "user-logout", accessToken, "refresh-logout", "oidc", "browser")
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/logout", nil)
	req.AddCookie(&http.Cookie{Name: middleware.CookieRefreshToken, Value: "refresh-logout"})
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "sid-logout"})
	req.Header.Set("Authorization", "Bearer "+accessToken)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	session, err := tokenSvc.GetSessionStore().Get(t.Context(), "sid-logout")
	require.NoError(t, err)
	assert.Nil(t, session)

	req = httptest.NewRequest(http.MethodPost, "/logout", nil).WithContext(canceledContext())
	req.AddCookie(&http.Cookie{Name: middleware.CookieRefreshToken, Value: "refresh-logout-fail"})
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "sid-logout"})
	req.Header.Set("Authorization", "Bearer "+accessToken)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "logout partially failed")
}

func TestLogout_UsesNativeSessionHeaderForOIDCSession(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, tokenSvc := newRefreshTestHandler(t, &fakeUserSyncRepo{})

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(middleware.CtxKeyUserID, "user-native-logout")
		c.Set(middleware.CtxKeyUsername, "native-logout-user")
		c.Set(middleware.CtxKeyRequestID, "req-native-logout")
		c.Next()
	})
	r.POST("/logout", h.Logout)

	_, err := h.svc.CreateSession(t.Context(), "sid-native-logout", "user-native-logout", "oidc-access-token", "oidc-refresh-token", "oidc-native", "ios")
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/logout", nil)
	req.AddCookie(&http.Cookie{Name: middleware.CookieRefreshToken, Value: "oidc-refresh-token"})
	req.Header.Set("Authorization", "Bearer oidc-access-token")
	req.Header.Set(nativeSessionIDHeader, "sid-native-logout")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	session, err := tokenSvc.GetSessionStore().Get(t.Context(), "sid-native-logout")
	require.NoError(t, err)
	assert.Nil(t, session)
}

func TestLogout_RejectsAmbiguousSessionIDSource(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, tokenSvc := newRefreshTestHandler(t, &fakeUserSyncRepo{})

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(middleware.CtxKeyUserID, "user-native-logout")
		c.Set(middleware.CtxKeyUsername, "native-logout-user")
		c.Set(middleware.CtxKeyRequestID, "req-native-logout")
		c.Next()
	})
	r.POST("/logout", h.Logout)

	_, err := h.svc.CreateSession(t.Context(), "sid-native-logout", "user-native-logout", "oidc-access-token", "oidc-refresh-token", "oidc-native", "ios")
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/logout", nil)
	req.Header.Set("Authorization", "Bearer oidc-access-token")
	req.Header.Set(nativeSessionIDHeader, "sid-native-logout")
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "sid-cookie"})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
	assert.Contains(t, w.Body.String(), string(errs.ErrInvalidParam))

	session, err := tokenSvc.GetSessionStore().Get(t.Context(), "sid-native-logout")
	require.NoError(t, err)
	require.NotNil(t, session)
}

func TestLogout_NativeOIDCRequiresTrackedSession(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, tokenSvc := newRefreshTestHandler(t, &fakeUserSyncRepo{})

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(middleware.CtxKeyUserID, "user-native-logout")
		c.Set(middleware.CtxKeyUsername, "native-logout-user")
		c.Set(middleware.CtxKeyRequestID, "req-native-logout")
		c.Next()
	})
	r.POST("/logout", h.Logout)

	_, err := h.svc.CreateSession(t.Context(), "sid-native-logout", "user-native-logout", "oidc-access-token", "oidc-refresh-token", "oidc-native", "ios")
	require.NoError(t, err)

	t.Run("missing session header is rejected", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/logout", nil)
		req.Header.Set("Authorization", "Bearer oidc-access-token")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		require.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "missing native session id")

		session, getErr := tokenSvc.GetSessionStore().Get(t.Context(), "sid-native-logout")
		require.NoError(t, getErr)
		require.NotNil(t, session)
	})

	t.Run("unknown session header is rejected", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/logout", nil)
		req.Header.Set("Authorization", "Bearer oidc-access-token")
		req.Header.Set(nativeSessionIDHeader, "sid-unknown")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		require.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "invalid native session id")

		session, getErr := tokenSvc.GetSessionStore().Get(t.Context(), "sid-native-logout")
		require.NoError(t, getErr)
		require.NotNil(t, session)
	})

	t.Run("session owned by another user is rejected", func(t *testing.T) {
		_, createErr := h.svc.CreateSession(
			t.Context(),
			"sid-native-logout-foreign",
			"other-user",
			"other-access-token",
			"other-refresh-token",
			"oidc-native",
			"ios",
		)
		require.NoError(t, createErr)

		req := httptest.NewRequest(http.MethodPost, "/logout", nil)
		req.Header.Set("Authorization", "Bearer oidc-access-token")
		req.Header.Set(nativeSessionIDHeader, "sid-native-logout-foreign")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		require.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "invalid native session id")
	})
}

func mustSignAccessTokenForHandler(t *testing.T, userID, sessionID string) string {
	t.Helper()
	require.NoError(t, crypto.InitHMACKey("test-auth-refresh-secret-32-bytes!!", false))
	tok, err := token.SignJWT(crypto.GetHMACKey(), token.JWTClaims{
		Sub:         userID,
		Name:        userID,
		DisplayName: userID,
		Roles:       []string{"user"},
		Typ:         token.JWTTokenTypeAccess,
		Sid:         sessionID,
	}, time.Minute)
	require.NoError(t, err)
	return tok
}

func canceledContext() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}
