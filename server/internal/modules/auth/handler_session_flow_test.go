package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/config"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/crypto"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/middleware"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/token"
	"git.stuhelper.com/StuHelper/StuHelper/internal/testutil/redisfixture"
)

type missingExternalUserRepo struct{ fakeUserSyncRepo }

func (missingExternalUserRepo) ExistsByExternalID(context.Context, string) (bool, error) {
	return false, nil
}

func newRefreshTestHandler(t *testing.T, repo UserSyncRepo) (*Handler, *token.Service) {
	t.Helper()
	require.NoError(t, crypto.InitHMACKey("test-auth-refresh-secret-32-bytes!!", false))

	fixture := redisfixture.Start(t)

	tokenSvc, err := token.NewService(token.ServiceConfig{RedisClient: fixture.Client, AccessTTL: 300, RefreshTTL: 600})
	require.NoError(t, err)
	t.Cleanup(tokenSvc.Close)

	tokenCfg := config.TokenConfig{AccessTokenTTL: 300, RefreshTokenTTL: 600}
	svc := NewService(tokenCfg, tokenSvc, repo)
	return &Handler{svc: svc, tokenService: tokenSvc, redisClient: fixture.Client, tokenConfig: tokenCfg}, tokenSvc
}

func marshalRefreshBody(t *testing.T, refreshToken string) *bytes.Buffer {
	t.Helper()
	body, err := json.Marshal(refreshTokenRequest{RefreshToken: refreshToken})
	require.NoError(t, err)
	return bytes.NewBuffer(body)
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
	assert.True(t, blacklisted)

	release()

	blacklisted, err = tokenSvc.GetBlacklist().IsBlacklisted(t.Context(), "refresh-token-1")
	require.NoError(t, err)
	assert.False(t, blacklisted)

	release2, ok := h.consumeRefreshToken(c, "refresh-token-1")
	require.True(t, ok)
	require.NotNil(t, release2)
}

func TestConsumeRefreshToken_Revoked(t *testing.T) {
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
	assert.Contains(t, w.Body.String(), "refresh token revoked")
}

func TestRefreshToken_SelfSignedSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, tokenSvc := newRefreshTestHandler(t, &fakeUserSyncRepo{})

	avatar := "https://cdn.example.com/avatar.png"
	phoneUser := &PhoneUser{
		ExternalID: "phone-user-1",
		Username:   "phone-user-1",
		Email:      "phone-user-1@example.com",
		AvatarURL:  &avatar,
	}
	sessionID := "sid-refresh-success"
	accessToken, refreshToken, err := h.svc.SignPhoneTokenPair(phoneUser, []string{"user"}, sessionID)
	require.NoError(t, err)
	_, err = h.svc.CreateSession(t.Context(), sessionID, phoneUser.ExternalID, accessToken, refreshToken, "phone", "ios")
	require.NoError(t, err)

	r := gin.New()
	r.POST("/refresh", h.RefreshToken)

	req := httptest.NewRequest(http.MethodPost, "/refresh", marshalRefreshBody(t, refreshToken))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Success bool `json:"success"`
		Data    struct {
			AccessToken  string `json:"accessToken"`
			RefreshToken string `json:"refreshToken"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.True(t, resp.Success)
	require.NotEmpty(t, resp.Data.AccessToken)
	require.NotEmpty(t, resp.Data.RefreshToken)

	newAccessClaims, err := token.VerifyJWTWithType(crypto.GetHMACKey(), resp.Data.AccessToken, token.JWTTokenTypeAccess)
	require.NoError(t, err)
	assert.Equal(t, phoneUser.Email, newAccessClaims.Email)
	assert.Equal(t, phoneUser.Username, newAccessClaims.DisplayName)
	assert.Equal(t, avatar, newAccessClaims.Avatar)

	newRefreshClaims, err := token.VerifyJWTWithType(crypto.GetHMACKey(), resp.Data.RefreshToken, token.JWTTokenTypeRefresh)
	require.NoError(t, err)
	assert.Equal(t, phoneUser.Email, newRefreshClaims.Email)
	assert.Equal(t, phoneUser.Username, newRefreshClaims.DisplayName)
	assert.Equal(t, avatar, newRefreshClaims.Avatar)

	blacklisted, err := tokenSvc.GetBlacklist().IsBlacklisted(t.Context(), refreshToken)
	require.NoError(t, err)
	assert.True(t, blacklisted)

	session, err := tokenSvc.GetSessionStore().Get(t.Context(), sessionID)
	require.NoError(t, err)
	require.NotNil(t, session)
	newRefreshHash, err := hashTokenForSession(resp.Data.RefreshToken)
	require.NoError(t, err)
	assert.Equal(t, newRefreshHash, session.RefreshTokenHash)
}

func TestRefreshToken_SelfSignedRejectsMissingUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, _ := newRefreshTestHandler(t, &missingExternalUserRepo{})

	phoneUser := &PhoneUser{ExternalID: "phone-user-missing", Username: "phone-user-missing"}
	sessionID := "sid-refresh-missing-user"
	_, refreshToken, err := h.svc.SignPhoneTokenPair(phoneUser, []string{"user"}, sessionID)
	require.NoError(t, err)

	r := gin.New()
	r.POST("/refresh", h.RefreshToken)

	req := httptest.NewRequest(http.MethodPost, "/refresh", marshalRefreshBody(t, refreshToken))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "failed to refresh token")
}

func TestRefreshSelfSignedToken_InvalidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, _ := newRefreshTestHandler(t, &fakeUserSyncRepo{})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/refresh", nil)

	ok := h.refreshSelfSignedToken(c, "not-a-jwt")
	assert.False(t, ok)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "invalid refresh token")
}

func TestRefreshToken_SelfSignedMissingSessionFailsRotation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, tokenSvc := newRefreshTestHandler(t, &fakeUserSyncRepo{})

	phoneUser := &PhoneUser{ExternalID: "phone-user-no-session", Username: "phone-user-no-session"}
	_, refreshToken, err := h.svc.SignPhoneTokenPair(phoneUser, []string{"user"}, "sid-missing-session")
	require.NoError(t, err)

	r := gin.New()
	r.POST("/refresh", h.RefreshToken)

	req := httptest.NewRequest(http.MethodPost, "/refresh", marshalRefreshBody(t, refreshToken))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "failed to refresh token")

	blacklisted, err := tokenSvc.GetBlacklist().IsBlacklisted(t.Context(), refreshToken)
	require.NoError(t, err)
	assert.True(t, blacklisted)
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

func TestLogoutAll_FailureBranch(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, _ := newRefreshTestHandler(t, &fakeUserSyncRepo{})

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(middleware.CtxKeyUserID, "user-logout-all")
		c.Set(middleware.CtxKeyUsername, "logout-all-tester")
		c.Next()
	})
	r.POST("/logout-all", h.LogoutAll)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	req := httptest.NewRequest(http.MethodPost, "/logout-all", nil).WithContext(ctx)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "failed to logout from all devices")
}

func TestExtractSessionIDBranches(t *testing.T) {
	require.NoError(t, crypto.InitHMACKey("test-auth-refresh-secret-32-bytes!!", false))

	assert.Empty(t, extractSessionID(""))
	assert.Empty(t, extractSessionID("plain-oidc-token"))

	accessToken, err := token.SignJWT(crypto.GetHMACKey(), token.JWTClaims{
		Sub:  "user-1",
		Name: "tester",
		Typ:  token.JWTTokenTypeAccess,
		Sid:  "sid-extract",
	}, time.Minute)
	require.NoError(t, err)
	assert.Equal(t, "sid-extract", extractSessionID(accessToken))
}
