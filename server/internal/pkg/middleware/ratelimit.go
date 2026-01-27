package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// RedisRateLimiter 基于 Redis 的速率限制器
type RedisRateLimiter struct {
	rdb    *redis.Client
	limit  int
	window time.Duration
}

// NewRedisRateLimiter 创建 Redis 速率限制器
func NewRedisRateLimiter(rdb *redis.Client, limit int, window time.Duration) *RedisRateLimiter {
	return &RedisRateLimiter{
		rdb:    rdb,
		limit:  limit,
		window: window,
	}
}

// Allow 检查是否允许请求（滑动窗口）
func (rl *RedisRateLimiter) Allow(ctx context.Context, key string) (bool, error) {
	script := redis.NewScript(`
local key = KEYS[1]
local now = tonumber(ARGV[1])
local window = tonumber(ARGV[2])
local limit = tonumber(ARGV[3])

redis.call('ZREMRANGEBYSCORE', key, 0, now - window)
local count = redis.call('ZCARD', key)
if count >= limit then
  return 0
end
redis.call('ZADD', key, now, now)
redis.call('EXPIRE', key, math.ceil(window/1000))
return 1
`)

	result, err := script.Run(ctx, rl.rdb, []string{key}, time.Now().UnixMilli(), rl.window.Milliseconds(), rl.limit).Int()
	if err != nil {
		return false, err
	}
	return result == 1, nil
}

// RateLimitMiddleware 速率限制中间件
func RateLimitMiddleware(limiter *RedisRateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := "rl:" + c.ClientIP()
		allowed, err := limiter.Allow(c.Request.Context(), key)
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": "rate limit service unavailable",
			})
			c.Abort()
			return
		}
		if !allowed {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "too many requests, please try again later",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}
