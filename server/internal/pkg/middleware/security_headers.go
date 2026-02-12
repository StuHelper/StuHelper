package middleware

import (
	"errors"
	"io"
	"net/http"

	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/response"
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

// auditedBody wraps http.MaxBytesReader to log when the body limit is exceeded,
// covering the case where Content-Length is absent and the early check is bypassed.
type auditedBody struct {
	io.ReadCloser
	logged   bool
	c        *gin.Context
	maxBytes int64
}

func (b *auditedBody) Read(p []byte) (n int, err error) {
	n, err = b.ReadCloser.Read(p)
	if err != nil && !b.logged {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			b.logged = true
			requestID := getRequestID(b.c)
			logger.L().Warn("request body exceeded limit (no Content-Length)",
				zap.String("request_id", requestID),
				zap.String("client_ip", b.c.ClientIP()),
				zap.String("method", b.c.Request.Method),
				zap.String("path", b.c.Request.URL.Path),
				zap.Int64("max_bytes", b.maxBytes),
				zap.String("user_agent", b.c.Request.UserAgent()),
			)
		}
	}
	return
}

// MaxBodySize 限制请求体大小的中间件
func MaxBodySize(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.ContentLength > maxBytes {
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
			response.Error(c, http.StatusRequestEntityTooLarge, "REQUEST_TOO_LARGE", "request body too large")
			c.Abort()
			return
		}
		c.Request.Body = &auditedBody{
			ReadCloser: http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes),
			c:          c,
			maxBytes:   maxBytes,
		}
		c.Next()
	}
}
