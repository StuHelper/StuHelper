package middleware

import (
	"context"
	"net/http"

	"go.uber.org/zap"

	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/errs"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/response"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/sso"
	"github.com/gin-gonic/gin"
)

// getCachedUserOrAbort 获取缓存用户信息，失败时根据错误类型返回不同响应。
// 返回 nil 表示已 abort，调用方应直接 return。
func getCachedUserOrAbort(c *gin.Context, ssoClient *sso.Client, userID, operation string) *sso.CachedUser {
	user, err := ssoClient.GetCachedUserByID(c.Request.Context(), userID)
	if err != nil {
		// 区分 context 取消（客户端断开）与超时
		ctxErr := c.Request.Context().Err()
		if ctxErr == context.Canceled {
			logger.L().Debug("permission check aborted: client disconnected",
				zap.String("operation", operation),
				zap.String("user_id", userID),
			)
			c.Abort()
			return nil
		}
		if ctxErr == context.DeadlineExceeded {
			logger.L().Warn("permission check timed out",
				zap.String("operation", operation),
				zap.String("user_id", userID),
				zap.Error(err),
			)
			response.Error(c, http.StatusGatewayTimeout, errs.ErrSSOTimeout, "identity service request timed out")
			c.Abort()
			return nil
		}
		// SSO 服务不可用：fail-closed（拒绝访问）+ 记录日志
		logger.L().Error("permission check failed: SSO unavailable",
			zap.String("operation", operation),
			zap.String("user_id", userID),
			zap.Error(err),
		)
		response.ServiceUnavailable(c, "identity service temporarily unavailable")
		c.Abort()
		return nil
	}
	return user
}

// RequireRole 角色检查中间件 - 要求用户拥有指定角色
func RequireRole(ssoClient *sso.Client, roleName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := GetUserID(c)
		if userID == "" {
			abortUnauthorized(c)
			return
		}

		user := getCachedUserOrAbort(c, ssoClient, userID, "require_role")
		if user == nil {
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
		userID := GetUserID(c)
		if userID == "" {
			abortUnauthorized(c)
			return
		}

		user := getCachedUserOrAbort(c, ssoClient, userID, "require_any_role")
		if user == nil {
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
		userID := GetUserID(c)
		if userID == "" {
			abortUnauthorized(c)
			return
		}

		user := getCachedUserOrAbort(c, ssoClient, userID, "require_permission")
		if user == nil {
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
		userID := GetUserID(c)
		if userID == "" {
			abortUnauthorized(c)
			return
		}

		user := getCachedUserOrAbort(c, ssoClient, userID, "require_admin")
		if user == nil {
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
		userID := GetUserID(c)
		if userID == "" {
			abortUnauthorized(c)
			return
		}

		allowed, err := ssoClient.Enforce(
			permissionId,
			ssoClient.GetOrganization(),
			userID,
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
	response.Unauthorized(c, "unauthorized", errs.ErrTokenMissing)
	c.Abort()
}

func abortForbidden(c *gin.Context) {
	response.Forbidden(c, "forbidden", errs.ErrAccessDenied)
	c.Abort()
}

func abortInternalError(c *gin.Context, message string) {
	response.InternalError(c, message)
	c.Abort()
}
