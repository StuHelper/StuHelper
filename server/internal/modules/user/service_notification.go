package user

import (
	"context"
	"time"

	"go.uber.org/zap"

	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/notification"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
)

const userNotificationTimeout = 5 * time.Second

func (s *Service) sendNotificationAsync(params notification.SendParams) {
	if s == nil || s.notifSender == nil {
		return
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), userNotificationTimeout)
		defer cancel()

		if err := s.notifSender.Send(ctx, params); err != nil {
			logger.L().Warn("failed to send user module notification",
				zap.Int64("user_id", params.UserID),
				zap.String("type", params.Type),
				zap.Error(err),
			)
		}
	}()
}
