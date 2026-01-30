package middleware

import (
	"errors"
	"net/http"
	"strings"

	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/jwt"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/token"
	"github.com/gin-gonic/gin"
)

// 上下文键名常量
const (
	CtxKeyUserID      = "user_id"
	CtxKeyUsername    = "username"
	CtxKeyEmail       = "email"
	CtxKeyDisplayName = "display_name"
	CtxKeyAccessToken = "access_token"
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
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "missing authentication token",
			})
			c.Abort()
			return
		}

		// 检查 token 是否在黑名单中
		isBlacklisted, err := tokenService.GetBlacklist().IsBlacklisted(c.Request.Context(), tokenString)
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": "service temporarily unavailable",
			})
			c.Abort()
			return
		}
		if isBlacklisted {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "token has been revoked",
			})
			c.Abort()
			return
		}

		// 使用增强的 JWT 验证器验证 token（校验 iss/aud/alg/exp）
		claims, err := tokenService.ValidateToken(tokenString)
		if err != nil {
			errorMsg := "invalid or expired token"

			// 根据错误类型返回更具体的错误信息
			switch {
			case errors.Is(err, jwt.ErrTokenExpired):
				errorMsg = "token has expired"
			case errors.Is(err, jwt.ErrTokenNotYetValid):
				errorMsg = "token not yet valid"
			case errors.Is(err, jwt.ErrInvalidIssuer):
				errorMsg = "invalid token issuer"
			case errors.Is(err, jwt.ErrInvalidAudience):
				errorMsg = "invalid token audience"
			case errors.Is(err, jwt.ErrAlgorithmNotAllowed):
				errorMsg = "token algorithm not allowed"
			case errors.Is(err, jwt.ErrInvalidSignature):
				errorMsg = "invalid token signature"
			}

			c.JSON(http.StatusUnauthorized, gin.H{
				"error": errorMsg,
			})
			c.Abort()
			return
		}

		// 将用户信息注入到上下文
		c.Set(CtxKeyUserID, claims.GetUserID())
		c.Set(CtxKeyUsername, claims.GetUsername())
		c.Set(CtxKeyEmail, claims.Email)
		c.Set(CtxKeyDisplayName, claims.DisplayName)
		c.Set(CtxKeyAccessToken, tokenString)

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

// getContextString 从上下文获取字符串值的通用函数
func getContextString(c *gin.Context, key string) string {
	if val, exists := c.Get(key); exists {
		if s, ok := val.(string); ok {
			return s
		}
	}
	return ""
}

// getTokenFromRequest 从请求中获取 token（优先 Cookie，其次 Header）
func getTokenFromRequest(c *gin.Context) string {
	// 优先从 Cookie 获取
	if token, err := c.Cookie(CookieAccessToken); err == nil && token != "" {
		return token
	}

	// 其次从 Authorization Header 获取
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
