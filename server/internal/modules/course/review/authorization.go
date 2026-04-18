package review

import "context"

// AuthorizationProvider is the review module's single authorization dependency.
// It serves both request-time permission checks and async relation projection.
type AuthorizationProvider interface {
	Check(ctx context.Context, user, relation, object string) (bool, error)
	WriteReviewRelations(ctx context.Context, reviewID, authorUserID, courseID, schoolID string) error
	WriteReportRelations(ctx context.Context, reportID, reporterUserID, reviewID, schoolID string) error
}

type failClosedAuthorizationProvider struct{}

func NewFailClosedAuthorizationProvider() AuthorizationProvider {
	return failClosedAuthorizationProvider{}
}

func (failClosedAuthorizationProvider) Check(context.Context, string, string, string) (bool, error) {
	return false, nil
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
