package rbac

import (
	"github.com/gin-gonic/gin"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/errs"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/middleware"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/response"
)

// RequireCapability 检查当前用户是否持有指定能力。
// 能力由 AuthMiddleware 从 Token 角色展开后注入 Gin context，零 DB 查询。
func RequireCapability(capName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !middleware.HasCapability(c, capName) {
			response.Forbidden(c, "insufficient permissions", errs.ErrPermissionDenied)
			c.Abort()
			return
		}
		c.Next()
	}
}

// RequireAnyCapability 检查当前用户是否持有任一指定能力（O(n) — 利用 capability set 的 O(1) 查找）
func RequireAnyCapability(capNames ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		for _, required := range capNames {
			if middleware.HasCapability(c, required) {
				c.Next()
				return
			}
		}
		response.Forbidden(c, "insufficient permissions", errs.ErrPermissionDenied)
		c.Abort()
	}
}
