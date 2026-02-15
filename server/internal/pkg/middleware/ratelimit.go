package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	mrand "math/rand/v2"
	"net"
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

// slidingWindowScript 滑动窗口限流 Lua 脚本（包级变量，仅初始化一次）
// 注意: ZREMRANGEBYSCORE 无 LIMIT 参数，对大集合为 O(N)。
// 由于限流 key 的成员数受 limit 参数约束（通常 ≤ 10000），不会出现大集合问题。
var slidingWindowScript = redis.NewScript(`
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
redis.call('PEXPIRE', key, math.max(1, window))
return 1
`)

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
	// 检查 context 是否已取消
	select {
	case <-ctx.Done():
		return false, ctx.Err()
	default:
	}

	// 生成唯一 member，避免毫秒内并发请求覆盖
	uniqueID := generateUniqueID()

	// 独立超时 context，防止 Redis 慢响应阻塞请求
	scriptCtx, scriptCancel := context.WithTimeout(ctx, 5*time.Second)
	defer scriptCancel()

	result, err := slidingWindowScript.Run(scriptCtx, rl.rdb, []string{key},
		time.Now().UnixMilli(),
		rl.window.Milliseconds(),
		rl.limit,
		uniqueID,
	).Int()
	if err != nil {
		// 区分 context 取消和 Redis 错误
		if ctx.Err() != nil {
			return false, ctx.Err()
		}
		return false, err
	}
	return result == 1, nil
}

// generateUniqueID 生成唯一标识符
// crypto/rand 失败时降级为 math/rand，避免 panic 导致服务中断
func generateUniqueID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		logger.L().Warn("crypto/rand.Read failed, falling back to math/rand", zap.Error(err))
		return fmt.Sprintf("%d", mrand.Int64())
	}
	return hex.EncodeToString(b)
}

// extractIP 从 ClientIP() 返回值中提取纯 IP 地址
// ClientIP() 可能返回 "IP:Port" 格式，使用 net.SplitHostPort 提取纯 IP
func extractIP(raw string) string {
	host, _, err := net.SplitHostPort(raw)
	if err != nil {
		// 无端口的纯 IP 地址，SplitHostPort 会报错，直接返回原值
		return raw
	}
	return host
}

// RateLimitMiddleware 速率限制中间件
func RateLimitMiddleware(limiter *RedisRateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := "rl:" + extractIP(c.ClientIP())
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
			logger.L().Warn("redis unavailable, rejecting request (fail-closed)",
				zap.String("client_ip", c.ClientIP()),
			)
			logger.L().Debug("global rate limit redis error detail", zap.Error(err))
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
			logger.L().Warn("redis unavailable, rejecting request (fail-closed)",
				zap.String("user_id", userID),
			)
			logger.L().Debug("user rate limit redis error detail", zap.Error(err))
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
			key = "rl:endpoint:" + endpoint + ":ip:" + extractIP(c.ClientIP())
		}
		allowed, err := limiter.Allow(c.Request.Context(), key)
		if err != nil {
			logger.L().Warn("redis unavailable, rejecting request (fail-closed)",
				zap.String("endpoint", endpoint),
			)
			logger.L().Debug("endpoint rate limit redis error detail", zap.Error(err))
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
