package token

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/crypto"
	"git.stuhelper.com/StuHelper/StuHelper/internal/testutil/redisfixture"
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
	assert.True(t, isBlacklisted)

	require.NoError(t, bl.ReleaseConsumedRefreshToken(ctx, "refresh-token"))

	isBlacklisted, err = bl.IsBlacklisted(ctx, "refresh-token")
	require.NoError(t, err)
	assert.False(t, isBlacklisted)

	consumed, err = bl.TryConsumeRefreshToken(ctx, "refresh-token", time.Hour)
	require.NoError(t, err)
	assert.True(t, consumed)
}
