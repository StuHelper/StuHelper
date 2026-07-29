package token

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/StuHelper/StuHelper/server/internal/pkg/circuitbreaker"
	"github.com/StuHelper/StuHelper/server/internal/pkg/crypto"
	"github.com/StuHelper/StuHelper/server/internal/testutil/redisfixture"
)

func setupTestRedis(t *testing.T) *redisfixture.Fixture {
	t.Helper()
	return redisfixture.Start(t)
}

func TestBlacklist_Add(t *testing.T) {
	// 初始化 HMAC key（Blacklist 内部使用 crypto.HMACHash）
	require.NoError(t, crypto.InitHMACKey("test-blacklist-secret", false))

	fixture := setupTestRedis(t)
	bl := NewBlacklist(fixture.Client)
	ctx := context.Background()

	err := bl.Add(ctx, "test-token", time.Hour)
	assert.NoError(t, err)

	// 验证 token 已加入黑名单
	isBlacklisted, err := bl.IsBlacklisted(ctx, "test-token")
	assert.NoError(t, err)
	assert.True(t, isBlacklisted)
}

func TestBlacklist_IsBlacklisted(t *testing.T) {
	// 显式初始化 HMAC key，确保单独运行时不依赖其他测试的副作用
	require.NoError(t, crypto.InitHMACKey("test-blacklist-secret", false))

	fixture := setupTestRedis(t)
	bl := NewBlacklist(fixture.Client)
	ctx := context.Background()

	// 未加入黑名单的 token
	isBlacklisted, err := bl.IsBlacklisted(ctx, "unknown-token")
	assert.NoError(t, err)
	assert.False(t, isBlacklisted)

	// 加入黑名单后
	err = bl.Add(ctx, "blacklisted-token", time.Hour)
	require.NoError(t, err)

	isBlacklisted, err = bl.IsBlacklisted(ctx, "blacklisted-token")
	assert.NoError(t, err)
	assert.True(t, isBlacklisted)
}

func TestBlacklist_TryConsumeRefreshToken(t *testing.T) {
	require.NoError(t, crypto.InitHMACKey("test-blacklist-secret", false))

	fixture := setupTestRedis(t)
	bl := NewBlacklist(fixture.Client)
	ctx := context.Background()

	consumed, err := bl.TryConsumeRefreshToken(ctx, "refresh-token", time.Hour)
	require.NoError(t, err)
	assert.True(t, consumed)

	consumed, err = bl.TryConsumeRefreshToken(ctx, "refresh-token", time.Hour)
	require.NoError(t, err)
	assert.False(t, consumed)

	isBlacklisted, err := bl.IsBlacklisted(ctx, "refresh-token")
	require.NoError(t, err)
	assert.False(t, isBlacklisted)

	require.NoError(t, bl.ReleaseConsumedRefreshToken(ctx, "refresh-token"))

	isBlacklisted, err = bl.IsBlacklisted(ctx, "refresh-token")
	require.NoError(t, err)
	assert.False(t, isBlacklisted)

	consumed, err = bl.TryConsumeRefreshToken(ctx, "refresh-token", time.Hour)
	require.NoError(t, err)
	assert.True(t, consumed)
}

func TestBlacklist_TryConsumeRefreshTokenDoesNotPolluteRevocationCache(t *testing.T) {
	require.NoError(t, crypto.InitHMACKey("test-blacklist-secret", false))

	fixture := setupTestRedis(t)
	bl := NewBlacklist(fixture.Client)
	ctx := context.Background()

	consumed, err := bl.TryConsumeRefreshToken(ctx, "reserved-refresh-token", time.Hour)
	require.NoError(t, err)
	require.True(t, consumed)

	hash, err := hashToken("reserved-refresh-token")
	require.NoError(t, err)
	_, ok := bl.cachedRevocation(hash)
	assert.False(t, ok)

	isBlacklisted, err := bl.IsBlacklisted(ctx, "reserved-refresh-token")
	require.NoError(t, err)
	assert.False(t, isBlacklisted)

	require.NoError(t, bl.Add(ctx, "reserved-refresh-token", time.Hour))

	isBlacklisted, err = bl.IsBlacklisted(ctx, "reserved-refresh-token")
	require.NoError(t, err)
	assert.True(t, isBlacklisted)

	require.NoError(t, bl.ReleaseConsumedRefreshToken(ctx, "reserved-refresh-token"))

	isBlacklisted, err = bl.IsBlacklisted(ctx, "reserved-refresh-token")
	require.NoError(t, err)
	assert.True(t, isBlacklisted)
}

