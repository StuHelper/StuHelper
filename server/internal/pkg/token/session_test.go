package token

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/StuHelper/StuHelper/server/internal/testutil/redisfixture"
)

func newTestSessionStore(t *testing.T) (*SessionStore, *Blacklist, *redisfixture.Fixture) {
	t.Helper()
	fixture := redisfixture.Start(t)

	store := NewSessionStore(fixture.Client, 10*time.Minute)
	bl := NewBlacklist(fixture.Client)
	t.Cleanup(bl.Close)

	return store, bl, fixture
}

func TestSessionStore_CreateAndGet(t *testing.T) {
	store, _, _ := newTestSessionStore(t)
	ctx := context.Background()

	data := SessionData{
		SessionID:            "sess-001",
		UserID:               "user-001",
		AccessTokenHash:      "acc-hash-001",
		AccessTokenExpiresAt: time.Now().Add(5 * time.Minute).Unix(),
		LoginMethod:          "phone",
		DeviceInfo:           "test-ua",
	}

	err := store.Create(ctx, data)
	require.NoError(t, err)

	got, err := store.Get(ctx, "sess-001")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "sess-001", got.SessionID)
	assert.Equal(t, "user-001", got.UserID)
	assert.Equal(t, "acc-hash-001", got.AccessTokenHash)
	assert.Equal(t, data.AccessTokenExpiresAt, got.AccessTokenExpiresAt)
	assert.Equal(t, "phone", got.LoginMethod)
	assert.Equal(t, "test-ua", got.DeviceInfo)
	assert.Greater(t, got.CreatedAt, int64(0))
	assert.Equal(t, got.CreatedAt, got.LastActiveAt)
}

