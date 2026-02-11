package middleware

import (
	"net"
	"net/http"
	"net/url"
	"os"
	"runtime/debug"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
)

const (
	CtxKeyRequestID = "request_id"
)

// 敏感 query 参数黑名单（这些参数的值会被脱敏）
var sensitiveQueryParams = map[string]bool{
	"code":          true, // OAuth authorization code
	"token":         true,
	"access_token":  true,
	"refresh_token": true,
	"id_token":      true,
	"password":      true,
	"secret":        true,
	"key":           true,
	"api_key":       true,
	"apikey":        true,
	"auth":          true,
	"authorization": true,
	"credential":    true,
	"credentials":   true,
	"state":         true, // OAuth state parameter
	"nonce":         true, // OAuth nonce
	"session":       true,
	"session_id":    true,
}

// RequestIDMiddleware 注入请求 ID
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.NewString()
		}
		c.Set(CtxKeyRequestID, requestID)
		c.Header("X-Request-ID", requestID)
		c.Next()
	}
}

// RequestLogger 结构化访问日志中间件
func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		// 对 query string 中的敏感参数进行脱敏
		query := maskSensitiveQueryParams(c.Request.URL.RawQuery)

		// 注入带 request_id 的 logger 到 context
		requestID := getRequestID(c)
		reqLogger := logger.L().With(zap.String("request_id", requestID))
		logger.GinContext(c, reqLogger)

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()

		fields := []zap.Field{
			zap.String("request_id", requestID),
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.String("query", query),
			zap.Int("status", status),
			zap.Duration("latency", latency),
			zap.Int("size", c.Writer.Size()),
			zap.String("client_ip", c.ClientIP()),
			zap.String("user_agent", c.Request.UserAgent()),
		}

		// 添加用户 ID（如果存在）
		if userID, exists := c.Get(CtxKeyUserID); exists {
			fields = append(fields, zap.String("user_id", toString(userID)))
		}

		// 根据状态码选择日志级别
		switch {
		case status >= 500:
			reqLogger.Error("http_request", fields...)
		case status >= 400:
			reqLogger.Warn("http_request", fields...)
		default:
			reqLogger.Info("http_request", fields...)
		}
	}
}

// Recovery 恢复中间件，捕获 panic 并记录日志
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// 检查是否是断开的连接
				var brokenPipe bool
				if ne, ok := err.(*net.OpError); ok {
					if se, ok := ne.Err.(*os.SyscallError); ok {
						if strings.Contains(strings.ToLower(se.Error()), "broken pipe") ||
							strings.Contains(strings.ToLower(se.Error()), "connection reset by peer") {
							brokenPipe = true
						}
					}
				}

				stack := string(debug.Stack())
				requestID := getRequestID(c)

				logger.L().Error("panic_recovered",
					zap.String("request_id", requestID),
					zap.Any("error", err),
					zap.String("path", c.Request.URL.Path),
					zap.String("method", c.Request.Method),
					zap.String("stack", stack),
				)

				if brokenPipe {
					c.Abort()
					return
				}

				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"error": "internal server error",
				})
			}
		}()
		c.Next()
	}
}

func getRequestID(c *gin.Context) string {
	if id, exists := c.Get(CtxKeyRequestID); exists {
		if s, ok := id.(string); ok {
			return s
		}
	}
	return ""
}

func toString(v interface{}) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

// maskSensitiveQueryParams 对 query string 中的敏感参数进行脱敏
func maskSensitiveQueryParams(rawQuery string) string {
	if rawQuery == "" {
		return ""
	}

	values, err := url.ParseQuery(rawQuery)
	if err != nil {
		// 解析失败时返回固定的脱敏标记
		return "[parse_error]"
	}

	for key := range values {
		// 检查参数名是否在敏感参数黑名单中（不区分大小写）
		if sensitiveQueryParams[strings.ToLower(key)] {
			values.Set(key, "[REDACTED]")
		}
	}

	return values.Encode()
}