func TestBlacklist_TryConsumeRefreshTokenDoesNotCacheFailedReservation(t *testing.T) {
	require.NoError(t, crypto.InitHMACKey("test-blacklist-secret", false))

	bl := &Blacklist{
		cb: circuitbreaker.NewNamed("token_blacklist_test_reservation_failure", circuitbreaker.Config{
			FailureThreshold: 1,
			SuccessThreshold: 1,
			Timeout:          time.Hour,
		}),
		stopCh: make(chan struct{}),
	}
	bl.cb.RecordFailure()

	consumed, err := bl.TryConsumeRefreshToken(context.Background(), "reservation-failure-token", time.Hour)
	require.Error(t, err)
	assert.False(t, consumed)

	hash, err := hashToken("reservation-failure-token")
	require.NoError(t, err)
	_, ok := bl.cachedRevocation(hash)
	assert.False(t, ok)

	blacklisted, err := bl.IsBlacklisted(context.Background(), "reservation-failure-token")
	require.Error(t, err)
	assert.True(t, blacklisted)
}

func TestBlacklist_NilRedisFailsClosedWithoutPanic(t *testing.T) {
	require.NoError(t, crypto.InitHMACKey("test-blacklist-secret", false))

	bl := NewBlacklist(nil)
	defer bl.Close()
	ctx := context.Background()

	assert.NotPanics(t, func() {
		err := bl.Add(ctx, "revoked-token", time.Hour)
		require.Error(t, err)
	})
	blacklisted, err := bl.IsBlacklisted(ctx, "revoked-token")
	require.NoError(t, err)
	assert.True(t, blacklisted)

	assert.NotPanics(t, func() {
		err := bl.AddByHash(ctx, "known-hash", time.Hour)
		require.Error(t, err)
	})

	assert.NotPanics(t, func() {
		consumed, err := bl.TryConsumeRefreshToken(ctx, "refresh-token", time.Hour)
		require.Error(t, err)
		assert.False(t, consumed)
	})

	assert.NotPanics(t, func() {
		err := bl.ReleaseConsumedRefreshToken(ctx, "refresh-token")
		require.Error(t, err)
	})

	assert.NotPanics(t, func() {
		blacklisted, err = bl.IsBlacklisted(ctx, "unknown-token")
		require.Error(t, err)
		assert.True(t, blacklisted)
	})
}

func TestBlacklistCloseIsNilAndZeroValueSafe(t *testing.T) {
	var nilBlacklist *Blacklist
	assert.NotPanics(t, func() {
		nilBlacklist.Close()
	})

	zeroBlacklist := &Blacklist{}
	assert.NotPanics(t, func() {
		zeroBlacklist.Close()
		zeroBlacklist.Close()
	})
}

func TestBlacklist_ReleaseFailureKeepsLocalRevocation(t *testing.T) {
	require.NoError(t, crypto.InitHMACKey("test-blacklist-secret", false))

	bl := &Blacklist{
		cb: circuitbreaker.NewNamed("token_blacklist_test_release_failure", circuitbreaker.Config{
			FailureThreshold: 1,
			SuccessThreshold: 1,
			Timeout:          time.Hour,
		}),
		stopCh: make(chan struct{}),
	}
	hash, err := hashToken("refresh-token")
	require.NoError(t, err)
	bl.cacheRevocation(hash)
	bl.cb.RecordFailure()

	err = bl.ReleaseConsumedRefreshToken(context.Background(), "refresh-token")
	require.Error(t, err)

	blacklisted, err := bl.IsBlacklisted(context.Background(), "refresh-token")
	require.NoError(t, err)
	assert.True(t, blacklisted)
}

func TestBlacklist_IsBlacklisted_DeniesWhenCircuitOpenAndOnlyNegativeCacheExists(t *testing.T) {
	require.NoError(t, crypto.InitHMACKey("test-blacklist-secret", false))

	bl := &Blacklist{
		cb: circuitbreaker.NewNamed("token_blacklist_test", circuitbreaker.Config{
			FailureThreshold: 1,
			SuccessThreshold: 1,
			Timeout:          time.Hour,
		}),
		stopCh: make(chan struct{}),
	}

	hash, err := hashToken("stale-negative-cache-token")
	require.NoError(t, err)
	bl.localCache.Store(hash, localCacheEntry{
		blacklisted: false,
		expiresAt:   time.Now().Add(time.Minute),
	})
	bl.cb.RecordFailure()

	blacklisted, err := bl.IsBlacklisted(context.Background(), "stale-negative-cache-token")
	require.Error(t, err)
	assert.True(t, blacklisted)
}

func TestBlacklist_IsBlacklisted_AllowsPositiveRevocationCacheWhenCircuitOpen(t *testing.T) {
	require.NoError(t, crypto.InitHMACKey("test-blacklist-secret", false))

	bl := &Blacklist{
		cb: circuitbreaker.NewNamed("token_blacklist_test_positive", circuitbreaker.Config{
			FailureThreshold: 1,
			SuccessThreshold: 1,
			Timeout:          time.Hour,
		}),
		stopCh: make(chan struct{}),
	}

	hash, err := hashToken("cached-revoked-token")
	require.NoError(t, err)
	bl.localCache.Store(hash, localCacheEntry{
		blacklisted: true,
		expiresAt:   time.Now().Add(time.Minute),
	})
	bl.cb.RecordFailure()

	blacklisted, err := bl.IsBlacklisted(context.Background(), "cached-revoked-token")
	require.NoError(t, err)
	assert.True(t, blacklisted)
}
