package rbac

import (
	"context"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/errs"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/middleware"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/response"
)

// PermissionService defines the RBAC methods required by permission middleware.
type PermissionService interface {
	GetInternalUserID(ctx context.Context, externalID string) (int64, error)
	CheckPermission(ctx context.Context, userID int64, permName string, schoolID *string) (bool, error)
}

// RequirePermission RBAC 权限检查中间件
// 检查当前用户是否拥有指定权限名，支持 scope 限制（school, role）
func RequirePermission(service PermissionService, permName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		externalID := middleware.GetUserID(c)
		if externalID == "" {
			response.Unauthorized(c, "authentication required", errs.ErrLoginRequired)
			c.Abort()
			return
		}

		// 将 Casdoor external_id 转换为内部 user.id
		userID, err := service.GetInternalUserID(c.Request.Context(), externalID)
		if err != nil {
			logger.FromGin(c).Error("failed to resolve user ID for permission check",
				zap.String("external_id", externalID),
				zap.Error(err),
			)
			response.InternalError(c, "permission check failed")
			c.Abort()
			return
		}

		// 从请求上下文获取用户的 schoolID（如果适用）
		var schoolID *string
		if sid := c.Query("schoolID"); sid != "" {
			schoolID = &sid
		}

		allowed, err := service.CheckPermission(c.Request.Context(), userID, permName, schoolID)
		if err != nil {
			logger.FromGin(c).Error("permission check failed",
				zap.Int64("user_id", userID),
				zap.String("permission", permName),
				zap.Error(err),
			)
			response.InternalError(c, "permission check failed")
			c.Abort()
			return
		}
		if !allowed {
			response.Forbidden(c, "insufficient permissions", errs.ErrPermissionDenied)
			c.Abort()
			return
		}

		c.Next()
	}
}
