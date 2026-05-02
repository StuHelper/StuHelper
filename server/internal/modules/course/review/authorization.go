package review

import "context"

// AuthorizationProvider 是评测模块的资源关系投影依赖。
// 请求时鉴权走 capability / DB scope / Authorization Service，不能在 handler 中直接查 OpenFGA。
type AuthorizationProvider interface {
	WriteReviewRelations(ctx context.Context, reviewID, authorUserID, courseID, schoolID string) error
	WriteReportRelations(ctx context.Context, reportID, reporterUserID, reviewID, schoolID string) error
}

type failClosedAuthorizationProvider struct{}

func NewFailClosedAuthorizationProvider() AuthorizationProvider {
	return failClosedAuthorizationProvider{}
}

func (failClosedAuthorizationProvider) WriteReviewRelations(context.Context, string, string, string, string) error {
	return nil
}

func (failClosedAuthorizationProvider) WriteReportRelations(context.Context, string, string, string, string) error {
	return nil
}

func normalizeAuthorizationProvider(provider AuthorizationProvider) AuthorizationProvider {
	if provider != nil {
		return provider
	}
	return NewFailClosedAuthorizationProvider()
}
