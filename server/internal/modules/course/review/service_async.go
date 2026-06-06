package review

import (
	"context"
	"time"

	"go.uber.org/zap"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
)

const asyncNotificationTimeout = 5 * time.Second

func (s *Service) dispatchNotification(parent context.Context, fn func(context.Context)) {
	if parent == nil {
		parent = context.Background()
	}
	baseCtx := context.WithoutCancel(parent)
	if s.asyncCtx != nil {
		baseCtx = s.asyncCtx
	}

	run := func(ctx context.Context) {
		defer func() {
			if recovered := recover(); recovered != nil {
				logger.L().Error(
					"review async notification panicked",
					zap.Any("panic", recovered),
					zap.Stack("stack"),
				)
			}
		}()

		notifCtx, cancel := context.WithTimeout(ctx, asyncNotificationTimeout)
		defer cancel()
		fn(notifCtx)
	}

	if s.asyncLaunch != nil {
		s.asyncLaunch("", run)
		return
	}

	go run(baseCtx)
}
