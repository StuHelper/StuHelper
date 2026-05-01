package auth

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/config"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/crypto/pii"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/token"
	"git.stuhelper.com/StuHelper/StuHelper/internal/testutil/redisfixture"
)

type fakeProviderRefreshRevoker struct {
	tokens []string
	err    error
}

func (f *fakeProviderRefreshRevoker) RevokeRefreshToken(_ context.Context, refreshToken string) error {
	f.tokens = append(f.tokens, refreshToken)
	return f.err
}

func newAuthServiceWithProviderRevoker(
	t *testing.T,
	revoker *fakeProviderRefreshRevoker,
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

func TestRevokeAllSessionsKeepsLocalSessionsWhenProviderRevokeFails(t *testing.T) {
	revoker := &fakeProviderRefreshRevoker{err: errors.New("provider revoke failed")}
	svc, tokenSvc := newAuthServiceWithProviderRevoker(t, revoker)
	createTrackedSession(t, svc, trackedSessionSeed{
		SessionID: "sid-a", UserID: "user-1", RefreshToken: "provider-refresh-a", LoginMethod: "oidc",
	})

	err := svc.RevokeAllSessions(t.Context(), "user-1")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "provider revoke failed")
	assert.NotNil(t, requireSession(t, tokenSvc, "sid-a"))
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
