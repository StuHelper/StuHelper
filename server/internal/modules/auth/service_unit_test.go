package auth

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/config"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/crypto"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/token"
	"git.stuhelper.com/StuHelper/StuHelper/internal/testutil/redisfixture"
)

type fakeUserSyncRepo struct{}

func (f *fakeUserSyncRepo) UpsertUser(ctx context.Context, input UserSyncInput) error { return nil }
func (f *fakeUserSyncRepo) ExistsByCasdoorSubject(ctx context.Context, casdoorSubject string) (bool, error) {
	return casdoorSubject != "", nil
}

func newAuthServiceForTest(t *testing.T, opts ...ServiceOption) (*Service, *token.Service) {
	t.Helper()
	require.NoError(t, crypto.InitHMACKey("test-auth-service-secret-32-bytes!", false))

	fixture := redisfixture.Start(t)

	tokenSvc, err := token.NewService(token.ServiceConfig{RedisClient: fixture.Client, AccessTTL: 300, RefreshTTL: 600})
	require.NoError(t, err)
	t.Cleanup(tokenSvc.Close)

	tokenCfg := config.TokenConfig{AccessTokenTTL: 300, RefreshTokenTTL: 600}
	return NewService(tokenCfg, tokenSvc, &fakeUserSyncRepo{}, opts...), tokenSvc
}

func TestSyncOIDCUser_ForwardsRoles(t *testing.T) {
	repo := &recordingUserSyncRepo{}
	svc, _ := newAuthServiceForTest(t)
	svc.userSyncRepo = repo

	input := UserSyncInput{
		CasdoorSubject: "oidc-admin",
		Username:       "admin",
		Roles:          []string{"super_admin"},
	}
	require.NoError(t, svc.SyncOIDCUser(context.Background(), input))
	assert.Equal(t, []string{"super_admin"}, repo.upsertInput.Roles)
}

func TestNewService(t *testing.T) {
	t.Run("panics when required deps missing", func(t *testing.T) {
		assert.Panics(t, func() { NewService(config.TokenConfig{}, nil, nil) })
		assert.Panics(t, func() { NewService(config.TokenConfig{}, nil, &fakeUserSyncRepo{}) })
		fixture := redisfixture.Start(t)
		tokenSvc, err := token.NewService(token.ServiceConfig{RedisClient: fixture.Client, AccessTTL: 300, RefreshTTL: 600})
		require.NoError(t, err)
		defer tokenSvc.Close()
		assert.Panics(t, func() { NewService(config.TokenConfig{}, tokenSvc, nil) })
	})

	t.Run("constructs with valid deps", func(t *testing.T) {
		svc, _ := newAuthServiceForTest(t)
		require.NotNil(t, svc)
		assert.Equal(t, 300, svc.tokenConfig.AccessTokenTTL)
		assert.Equal(t, 600, svc.tokenConfig.RefreshTokenTTL)
	})

	t.Run("ignores nil options", func(t *testing.T) {
		svc, _ := newAuthServiceForTest(t, nil)
		require.NotNil(t, svc)
	})
}

func TestCreateSession(t *testing.T) {
	svc, tokenSvc := newAuthServiceForTest(t)
	ctx := context.Background()

	_, err := svc.CreateSession(ctx, "", "user-1", "access", "refresh", "oidc", "browser")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "sessionID is required")

	_, err = svc.CreateSession(ctx, "sid-1", "", "access", "refresh", "oidc", "browser")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "userID is required")

	info, err := svc.CreateSession(ctx, "sid-1", "user-1", "access-token", "refresh-token", "oidc", "browser")
	require.NoError(t, err)
	assert.Equal(t, "sid-1", info.SessionID)

	session, err := tokenSvc.GetSessionStore().Get(ctx, "sid-1")
	require.NoError(t, err)
	require.NotNil(t, session)
	assert.Equal(t, "user-1", session.UserID)
	assert.Equal(t, "oidc", session.LoginMethod)
	assert.Equal(t, "browser", session.DeviceInfo)
	assert.NotEmpty(t, session.AccessTokenHash)
	assert.NotEmpty(t, session.RefreshTokenHash)
}

func TestRotateSession_BlacklistsOldRefreshAndTouchesSession(t *testing.T) {
	svc, tokenSvc := newAuthServiceForTest(t)
	ctx := context.Background()

	_, err := svc.CreateSession(ctx, "sid-1", "user-1", "old-access", "old-refresh", "oidc", "browser")
	require.NoError(t, err)

	err = svc.RotateSession(ctx, "sid-1", "user-1", "old-refresh", "new-access", "new-refresh")
	require.NoError(t, err)

	blacklisted, err := tokenSvc.GetBlacklist().IsBlacklisted(ctx, "old-refresh")
	require.NoError(t, err)
	assert.True(t, blacklisted)

	session, err := tokenSvc.GetSessionStore().Get(ctx, "sid-1")
	require.NoError(t, err)
	require.NotNil(t, session)
	newAccessHash, err := hashTokenForSession("new-access")
	require.NoError(t, err)
	newRefreshHash, err := hashTokenForSession("new-refresh")
	require.NoError(t, err)
	assert.Equal(t, newAccessHash, session.AccessTokenHash)
	assert.Equal(t, newRefreshHash, session.RefreshTokenHash)
}

