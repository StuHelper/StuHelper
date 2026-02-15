package middleware

import (
	"errors"
	"strings"

	"go.uber.org/zap"

	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/jwt"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/response"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/token"
	"github.com/gin-gonic/gin"
)

// 上下文键名常量
const (
	CtxKeyUserID      = "user_id"
	CtxKeyUsername    = "username"
	CtxKeyEmail       = "email"
	CtxKeyDisplayName = "display_name"
	CtxKeyIsAdmin = "is_admin"
)

// Cookie 名称常量
const (
	CookieAccessToken  = "access_token"
	CookieRefreshToken = "refresh_token"
)

// AuthMiddleware 认证中间件
func AuthMiddleware(tokenService *token.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 优先从 Cookie 获取 token，其次从 Header
		tokenString := getTokenFromRequest(c)
		if tokenString == "" {
			response.Unauthorized(c, "missing authentication token")
			c.Abort()
			return
		}

		// 检查 token 是否在黑名单中
		isBlacklisted, err := tokenService.GetBlacklist().IsBlacklisted(c.Request.Context(), tokenString)
		if err != nil {
			response.ServiceUnavailable(c, "service temporarily unavailable")
			c.Abort()
			return
		}
		if isBlacklisted {
			response.Unauthorized(c, "token has been revoked")
			c.Abort()
			return
		}

		// 使用增强的 JWT 验证器验证 token（校验 iss/aud/alg/exp）
		claims, err := tokenService.ValidateToken(tokenString)
		if err != nil {
			// 仅在服务端日志记录具体错误类型，避免向客户端泄露验证细节
			logger.L().Debug("JWT validation failed",
				zap.String("reason", classifyJWTError(err)),
				zap.Error(err),
			)

			response.Unauthorized(c, "invalid or expired token")
			c.Abort()
			return
		}

		// 将用户信息注入到上下文
		c.Set(CtxKeyUserID, claims.GetUserID())
		c.Set(CtxKeyUsername, claims.GetUsername())
		c.Set(CtxKeyEmail, claims.Email)
		c.Set(CtxKeyDisplayName, claims.DisplayName)
		c.Set(CtxKeyIsAdmin, claims.IsAdmin)

		c.Next()
	}
}

// GetUserID 从上下文获取用户 ID
func GetUserID(c *gin.Context) string {
	return getContextString(c, CtxKeyUserID)
}

// GetUsername 从上下文获取用户名
func GetUsername(c *gin.Context) string {
	return getContextString(c, CtxKeyUsername)
}

// GetEmail 从上下文获取邮箱
func GetEmail(c *gin.Context) string {
	return getContextString(c, CtxKeyEmail)
}

// GetDisplayName 从上下文获取显示名称
func GetDisplayName(c *gin.Context) string {
	return getContextString(c, CtxKeyDisplayName)
}

// GetIsAdmin 从上下文获取管理员状态
func GetIsAdmin(c *gin.Context) bool {
	if val, exists := c.Get(CtxKeyIsAdmin); exists {
		if b, ok := val.(bool); ok {
			return b
		}
	}
	return false
}

// getContextString 从上下文获取字符串值的通用函数
func getContextString(c *gin.Context, key string) string {
	if val, exists := c.Get(key); exists {
		if s, ok := val.(string); ok {
			return s
		}
	}
	return ""
}

// getTokenFromRequest 从请求中获取 access token。
//
// Token 来源优先级：Cookie > Authorization Header
//   1. Cookie（access_token）：浏览器端自动携带，CSRF 中间件配合防护
//   2. Authorization Header（Bearer <token>）：供非浏览器客户端（移动端、API 调试工具）使用
//
// Cookie 优先的原因：浏览器场景下 HttpOnly Cookie 比 localStorage 存储的 Bearer token 更安全，
// 不受 XSS 攻击窃取；同时 Cookie 由服务端 Set-Cookie 自动管理，前端无需手动处理 token 存储。
func getTokenFromRequest(c *gin.Context) string {
	// 1. 优先从 Cookie 获取（浏览器客户端）
	if token, err := c.Cookie(CookieAccessToken); err == nil && token != "" {
		return token
	}

	// 2. 其次从 Authorization Header 获取（非浏览器客户端）
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return ""
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}

	return strings.TrimSpace(parts[1])
}

// classifyJWTError 将 JWT 错误分类为日志友好的字符串（仅用于服务端日志，不暴露给客户端）
func classifyJWTError(err error) string {
	switch {
	case errors.Is(err, jwt.ErrTokenExpired):
		return "token_expired"
	case errors.Is(err, jwt.ErrTokenNotYetValid):
		return "token_not_yet_valid"
	case errors.Is(err, jwt.ErrInvalidIssuer):
		return "invalid_issuer"
	case errors.Is(err, jwt.ErrInvalidAudience):
		return "invalid_audience"
	case errors.Is(err, jwt.ErrAlgorithmNotAllowed):
		return "algorithm_not_allowed"
	case errors.Is(err, jwt.ErrInvalidSignature):
		return "invalid_signature"
	default:
		return "unknown"
	}
}
