package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// RateLimiter 简单的内存速率限制器
type RateLimiter struct {
	requests map[string][]time.Time
	mu       sync.RWMutex
	limit    int
	window   time.Duration
}

// NewRateLimiter 创建速率限制器
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		requests: make(map[string][]time.Time),
		limit:    limit,
		window:   window,
	}
	// 定期清理过期记录
	go rl.cleanup()
	return rl
}

// Allow 检查是否允许请求
func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-rl.window)

	// 过滤掉窗口外的请求
	var valid []time.Time
	for _, t := range rl.requests[key] {
		if t.After(windowStart) {
			valid = append(valid, t)
		}
	}

	if len(valid) >= rl.limit {
		rl.requests[key] = valid
		return false
	}

	rl.requests[key] = append(valid, now)
	return true
}

// cleanup 定期清理过期记录
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(rl.window)
	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		windowStart := now.Add(-rl.window)
		for key, times := range rl.requests {
			var valid []time.Time
			for _, t := range times {
				if t.After(windowStart) {
					valid = append(valid, t)
				}
			}
			if len(valid) == 0 {
				delete(rl.requests, key)
			} else {
				rl.requests[key] = valid
			}
		}
		rl.mu.Unlock()
	}
}

// RateLimitMiddleware 速率限制中间件
func RateLimitMiddleware(limiter *RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.ClientIP()
		if !limiter.Allow(key) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "too many requests, please try again later",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}
