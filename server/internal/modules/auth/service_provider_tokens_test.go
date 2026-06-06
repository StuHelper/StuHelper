package auth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/config"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/crypto/pii"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/oidc"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/token"
	"git.stuhelper.com/StuHelper/StuHelper/internal/testutil/redisfixture"
)

type fakeProviderRefreshRevoker struct {
	tokens []string
	err    error
	ctxErr error
}

func (f *fakeProviderRefreshRevoker) RevokeRefreshToken(ctx context.Context, refreshToken string) error {
	f.ctxErr = ctx.Err()
	f.tokens = append(f.tokens, refreshToken)
	return f.err
}

type providerRefreshRevokerFunc func(ctx context.Context, refreshToken string) error

func (f providerRefreshRevokerFunc) RevokeRefreshToken(ctx context.Context, refreshToken string) error {
	return f(ctx, refreshToken)
}

type providerRefreshTokenRevokeCall struct {
	appKey       string
	refreshToken string
}

type fakeApplicationProviderRefreshRevoker struct {
	calls  []providerRefreshTokenRevokeCall
	err    error
	ctxErr error
}

func (f *fakeApplicationProviderRefreshRevoker) RevokeRefreshToken(ctx context.Context, refreshToken string) error {
	f.ctxErr = ctx.Err()
	f.calls = append(f.calls, providerRefreshTokenRevokeCall{
		appKey:       oidc.ApplicationWeb,
		refreshToken: refreshToken,
	})
	return f.err
}

func (f *fakeApplicationProviderRefreshRevoker) RevokeRefreshTokenForApplication(ctx context.Context, appKey, refreshToken string) error {
	f.ctxErr = ctx.Err()
	f.calls = append(f.calls, providerRefreshTokenRevokeCall{
		appKey:       appKey,
		refreshToken: refreshToken,
	})
	return f.err
}

func newAuthServiceWithProviderRevoker(
	t *testing.T,
	revoker ProviderRefreshTokenRevoker,
) (*Service, *token.Service) {
	t.Helper()
	fixture := redisfixture.Start(t)
	tokenSvc, err := token.NewService(token.ServiceConfig{RedisClient: fixture.Client, AccessTTL: 300, RefreshTTL: 600})
	require.NoError(t, err)
	t.Cleanup(tokenSvc.Close)

	cipher, err := pii.NewCipher(1, map[uint8][]byte{1: []byte("0123456789abcdef0123456789abcdef")})
	require.NoError(t, err)
	tokenCfg := config.TokenConfig{AccessTokenTTL: 300, RefreshTokenTTL: 600}
	svc := NewService(tokenCfg, tokenSvc, &fakeUserSyncRepo{}, WithProviderRefreshTokenRevocation(revoker, cipher))
	return svc, tokenSvc
}

func TestProviderRefreshTokenRevokedOnSessionLifecycle(t *testing.T) {
	revoker := &fakeProviderRefreshRevoker{}
	svc, tokenSvc := newAuthServiceWithProviderRevoker(t, revoker)

	_, err := svc.CreateSession(t.Context(), "sid-oidc", "user-1", "old-access", "old-refresh", "oidc", "browser")
	require.NoError(t, err)
	session := requireSession(t, tokenSvc, "sid-oidc")
	assert.NotEmpty(t, session.ProviderRefreshTokenEnc)
	assert.NotContains(t, session.ProviderRefreshTokenEnc, "old-refresh")

	err = svc.RotateSession(t.Context(), "sid-oidc", "user-1", "old-refresh", "new-access", "new-refresh")
	require.NoError(t, err)
	assert.Equal(t, []string{"old-refresh"}, revoker.tokens)

	err = svc.RevokeSession(t.Context(), "sid-oidc", "user-1", "new-access", "new-refresh")
	require.NoError(t, err)
	assert.Equal(t, []string{"old-refresh", "new-refresh"}, revoker.tokens)
	assert.Nil(t, requireSession(t, tokenSvc, "sid-oidc"))
}

func TestProviderRefreshTokenRevokedAfterSessionTouch(t *testing.T) {
	var tokenSvc *token.Service
	revoker := providerRefreshRevokerFunc(func(ctx context.Context, refreshToken string) error {
		assert.Equal(t, "old-refresh", refreshToken)
		session := requireSession(t, tokenSvc, "sid-provider-order")
		newRefreshHash, err := hashTokenForSession("new-refresh")
		require.NoError(t, err)
		assert.Equal(t, newRefreshHash, session.RefreshTokenHash)
		return nil
	})
	svc, tokenSvc := newAuthServiceWithProviderRevoker(t, revoker)

	_, err := svc.CreateSession(t.Context(), "sid-provider-order", "user-1", "old-access", "old-refresh", "oidc", "browser")
	require.NoError(t, err)

	err = svc.RotateSession(t.Context(), "sid-provider-order", "user-1", "old-refresh", "new-access", "new-refresh")
	require.NoError(t, err)
}

