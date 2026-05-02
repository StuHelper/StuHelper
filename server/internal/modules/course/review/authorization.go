package review

import (
	"context"
	"errors"
)

// AuthorizationProvider 是评测模块的资源关系投影依赖。
// 请求时鉴权走 capability / DB scope / Authorization Service，不能在 handler 中直接查 OpenFGA。
type AuthorizationProvider interface {
	WriteReviewRelations(ctx context.Context, reviewID, authorUserID, courseID, schoolID string) error
	WriteReportRelations(ctx context.Context, reportID, reporterUserID, reviewID, schoolID string) error
}

var errAuthorizationProviderNotConfigured = errors.New("review authorization provider is not configured")

type failClosedAuthorizationProvider struct{}

func NewFailClosedAuthorizationProvider() AuthorizationProvider {
	return failClosedAuthorizationProvider{}
}

func (failClosedAuthorizationProvider) WriteReviewRelations(context.Context, string, string, string, string) error {
	return errAuthorizationProviderNotConfigured
}

func (failClosedAuthorizationProvider) WriteReportRelations(context.Context, string, string, string, string) error {
	return errAuthorizationProviderNotConfigured
}

func normalizeAuthorizationProvider(provider AuthorizationProvider) AuthorizationProvider {
	if provider != nil {
		return provider
	}
	return NewFailClosedAuthorizationProvider()
}
