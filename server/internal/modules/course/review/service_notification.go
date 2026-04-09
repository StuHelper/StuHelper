package review

import (
	"context"
	"time"

	"go.uber.org/zap"

	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/notification"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
)

const reviewNotificationTimeout = 5 * time.Second

func (s *Service) sendNotificationAsync(params notification.SendParams) {
	if s == nil || s.notifSender == nil {
		return
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), reviewNotificationTimeout)
		defer cancel()

		if err := s.notifSender.Send(ctx, params); err != nil {
			logger.L().Warn("failed to send review notification",
				zap.Int64("user_id", params.UserID),
				zap.String("type", params.Type),
				zap.String("source_id", params.SourceID),
				zap.Error(err),
			)
		}
	}()
}
