package middleware

import (
	"net/http"
	"strings"

	"github.com/casdoor/casdoor-go-sdk/casdoorsdk"
	"github.com/gin-gonic/gin"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/token"
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

		// 使用 Casdoor SDK 验证 token
		claims, err := casdoorsdk.ParseJwtToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "invalid or expired token",
			})
			c.Abort()
			return
		}

		// 将用户信息注入到上下文
		c.Set(CtxKeyUserID, claims.User.Id)
		c.Set(CtxKeyUsername, claims.User.Name)
		c.Set(CtxKeyEmail, claims.User.Email)
		c.Set(CtxKeyDisplayName, claims.User.DisplayName)
		c.Set(CtxKeyAccessToken, tokenString)

		c.Next()
	}
}

// GetUserID 从上下文获取用户 ID
func GetUserID(c *gin.Context) string {
	if userID, exists := c.Get(CtxKeyUserID); exists {
		if id, ok := userID.(string); ok {
			return id
		}
	}
	return ""
}

// GetUsername 从上下文获取用户名
func GetUsername(c *gin.Context) string {
	if username, exists := c.Get(CtxKeyUsername); exists {
		if name, ok := username.(string); ok {
			return name
		}
	}
	return ""
}

// GetEmail 从上下文获取邮箱
func GetEmail(c *gin.Context) string {
	if email, exists := c.Get(CtxKeyEmail); exists {
		if e, ok := email.(string); ok {
			return e
		}
	}
	return ""
}

// GetDisplayName 从上下文获取显示名称
func GetDisplayName(c *gin.Context) string {
	if displayName, exists := c.Get(CtxKeyDisplayName); exists {
		if name, ok := displayName.(string); ok {
			return name
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
