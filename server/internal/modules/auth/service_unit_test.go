package auth

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/StuHelper/StuHelper/server/internal/pkg/config"
	"github.com/StuHelper/StuHelper/server/internal/pkg/crypto"
	"github.com/StuHelper/StuHelper/server/internal/pkg/token"
	"github.com/StuHelper/StuHelper/server/internal/testutil/redisfixture"
)

type fakeUserSyncRepo struct{}

func (f *fakeUserSyncRepo) UpsertUser(ctx context.Context, input UserSyncInput) error { return nil }
func (f *fakeUserSyncRepo) ExistsByCasdoorSubject(ctx context.Context, casdoorSubject string) (bool, error) {
	return casdoorSubject != "", nil
}

func newAuthServiceForTest(t *testing.T, opts ...ServiceOption) (*Service, *token.Service) {
	t.Helper()
	svc, tokenSvc, _ := newAuthServiceForTestWithFixture(t, opts...)
	return svc, tokenSvc
}

func newAuthServiceForTestWithFixture(t *testing.T, opts ...ServiceOption) (*Service, *token.Service, *redisfixture.Fixture) {
	t.Helper()
	require.NoError(t, crypto.InitHMACKey("test-auth-service-secret-32-bytes!", false))

	fixture := redisfixture.Start(t)

	tokenSvc, err := token.NewService(token.ServiceConfig{RedisClient: fixture.Client, AccessTTL: 300, RefreshTTL: 600})
	require.NoError(t, err)
	t.Cleanup(tokenSvc.Close)

	tokenCfg := config.TokenConfig{AccessTokenTTL: 300, RefreshTokenTTL: 600}
	return NewService(tokenCfg, tokenSvc, &fakeUserSyncRepo{}, opts...), tokenSvc, fixture
}

func refreshTokenRefKeyForTest(refreshHash string) string {
	return "session:refresh:" + refreshHash
}

func futureAccessTokenExpiry() time.Time {
	return time.Now().Add(8 * time.Minute)
}

func futureAccessTokenExpiryUnix() int64 {
	return futureAccessTokenExpiry().Unix()
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

	_, err = svc.CreateSession(ctx, "sid-1", "   ", "access", "refresh", "oidc", "browser")
	require.ErrorIs(t, err, errSessionUserRequired)

	_, err = svc.CreateSession(ctx, " \t\n ", "user-1", "access", "refresh", "oidc", "browser")
	require.ErrorIs(t, err, errSessionIDRequired)

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
	assert.Greater(t, session.AccessTokenExpiresAt, time.Now().Add(4*time.Minute).Unix())
	assert.LessOrEqual(t, session.AccessTokenExpiresAt, time.Now().Add(5*time.Minute).Unix())
}

func TestCreateSessionForApplicationRequiresAndStoresVerifiedExpiry(t *testing.T) {
	svc, tokenSvc := newAuthServiceForTest(t)
	ctx := t.Context()
	expiresAt := time.Now().Add(8 * time.Minute).Unix()

	_, err := svc.CreateSessionForApplication(
		ctx,
		"sid-provider-expiry",
		"user-provider-expiry",
		"provider-access-token",
		0,
		"provider-access-token",
		"provider-refresh-token",
		"oidc",
		"web",
		"browser",
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "verified access token expiry is required")

	_, err = svc.CreateSessionForApplication(
		ctx,
		"sid-provider-expiry",
		"user-provider-expiry",
		"provider-access-token",
		expiresAt,
		"provider-access-token",
		"provider-refresh-token",
		"oidc",
		"web",
		"browser",
	)
	require.NoError(t, err)

	session, err := tokenSvc.GetSessionStore().Get(ctx, "sid-provider-expiry")
	require.NoError(t, err)
	require.NotNil(t, session)
	assert.Equal(t, expiresAt, session.AccessTokenExpiresAt)
}

func TestRotateSession_BlacklistsOldRefreshAndTouchesSession(t *testing.T) {
	svc, tokenSvc := newAuthServiceForTest(t)
	ctx := context.Background()
	newAccessExpiry := futureAccessTokenExpiryUnix()

	_, err := svc.CreateSession(ctx, "sid-1", "user-1", "old-access", "old-refresh", "oidc", "browser")
	require.NoError(t, err)

	err = svc.RotateSession(
		ctx,
		"sid-1",
		"user-1",
		"old-refresh",
		"new-access",
		newAccessExpiry,
		"new-access",
		"new-refresh",
	)
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
	assert.Equal(t, newAccessExpiry, session.AccessTokenExpiresAt)
	assert.Equal(t, newRefreshHash, session.RefreshTokenHash)
}

