package review

import (
	"context"
	"time"

	"github.com/StuHelper/StuHelper/server/internal/pkg/ctxutil"
)

func detachedRefreshContext(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	return ctxutil.DetachedTimeout(parent, timeout)
}

func (s *Service) cacheRefreshContext(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if s != nil && s.asyncCtx != nil {
		return context.WithTimeout(s.asyncCtx, timeout)
	}
	return detachedRefreshContext(parent, timeout)
}
