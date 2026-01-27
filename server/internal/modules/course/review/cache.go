package review

import (
	"context"
	"encoding/json"
	"time"

	"go.uber.org/zap"

	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
)

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

func (h *Handler) invalidateCache(ctx context.Context, prefix string) error {
	if h.cache == nil {
		return nil
	}

	const maxKeysToDelete = 1000
	var keys []string
	var cursor uint64
	for {
		var batch []string
		var err error
		batch, cursor, err = h.cache.Scan(ctx, cursor, prefix+"*", 100).Result()
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

	pipe := h.cache.Pipeline()
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

func (h *Handler) count(ctx context.Context, query string) (int, error) {
	var total int
	if err := h.db.QueryRow(ctx, query).Scan(&total); err != nil {
		return 0, err
	}
	return total, nil
}
