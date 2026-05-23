package identityserver

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/openplatform"
	"git.stuhelper.com/StuHelper/StuHelper/internal/testutil/redisfixture"
)

func TestBeginRejectsUnregisteredRedirectURIForPreConsentedApp(t *testing.T) {
	ctx := context.Background()
	gateway := newFakeIdentityOpenPlatform()
	service, fixture := newIdentityTestService(t, gateway)

	redirectURL, err := service.Begin(ctx, AuthorizeRequest{
		ResponseType: "code",
		ClientID:     gateway.app.ClientID,
		RedirectURI:  "https://evil.example.com/callback",
		Scope:        "openid profile",
		State:        "state-1",
	}, 42)

	require.ErrorIs(t, err, openplatform.ErrRedirectURINotAllowed)
	assert.Empty(t, redirectURL)
	keys, err := fixture.Client.Keys(ctx, authCodeRedisPrefix+"*").Result()
	require.NoError(t, err)
	assert.Empty(t, keys)
}

func TestAuthorizationCodeExchangeUserInfoIntrospectAndRevoke(t *testing.T) {
	ctx := context.Background()
	gateway := newFakeIdentityOpenPlatform()
	service, _ := newIdentityTestService(t, gateway)
	verifier := "this-is-a-long-enough-test-code-verifier"

	redirectURL, err := service.Begin(ctx, AuthorizeRequest{
		ResponseType:        "code",
		ClientID:            gateway.app.ClientID,
		RedirectURI:         "https://client.example.com/callback",
		Scope:               "openid profile email",
		State:               "state-2",
		CodeChallenge:       s256Challenge(verifier),
		CodeChallengeMethod: "S256",
		Nonce:               "nonce-1",
	}, 42)
	require.NoError(t, err)

	callback, err := url.Parse(redirectURL)
	require.NoError(t, err)
	assert.Equal(t, "https://client.example.com/callback", callback.Scheme+"://"+callback.Host+callback.Path)
	assert.Equal(t, "state-2", callback.Query().Get("state"))
	code := callback.Query().Get("code")
	require.NotEmpty(t, code)

	tokenSet, err := service.ExchangeCode(ctx, TokenRequest{
		GrantType:    "authorization_code",
		Code:         code,
		RedirectURI:  "https://client.example.com/callback",
		CodeVerifier: verifier,
		ClientID:     gateway.app.ClientID,
		ClientSecret: gateway.clientSecret,
	})
	require.NoError(t, err)

	accessToken, ok := tokenSet["access_token"].(string)
	require.True(t, ok)
	require.NotEmpty(t, accessToken)
	idToken, ok := tokenSet["id_token"].(string)
	require.True(t, ok)
	require.NotEmpty(t, idToken)
	assert.Equal(t, "Bearer", tokenSet["token_type"])
	assert.Equal(t, "openid profile email", tokenSet["scope"])

	claims, err := service.signer.VerifyAccessToken(accessToken)
	require.NoError(t, err)
	assert.Equal(t, "stuhelper:42", claims.Subject)
	assert.Equal(t, gateway.app.ClientID, claims.ClientID)
	assert.Equal(t, int64(42), claims.UserID)
	assert.Equal(t, []string{"openid", "profile", "email"}, claims.Scopes)

	userInfo, err := service.UserInfo(ctx, accessToken)
	require.NoError(t, err)
	assert.Equal(t, "alice", userInfo["preferred_username"])
	assert.Equal(t, "alice@example.com", userInfo["email"])
	assert.Equal(t, true, userInfo["email_verified"])

	active := service.Introspect(ctx, accessToken)
	assert.Equal(t, true, active["active"])
	assert.Equal(t, gateway.app.ClientID, active["client_id"])
	assert.Equal(t, "openid profile email", active["scope"])

	require.NoError(t, service.Revoke(ctx, accessToken))
	revoked := service.Introspect(ctx, accessToken)
	assert.Equal(t, false, revoked["active"])

	_, err = service.ExchangeCode(ctx, TokenRequest{
		GrantType:    "authorization_code",
		Code:         code,
		RedirectURI:  "https://client.example.com/callback",
		CodeVerifier: verifier,
		ClientID:     gateway.app.ClientID,
		ClientSecret: gateway.clientSecret,
	})
	require.ErrorIs(t, err, ErrInvalidGrant)
}

