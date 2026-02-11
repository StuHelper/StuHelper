package cache

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/metrics"
)

const (
	// DefaultTTL 默认缓存过期时间
	DefaultTTL = 5 * time.Minute
	// VersionKeyTTL 版本号 key 的过期时间
	VersionKeyTTL = 24 * time.Hour
)

// Helper Redis 缓存辅助工具
type Helper struct {
	client *redis.Client
}

// NewHelper 创建缓存辅助工具
func NewHelper(client *redis.Client) *Helper {
	return &Helper{client: client}
}

// Client 返回底层 Redis 客户端（用于需要直接访问的场景）
func (h *Helper) Client() *redis.Client {
	return h.client
}

// Get 获取缓存值
func (h *Helper) Get(ctx context.Context, key string) (any, bool) {
	if h.client == nil {
		return nil, false
	}
	start := time.Now()
	data, err := h.client.Get(ctx, key).Bytes()
	metrics.CacheOperationDuration.WithLabelValues("get", "redis").Observe(time.Since(start).Seconds())
	if err != nil {
		metrics.CacheMissesTotal.WithLabelValues("redis").Inc()
		return nil, false
	}
	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		return nil, false
	}
	metrics.CacheHitsTotal.WithLabelValues("redis").Inc()
	return v, true
}

// GetAs 获取缓存值并反序列化为指定类型（泛型版本，避免 any 类型丢失问题）
func GetAs[T any](h *Helper, ctx context.Context, key string) (T, bool) {
	var zero T
	if h.client == nil {
		return zero, false
	}
	start := time.Now()
	data, err := h.client.Get(ctx, key).Bytes()
	metrics.CacheOperationDuration.WithLabelValues("get", "redis").Observe(time.Since(start).Seconds())
	if err != nil {
		metrics.CacheMissesTotal.WithLabelValues("redis").Inc()
		return zero, false
	}
	var v T
	if err := json.Unmarshal(data, &v); err != nil {
		return zero, false
	}
	metrics.CacheHitsTotal.WithLabelValues("redis").Inc()
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
	start := time.Now()
	defer func() {
		metrics.CacheOperationDuration.WithLabelValues("set", "redis").Observe(time.Since(start).Seconds())
	}()
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
// 注意：优先使用 InvalidateByVersion 代替此方法，避免 SCAN 的性能问题
func (h *Helper) Invalidate(ctx context.Context, prefix string) error {
	if h.client == nil {
		return nil
	}

	// 添加超时保护，防止 SCAN 长时间阻塞
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

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

// VersionKey 返回版本号的 Redis key
func VersionKey(prefix string) string {
	return "cache:version:" + prefix
}

// GetVersion 获取缓存版本号
func (h *Helper) GetVersion(ctx context.Context, prefix string) string {
	if h.client == nil {
		return "0"
	}
	version, err := h.client.Get(ctx, VersionKey(prefix)).Result()
	if err != nil {
		return "0"
	}
	return version
}

// BuildVersionedKey 构建带版本号的缓存 key
func (h *Helper) BuildVersionedKey(ctx context.Context, prefix, key string) string {
	version := h.GetVersion(ctx, prefix)
	return prefix + ":v" + version + ":" + key
}

// InvalidateByVersion 通过递增版本号使缓存失效
// 旧的缓存 key 会根据 TTL 自然过期，避免 SCAN 操作的性能问题
func (h *Helper) InvalidateByVersion(ctx context.Context, prefix string) error {
	if h.client == nil {
		return nil
	}

	versionKey := VersionKey(prefix)
	newVersion, err := h.client.Incr(ctx, versionKey).Result()
	if err != nil {
		logger.L().Warn("failed to increment cache version",
			zap.String("prefix", prefix),
			zap.Error(err),
		)
		return err
	}

	// 设置版本号 key 的过期时间，防止无限增长
	if err := h.client.Expire(ctx, versionKey, VersionKeyTTL).Err(); err != nil {
		logger.L().Warn("failed to set version key expiry",
			zap.String("prefix", prefix),
			zap.Error(err),
		)
	}

	logger.L().Debug("cache invalidated by version increment",
		zap.String("prefix", prefix),
		zap.Int64("new_version", newVersion),
	)
	return nil
}
