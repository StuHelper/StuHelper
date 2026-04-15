package token

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestSessionStore(t *testing.T) (*SessionStore, *Blacklist, *miniredis.Miniredis) {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	store := NewSessionStore(rdb, 10*time.Minute)
	bl := NewBlacklist(rdb)
	t.Cleanup(bl.Close)

	return store, bl, mr
}

func TestSessionStore_CreateAndGet(t *testing.T) {
	store, _, _ := newTestSessionStore(t)
	ctx := context.Background()

	data := SessionData{
		SessionID:       "sess-001",
		UserID:          "user-001",
		AccessTokenHash: "acc-hash-001",
		LoginMethod:     "phone",
		DeviceInfo:      "test-ua",
	}

	err := store.Create(ctx, data)
	require.NoError(t, err)

	got, err := store.Get(ctx, "sess-001")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "sess-001", got.SessionID)
	assert.Equal(t, "user-001", got.UserID)
	assert.Equal(t, "acc-hash-001", got.AccessTokenHash)
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
	err := store.Touch(ctx, "sess-002", "new-acc", "new-ref")
	require.NoError(t, err)

	got, err := store.Get(ctx, "sess-002")
	require.NoError(t, err)
	assert.Equal(t, "new-acc", got.AccessTokenHash)
	assert.Equal(t, "new-ref", got.RefreshTokenHash)
	assert.GreaterOrEqual(t, got.LastActiveAt, got.CreatedAt)
}

func TestSessionStore_Revoke(t *testing.T) {
	store, bl, _ := newTestSessionStore(t)
	ctx := context.Background()

	data := SessionData{
		SessionID:        "sess-003",
		UserID:           "user-003",
		AccessTokenHash:  "acc-hash-003",
		RefreshTokenHash: "ref-hash-003",
	}
	require.NoError(t, store.Create(ctx, data))

	revoked, err := store.Revoke(ctx, "sess-003", bl, 5*time.Minute, 10*time.Minute)
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

	isBlocked2, err := bl.rdb.Exists(ctx, blacklistPrefix+"ref-hash-003").Result()
	require.NoError(t, err)
	assert.Equal(t, int64(1), isBlocked2)
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

	err = store.RevokeAll(ctx, "user-multi", bl, 5*time.Minute, 10*time.Minute)
	require.NoError(t, err)

	sessions, err = store.ListUserSessions(ctx, "user-multi")
	require.NoError(t, err)
	assert.Len(t, sessions, 0)
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

func TestGenerateSessionID(t *testing.T) {
	id1, err := GenerateSessionID()
	require.NoError(t, err)
	assert.Len(t, id1, 32) // 16 字节 hex = 32 字符

	id2, err := GenerateSessionID()
	require.NoError(t, err)
	assert.NotEqual(t, id1, id2)
}
