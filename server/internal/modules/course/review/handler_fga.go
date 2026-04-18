package review

import (
	"context"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
	"go.uber.org/zap"
)

// checkFGA 检查 FGA 权限（FGA 未配置时返回 false）
func (h *Handler) checkFGA(ctx context.Context, user, relation, object string) bool {
	allowed, err := h.fga.Check(ctx, user, relation, object)
	if err != nil {
		logger.L().Warn("FGA check failed, denying",
			zap.String("user", user),
			zap.String("relation", relation),
			zap.String("object", object),
			zap.Error(err),
		)
		return false
	}
	return allowed
}

func reviewPermissionRelationForAction(action string) string {
	if action == "delete" {
		return "can_delete"
	}
	return "can_hide"
}
