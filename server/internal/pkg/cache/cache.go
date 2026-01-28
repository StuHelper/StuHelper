package cache

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
)

// Helper Redis 缓存辅助工具
type Helper struct {
	client *redis.Client
}

// NewHelper 创建缓存辅助工具
func NewHelper(client *redis.Client) *Helper {
	return &Helper{client: client}
}

// Get 获取缓存值
func (h *Helper) Get(ctx context.Context, key string) (any, bool) {
	if h.client == nil {
		return nil, false
	}
	data, err := h.client.Get(ctx, key).Bytes()
	if err != nil {
		return nil, false
	}
	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		return nil, false
	}
	return v, true
}

// Set 设置缓存值
func (h *Helper) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	if h.client == nil {
		return nil
	}
	data, err := json.Marshal(value)
	if err != nil {
		logger.L().Warn("failed to marshal cache value",
			zap.String("key", key),
			zap.Error(err),
		)
		return err
	}
	if err := h.client.Set(ctx, key, data, ttl).Err(); err != nil {
		logger.L().Warn("failed to set cache",
			zap.String("key", key),
			zap.Error(err),
		)
		return err
	}
	return nil
}

// Invalidate 批量删除匹配前缀的缓存
func (h *Helper) Invalidate(ctx context.Context, prefix string) error {
	if h.client == nil {
		return nil
	}

	const maxKeysToDelete = 1000
	var keys []string
	var cursor uint64
	for {
		var batch []string
		var err error
		batch, cursor, err = h.client.Scan(ctx, cursor, prefix+"*", 100).Result()
		if err != nil {
			logger.L().Warn("failed to scan cache keys",
				zap.String("prefix", prefix),
				zap.Error(err),
			)
			return err
		}
		keys = append(keys, batch...)
		if cursor == 0 || len(keys) >= maxKeysToDelete {
			break
		}
	}

	if len(keys) == 0 {
		return nil
	}

	pipe := h.client.Pipeline()
	for _, key := range keys {
		pipe.Del(ctx, key)
	}
	_, err := pipe.Exec(ctx)
	if err != nil {
		logger.L().Warn("failed to invalidate cache",
			zap.String("prefix", prefix),
			zap.Int("key_count", len(keys)),
			zap.Error(err),
		)
	}
	return err
}

// GetInt 获取整数缓存值
func (h *Helper) GetInt(ctx context.Context, key string) (int, bool) {
	if h.client == nil {
		return 0, false
	}
	val, err := h.client.Get(ctx, key).Int()
	if err != nil {
		return 0, false
	}
	return val, true
}

// SetInt 设置整数缓存值
func (h *Helper) SetInt(ctx context.Context, key string, value int, ttl time.Duration) error {
	if h.client == nil {
		return nil
	}
	if err := h.client.Set(ctx, key, value, ttl).Err(); err != nil {
		logger.L().Warn("failed to set int cache",
			zap.String("key", key),
			zap.Error(err),
		)
		return err
	}
	return nil
}
