package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/StuHelper/StuHelper/server/internal/pkg/config"
	"github.com/StuHelper/StuHelper/server/internal/pkg/crypto"
	"github.com/StuHelper/StuHelper/server/internal/pkg/errs"
	"github.com/StuHelper/StuHelper/server/internal/pkg/middleware"
	"github.com/StuHelper/StuHelper/server/internal/pkg/token"
	"github.com/StuHelper/StuHelper/server/internal/testutil/redisfixture"
)

func newRefreshTestHandler(t *testing.T, repo UserSyncRepo) (*Handler, *token.Service) {
	t.Helper()
	h, tokenSvc, _ := newRefreshTestHandlerWithFixture(t, repo)
	return h, tokenSvc
}

func newRefreshTestHandlerWithFixture(t *testing.T, repo UserSyncRepo) (*Handler, *token.Service, *redisfixture.Fixture) {
	t.Helper()
	require.NoError(t, crypto.InitHMACKey("test-auth-refresh-secret-32-bytes!!", false))

	fixture := redisfixture.Start(t)

	tokenSvc, err := token.NewService(token.ServiceConfig{RedisClient: fixture.Client, AccessTTL: 300, RefreshTTL: 7200})
	require.NoError(t, err)
	t.Cleanup(tokenSvc.Close)

	tokenCfg := config.TokenConfig{AccessTokenTTL: 300, RefreshTokenTTL: 7200}
	svc := NewService(tokenCfg, tokenSvc, repo)
	return &Handler{
		svc:              svc,
		tokenService:     tokenSvc,
		redisClient:      fixture.Client,
		tokenConfig:      tokenCfg,
		authFailureGuard: NewAuthFailureGuard(fixture.Client),
	}, tokenSvc, fixture
}

func marshalRefreshBody(t *testing.T, refreshToken string) *bytes.Buffer {
	t.Helper()
	body, err := json.Marshal(refreshTokenRequest{RefreshToken: refreshToken})
	require.NoError(t, err)
	return bytes.NewBuffer(body)
}

func assertNoIssuedTokenCookies(t *testing.T, recorder *httptest.ResponseRecorder) {
	t.Helper()
	for _, header := range recorder.Header().Values("Set-Cookie") {
		if strings.HasPrefix(header, middleware.CookieAccessToken+"=;") ||
			strings.HasPrefix(header, middleware.CookieRefreshToken+"=;") {
			continue
		}
		assert.False(t, strings.HasPrefix(header, middleware.CookieAccessToken+"="), header)
		assert.False(t, strings.HasPrefix(header, middleware.CookieRefreshToken+"="), header)
	}
}

func assertNoClearedTokenCookies(t *testing.T, recorder *httptest.ResponseRecorder) {
	t.Helper()
	for _, header := range recorder.Header().Values("Set-Cookie") {
		assert.False(t, strings.HasPrefix(header, middleware.CookieAccessToken+"=;"), header)
		assert.False(t, strings.HasPrefix(header, middleware.CookieRefreshToken+"=;"), header)
		assert.False(t, strings.HasPrefix(header, middleware.CSRFCookieName+"=;"), header)
		assert.False(t, strings.HasPrefix(header, sessionCookieName+"=;"), header)
	}
}

func TestConsumeRefreshToken_ReserveReleaseAndRetry(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, tokenSvc := newRefreshTestHandler(t, &fakeUserSyncRepo{})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/refresh", nil)

	release, ok := h.consumeRefreshToken(c, "refresh-token-1")
	require.True(t, ok)
	require.NotNil(t, release)

	blacklisted, err := tokenSvc.GetBlacklist().IsBlacklisted(t.Context(), "refresh-token-1")
	require.NoError(t, err)
	assert.False(t, blacklisted)

	release()

	blacklisted, err = tokenSvc.GetBlacklist().IsBlacklisted(t.Context(), "refresh-token-1")
	require.NoError(t, err)
	assert.False(t, blacklisted)

	release2, ok := h.consumeRefreshToken(c, "refresh-token-1")
	require.True(t, ok)
	require.NotNil(t, release2)
}

func TestConsumeRefreshToken_ReservationUsesShortTTL(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, tokenSvc, fixture := newRefreshTestHandlerWithFixture(t, &fakeUserSyncRepo{})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/refresh", nil)

	release, ok := h.consumeRefreshToken(c, "refresh-token-short-reservation")
	require.True(t, ok)
	require.NotNil(t, release)
	t.Cleanup(release)

	reservationKey := refreshReservationRedisKey(t, "refresh-token-short-reservation")
	ttl := fixture.Server.TTL(reservationKey)
	assert.Greater(t, ttl, time.Duration(0))
	assert.LessOrEqual(t, ttl, refreshReservationTTL)
	assert.Greater(t, ttl, refreshReservationTTL-5*time.Second)
	assert.Less(t, ttl, tokenSvc.GetRefreshTokenTTL())
}