func TestRotateSession_ExtendsOldRefreshRefTTLToBlacklistTTL(t *testing.T) {
	svc, tokenSvc, fixture := newAuthServiceForTestWithFixture(t)
	ctx := context.Background()
	oldRefresh := "old-refresh-ttl"
	refreshTTL := tokenSvc.GetRefreshTokenTTL()

	_, err := svc.CreateSession(ctx, "sid-ttl", "user-ttl", "old-access", oldRefresh, "oidc", "browser")
	require.NoError(t, err)
	oldRefreshHash, err := hashTokenForSession(oldRefresh)
	require.NoError(t, err)
	oldRefKey := refreshTokenRefKeyForTest(oldRefreshHash)
	require.NoError(t, fixture.Client.PExpire(ctx, oldRefKey, 50*time.Millisecond).Err())

	err = svc.RotateSession(
		ctx,
		"sid-ttl",
		"user-ttl",
		oldRefresh,
		"new-access",
		futureAccessTokenExpiryUnix(),
		"new-access",
		"new-refresh",
	)
	require.NoError(t, err)

	ref, err := tokenSvc.GetSessionStore().LookupRefreshTokenHash(ctx, oldRefreshHash)
	require.NoError(t, err)
	require.NotNil(t, ref)
	assert.Equal(t, "sid-ttl", ref.SessionID)
	assert.Equal(t, "user-ttl", ref.UserID)

	refTTL, err := fixture.Client.TTL(ctx, oldRefKey).Result()
	require.NoError(t, err)
	assert.Greater(t, refTTL, refreshTTL-5*time.Second)
	assert.LessOrEqual(t, refTTL, refreshTTL)
}

func TestSessionLifecycleRejectsMissingUserIDBeforeStateChanges(t *testing.T) {
	svc, tokenSvc := newAuthServiceForTest(t)
	ctx := context.Background()

	_, err := svc.CreateSession(ctx, "sid-missing-user", "user-1", "access-1", "refresh-1", "oidc", "browser")
	require.NoError(t, err)

	err = svc.RotateSession(
		ctx,
		"sid-missing-user",
		"   ",
		"refresh-1",
		"new-access",
		futureAccessTokenExpiryUnix(),
		"new-access",
		"new-refresh",
	)
	require.ErrorIs(t, err, errSessionUserRequired)
	blacklisted, err := tokenSvc.GetBlacklist().IsBlacklisted(ctx, "refresh-1")
	require.NoError(t, err)
	assert.False(t, blacklisted)

	err = svc.RevokeSession(ctx, "sid-missing-user", "", "access-1", "refresh-1", futureAccessTokenExpiry())
	require.ErrorIs(t, err, errSessionUserRequired)
	session, err := tokenSvc.GetSessionStore().Get(ctx, "sid-missing-user")
	require.NoError(t, err)
	require.NotNil(t, session)

	err = svc.RevokeAllSessions(ctx, " ")
	require.ErrorIs(t, err, errSessionUserRequired)
	sessions, err := tokenSvc.GetSessionStore().ListUserSessions(ctx, "user-1")
	require.NoError(t, err)
	require.Len(t, sessions, 1)
}

func TestRotateSession_RejectsTrackedSessionMismatch(t *testing.T) {
	svc, tokenSvc := newAuthServiceForTest(t)
	ctx := context.Background()

	_, err := svc.CreateSession(ctx, "sid-1", "user-1", "old-access", "old-refresh", "oidc", "browser")
	require.NoError(t, err)

	err = svc.RotateSession(
		ctx,
		"sid-1",
		"user-2",
		"old-refresh",
		"new-access",
		futureAccessTokenExpiryUnix(),
		"new-access",
		"new-refresh",
	)
	require.Error(t, err)
	assert.ErrorIs(t, err, errSessionUserMismatch)

	err = svc.RotateSession(
		ctx,
		"sid-1",
		"user-1",
		"other-refresh",
		"new-access",
		futureAccessTokenExpiryUnix(),
		"new-access",
		"new-refresh",
	)
	require.Error(t, err)
	assert.ErrorIs(t, err, errSessionRefreshTokenMismatch)

	blacklisted, err := tokenSvc.GetBlacklist().IsBlacklisted(ctx, "old-refresh")
	require.NoError(t, err)
	assert.False(t, blacklisted)

	blacklisted, err = tokenSvc.GetBlacklist().IsBlacklisted(ctx, "other-refresh")
	require.NoError(t, err)
	assert.False(t, blacklisted)
}

func TestRotateSession_WithoutSessionIDRejectsRequestWithoutBlacklistingOldRefresh(t *testing.T) {
	svc, tokenSvc := newAuthServiceForTest(t)
	ctx := context.Background()

	err := svc.RotateSession(
		ctx,
		"",
		"user-1",
		"old-refresh",
		"new-access",
		futureAccessTokenExpiryUnix(),
		"new-access",
		"new-refresh",
	)
	require.ErrorIs(t, err, errSessionIDRequired)

	blacklisted, err := tokenSvc.GetBlacklist().IsBlacklisted(ctx, "old-refresh")
	require.NoError(t, err)
	assert.False(t, blacklisted)
}

