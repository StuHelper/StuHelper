package review

import (
	"context"
	"time"
)

func detachedRefreshContext(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if parent == nil {
		parent = context.Background()
	}
	return context.WithTimeout(context.WithoutCancel(parent), timeout)
}

func (s *Service) cacheRefreshContext(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if s != nil && s.asyncCtx != nil {
		return context.WithTimeout(s.asyncCtx, timeout)
	}
	return detachedRefreshContext(parent, timeout)
}