func TestConsumeRefreshToken_ReleaseSurvivesRequestCancellation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, tokenSvc := newRefreshTestHandler(t, &fakeUserSyncRepo{})

	reqCtx, cancelReq := context.WithCancel(context.Background())
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/refresh", nil).WithContext(reqCtx)

	release, ok := h.consumeRefreshToken(c, "refresh-token-cancelled")
	require.True(t, ok)
	require.NotNil(t, release)

	cancelReq()
	release()

	blacklisted, err := tokenSvc.GetBlacklist().IsBlacklisted(t.Context(), "refresh-token-cancelled")
	require.NoError(t, err)
	assert.False(t, blacklisted)
}

func TestConsumeRefreshToken_InFlightReservationRejectedWithoutReuse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, _ := newRefreshTestHandler(t, &fakeUserSyncRepo{})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/refresh", nil)

	release, ok := h.consumeRefreshToken(c, "refresh-token-revoked")
	require.True(t, ok)
	require.NotNil(t, release)

	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/refresh", nil)
	release2, ok := h.consumeRefreshToken(c, "refresh-token-revoked")
	require.False(t, ok)
	assert.Nil(t, release2)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "refresh token rotation already in progress")
	assert.Contains(t, w.Body.String(), string(errs.ErrRefreshTokenInvalid))
}

func TestRefreshToken_InFlightReservationDoesNotRevokeSessionsOrClearCookies(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, tokenSvc := newRefreshTestHandler(t, &fakeUserSyncRepo{})

	_, err := h.svc.CreateSession(
		t.Context(),
		"sid-refresh-reserved",
		"user-refresh-reserved",
		"old-access-token",
		"reserved-refresh-token",
		"oidc-native",
		"ios",
	)
	require.NoError(t, err)

	consumed, err := tokenSvc.GetBlacklist().TryConsumeRefreshToken(t.Context(), "reserved-refresh-token", refreshReservationTTL)
	require.NoError(t, err)
	require.True(t, consumed)

	r := gin.New()
	r.POST("/refresh", h.RefreshToken)

	req := httptest.NewRequest(http.MethodPost, "/refresh", marshalRefreshBody(t, "reserved-refresh-token"))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(nativeSessionIDHeader, "sid-refresh-reserved")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusUnauthorized, w.Code, w.Body.String())
	assert.Contains(t, w.Body.String(), "refresh token rotation already in progress")
	assert.Contains(t, w.Body.String(), string(errs.ErrRefreshTokenInvalid))
	assertNoClearedTokenCookies(t, w)

	session, err := tokenSvc.GetSessionStore().Get(t.Context(), "sid-refresh-reserved")
	require.NoError(t, err)
	require.NotNil(t, session)
	assert.Equal(t, "user-refresh-reserved", session.UserID)

	blacklisted, err := tokenSvc.GetBlacklist().IsBlacklisted(t.Context(), "reserved-refresh-token")
	require.NoError(t, err)
	assert.False(t, blacklisted)
}

func TestRefreshToken_RejectsLegacyCSRFWithoutReservingRefreshToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, tokenSvc, fixture := newRefreshTestHandlerWithFixture(t, &fakeUserSyncRepo{})

	r := gin.New()
	r.POST("/refresh", h.RefreshToken)

	req := httptest.NewRequest(http.MethodPost, "/refresh", nil)
	req.AddCookie(&http.Cookie{Name: middleware.CookieRefreshToken, Value: "legacy-csrf-refresh-token"})
	req.AddCookie(&http.Cookie{Name: middleware.CSRFCookieName, Value: "legacy-csrf-token"})
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "sid-legacy-csrf"})
	req.Header.Set(middleware.CSRFHeaderName, "legacy-csrf-token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusForbidden, w.Code, w.Body.String())
	assert.Contains(t, w.Body.String(), string(errs.ErrCSRFTokenInvalid))
	assertNoClearedTokenCookies(t, w)

	blacklisted, err := tokenSvc.GetBlacklist().IsBlacklisted(t.Context(), "legacy-csrf-refresh-token")
	require.NoError(t, err)
	assert.False(t, blacklisted)
	assert.False(t, fixture.Server.Exists(refreshReservationRedisKey(t, "legacy-csrf-refresh-token")))
}

