package middleware

import (
	"net/http"

	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// SecurityHeadersMiddleware 添加安全响应头
func SecurityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		// c.Header("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
		// API 服务器的 CSP 策略：只允许 JSON 响应，禁止脚本和样式
		c.Header("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
		c.Next()
	}
}

// SecurityHeadersWithHSTS 添加安全响应头（包含 HSTS，用于生产环境）
func SecurityHeadersWithHSTS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("Referrer-Policy", "no-referrer")
		c.Header("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
		c.Header("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
		// HSTS: 强制 HTTPS，有效期 1 年，包含子域名
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		// CORP: 限制资源只能被同源页面加载
		c.Header("Cross-Origin-Resource-Policy", "same-origin")
		// COOP: 隔离浏览上下文，防止跨源攻击
		c.Header("Cross-Origin-Opener-Policy", "same-origin")
		c.Next()
	}
}

// MaxBodySize 限制请求体大小的中间件
func MaxBodySize(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.ContentLength > maxBytes {
			// 记录请求体过大的安全审计日志
			requestID := getRequestID(c)
			logger.L().Warn("request body too large",
				zap.String("request_id", requestID),
				zap.String("client_ip", c.ClientIP()),
				zap.String("method", c.Request.Method),
				zap.String("path", c.Request.URL.Path),
				zap.Int64("content_length", c.Request.ContentLength),
				zap.Int64("max_bytes", maxBytes),
				zap.String("user_agent", c.Request.UserAgent()),
			)
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{
				"error": "request body too large",
			})
			c.Abort()
			return
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		c.Next()
	}
}
