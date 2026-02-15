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
	// state 生成最大重试次数（防止碰撞时无限递归）
	stateMaxRetries = 3
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
	for i := 0; i < stateMaxRetries; i++ {
		b := make([]byte, stateLength)
		if _, err := rand.Read(b); err != nil {
			return "", fmt.Errorf("failed to generate random state: %w", err)
		}

		state := base64.RawURLEncoding.EncodeToString(b)
		key := stateKeyPrefix + state

		// 使用 SetNX 原子性存储 state，防止极端并发下碰撞覆盖
		ok, err := m.rdb.SetNX(ctx, key, "1", stateTTL).Result()
		if err != nil {
			return "", fmt.Errorf("failed to store state: %w", err)
		}
		if ok {
			return state, nil
		}
		// 极端情况：随机 state 碰撞，重试
	}
	return "", fmt.Errorf("failed to generate unique state after %d attempts", stateMaxRetries)
}

// validateStateScript 使用 Lua 脚本原子性地 GET+DEL state，
// 防止攻击者在 GET 和 DEL 之间重放 state 参数（时序攻击）
var validateStateScript = redis.NewScript(`
local v = redis.call('GET', KEYS[1])
if v then
	redis.call('DEL', KEYS[1])
	return 1
end
return 0
`)

// Validate 验证并消费 state（一次性使用，防止回放攻击）
// 使用 Lua 脚本保证 GET+DEL 原子性，单次 RTT 完成
func (m *StateManager) Validate(ctx context.Context, state string) (bool, error) {
	if state == "" {
		return false, nil
	}

	key := stateKeyPrefix + state

	result, err := validateStateScript.Run(ctx, m.rdb, []string{key}).Int64()
	if err != nil {
		return false, fmt.Errorf("failed to validate state: %w", err)
	}

	return result == 1, nil
}
