package sso

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	// OAuth state 的 Redis key 前缀
	stateKeyPrefix = "oauth:state:"
	// state 有效期（5分钟）
	stateTTL = 5 * time.Minute
	// state 长度（字节）
	stateLength = 32
)

// StateManager OAuth state 管理器，用于防止 CSRF 和回放攻击
type StateManager struct {
	rdb *redis.Client
}

// NewStateManager 创建 state 管理器
func NewStateManager(rdb *redis.Client) *StateManager {
	return &StateManager{rdb: rdb}
}

// Generate 生成随机 state 并存储到 Redis
func (m *StateManager) Generate(ctx context.Context) (string, error) {
	b := make([]byte, stateLength)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate random state: %w", err)
	}

	state := base64.RawURLEncoding.EncodeToString(b)
	key := stateKeyPrefix + state

	// 存储 state 到 Redis，设置 TTL
	if err := m.rdb.Set(ctx, key, "1", stateTTL).Err(); err != nil {
		return "", fmt.Errorf("failed to store state: %w", err)
	}

	return state, nil
}

// Validate 验证并消费 state（一次性使用，防止回放攻击）
func (m *StateManager) Validate(ctx context.Context, state string) (bool, error) {
	if state == "" {
		return false, nil
	}

	key := stateKeyPrefix + state

	// 使用 DEL 命令原子性地删除并检查是否存在
	deleted, err := m.rdb.Del(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to validate state: %w", err)
	}

	// deleted > 0 表示 key 存在且已被删除
	return deleted > 0, nil
}
