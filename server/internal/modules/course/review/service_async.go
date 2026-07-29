package review

import (
	"context"
	"time"

	"go.uber.org/zap"

	"github.com/StuHelper/StuHelper/server/internal/pkg/ctxutil"
	"github.com/StuHelper/StuHelper/server/internal/pkg/logger"
)

const asyncNotificationTimeout = 5 * time.Second

func (s *Service) dispatchNotification(parent context.Context, fn func(context.Context)) {
	baseCtx := ctxutil.WithoutCancel(parent)
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

		notifCtx, cancel := ctxutil.Timeout(ctx, asyncNotificationTimeout)
		defer cancel()
		fn(notifCtx)
	}

	if s.asyncLaunch != nil {
		s.asyncLaunch("", run)
		return
	}

	go run(baseCtx)
}
