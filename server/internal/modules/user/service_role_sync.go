package user

import (
	"context"
	"time"

	"go.uber.org/zap"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
)

const roleSyncTimeout = 15 * time.Second

func (s *Service) syncRole(ctx context.Context, userID int64, role string, approved bool) {
	if s.onRoleSync == nil {
		return
	}

	roleCtx, cancel := context.WithTimeout(ctx, roleSyncTimeout)
	defer cancel()

	if err := s.onRoleSync(roleCtx, userID, role, approved); err != nil {
		logger.L().Error("role sync failed",
			zap.Int64("user_id", userID),
			zap.String("role", role),
			zap.Bool("approved", approved),
			zap.Error(err),
		)
	}
}