func TestRotateSession_RejectsTrackedSessionMismatch(t *testing.T) {
	svc, _ := newAuthServiceForTest(t)
	ctx := context.Background()

	_, err := svc.CreateSession(ctx, "sid-1", "user-1", "old-access", "old-refresh", "oidc", "browser")
	require.NoError(t, err)

	err = svc.RotateSession(ctx, "sid-1", "user-2", "old-refresh", "new-access", "new-refresh")
	require.Error(t, err)
	assert.ErrorIs(t, err, errSessionUserMismatch)

	err = svc.RotateSession(ctx, "sid-1", "user-1", "other-refresh", "new-access", "new-refresh")
	require.Error(t, err)
	assert.ErrorIs(t, err, errSessionRefreshTokenMismatch)
}

func TestRotateSession_WithoutSessionIDRejectsRequestAndBlacklistsOldRefresh(t *testing.T) {
	svc, tokenSvc := newAuthServiceForTest(t)
	ctx := context.Background()

	err := svc.RotateSession(ctx, "", "user-1", "old-refresh", "new-access", "new-refresh")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "sessionID is required")

	blacklisted, err := tokenSvc.GetBlacklist().IsBlacklisted(ctx, "old-refresh")
	require.NoError(t, err)
	assert.True(t, blacklisted)
}

func TestHashTokenForSession(t *testing.T) {
	hash, err := hashTokenForSession("sample-token")
	require.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.NotEqual(t, "sample-token", hash)
}

func TestRevokeSessionAndRevokeAll(t *testing.T) {
	svc, tokenSvc := newAuthServiceForTest(t)
	ctx := context.Background()
	_, err := svc.CreateSession(ctx, "sid-1", "user-1", "access-1", "refresh-1", "oidc", "browser")
	require.NoError(t, err)
	_, err = svc.CreateSession(ctx, "sid-2", "user-1", "access-2", "refresh-2", "oidc", "browser")
	require.NoError(t, err)

	err = svc.RevokeSession(ctx, "sid-1", "user-1", "access-1", "refresh-1")
	require.NoError(t, err)
	session, err := tokenSvc.GetSessionStore().Get(ctx, "sid-1")
	require.NoError(t, err)
	assert.Nil(t, session)

	err = svc.RevokeAllSessions(ctx, "user-1")
	require.NoError(t, err)
	sessions, err := tokenSvc.GetSessionStore().ListUserSessions(ctx, "user-1")
	require.NoError(t, err)
	assert.Empty(t, sessions)
}

func TestRevokeSession_RejectsTrackedSessionMismatch(t *testing.T) {
	svc, tokenSvc := newAuthServiceForTest(t)
	ctx := context.Background()

	_, err := svc.CreateSession(ctx, "sid-1", "user-1", "access-1", "refresh-1", "oidc", "browser")
	require.NoError(t, err)

	err = svc.RevokeSession(ctx, "sid-1", "user-2", "access-1", "refresh-1")
	require.Error(t, err)
	assert.ErrorIs(t, err, errSessionUserMismatch)

	session, err := tokenSvc.GetSessionStore().Get(ctx, "sid-1")
	require.NoError(t, err)
	require.NotNil(t, session)

	err = svc.RevokeSession(ctx, "sid-1", "user-1", "access-other", "refresh-1")
	require.Error(t, err)
	assert.ErrorIs(t, err, errSessionAccessTokenMismatch)
}

func TestRotateSession_ContextCanceled(t *testing.T) {
	svc, _ := newAuthServiceForTest(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := svc.RotateSession(ctx, "sid-canceled", "user-1", "old-refresh", "new-access", "new-refresh")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "blacklist old refresh token")
}

func TestRevokeSession_FallbackBlacklistFailureWithoutSessionID(t *testing.T) {
	svc, _ := newAuthServiceForTest(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := svc.RevokeSession(ctx, "", "user-1", "access-fallback", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "revoke access token")
}

func TestRevokeAllSessions_ContextCanceled(t *testing.T) {
	svc, _ := newAuthServiceForTest(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := svc.RevokeAllSessions(ctx, "user-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "revoke all sessions")
}

func TestCreateSession_StoresFreshTimestamps(t *testing.T) {
	svc, tokenSvc := newAuthServiceForTest(t)
	ctx := context.Background()
	before := time.Now().Unix()
	_, err := svc.CreateSession(ctx, "sid-time", "user-1", "access-time", "refresh-time", "oidc", "browser")
	require.NoError(t, err)
	after := time.Now().Unix()

	session, err := tokenSvc.GetSessionStore().Get(ctx, "sid-time")
	require.NoError(t, err)
	require.NotNil(t, session)
	assert.GreaterOrEqual(t, session.CreatedAt, before)
	assert.LessOrEqual(t, session.CreatedAt, after)
	assert.Equal(t, session.CreatedAt, session.LastActiveAt)
}
