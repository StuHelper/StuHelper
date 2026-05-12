package review

import (
	"context"
	"errors"
)

// AuthorizationProvider 是评课模块的资源关系与资源级鉴权依赖。
// 写路径负责把业务事实投影到 OpenFGA；admin mutation 读路径通过 Check
// 以 OpenFGA 作为单条资源操作的权威决策点。
type AuthorizationProvider interface {
	Check(ctx context.Context, user, relation, object string) (bool, error)
	// authorUserID may be empty for legacy/imported reviews; school/section relations remain required.
	WriteReviewRelations(ctx context.Context, reviewID, authorUserID, schoolID string) error
	WriteReportRelations(ctx context.Context, reportID, schoolID string) error
}

var errAuthorizationProviderNotConfigured = errors.New("review authorization provider is not configured")

type failClosedAuthorizationProvider struct{}

func NewFailClosedAuthorizationProvider() AuthorizationProvider {
	return failClosedAuthorizationProvider{}
}

func (failClosedAuthorizationProvider) Check(context.Context, string, string, string) (bool, error) {
	return false, errAuthorizationProviderNotConfigured
}

func (failClosedAuthorizationProvider) WriteReviewRelations(context.Context, string, string, string) error {
	return errAuthorizationProviderNotConfigured
}

func (failClosedAuthorizationProvider) WriteReportRelations(context.Context, string, string) error {
	return errAuthorizationProviderNotConfigured
}

func normalizeAuthorizationProvider(provider AuthorizationProvider) AuthorizationProvider {
	if provider != nil {
		return provider
	}
	return NewFailClosedAuthorizationProvider()
}
