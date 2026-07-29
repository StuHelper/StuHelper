package review

import (
	"context"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/StuHelper/StuHelper/server/internal/pkg/cache"
	"github.com/StuHelper/StuHelper/server/internal/pkg/logger"
	"github.com/StuHelper/StuHelper/server/internal/pkg/response"
)

func respondWithCachedData[T any](
	h *Handler,
	c *gin.Context,
	prefix string,
	key string,
	fetch func(context.Context) (T, error),
	wrap func(T) any,
	logMessage string,
	userMessage string,
	handleErr func(*gin.Context, error) bool,
) {
	ctx := c.Request.Context()
	cacheKey := h.cache.BuildVersionedKey(ctx, prefix, key)
	if cached, ok := h.cache.GetRaw(ctx, cacheKey); ok {
		response.Success(c, cached)
		return
	}

	value, err := fetch(ctx)
	if err != nil {
		if handleErr != nil && handleErr(c, err) {
			return
		}
		logger.FromGin(c).Error(logMessage, zap.Error(err))
		response.InternalError(c, userMessage)
		return
	}

	payload := any(value)
	if wrap != nil {
		payload = wrap(value)
	}
	if err := h.cache.Set(ctx, cacheKey, payload, cache.JitteredTTL(cache.DefaultTTL)); err != nil {
		logger.FromGin(c).Warn("failed to set cache", zap.Error(err))
	}
	response.Success(c, payload)
}
