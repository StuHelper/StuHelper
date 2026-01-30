package course

import (
	"context"
	"encoding/json"
	"time"

	"go.uber.org/zap"

	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
)

const cacheTTL = 5 * time.Minute

// cacheVersionKey 返回版本号的 Redis key
func cacheVersionKey(prefix string) string {
	return "cache:version:" + prefix
}

// getCacheVersion 获取缓存版本号
func (h *Handler) getCacheVersion(ctx context.Context, prefix string) string {
	if h.cache == nil {
		return "0"
	}
	version, err := h.cache.Get(ctx, cacheVersionKey(prefix)).Result()
	if err != nil {
		return "0"
	}
	return version
}

// buildCacheKey 构建带版本号的缓存 key
func (h *Handler) buildCacheKey(ctx context.Context, prefix, key string) string {
	version := h.getCacheVersion(ctx, prefix)
	return prefix + ":v" + version + ":" + key
}

func (h *Handler) getCache(ctx context.Context, key string) (interface{}, bool) {
	if h.cache == nil {
		return nil, false
	}
	data, err := h.cache.Get(ctx, key).Bytes()
	if err != nil {
		return nil, false
	}
	var v interface{}
	if err := json.Unmarshal(data, &v); err != nil {
		return nil, false
	}
	return v, true
}

func (h *Handler) setCache(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	if h.cache == nil {
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
	if err := h.cache.Set(ctx, key, data, ttl).Err(); err != nil {
		logger.L().Warn("failed to set cache",
			zap.String("key", key),
			zap.Error(err),
		)
		return err
	}
	return nil
}

// invalidateCache 通过递增版本号使缓存失效
// 旧的缓存 key 会根据 TTL 自然过期，避免 SCAN 操作的性能问题
func (h *Handler) invalidateCache(ctx context.Context, prefix string) error {
	if h.cache == nil {
		return nil
	}

	versionKey := cacheVersionKey(prefix)
	newVersion, err := h.cache.Incr(ctx, versionKey).Result()
	if err != nil {
		logger.L().Warn("failed to increment cache version",
			zap.String("prefix", prefix),
			zap.Error(err),
		)
		return err
	}

	// 设置版本号 key 的过期时间，防止无限增长
	// 使用较长的 TTL，确保在缓存有效期内版本号不会过期
	if err := h.cache.Expire(ctx, versionKey, 24*time.Hour).Err(); err != nil {
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
