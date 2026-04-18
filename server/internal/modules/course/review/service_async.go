package review

import (
	"context"
	"time"
)

const asyncNotificationTimeout = 5 * time.Second

func (s *Service) dispatchNotification(parent context.Context, fn func(context.Context)) {
	baseCtx := parent
	if s.asyncCtx != nil {
		baseCtx = s.asyncCtx
	}

	go func(ctx context.Context) {
		notifCtx, cancel := context.WithTimeout(ctx, asyncNotificationTimeout)
		defer cancel()
		fn(notifCtx)
	}(baseCtx)
}
