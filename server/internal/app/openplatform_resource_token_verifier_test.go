package app

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/StuHelper/StuHelper/server/internal/modules/openplatform"
	"github.com/StuHelper/StuHelper/server/internal/pkg/oidc"
)

type fakeOIDCTokenIntrospector struct {
	result *oidc.IntrospectionResult
	err    error
	seen   []string
}

func (f *fakeOIDCTokenIntrospector) IntrospectToken(
	_ context.Context,
	accessToken string,
) (*oidc.IntrospectionResult, error) {
	f.seen = append(f.seen, accessToken)
	if f.err != nil {
		return nil, f.err
	}
	return f.result, nil
}

func TestOpenPlatformResourceAccessTokenVerifierAcceptsActiveIntrospectionToken(t *testing.T) {
	introspector := &fakeOIDCTokenIntrospector{result: &oidc.IntrospectionResult{
		Active: true,
		AppID:  "third-party-client",
		Scope:  "openid profile resource.read",
	}}
	verifier := newOpenPlatformResourceAccessTokenVerifier(introspector)

	token, err := verifier.VerifyOpenPlatformResourceAccessToken(context.Background(), "access-token")

	require.NoError(t, err)
	assert.Equal(t, "third-party-client", token.ClientID)
	assert.Equal(t, []string{"openid", "profile", openplatform.ScopeResourceRead}, token.Scopes)
	assert.Equal(t, []string{"access-token"}, introspector.seen)
}

func TestOpenPlatformResourceAccessTokenVerifierRejectsInactiveOrUnboundTokens(t *testing.T) {
	for _, tt := range []struct {
		name   string
		result *oidc.IntrospectionResult
	}{
		{name: "nil result", result: nil},
		{name: "inactive", result: &oidc.IntrospectionResult{Active: false, AppID: "third-party-client"}},
		{name: "missing app id", result: &oidc.IntrospectionResult{Active: true}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			verifier := newOpenPlatformResourceAccessTokenVerifier(&fakeOIDCTokenIntrospector{result: tt.result})

			_, err := verifier.VerifyOpenPlatformResourceAccessToken(context.Background(), "access-token")

			require.ErrorIs(t, err, openplatform.ErrInvalidResourceAccessToken)
		})
	}
}

func TestOpenPlatformResourceAccessTokenVerifierClassifiesProviderOutage(t *testing.T) {
	verifier := newOpenPlatformResourceAccessTokenVerifier(&fakeOIDCTokenIntrospector{
		err: fmt.Errorf("introspection failed: %w", oidc.ErrProviderUnavailable),
	})

	_, err := verifier.VerifyOpenPlatformResourceAccessToken(context.Background(), "access-token")

	require.ErrorIs(t, err, openplatform.ErrResourceAccessUnavailable)
}