func TestProviderRefreshTokenNotRevokedWhenRotationValidationFails(t *testing.T) {
	revoker := &fakeProviderRefreshRevoker{}
	svc, _ := newAuthServiceWithProviderRevoker(t, revoker)

	_, err := svc.CreateSession(t.Context(), "sid-oidc-mismatch", "user-1", "old-access", "old-refresh", "oidc", "browser")
	require.NoError(t, err)

	err = svc.RotateSession(t.Context(), "sid-oidc-mismatch", "user-1", "other-refresh", "new-access", "new-refresh")
	require.ErrorIs(t, err, errSessionRefreshTokenMismatch)
	assert.Empty(t, revoker.tokens)
}

func TestProviderRefreshTokenInputsAreNormalized(t *testing.T) {
	revoker := &fakeApplicationProviderRefreshRevoker{}
	svc, tokenSvc := newAuthServiceWithProviderRevoker(t, revoker)

	_, err := svc.CreateSessionForApplication(
		t.Context(),
		"sid-provider-trim",
		"user-1",
		"access-token",
		" \tprovider-refresh\n ",
		"oidc-native",
		" \t"+oidc.ApplicationUniapp+"\n ",
		"ios",
	)
	require.NoError(t, err)

	session := requireSession(t, tokenSvc, "sid-provider-trim")
	require.NotEmpty(t, session.ProviderRefreshTokenEnc)
	assert.Equal(t, oidc.ApplicationUniapp, session.ProviderAppKey)

	rawProviderRefresh, err := svc.decryptProviderRefreshToken(session.ProviderRefreshTokenEnc)
	require.NoError(t, err)
	assert.Equal(t, "provider-refresh", rawProviderRefresh)

	appKey, err := svc.OIDCApplicationForRefresh(t.Context(), "sid-provider-trim", " \tprovider-refresh\n ")
	require.NoError(t, err)
	assert.Equal(t, oidc.ApplicationUniapp, appKey)

	err = svc.RevokeSession(t.Context(), "sid-provider-trim", "user-1", "access-token", " \tprovider-refresh\n ")
	require.NoError(t, err)
	assert.Equal(t, []providerRefreshTokenRevokeCall{
		{appKey: oidc.ApplicationUniapp, refreshToken: "provider-refresh"},
	}, revoker.calls)
}

func TestProviderRefreshTokenWhitespaceOnlySkipped(t *testing.T) {
	revoker := &fakeApplicationProviderRefreshRevoker{}
	svc, tokenSvc := newAuthServiceWithProviderRevoker(t, revoker)

	_, err := svc.CreateSessionForApplication(
		t.Context(),
		"sid-provider-blank",
		"user-1",
		"access-token",
		" \t\n ",
		"oidc-native",
		" \t\n ",
		"ios",
	)
	require.NoError(t, err)

	session := requireSession(t, tokenSvc, "sid-provider-blank")
	assert.Empty(t, session.ProviderRefreshTokenEnc)
	assert.Equal(t, oidc.ApplicationUniapp, session.ProviderAppKey)

	err = svc.RevokeSession(t.Context(), "sid-provider-blank", "user-1", "access-token", " \t\n ")
	require.NoError(t, err)
	assert.Empty(t, revoker.calls)
}

func TestRevokeRawProviderRefreshTokenNormalizesInputs(t *testing.T) {
	revoker := &fakeApplicationProviderRefreshRevoker{}
	svc, _ := newAuthServiceWithProviderRevoker(t, revoker)

	err := svc.revokeRawProviderRefreshToken(t.Context(), " \t"+oidc.ApplicationUniapp+"\n ", " \tprovider-refresh\n ")
	require.NoError(t, err)
	assert.Equal(t, []providerRefreshTokenRevokeCall{
		{appKey: oidc.ApplicationUniapp, refreshToken: "provider-refresh"},
	}, revoker.calls)

	err = svc.revokeRawProviderRefreshToken(t.Context(), oidc.ApplicationWeb, " \t\n ")
	require.NoError(t, err)
	assert.Len(t, revoker.calls, 1)
}

func TestRevokeAllSessionsRevokesProviderRefreshTokens(t *testing.T) {
	revoker := &fakeProviderRefreshRevoker{}
	svc, tokenSvc := newAuthServiceWithProviderRevoker(t, revoker)

	createTrackedSession(t, svc, trackedSessionSeed{
		SessionID: "sid-a", UserID: "user-1", RefreshToken: "provider-refresh-a", LoginMethod: "oidc",
	})
	createTrackedSession(t, svc, trackedSessionSeed{
		SessionID: "sid-b", UserID: "user-1", RefreshToken: "provider-refresh-b", LoginMethod: "oidc-native",
	})
	createTrackedSession(t, svc, trackedSessionSeed{
		SessionID: "sid-phone", UserID: "user-1", RefreshToken: "self-refresh", LoginMethod: "phone",
	})

	err := svc.RevokeAllSessions(t.Context(), "user-1")

	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"provider-refresh-a", "provider-refresh-b"}, revoker.tokens)
	sessions, err := tokenSvc.GetSessionStore().ListUserSessions(t.Context(), "user-1")
	require.NoError(t, err)
	assert.Empty(t, sessions)
}

