package logger

import (
	"context"

	"go.uber.org/zap"
)

type ctxKey struct{}

// WithContext 将 Logger 注入到 Context
func WithContext(ctx context.Context, l *zap.Logger) context.Context {
	return context.WithValue(ctx, ctxKey{}, l)
}

// FromContext 从 Context 提取 Logger
func FromContext(ctx context.Context) *zap.Logger {
	if l, ok := ctx.Value(ctxKey{}).(*zap.Logger); ok {
		return l
	}
	return L()
}

// WithFields 创建带有额外字段的 Logger 并注入 Context
func WithFields(ctx context.Context, fields ...zap.Field) context.Context {
	l := FromContext(ctx).With(fields...)
	return WithContext(ctx, l)
}
