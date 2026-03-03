package middleware

import (
	"net"
	"net/url"
	"os"
	"regexp"
	"runtime/debug"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/response"
)

const (
	CtxKeyRequestID = "request_id"
	// maxRequestIDLen X-Request-ID 最大允许长度
	maxRequestIDLen = 128
)

// validRequestID 仅允许字母、数字和连字符（排除点号和下划线以减少注入面）
var validRequestID = regexp.MustCompile(`^[a-zA-Z0-9\-]+$`)

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
// 客户端提供的 X-Request-ID 必须通过长度和字符校验，否则生成新 UUID
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" || len(requestID) > maxRequestIDLen || !validRequestID.MatchString(requestID) {
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
		requestID := GetRequestID(c)
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
			zap.String("user_agent", truncateString(c.Request.UserAgent(), 256)),
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
				requestID := GetRequestID(c)

				// 按行拆分栈追踪，便于日志聚合系统解析
				stackLines := strings.Split(stack, "\n")

				logger.L().Error("panic_recovered",
					zap.String("request_id", requestID),
					zap.Any("error", err),
					zap.String("path", c.Request.URL.Path),
					zap.String("method", c.Request.Method),
					zap.Strings("stack", stackLines),
				)

				if brokenPipe {
					c.Abort()
					return
				}

				response.InternalError(c, "internal server error")
				c.Abort()
			}
		}()
		c.Next()
	}
}

// GetRequestID 从 Gin context 中提取请求 ID
func GetRequestID(c *gin.Context) string {
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

// truncateString 按 rune 截断字符串，避免切断多字节 UTF-8 字符
func truncateString(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen])
}

// maskSensitiveQueryParams 对 query string 中的敏感参数进行脱敏
func maskSensitiveQueryParams(rawQuery string) string {
	if rawQuery == "" {
		return ""
	}

	values, err := url.ParseQuery(rawQuery)
	if err != nil {
		// 解析失败时记录警告并返回固定的脱敏标记
		logger.L().Warn("failed to parse query string for sensitive param masking",
			zap.String("raw_query", truncateString(rawQuery, 128)),
			zap.Error(err),
		)
		return "[parse_error]"
	}

	for key := range values {
		// 检查参数名是否在敏感参数黑名单中（不区分大小写）
		lowerKey := strings.ToLower(key)
		// M-67: 同时匹配 token 和 token[] 形式的数组参数
		baseKey := strings.TrimSuffix(lowerKey, "[]")
		if sensitiveQueryParams[lowerKey] || sensitiveQueryParams[baseKey] {
			// 遍历所有值（处理 ?token[]=xxx&token[]=yyy 数组形式）
			redacted := make([]string, len(values[key]))
			for i := range redacted {
				redacted[i] = "[REDACTED]"
			}
			values[key] = redacted
		}
	}

	return values.Encode()
}
