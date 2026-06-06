package ctxutil

import (
	"context"
	"time"
)

// Normalize returns context.Background when ctx is nil.
func Normalize(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}

// WithoutCancel returns a context that keeps values from ctx but ignores its cancellation.
func WithoutCancel(ctx context.Context) context.Context {
	return context.WithoutCancel(Normalize(ctx))
}

// Timeout derives a timeout context from ctx, defaulting nil ctx to context.Background.
func Timeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(Normalize(ctx), timeout)
}

// DetachedTimeout derives a timeout context that keeps values from ctx but ignores its cancellation.
func DetachedTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(WithoutCancel(ctx), timeout)
}
