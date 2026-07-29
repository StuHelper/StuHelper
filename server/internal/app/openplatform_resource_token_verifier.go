package app

import (
	"context"
	"errors"
	"fmt"

	"github.com/StuHelper/StuHelper/server/internal/modules/openplatform"
	"github.com/StuHelper/StuHelper/server/internal/pkg/oidc"
)

type oidcOpenPlatformResourceAccessTokenVerifier struct {
	client oidcTokenIntrospector
}

type oidcTokenIntrospector interface {
	IntrospectToken(ctx context.Context, accessToken string) (*oidc.IntrospectionResult, error)
}

func newOpenPlatformResourceAccessTokenVerifier(
	client oidcTokenIntrospector,
) *oidcOpenPlatformResourceAccessTokenVerifier {
	if client == nil {
		return nil
	}
	return &oidcOpenPlatformResourceAccessTokenVerifier{client: client}
}

func (v *oidcOpenPlatformResourceAccessTokenVerifier) VerifyOpenPlatformResourceAccessToken(
	ctx context.Context,
	rawToken string,
) (openplatform.ResourceAccessToken, error) {
	if v == nil || v.client == nil {
		return openplatform.ResourceAccessToken{}, openplatform.ErrInvalidResourceAccessToken
	}
	result, err := v.client.IntrospectToken(ctx, rawToken)
	if err != nil {
		if isOIDCProviderUnavailable(err) {
			return openplatform.ResourceAccessToken{}, fmt.Errorf("%w: %v", openplatform.ErrResourceAccessUnavailable, err)
		}
		return openplatform.ResourceAccessToken{}, openplatform.ErrInvalidResourceAccessToken
	}
	if result == nil || !result.Active || result.GetAppID() == "" {
		return openplatform.ResourceAccessToken{}, openplatform.ErrInvalidResourceAccessToken
	}
	return openplatform.ResourceAccessToken{
		ClientID: result.GetAppID(),
		Scopes:   result.Scopes(),
	}, nil
}

func isOIDCProviderUnavailable(err error) bool {
	return errors.Is(err, oidc.ErrProviderUnavailable)
}