func TestRotateSession_WithBlankSessionIDRejectsRequestWithoutBlacklistingOldRefresh(t *testing.T) {
	svc, tokenSvc := newAuthServiceForTest(t)
	ctx := context.Background()

	err := svc.RotateSession(
		ctx,
		" \t\n ",
		"user-1",
		"blank-sid-refresh",
		"new-access",
		futureAccessTokenExpiryUnix(),
		"new-access",
		"new-refresh",
	)
	require.ErrorIs(t, err, errSessionIDRequired)

	blacklisted, err := tokenSvc.GetBlacklist().IsBlacklisted(ctx, "blank-sid-refresh")
	require.NoError(t, err)
	assert.False(t, blacklisted)
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

	err = svc.RevokeSession(ctx, "sid-1", "user-1", "access-1", "refresh-1", futureAccessTokenExpiry())
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

func TestRevokeSessionKeepsTrackedAccessRevokedUntilProviderExpiry(t *testing.T) {
	svc, _, fixture := newAuthServiceForTestWithFixture(t)
	ctx := t.Context()
	expiresAt := time.Now().Add(8 * time.Minute)

	_, err := svc.CreateSessionForApplication(
		ctx,
		"sid-provider-revoke-expiry",
		"user-provider-revoke-expiry",
		"provider-access-expiry",
		expiresAt.Unix(),
		"provider-access-expiry",
		"provider-refresh-expiry",
		"oidc",
		"web",
		"browser",
	)
	require.NoError(t, err)

	err = svc.RevokeSession(
		ctx,
		"sid-provider-revoke-expiry",
		"user-provider-revoke-expiry",
		"provider-access-expiry",
		"provider-refresh-expiry",
		time.Now().Add(time.Minute),
	)
	require.NoError(t, err)

	accessHash, err := hashTokenForSession("provider-access-expiry")
	require.NoError(t, err)
	ttl, err := fixture.Client.TTL(ctx, "token:blacklist:"+accessHash).Result()
	require.NoError(t, err)
	assert.Greater(t, ttl, 7*time.Minute+50*time.Second)
	assert.LessOrEqual(t, ttl, 8*time.Minute)
}

func TestRevokeSession_RejectsTrackedSessionMismatch(t *testing.T) {
	svc, tokenSvc := newAuthServiceForTest(t)
	ctx := context.Background()

	_, err := svc.CreateSession(ctx, "sid-1", "user-1", "access-1", "refresh-1", "oidc", "browser")
	require.NoError(t, err)

	err = svc.RevokeSession(ctx, "sid-1", "user-2", "access-1", "refresh-1", futureAccessTokenExpiry())
	require.Error(t, err)
	assert.ErrorIs(t, err, errSessionUserMismatch)

	session, err := tokenSvc.GetSessionStore().Get(ctx, "sid-1")
	require.NoError(t, err)
	require.NotNil(t, session)

	err = svc.RevokeSession(ctx, "sid-1", "user-1", "access-other", "refresh-1", futureAccessTokenExpiry())
	require.Error(t, err)
	assert.ErrorIs(t, err, errSessionAccessTokenMismatch)
}

func TestRotateSession_ContextCanceled(t *testing.T) {
	svc, _ := newAuthServiceForTest(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := svc.RotateSession(
		ctx,
		"sid-canceled",
		"user-1",
		"old-refresh",
		"new-access",
		futureAccessTokenExpiryUnix(),
		"new-access",
		"new-refresh",
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "load session")
}

func TestRevokeSession_FallbackBlacklistFailureWithoutSessionID(t *testing.T) {
	svc, _ := newAuthServiceForTest(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := svc.RevokeSession(ctx, "", "user-1", "access-fallback", "", futureAccessTokenExpiry())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "revoke access token")
}

func TestRevokeSession_BlankSessionIDUsesTokenBlacklistFallback(t *testing.T) {
	svc, tokenSvc, fixture := newAuthServiceForTestWithFixture(t)
	ctx := context.Background()
	expiresAt := time.Now().Add(8 * time.Minute)

	err := svc.RevokeSession(ctx, " \t\n ", "user-1", "access-fallback", "refresh-fallback", expiresAt)
	require.NoError(t, err)

	accessBlacklisted, err := tokenSvc.GetBlacklist().IsBlacklisted(ctx, "access-fallback")
	require.NoError(t, err)
	assert.True(t, accessBlacklisted)
	refreshBlacklisted, err := tokenSvc.GetBlacklist().IsBlacklisted(ctx, "refresh-fallback")
	require.NoError(t, err)
	assert.True(t, refreshBlacklisted)

	accessHash, err := hashTokenForSession("access-fallback")
	require.NoError(t, err)
	ttl, err := fixture.Client.TTL(ctx, "token:blacklist:"+accessHash).Result()
	require.NoError(t, err)
	assert.Greater(t, ttl, 7*time.Minute+50*time.Second)
	assert.LessOrEqual(t, ttl, 8*time.Minute)
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
