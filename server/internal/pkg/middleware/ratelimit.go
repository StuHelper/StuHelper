package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/response"
)

// RateLimitConfig 限流配置
type RateLimitConfig struct {
	// 全局限流（所有请求）
	GlobalLimit  int
	GlobalWindow time.Duration
	// IP 限流
	IPLimit  int
	IPWindow time.Duration
	// 用户限流（已认证用户）
	UserLimit  int
	UserWindow time.Duration
}

// DefaultRateLimitConfig 默认限流配置
func DefaultRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		GlobalLimit:  10000,            // 全局每分钟 10000 请求
		GlobalWindow: time.Minute,
		IPLimit:      100,              // 每 IP 每分钟 100 请求
		IPWindow:     time.Minute,
		UserLimit:    200,              // 每用户每分钟 200 请求
		UserWindow:   time.Minute,
	}
}

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
	// 生成唯一 member，避免毫秒内并发请求覆盖
	uniqueID := generateUniqueID()

	script := redis.NewScript(`
local key = KEYS[1]
local now = tonumber(ARGV[1])
local window = tonumber(ARGV[2])
local limit = tonumber(ARGV[3])
local member = ARGV[4]

redis.call('ZREMRANGEBYSCORE', key, 0, now - window)
local count = redis.call('ZCARD', key)
if count >= limit then
  return 0
end
redis.call('ZADD', key, now, member)
redis.call('EXPIRE', key, math.ceil(window/1000))
return 1
`)

	result, err := script.Run(ctx, rl.rdb, []string{key},
		time.Now().UnixMilli(),
		rl.window.Milliseconds(),
		rl.limit,
		uniqueID,
	).Int()
	if err != nil {
		return false, err
	}
	return result == 1, nil
}

// generateUniqueID 生成唯一标识符
func generateUniqueID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		// crypto/rand.Read 在现代 Go 中不会失败，但遵循错误处理规范
		panic("crypto/rand.Read failed: " + err.Error())
	}
	return hex.EncodeToString(b)
}

// RateLimitMiddleware 速率限制中间件
func RateLimitMiddleware(limiter *RedisRateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := "rl:" + c.ClientIP()
		allowed, err := limiter.Allow(c.Request.Context(), key)
		if err != nil {
			response.ServiceUnavailable(c, "rate limit service unavailable")
			c.Abort()
			return
		}
		if !allowed {
			response.RateLimitExceeded(c)
			c.Abort()
			return
		}
		c.Next()
	}
}

// GlobalRateLimitMiddleware 全局限流中间件
func GlobalRateLimitMiddleware(limiter *RedisRateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := "rl:global"
		allowed, err := limiter.Allow(c.Request.Context(), key)
		if err != nil {
			logger.L().Warn("global rate limit check failed, rejecting request (fail-closed)",
				zap.Error(err),
				zap.String("client_ip", c.ClientIP()),
			)
			response.ServiceUnavailable(c, "service temporarily unavailable")
			c.Abort()
			return
		}
		if !allowed {
			response.ServiceUnavailable(c, "service temporarily unavailable due to high load")
			c.Abort()
			return
		}
		c.Next()
	}
}

// UserRateLimitMiddleware 用户维度限流中间件
func UserRateLimitMiddleware(limiter *RedisRateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := GetUserID(c)
		if userID == "" {
			// 未认证用户使用 IP 限流
			c.Next()
			return
		}
		key := "rl:user:" + userID
		allowed, err := limiter.Allow(c.Request.Context(), key)
		if err != nil {
			logger.L().Warn("user rate limit check failed, rejecting request (fail-closed)",
				zap.Error(err),
				zap.String("user_id", userID),
			)
			response.ServiceUnavailable(c, "service temporarily unavailable")
			c.Abort()
			return
		}
		if !allowed {
			response.RateLimitExceeded(c)
			c.Abort()
			return
		}
		c.Next()
	}
}

// EndpointRateLimitMiddleware 端点限流中间件（用于敏感操作）
func EndpointRateLimitMiddleware(limiter *RedisRateLimiter, endpoint string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := GetUserID(c)
		var key string
		if userID != "" {
			key = "rl:endpoint:" + endpoint + ":user:" + userID
		} else {
			key = "rl:endpoint:" + endpoint + ":ip:" + c.ClientIP()
		}
		allowed, err := limiter.Allow(c.Request.Context(), key)
		if err != nil {
			logger.L().Warn("endpoint rate limit check failed, rejecting request (fail-closed)",
				zap.Error(err),
				zap.String("endpoint", endpoint),
			)
			response.ServiceUnavailable(c, "service temporarily unavailable")
			c.Abort()
			return
		}
		if !allowed {
			response.RateLimitExceeded(c)
			c.Abort()
			return
		}
		c.Next()
	}
}