func TestRefreshToken_BlacklistedRefreshReuseRevokesAllSessions(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, tokenSvc := newRefreshTestHandler(t, &fakeUserSyncRepo{})

	_, err := h.svc.CreateSession(
		t.Context(),
		"sid-refresh-reuse-a",
		"user-refresh-reuse",
		"old-access-token",
		"reused-refresh-token",
		"oidc",
		"browser",
	)
	require.NoError(t, err)
	_, err = h.svc.CreateSession(
		t.Context(),
		"sid-refresh-reuse-b",
		"user-refresh-reuse",
		"other-access-token",
		"other-refresh-token",
		"oidc",
		"browser",
	)
	require.NoError(t, err)
	require.NoError(t, tokenSvc.GetBlacklist().Add(t.Context(), "reused-refresh-token", tokenSvc.GetRefreshTokenTTL()))

	r := gin.New()
	r.POST("/refresh", h.RefreshToken)

	csrfToken := mustGenerateCSRFTokenForSession(t, "sid-refresh-reuse-a")
	req := httptest.NewRequest(http.MethodPost, "/refresh", nil)
	req.AddCookie(&http.Cookie{Name: middleware.CookieRefreshToken, Value: "reused-refresh-token"})
	req.AddCookie(&http.Cookie{Name: middleware.CSRFCookieName, Value: csrfToken})
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "sid-refresh-reuse-a"})
	req.Header.Set(middleware.CSRFHeaderName, csrfToken)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusUnauthorized, w.Code, w.Body.String())
	assert.Contains(t, w.Body.String(), "refresh token reuse detected")
	assert.Contains(t, w.Body.String(), string(errs.ErrTokenRevoked))

	sessions, err := tokenSvc.GetSessionStore().ListUserSessions(t.Context(), "user-refresh-reuse")
	require.NoError(t, err)
	assert.Empty(t, sessions)
}

func TestRefreshToken_BlacklistedRefreshReuseRevokesAllSessionsAfterOldRefTTLWasNearExpiry(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, tokenSvc, fixture := newRefreshTestHandlerWithFixture(t, &fakeUserSyncRepo{})
	ctx := t.Context()
	oldRefresh := "old-refresh-near-expiry"

	_, err := h.svc.CreateSession(
		ctx,
		"sid-refresh-near-expiry-a",
		"user-refresh-near-expiry",
		"old-access-token",
		oldRefresh,
		"oidc",
		"browser",
	)
	require.NoError(t, err)
	_, err = h.svc.CreateSession(
		ctx,
		"sid-refresh-near-expiry-b",
		"user-refresh-near-expiry",
		"other-access-token",
		"other-refresh-token",
		"oidc",
		"browser",
	)
	require.NoError(t, err)

	oldRefreshHash, err := hashTokenForSession(oldRefresh)
	require.NoError(t, err)
	require.NoError(t, fixture.Client.PExpire(ctx, refreshTokenRefKeyForTest(oldRefreshHash), 50*time.Millisecond).Err())

	err = h.svc.RotateSession(
		ctx,
		"sid-refresh-near-expiry-a",
		"user-refresh-near-expiry",
		oldRefresh,
		"new-access-token",
		futureAccessTokenExpiryUnix(),
		"new-refresh-token",
	)
	require.NoError(t, err)
	time.Sleep(100 * time.Millisecond)

	r := gin.New()
	r.POST("/refresh", h.RefreshToken)

	csrfToken := mustGenerateCSRFTokenForSession(t, "sid-refresh-near-expiry-a")
	req := httptest.NewRequest(http.MethodPost, "/refresh", nil)
	req.AddCookie(&http.Cookie{Name: middleware.CookieRefreshToken, Value: oldRefresh})
	req.AddCookie(&http.Cookie{Name: middleware.CSRFCookieName, Value: csrfToken})
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "sid-refresh-near-expiry-a"})
	req.Header.Set(middleware.CSRFHeaderName, csrfToken)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusUnauthorized, w.Code, w.Body.String())
	assert.Contains(t, w.Body.String(), "refresh token reuse detected")
	assert.Contains(t, w.Body.String(), string(errs.ErrTokenRevoked))

	sessions, err := tokenSvc.GetSessionStore().ListUserSessions(ctx, "user-refresh-near-expiry")
	require.NoError(t, err)
	assert.Empty(t, sessions)
}