func TestSessionStore_GetNonExistent(t *testing.T) {
	store, _, _ := newTestSessionStore(t)
	ctx := context.Background()

	got, err := store.Get(ctx, "nonexistent")
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestSessionStore_CreateRejectsInvalidAccessTokenExpiry(t *testing.T) {
	store, _, _ := newTestSessionStore(t)

	err := store.Create(t.Context(), SessionData{
		SessionID:            "sess-expired-access",
		UserID:               "user-expired-access",
		AccessTokenHash:      "expired-access-hash",
		AccessTokenExpiresAt: time.Now().Add(-time.Minute).Unix(),
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expiry must be in the future")

	err = store.Create(t.Context(), SessionData{
		SessionID:            "sess-beyond-session-lease",
		UserID:               "user-beyond-session-lease",
		AccessTokenHash:      "beyond-session-lease-hash",
		AccessTokenExpiresAt: time.Now().Add(11 * time.Minute).Unix(),
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds session lease")

	err = store.Create(t.Context(), SessionData{
		SessionID:            "sess-overlong-access",
		UserID:               "user-overlong-access",
		AccessTokenHash:      "overlong-access-hash",
		AccessTokenExpiresAt: time.Now().Add(maxBlacklistTTL + time.Hour).Unix(),
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds hard maximum")
}

func TestSessionStore_Touch(t *testing.T) {
	store, _, _ := newTestSessionStore(t)
	ctx := context.Background()

	data := SessionData{
		SessionID:        "sess-002",
		UserID:           "user-002",
		AccessTokenHash:  "old-acc",
		RefreshTokenHash: "old-ref",
	}
	require.NoError(t, store.Create(ctx, data))

	time.Sleep(10 * time.Millisecond)
	newAccessExpiry := time.Now().Add(5 * time.Minute).Unix()
	err := store.Touch(ctx, "sess-002", SessionTouchUpdate{
		AccessTokenHash:      "new-acc",
		AccessTokenExpiresAt: newAccessExpiry,
		RefreshTokenHash:     "new-ref",
	})
	require.NoError(t, err)

	got, err := store.Get(ctx, "sess-002")
	require.NoError(t, err)
	assert.Equal(t, "new-acc", got.AccessTokenHash)
	assert.Equal(t, newAccessExpiry, got.AccessTokenExpiresAt)
	assert.Equal(t, "new-ref", got.RefreshTokenHash)
	assert.GreaterOrEqual(t, got.LastActiveAt, got.CreatedAt)

	oldRef, err := store.LookupRefreshTokenHash(ctx, "old-ref")
	require.NoError(t, err)
	require.NotNil(t, oldRef)
	assert.Equal(t, "sess-002", oldRef.SessionID)
	assert.Equal(t, "user-002", oldRef.UserID)

	newRef, err := store.LookupRefreshTokenHash(ctx, "new-ref")
	require.NoError(t, err)
	require.NotNil(t, newRef)
	assert.Equal(t, "sess-002", newRef.SessionID)
	assert.Equal(t, "user-002", newRef.UserID)
}

func TestSessionStore_TouchRequiresExpiryWithNewAccessHash(t *testing.T) {
	store, _, _ := newTestSessionStore(t)
	ctx := t.Context()
	require.NoError(t, store.Create(ctx, SessionData{
		SessionID:       "sess-touch-expiry-required",
		UserID:          "user-touch-expiry-required",
		AccessTokenHash: "old-access",
	}))

	err := store.Touch(ctx, "sess-touch-expiry-required", SessionTouchUpdate{
		AccessTokenHash: "new-access",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "verified access token expiry is required")

	got, getErr := store.Get(ctx, "sess-touch-expiry-required")
	require.NoError(t, getErr)
	require.NotNil(t, got)
	assert.Equal(t, "old-access", got.AccessTokenHash)
	assert.Zero(t, got.AccessTokenExpiresAt)
}

func TestSessionStore_TouchRestoresUserSessionIndex(t *testing.T) {
	store, _, _ := newTestSessionStore(t)
	ctx := context.Background()

	require.NoError(t, store.Create(ctx, SessionData{
		SessionID:        "sess-touch-index",
		UserID:           "user-touch-index",
		AccessTokenHash:  "old-acc",
		RefreshTokenHash: "old-ref",
	}))
	require.NoError(t, store.rdb.Del(ctx, userSessionsPrefix+"user-touch-index").Err())

	sessions, err := store.ListUserSessions(ctx, "user-touch-index")
	require.NoError(t, err)
	assert.Empty(t, sessions)

	err = store.Touch(ctx, "sess-touch-index", SessionTouchUpdate{
		AccessTokenHash:      "new-acc",
		AccessTokenExpiresAt: time.Now().Add(5 * time.Minute).Unix(),
		RefreshTokenHash:     "new-ref",
	})
	require.NoError(t, err)

	sessions, err = store.ListUserSessions(ctx, "user-touch-index")
	require.NoError(t, err)
	require.Len(t, sessions, 1)
	assert.Equal(t, "sess-touch-index", sessions[0].SessionID)

	ttl, err := store.rdb.TTL(ctx, userSessionsPrefix+"user-touch-index").Result()
	require.NoError(t, err)
	assert.Positive(t, ttl)
}

func TestSessionStore_LookupRefreshTokenHashHash(t *testing.T) {
	store, _, _ := newTestSessionStore(t)
	ctx := context.Background()

	require.NoError(t, store.Create(ctx, SessionData{
		SessionID:        "sess-refresh-ref",
		UserID:           "user-refresh-ref",
		AccessTokenHash:  "acc-refresh-ref",
		RefreshTokenHash: "ref-refresh-ref",
	}))

	ref, err := store.LookupRefreshTokenHash(ctx, "ref-refresh-ref")
	require.NoError(t, err)
	require.NotNil(t, ref)
	assert.Equal(t, "sess-refresh-ref", ref.SessionID)
	assert.Equal(t, "user-refresh-ref", ref.UserID)

	missingRef, err := store.LookupRefreshTokenHash(ctx, "missing-ref")
	require.NoError(t, err)
	assert.Nil(t, missingRef)
}

func TestSessionStore_Revoke(t *testing.T) {
	store, bl, _ := newTestSessionStore(t)
	ctx := context.Background()

	data := SessionData{
		SessionID:            "sess-003",
		UserID:               "user-003",
		AccessTokenHash:      "acc-hash-003",
		AccessTokenExpiresAt: time.Now().Add(8 * time.Minute).Unix(),
		RefreshTokenHash:     "ref-hash-003",
	}
	require.NoError(t, store.Create(ctx, data))

	revoked, err := store.Revoke(ctx, "sess-003", bl, 10*time.Minute)
	require.NoError(t, err)
	require.NotNil(t, revoked)
	assert.Equal(t, "sess-003", revoked.SessionID)

	// session 应被删除
	got, err := store.Get(ctx, "sess-003")
	require.NoError(t, err)
	assert.Nil(t, got)

	// token hash 应在黑名单中
	isBlocked, err := bl.rdb.Exists(ctx, blacklistPrefix+"acc-hash-003").Result()
	require.NoError(t, err)
	assert.Equal(t, int64(1), isBlocked)
	accessBlacklistTTL, err := bl.rdb.TTL(ctx, blacklistPrefix+"acc-hash-003").Result()
	require.NoError(t, err)
	assert.Greater(t, accessBlacklistTTL, 7*time.Minute+50*time.Second)
	assert.LessOrEqual(t, accessBlacklistTTL, 8*time.Minute)

	isBlocked2, err := bl.rdb.Exists(ctx, blacklistPrefix+"ref-hash-003").Result()
	require.NoError(t, err)
	assert.Equal(t, int64(1), isBlocked2)
}

func TestSessionStore_RevokeLegacySessionUsesActualRedisPTTL(t *testing.T) {
	store, bl, _ := newTestSessionStore(t)
	ctx := t.Context()
	require.NoError(t, store.Create(ctx, SessionData{
		SessionID:       "sess-legacy-pttl",
		UserID:          "user-legacy-pttl",
		AccessTokenHash: "legacy-access-hash",
	}))
	require.NoError(t, store.rdb.PExpire(ctx, sessionPrefix+"sess-legacy-pttl", 4*time.Minute).Err())

	_, err := store.Revoke(ctx, "sess-legacy-pttl", bl, 10*time.Minute)
	require.NoError(t, err)

	ttl, err := bl.rdb.TTL(ctx, blacklistPrefix+"legacy-access-hash").Result()
	require.NoError(t, err)
	assert.Greater(t, ttl, 3*time.Minute+50*time.Second)
	assert.LessOrEqual(t, ttl, 4*time.Minute)
}

func TestSessionStore_RevokeAllUsesEachAccessTokenExpiry(t *testing.T) {
	store, bl, _ := newTestSessionStore(t)
	ctx := t.Context()
	now := time.Now()
	require.NoError(t, store.Create(ctx, SessionData{
		SessionID:            "sess-expiry-short",
		UserID:               "user-expiry-per-session",
		AccessTokenHash:      "access-expiry-short",
		AccessTokenExpiresAt: now.Add(2 * time.Minute).Unix(),
	}))
	require.NoError(t, store.Create(ctx, SessionData{
		SessionID:            "sess-expiry-long",
		UserID:               "user-expiry-per-session",
		AccessTokenHash:      "access-expiry-long",
		AccessTokenExpiresAt: now.Add(8 * time.Minute).Unix(),
	}))

	require.NoError(t, store.RevokeAll(ctx, "user-expiry-per-session", bl, 10*time.Minute))

	shortTTL, err := bl.rdb.TTL(ctx, blacklistPrefix+"access-expiry-short").Result()
	require.NoError(t, err)
	longTTL, err := bl.rdb.TTL(ctx, blacklistPrefix+"access-expiry-long").Result()
	require.NoError(t, err)
	assert.Greater(t, shortTTL, time.Minute+50*time.Second)
	assert.LessOrEqual(t, shortTTL, 2*time.Minute)
	assert.Greater(t, longTTL, 7*time.Minute+50*time.Second)
	assert.LessOrEqual(t, longTTL, 8*time.Minute)
	assert.Greater(t, longTTL-shortTTL, 5*time.Minute)
}

func TestSessionStore_RevokeExtendsRefreshRefToBlacklistTTL(t *testing.T) {
	store, bl, _ := newTestSessionStore(t)
	ctx := context.Background()
	refreshTTL := 10 * time.Minute
	refreshHash := "ref-hash-revoke-ttl"

	require.NoError(t, store.Create(ctx, SessionData{
		SessionID:        "sess-revoke-ttl",
		UserID:           "user-revoke-ttl",
		AccessTokenHash:  "acc-revoke-ttl",
		RefreshTokenHash: refreshHash,
	}))
	require.NoError(t, store.rdb.PExpire(ctx, refreshTokenRefKey(refreshHash), 50*time.Millisecond).Err())

	revoked, err := store.Revoke(ctx, "sess-revoke-ttl", bl, refreshTTL)
	require.NoError(t, err)
	require.NotNil(t, revoked)

	ref, err := store.LookupRefreshTokenHash(ctx, refreshHash)
	require.NoError(t, err)
	require.NotNil(t, ref)
	assert.Equal(t, "sess-revoke-ttl", ref.SessionID)
	assert.Equal(t, "user-revoke-ttl", ref.UserID)

	refTTL, err := store.rdb.TTL(ctx, refreshTokenRefKey(refreshHash)).Result()
	require.NoError(t, err)
	assert.Greater(t, refTTL, refreshTTL-5*time.Second)
	assert.LessOrEqual(t, refTTL, refreshTTL)
}

func TestSessionStore_RevokeAll(t *testing.T) {
	store, bl, _ := newTestSessionStore(t)
	ctx := context.Background()

	for i, sid := range []string{"sess-a", "sess-b", "sess-c"} {
		require.NoError(t, store.Create(ctx, SessionData{
			SessionID:       sid,
			UserID:          "user-multi",
			AccessTokenHash: "acc-" + sid,
			LoginMethod:     []string{"oidc", "phone", "oidc-native"}[i],
		}))
	}

	sessions, err := store.ListUserSessions(ctx, "user-multi")
	require.NoError(t, err)
	assert.Len(t, sessions, 3)

	err = store.RevokeAll(ctx, "user-multi", bl, 10*time.Minute)
	require.NoError(t, err)

	sessions, err = store.ListUserSessions(ctx, "user-multi")
	require.NoError(t, err)
	assert.Len(t, sessions, 0)
}

func TestSessionStore_RevokeAll_ReturnsErrorAndKeepsIndexForLegacySessionWithoutTTL(t *testing.T) {
	store, bl, _ := newTestSessionStore(t)
	ctx := context.Background()

	require.NoError(t, store.Create(ctx, SessionData{
		SessionID:       "sess-fail-no-ttl",
		UserID:          "user-revoke-fail",
		AccessTokenHash: "acc-fail-no-ttl",
	}))
	require.NoError(t, store.rdb.Persist(ctx, sessionPrefix+"sess-fail-no-ttl").Err())

	err := store.RevokeAll(ctx, "user-revoke-fail", bl, 10*time.Minute)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "session revoke all")
	assert.Contains(t, err.Error(), "legacy session has no Redis expiry")

	sessions, err := store.ListUserSessions(ctx, "user-revoke-fail")
	require.NoError(t, err)
	assert.Len(t, sessions, 1)

	session, err := store.Get(ctx, "sess-fail-no-ttl")
	require.NoError(t, err)
	assert.NotNil(t, session)
}

func TestSessionStore_ListUserSessions(t *testing.T) {
	store, _, _ := newTestSessionStore(t)
	ctx := context.Background()

	require.NoError(t, store.Create(ctx, SessionData{
		SessionID: "s1", UserID: "u1", LoginMethod: "phone",
	}))
	require.NoError(t, store.Create(ctx, SessionData{
		SessionID: "s2", UserID: "u1", LoginMethod: "oidc",
	}))
	require.NoError(t, store.Create(ctx, SessionData{
		SessionID: "s3", UserID: "u2", LoginMethod: "phone",
	}))

	u1Sessions, err := store.ListUserSessions(ctx, "u1")
	require.NoError(t, err)
	assert.Len(t, u1Sessions, 2)

	u2Sessions, err := store.ListUserSessions(ctx, "u2")
	require.NoError(t, err)
	assert.Len(t, u2Sessions, 1)
}

func TestSessionStore_ListUserSessions_CleansMissingSessionMembers(t *testing.T) {
	store, _, fixture := newTestSessionStore(t)
	ctx := context.Background()

	require.NoError(t, store.Create(ctx, SessionData{
		SessionID: "active-session",
		UserID:    "u-cleanup",
	}))
	require.NoError(t, store.Create(ctx, SessionData{
		SessionID: "expired-session",
		UserID:    "u-cleanup",
	}))

	fixture.Server.Del(sessionPrefix + "expired-session")

	sessions, err := store.ListUserSessions(ctx, "u-cleanup")
	require.NoError(t, err)
	require.Len(t, sessions, 1)
	assert.Equal(t, "active-session", sessions[0].SessionID)

	sessionIDs, err := store.rdb.SMembers(ctx, userSessionsPrefix+"u-cleanup").Result()
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"active-session"}, sessionIDs)
}

func TestSessionStore_ListUserSessions_CleansCrossUserSessionMembers(t *testing.T) {
	store, _, _ := newTestSessionStore(t)
	ctx := context.Background()

	require.NoError(t, store.Create(ctx, SessionData{
		SessionID:       "session-owned",
		UserID:          "user-owned",
		AccessTokenHash: "acc-owned",
	}))
	require.NoError(t, store.Create(ctx, SessionData{
		SessionID:       "session-other",
		UserID:          "user-other",
		AccessTokenHash: "acc-other",
	}))
	require.NoError(t, store.rdb.SAdd(ctx, userSessionsPrefix+"user-owned", "session-other").Err())

	sessions, err := store.ListUserSessions(ctx, "user-owned")
	require.NoError(t, err)
	require.Len(t, sessions, 1)
	assert.Equal(t, "session-owned", sessions[0].SessionID)

	ownedSessionIDs, err := store.rdb.SMembers(ctx, userSessionsPrefix+"user-owned").Result()
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"session-owned"}, ownedSessionIDs)

	other, err := store.Get(ctx, "session-other")
	require.NoError(t, err)
	require.NotNil(t, other)
	assert.Equal(t, "user-other", other.UserID)
}

func TestSessionStore_RevokeAllSkipsCrossUserSessionMembers(t *testing.T) {
	store, bl, _ := newTestSessionStore(t)
	ctx := context.Background()

	require.NoError(t, store.Create(ctx, SessionData{
		SessionID:        "session-owned",
		UserID:           "user-owned",
		AccessTokenHash:  "acc-owned",
		RefreshTokenHash: "ref-owned",
	}))
	require.NoError(t, store.Create(ctx, SessionData{
		SessionID:        "session-other",
		UserID:           "user-other",
		AccessTokenHash:  "acc-other",
		RefreshTokenHash: "ref-other",
	}))
	require.NoError(t, store.rdb.SAdd(ctx, userSessionsPrefix+"user-owned", "session-other").Err())

	err := store.RevokeAll(ctx, "user-owned", bl, 10*time.Minute)
	require.NoError(t, err)

	owned, err := store.Get(ctx, "session-owned")
	require.NoError(t, err)
	assert.Nil(t, owned)

	other, err := store.Get(ctx, "session-other")
	require.NoError(t, err)
	require.NotNil(t, other)
	assert.Equal(t, "user-other", other.UserID)

	isOtherAccessBlocked, err := bl.rdb.Exists(ctx, blacklistPrefix+"acc-other").Result()
	require.NoError(t, err)
	assert.Zero(t, isOtherAccessBlocked)

	isOtherRefreshBlocked, err := bl.rdb.Exists(ctx, blacklistPrefix+"ref-other").Result()
	require.NoError(t, err)
	assert.Zero(t, isOtherRefreshBlocked)

	otherSessionIDs, err := store.rdb.SMembers(ctx, userSessionsPrefix+"user-other").Result()
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"session-other"}, otherSessionIDs)
}

func TestGenerateSessionID(t *testing.T) {
	id1, err := GenerateSessionID()
	require.NoError(t, err)
	assert.Len(t, id1, 32) // 16 字节 hex = 32 字符

	id2, err := GenerateSessionID()
	require.NoError(t, err)
	assert.NotEqual(t, id1, id2)
}
