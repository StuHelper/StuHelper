package middleware

import (
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/StuHelper/StuHelper/server/internal/pkg/errs"
	"github.com/StuHelper/StuHelper/server/internal/pkg/logger"
	"github.com/StuHelper/StuHelper/server/internal/pkg/response"
)

// SecurityHeadersMiddleware 添加安全响应头
func SecurityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		applySecurityHeaders(c, false)
		c.Next()
	}
}

// SecurityHeadersWithHSTS 添加安全响应头（包含 HSTS，用于生产环境）
func SecurityHeadersWithHSTS() gin.HandlerFunc {
	return func(c *gin.Context) {
		applySecurityHeaders(c, true)
		c.Next()
	}
}

func applySecurityHeaders(c *gin.Context, includeHSTS bool) {
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("X-Frame-Options", "DENY")
	c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
	c.Header("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
	c.Header("X-Permitted-Cross-Domain-Policies", "none")
	c.Header("Cross-Origin-Resource-Policy", "same-origin")
	c.Header("Cross-Origin-Opener-Policy", "same-origin")

	// Swagger UI 需要加载脚本和样式，使用宽松 CSP
	if strings.HasPrefix(c.Request.URL.Path, "/docs") {
		c.Header("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; frame-ancestors 'none'")
	} else {
		// API 服务器的 CSP 策略：只允许 JSON 响应，禁止脚本和样式
		c.Header("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
	}

	if includeHSTS {
		// HSTS: 强制 HTTPS，有效期 1 年，包含子域名
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
	}
}

// auditedBody 包装 http.MaxBytesReader，在请求体超限时补充日志。
// 这能覆盖缺少 Content-Length、从而绕过前置大小检查的场景。
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
			requestID := GetRequestID(b.c)
			logger.L().Warn("request body exceeded limit (no Content-Length)",
				zap.String("request_id", requestID),
				zap.String("client_ip", b.c.ClientIP()),
				zap.String("method", b.c.Request.Method),
				zap.String("path", requestLogRoute(b.c)),
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
			requestID := GetRequestID(c)
			logger.L().Warn("request body too large",
				zap.String("request_id", requestID),
				zap.String("client_ip", c.ClientIP()),
				zap.String("method", c.Request.Method),
				zap.String("path", requestLogRoute(c)),
				zap.Int64("content_length", c.Request.ContentLength),
				zap.Int64("max_bytes", maxBytes),
				zap.String("user_agent", c.Request.UserAgent()),
			)
			response.Error(c, http.StatusRequestEntityTooLarge, errs.ErrPayloadTooLarge, "request body too large")
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
