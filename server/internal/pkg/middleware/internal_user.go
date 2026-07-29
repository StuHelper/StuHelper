package middleware

import (
	"context"
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"

	"github.com/StuHelper/StuHelper/server/internal/pkg/errs"
	"github.com/StuHelper/StuHelper/server/internal/pkg/logger"
	"github.com/StuHelper/StuHelper/server/internal/pkg/response"
)

// InternalUserIDResolver 根据 Casdoor subject 解析内部 user ID。
type InternalUserIDResolver func(ctx context.Context, casdoorSubject string) (int64, error)

// ResolveRequiredInternalUserID 从认证上下文读取 Casdoor subject 并解析内部 user ID。
// 缺少认证信息时统一返回 401；解析失败时统一返回 500。
func ResolveRequiredInternalUserID(
	c *gin.Context,
	resolver InternalUserIDResolver,
	failureMessage string,
) (int64, bool) {
	casdoorSubject := GetUserID(c)
	if casdoorSubject == "" {
		response.Unauthorized(c, "authentication required", errs.ErrLoginRequired)
		return 0, false
	}

	userID, err := resolver(c.Request.Context(), casdoorSubject)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			response.Forbidden(c, "user has not completed provisioning", errs.ErrUserNotFound)
			return 0, false
		}
		logger.FromGin(c).Error("failed to resolve user ID",
			zap.String("casdoor_subject", casdoorSubject),
			zap.Error(err),
		)
		response.InternalError(c, failureMessage)
		return 0, false
	}
	if userID <= 0 {
		logger.FromGin(c).Warn("resolved invalid internal user ID",
			zap.String("casdoor_subject", casdoorSubject),
			zap.Int64("user_id", userID),
		)
		response.Forbidden(c, "user has not completed provisioning", errs.ErrUserNotFound)
		return 0, false
	}

	return userID, true
}