func TestExchangeCodeRejectsPKCEMismatch(t *testing.T) {
	ctx := context.Background()
	gateway := newFakeIdentityOpenPlatform()
	service, _ := newIdentityTestService(t, gateway)

	redirectURL, err := service.Begin(ctx, AuthorizeRequest{
		ResponseType:        "code",
		ClientID:            gateway.app.ClientID,
		RedirectURI:         "https://client.example.com/callback",
		Scope:               "openid profile",
		CodeChallenge:       s256Challenge("expected-verifier"),
		CodeChallengeMethod: "S256",
	}, 42)
	require.NoError(t, err)
	callback, err := url.Parse(redirectURL)
	require.NoError(t, err)

	_, err = service.ExchangeCode(ctx, TokenRequest{
		GrantType:    "authorization_code",
		Code:         callback.Query().Get("code"),
		RedirectURI:  "https://client.example.com/callback",
		CodeVerifier: "wrong-verifier",
		ClientID:     gateway.app.ClientID,
		ClientSecret: gateway.clientSecret,
	})
	require.ErrorIs(t, err, ErrInvalidGrant)
}

func newIdentityTestService(t *testing.T, gateway *fakeIdentityOpenPlatform) (*Service, *redisfixture.Fixture) {
	t.Helper()
	fixture := redisfixture.Start(t)
	signer, err := NewSigner("https://id.example.com", "identity-test-key", "")
	require.NoError(t, err)
	service, err := NewService(gateway, fixture.Client, signer, "https://id.example.com", 5*time.Minute, 15*time.Minute)
	require.NoError(t, err)
	return service, fixture
}

func s256Challenge(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

type fakeIdentityOpenPlatform struct {
	app          *openplatform.App
	clientSecret string
	projection   *openplatform.UserProjection
	profile      map[string]any
}

func newFakeIdentityOpenPlatform() *fakeIdentityOpenPlatform {
	return &fakeIdentityOpenPlatform{
		app: &openplatform.App{
			ID:           7,
			ClientID:     "client-1",
			RedirectURIs: []string{"https://client.example.com/callback"},
			Status:       openplatform.AppStatusApproved,
		},
		clientSecret: "secret-1",
		projection: &openplatform.UserProjection{
			Username: "alice",
			Email:    "alice@example.com",
		},
		profile: map[string]any{
			"sub":      "stuhelper:42",
			"username": "alice",
			"email":    "alice@example.com",
		},
	}
}

func (f *fakeIdentityOpenPlatform) BeginAuthorization(_ context.Context, req openplatform.AuthorizeRequest, userID int64) (*openplatform.AuthorizationDecision, error) {
	return &openplatform.AuthorizationDecision{
		App:    f.app,
		UserID: userID,
		Scopes: req.Scopes,
	}, nil
}

func (f *fakeIdentityOpenPlatform) LoadConsentChallenge(context.Context, string) (*openplatform.ConsentChallenge, error) {
	return nil, openplatform.ErrConsentTokenInvalid
}

func (f *fakeIdentityOpenPlatform) AppByID(_ context.Context, appID int64) (*openplatform.App, error) {
	if appID == f.app.ID {
		return f.app, nil
	}
	return nil, openplatform.ErrAppNotFound
}

func (f *fakeIdentityOpenPlatform) UserProjection(context.Context, int64) (*openplatform.UserProjection, error) {
	return f.projection, nil
}

func (f *fakeIdentityOpenPlatform) UserInfoForIdentityToken(_ context.Context, clientID string, userID int64, subject string, _ []string) (map[string]any, error) {
	if clientID != f.app.ClientID || userID != 42 || subject != identitySubject(userID) {
		return nil, openplatform.ErrDisclosureUnavailable
	}
	payload := make(map[string]any, len(f.profile))
	for key, value := range f.profile {
		payload[key] = value
	}
	return payload, nil
}

func (f *fakeIdentityOpenPlatform) DeleteConsentChallenge(context.Context, string) error {
	return nil
}

func (f *fakeIdentityOpenPlatform) VerifyClientSecret(_ context.Context, clientID, clientSecret string) (*openplatform.App, error) {
	if clientID == f.app.ClientID && clientSecret == f.clientSecret {
		return f.app, nil
	}
	return nil, openplatform.ErrAppNotFound
}
