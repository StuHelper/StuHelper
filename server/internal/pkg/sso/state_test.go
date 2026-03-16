package sso

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 跳过需要 Redis 的测试（如果没有可用的 Redis）
func skipIfNoRedis(t *testing.T, client *redis.Client) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skip("跳过测试：Redis 不可用")
	}
}

func setupTestRedis(_ *testing.T) (*redis.Client, func()) {
	// 尝试连接本地 Redis
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   15, // 使用 DB 15 进行测试，避免影响其他数据
	})

	cleanup := func() {
		// 清理测试数据
		ctx := context.Background()
		keys, _ := client.Keys(ctx, stateKeyPrefix+"*").Result()
		if len(keys) > 0 {
			client.Del(ctx, keys...)
		}
		_ = client.Close() // test cleanup, error not actionable
	}

	return client, cleanup
}

func TestStateManager_Generate(t *testing.T) {
	client, cleanup := setupTestRedis(t)
	defer cleanup()
	skipIfNoRedis(t, client)

	sm := NewStateManager(client)
	ctx := context.Background()

	// 生成 state
	state, err := sm.Generate(ctx)
	assert.NoError(t, err)
	assert.NotEmpty(t, state)
	assert.Len(t, state, 43) // base64 编码的 32 字节

	// 验证 state 已存储到 Redis
	exists, err := client.Exists(ctx, stateKeyPrefix+state).Result()
	assert.NoError(t, err)
	assert.Equal(t, int64(1), exists)
}

func TestStateManager_Generate_Uniqueness(t *testing.T) {
	client, cleanup := setupTestRedis(t)
	defer cleanup()
	skipIfNoRedis(t, client)

	sm := NewStateManager(client)
	ctx := context.Background()

	// 生成多个 state，应该都不同
	states := make(map[string]bool)
	for range 100 {
		state, err := sm.Generate(ctx)
		assert.NoError(t, err)
		assert.False(t, states[state], "生成了重复的 state")
		states[state] = true
	}
}

func TestStateManager_Validate_Success(t *testing.T) {
	client, cleanup := setupTestRedis(t)
	defer cleanup()
	skipIfNoRedis(t, client)

	sm := NewStateManager(client)
	ctx := context.Background()

	// 生成 state
	state, err := sm.Generate(ctx)
	require.NoError(t, err)

	// 验证 state（应该成功）
	valid, err := sm.Validate(ctx, state)
	assert.NoError(t, err)
	assert.True(t, valid)

	// 再次验证同一个 state（应该失败，因为已被消费）
	valid, err = sm.Validate(ctx, state)
	assert.NoError(t, err)
	assert.False(t, valid, "state 应该只能使用一次")
}

func TestStateManager_Validate_InvalidState(t *testing.T) {
	client, cleanup := setupTestRedis(t)
	defer cleanup()
	skipIfNoRedis(t, client)

	sm := NewStateManager(client)
	ctx := context.Background()

	// 验证不存在的 state
	valid, err := sm.Validate(ctx, "invalid-state")
	assert.NoError(t, err)
	assert.False(t, valid)
}

func TestStateManager_Validate_EmptyState(t *testing.T) {
	client, cleanup := setupTestRedis(t)
	defer cleanup()
	skipIfNoRedis(t, client)

	sm := NewStateManager(client)
	ctx := context.Background()

	// 验证空 state
	valid, err := sm.Validate(ctx, "")
	assert.NoError(t, err)
	assert.False(t, valid)
}