func TestRevokeAllSessionsRevokesLocalSessionsWhenProviderRevokeFails(t *testing.T) {
	revoker := &fakeProviderRefreshRevoker{err: errors.New("provider revoke failed")}
	svc, tokenSvc := newAuthServiceWithProviderRevoker(t, revoker)
	createTrackedSession(t, svc, trackedSessionSeed{
		SessionID: "sid-a", UserID: "user-1", RefreshToken: "provider-refresh-a", LoginMethod: "oidc",
	})

	err := svc.RevokeAllSessions(t.Context(), "user-1")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "provider revoke failed")
	assert.Nil(t, requireSession(t, tokenSvc, "sid-a"))
}

func TestRevokeSessionRevokesLocalSessionWhenProviderRevokeFails(t *testing.T) {
	revoker := &fakeProviderRefreshRevoker{err: errors.New("provider revoke failed")}
	svc, tokenSvc := newAuthServiceWithProviderRevoker(t, revoker)
	createTrackedSession(t, svc, trackedSessionSeed{
		SessionID: "sid-a", UserID: "user-1", RefreshToken: "provider-refresh-a", LoginMethod: "oidc",
	})

	err := svc.RevokeSession(t.Context(), "sid-a", "user-1", "access-sid-a", "provider-refresh-a")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "provider revoke failed")
	assert.Nil(t, requireSession(t, tokenSvc, "sid-a"))
}

func TestRotateOIDCSessionRevokesNewProviderRefreshOnLocalFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	revoker := &fakeProviderRefreshRevoker{}
	svc, _ := newAuthServiceWithProviderRevoker(t, revoker)
	createTrackedSession(t, svc, trackedSessionSeed{
		SessionID: "sid-local-failure", UserID: "other-user", RefreshToken: "old-provider-refresh", LoginMethod: "oidc",
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
	h := &Handler{svc: svc}

	ok := h.rotateOIDCSession(c, oidcSessionRotation{
		sessionID:       "sid-local-failure",
		appKey:          oidc.ApplicationWeb,
		userID:          "oidc-user-1",
		oldRefreshToken: "old-provider-refresh",
		payload: oidcRefreshPayload{
			rawIDToken:   "new-id-token",
			refreshToken: "new-provider-refresh",
			userID:       "oidc-user-1",
		},
	})

	require.False(t, ok)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Equal(t, []string{"new-provider-refresh"}, revoker.tokens)
	assertNoIssuedTokenCookies(t, w)
}

func TestRotateOIDCSessionProviderRefreshCompensationSurvivesRequestCancellation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	revoker := &fakeProviderRefreshRevoker{}
	svc, _ := newAuthServiceWithProviderRevoker(t, revoker)
	createTrackedSession(t, svc, trackedSessionSeed{
		SessionID: "sid-canceled-rotation", UserID: "oidc-user-1", RefreshToken: "old-provider-refresh", LoginMethod: "oidc",
	})

	reqCtx, cancelReq := context.WithCancel(context.Background())
	cancelReq()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil).WithContext(reqCtx)
	h := &Handler{svc: svc}

	ok := h.rotateOIDCSession(c, oidcSessionRotation{
		sessionID:       "sid-canceled-rotation",
		appKey:          oidc.ApplicationWeb,
		userID:          "oidc-user-1",
		oldRefreshToken: "old-provider-refresh",
		payload: oidcRefreshPayload{
			rawIDToken:   "new-id-token",
			refreshToken: "new-provider-refresh",
			userID:       "oidc-user-1",
		},
	})

	require.False(t, ok)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, []string{"new-provider-refresh"}, revoker.tokens)
	assert.NoError(t, revoker.ctxErr)
}

type trackedSessionSeed struct {
	SessionID    string
	UserID       string
	RefreshToken string
	LoginMethod  string
}

func createTrackedSession(t *testing.T, svc *Service, seed trackedSessionSeed) {
	t.Helper()
	_, err := svc.CreateSession(
		context.Background(),
		seed.SessionID,
		seed.UserID,
		"access-"+seed.SessionID,
		seed.RefreshToken,
		seed.LoginMethod,
		"test-device",
	)
	require.NoError(t, err)
}

func requireSession(t *testing.T, tokenSvc *token.Service, sessionID string) *token.SessionData {
	t.Helper()
	session, err := tokenSvc.GetSessionStore().Get(t.Context(), sessionID)
	require.NoError(t, err)
	return session
}