func TestRefreshToken_BlacklistedRefreshWithoutAttributionRemainsRevoked(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, tokenSvc, fixture := newRefreshTestHandlerWithFixture(t, &fakeUserSyncRepo{})
	ctx := t.Context()
	refreshToken := "legacy-revoked-refresh"

	_, err := h.svc.CreateSession(
		ctx,
		"sid-legacy-revoked-a",
		"user-legacy-revoked",
		"legacy-access-a",
		refreshToken,
		"oidc",
		"browser",
	)
	require.NoError(t, err)
	_, err = h.svc.CreateSession(
		ctx,
		"sid-legacy-revoked-b",
		"user-legacy-revoked",
		"legacy-access-b",
		"legacy-refresh-b",
		"oidc",
		"browser",
	)
	require.NoError(t, err)
	require.NoError(t, tokenSvc.GetBlacklist().Add(ctx, refreshToken, tokenSvc.GetRefreshTokenTTL()))

	refreshHash, err := hashTokenForSession(refreshToken)
	require.NoError(t, err)
	require.NoError(t, fixture.Client.Del(ctx, refreshTokenRefKeyForTest(refreshHash)).Err())

	r := gin.New()
	r.POST("/refresh", h.RefreshToken)

	csrfToken := mustGenerateCSRFTokenForSession(t, "sid-legacy-revoked-a")
	req := httptest.NewRequest(http.MethodPost, "/refresh", nil)
	req.AddCookie(&http.Cookie{Name: middleware.CookieRefreshToken, Value: refreshToken})
	req.AddCookie(&http.Cookie{Name: middleware.CSRFCookieName, Value: csrfToken})
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "sid-legacy-revoked-a"})
	req.Header.Set(middleware.CSRFHeaderName, csrfToken)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusUnauthorized, w.Code, w.Body.String())
	assert.Contains(t, w.Body.String(), "refresh token revoked")
	assert.NotContains(t, w.Body.String(), "refresh token reuse detected")

	sessions, err := tokenSvc.GetSessionStore().ListUserSessions(ctx, "user-legacy-revoked")
	require.NoError(t, err)
	require.Len(t, sessions, 2)
}

func TestRefreshToken_RejectsSelfSignedRefreshToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, tokenSvc := newRefreshTestHandler(t, &fakeUserSyncRepo{})
	refreshToken := mustSignRefreshToken(t, "legacy-phone-user", "sid-legacy-phone")

	r := gin.New()
	r.POST("/refresh", h.RefreshToken)

	req := httptest.NewRequest(http.MethodPost, "/refresh", marshalRefreshBody(t, refreshToken))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "unsupported refresh token")
	assertNoIssuedTokenCookies(t, w)

	blacklisted, err := tokenSvc.GetBlacklist().IsBlacklisted(t.Context(), refreshToken)
	require.NoError(t, err)
	assert.False(t, blacklisted)
}

func TestLogoutAll_RevokesAllSessions(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, tokenSvc := newRefreshTestHandler(t, &fakeUserSyncRepo{})

	_, err := h.svc.CreateSession(t.Context(), "sid-a", "user-logout-all", "access-a", "refresh-a", "oidc", "browser")
	require.NoError(t, err)
	_, err = h.svc.CreateSession(t.Context(), "sid-b", "user-logout-all", "access-b", "refresh-b", "oidc", "browser")
	require.NoError(t, err)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(middleware.CtxKeyUserID, "user-logout-all")
		c.Set(middleware.CtxKeyUsername, "logout-all-tester")
		c.Next()
	})
	r.POST("/logout-all", h.LogoutAll)

	req := httptest.NewRequest(http.MethodPost, "/logout-all", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	sessions, err := tokenSvc.GetSessionStore().ListUserSessions(t.Context(), "user-logout-all")
	require.NoError(t, err)
	assert.Empty(t, sessions)
}

func TestLogoutAll_UsesDetachedContextAfterRequestCancellation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, tokenSvc := newRefreshTestHandler(t, &fakeUserSyncRepo{})

	_, err := h.svc.CreateSession(t.Context(), "sid-canceled-a", "user-logout-all-canceled", "access-canceled-a", "refresh-canceled-a", "oidc", "browser")
	require.NoError(t, err)
	_, err = h.svc.CreateSession(t.Context(), "sid-canceled-b", "user-logout-all-canceled", "access-canceled-b", "refresh-canceled-b", "oidc", "browser")
	require.NoError(t, err)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(middleware.CtxKeyUserID, "user-logout-all-canceled")
		c.Set(middleware.CtxKeyUsername, "logout-all-tester")
		c.Next()
	})
	r.POST("/logout-all", h.LogoutAll)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	req := httptest.NewRequest(http.MethodPost, "/logout-all", nil).WithContext(ctx)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	sessions, err := tokenSvc.GetSessionStore().ListUserSessions(t.Context(), "user-logout-all-canceled")
	require.NoError(t, err)
	assert.Empty(t, sessions)
}

func mustSignRefreshToken(t *testing.T, userID, sessionID string) string {
	t.Helper()
	require.NoError(t, crypto.InitHMACKey("test-auth-refresh-secret-32-bytes!!", false))
	tok, err := token.SignJWT(crypto.GetHMACKey(), token.JWTClaims{
		Sub:  userID,
		Name: userID,
		Typ:  token.JWTTokenTypeRefresh,
		Sid:  sessionID,
	}, time.Minute)
	require.NoError(t, err)
	return tok
}

func refreshReservationRedisKey(t *testing.T, refreshToken string) string {
	t.Helper()
	hash, err := crypto.HMACHash(refreshToken)
	require.NoError(t, err)
	return "token:refresh:consumed:" + hash
}
