package middleware

import (
	"context"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/errs"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/response"
)

// InternalUserIDResolver 根据 external user ID 解析内部 user ID。
type InternalUserIDResolver func(ctx context.Context, externalID string) (int64, error)

// ResolveRequiredInternalUserID 从认证上下文读取 external user ID 并解析内部 user ID。
// 缺少认证信息时统一返回 401；解析失败时统一返回 500。
func ResolveRequiredInternalUserID(
	c *gin.Context,
	resolver InternalUserIDResolver,
	failureMessage string,
) (int64, bool) {
	externalID := GetUserID(c)
	if externalID == "" {
		response.Unauthorized(c, "authentication required", errs.ErrLoginRequired)
		return 0, false
	}

	userID, err := resolver(c.Request.Context(), externalID)
	if err != nil {
		logger.FromGin(c).Error("failed to resolve user ID",
			zap.String("external_id", externalID),
			zap.Error(err),
		)
		response.InternalError(c, failureMessage)
		return 0, false
	}

	return userID, true
}
