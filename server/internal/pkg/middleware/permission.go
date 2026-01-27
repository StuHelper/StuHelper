package middleware

import (
	"net/http"

	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/sso"
	"github.com/gin-gonic/gin"
)

// RequireRole 角色检查中间件 - 要求用户拥有指定角色
func RequireRole(ssoClient *sso.Client, roleName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		username := GetUsername(c)
		if username == "" {
			abortUnauthorized(c)
			return
		}

		// 使用缓存获取用户信息
		user, err := ssoClient.GetCachedUser(c.Request.Context(), username)
		if err != nil {
			abortInternalError(c, "failed to check role")
			return
		}

		if !user.HasRole(roleName) {
			abortForbidden(c)
			return
		}

		c.Next()
	}
}

// RequireAnyRole 角色检查中间件 - 要求用户拥有任一指定角色
func RequireAnyRole(ssoClient *sso.Client, roleNames ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		username := GetUsername(c)
		if username == "" {
			abortUnauthorized(c)
			return
		}

		// 使用缓存获取用户信息
		user, err := ssoClient.GetCachedUser(c.Request.Context(), username)
		if err != nil {
			abortInternalError(c, "failed to check roles")
			return
		}

		if !user.HasAnyRole(roleNames...) {
			abortForbidden(c)
			return
		}

		c.Next()
	}
}

// RequirePermission 权限检查中间件 - 要求用户拥有指定权限
func RequirePermission(ssoClient *sso.Client, permissionName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		username := GetUsername(c)
		if username == "" {
			abortUnauthorized(c)
			return
		}

		// 使用缓存获取用户信息
		user, err := ssoClient.GetCachedUser(c.Request.Context(), username)
		if err != nil {
			abortInternalError(c, "failed to check permission")
			return
		}

		if !user.HasPermission(permissionName) {
			abortForbidden(c)
			return
		}

		c.Next()
	}
}

// RequireAdmin 管理员检查中间件
func RequireAdmin(ssoClient *sso.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		username := GetUsername(c)
		if username == "" {
			abortUnauthorized(c)
			return
		}

		// 使用缓存获取用户信息
		user, err := ssoClient.GetCachedUser(c.Request.Context(), username)
		if err != nil {
			abortInternalError(c, "failed to check admin status")
			return
		}

		if !user.IsAdmin {
			abortForbidden(c)
			return
		}

		c.Next()
	}
}

// CasbinEnforce Casbin 权限检查中间件
// permissionId: Casdoor 中配置的权限ID
// obj: 资源, act: 操作
func CasbinEnforce(ssoClient *sso.Client, permissionId, obj, act string) gin.HandlerFunc {
	return func(c *gin.Context) {
		username := GetUsername(c)
		if username == "" {
			abortUnauthorized(c)
			return
		}

		allowed, err := ssoClient.Enforce(
			permissionId,
			ssoClient.GetOrganization(),
			username,
			obj,
			act,
		)
		if err != nil {
			abortInternalError(c, "failed to enforce permission")
			return
		}

		if !allowed {
			abortForbidden(c)
			return
		}

		c.Next()
	}
}

// 辅助函数
func abortUnauthorized(c *gin.Context) {
	c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
	c.Abort()
}

func abortForbidden(c *gin.Context) {
	c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
	c.Abort()
}

func abortInternalError(c *gin.Context, message string) {
	c.JSON(http.StatusInternalServerError, gin.H{"error": message})
	c.Abort()
}
