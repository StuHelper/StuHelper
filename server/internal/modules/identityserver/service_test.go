package identityserver

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/url"
	"testing"
	"time"

	jwt "github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/openplatform"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/audit"
	"git.stuhelper.com/StuHelper/StuHelper/internal/testutil/redisfixture"
)

func TestBeginRejectsUnregisteredRedirectURIForPreConsentedApp(t *testing.T) {
	ctx := context.Background()
	gateway := newFakeIdentityOpenPlatform()
	service, fixture := newIdentityTestService(t, gateway)
	verifier := validTestPKCEVerifier()

	redirectURL, err := service.Begin(ctx, AuthorizeRequest{
		ResponseType:        "code",
		ClientID:            gateway.app.ClientID,
		RedirectURI:         "https://evil.example.com/callback",
		Scope:               "openid profile",
		State:               "state-1",
		CodeChallenge:       s256Challenge(verifier),
		CodeChallengeMethod: "S256",
	}, 42)

	require.ErrorIs(t, err, openplatform.ErrRedirectURINotAllowed)
	assert.Empty(t, redirectURL)
	keys, err := fixture.Client.Keys(ctx, authCodeRedisPrefix+"*").Result()
	require.NoError(t, err)
	assert.Empty(t, keys)
}

func TestBeginRequiresS256PKCE(t *testing.T) {
	ctx := context.Background()
	gateway := newFakeIdentityOpenPlatform()
	service, fixture := newIdentityTestService(t, gateway)

	tests := []struct {
		name      string
		challenge string
		method    string
	}{
		{name: "missing challenge", challenge: "", method: ""},
		{name: "plain method", challenge: s256Challenge(validTestPKCEVerifier()), method: "plain"},
		{name: "invalid challenge", challenge: "not-a-valid-s256-challenge", method: "S256"},
		{name: "padded challenge", challenge: " " + s256Challenge(validTestPKCEVerifier()), method: "S256"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			redirectURL, err := service.Begin(ctx, AuthorizeRequest{
				ResponseType:        "code",
				ClientID:            gateway.app.ClientID,
				RedirectURI:         "https://client.example.com/callback",
				Scope:               "openid profile",
				CodeChallenge:       tt.challenge,
				CodeChallengeMethod: tt.method,
			}, 42)

			require.ErrorIs(t, err, ErrInvalidAuthorizeRequest)
			assert.Empty(t, redirectURL)
			keys, err := fixture.Client.Keys(ctx, authCodeRedisPrefix+"*").Result()
			require.NoError(t, err)
			assert.Empty(t, keys)
		})
	}
}

func TestBeginRejectsUnsupportedAuthorizeRequestParameters(t *testing.T) {
	ctx := context.Background()
	gateway := newFakeIdentityOpenPlatform()
	service, fixture := newIdentityTestService(t, gateway)
	verifier := validTestPKCEVerifier()

	tests := []struct {
		name         string
		responseMode string
		prompt       string
		maxAge       string
		scope        string
	}{
		{name: "unsupported response mode", responseMode: "fragment"},
		{name: "whitespace response mode", responseMode: " query "},
		{name: "unsupported prompt", prompt: "select_account"},
		{name: "prompt none combined", prompt: "none consent"},
		{name: "negative max age", maxAge: "-1"},
		{name: "non numeric max age", maxAge: "soon"},
		{name: "offline access without openid", scope: "profile offline_access"},
		{name: "resource offline access without openid", scope: "resource.read offline_access"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scope := tt.scope
			if scope == "" {
				scope = "openid profile"
			}
			redirectURL, err := service.Begin(ctx, AuthorizeRequest{
				ResponseType:        "code",
				ClientID:            gateway.app.ClientID,
				RedirectURI:         "https://client.example.com/callback",
				Scope:               scope,
				ResponseMode:        tt.responseMode,
				CodeChallenge:       s256Challenge(verifier),
				CodeChallengeMethod: "S256",
				Prompt:              tt.prompt,
				MaxAge:              tt.maxAge,
			}, 42)

			require.ErrorIs(t, err, ErrInvalidAuthorizeRequest)
			assert.Empty(t, redirectURL)
			keys, err := fixture.Client.Keys(ctx, authCodeRedisPrefix+"*").Result()
			require.NoError(t, err)
			assert.Empty(t, keys)
		})
	}
}

func TestBeginAcceptsExplicitQueryResponseMode(t *testing.T) {
	ctx := context.Background()
	gateway := newFakeIdentityOpenPlatform()
	service, _ := newIdentityTestService(t, gateway)
	verifier := validTestPKCEVerifier()

	redirectURL, err := service.Begin(ctx, AuthorizeRequest{
		ResponseType:        "code",
		ResponseMode:        "query",
		ClientID:            gateway.app.ClientID,
		RedirectURI:         "https://client.example.com/callback",
		Scope:               "openid profile",
		State:               "state-response-mode",
		CodeChallenge:       s256Challenge(verifier),
		CodeChallengeMethod: "S256",
	}, 42)

	require.NoError(t, err)
	callback, err := url.Parse(redirectURL)
	require.NoError(t, err)
	assert.Equal(t, "https://client.example.com/callback", callback.Scheme+"://"+callback.Host+callback.Path)
	assert.NotEmpty(t, callback.Query().Get("code"))
	assert.Equal(t, "state-response-mode", callback.Query().Get("state"))
	assert.NotEmpty(t, callback.Query().Get("iss"))
	assert.Empty(t, callback.Fragment)
}

func TestOAuthRedirectsPreserveOpaqueState(t *testing.T) {
	ctx := context.Background()
	gateway := newFakeIdentityOpenPlatform()
	service, _ := newIdentityTestService(t, gateway)
	verifier := validTestPKCEVerifier()
	state := " state with surrounding spaces "

	redirectURL, err := service.Begin(ctx, AuthorizeRequest{
		ResponseType:        "code",
		ClientID:            gateway.app.ClientID,
		RedirectURI:         "https://client.example.com/callback",
		Scope:               "openid profile",
		State:               state,
		CodeChallenge:       s256Challenge(verifier),
		CodeChallengeMethod: "S256",
	}, 42)
	require.NoError(t, err)
	callback, err := url.Parse(redirectURL)
	require.NoError(t, err)
	assert.Equal(t, state, callback.Query().Get("state"))

	errorRedirect, err := service.AuthorizationErrorRedirect(ctx, AuthorizeRequest{
		ResponseType:        "code",
		ClientID:            gateway.app.ClientID,
		RedirectURI:         "https://client.example.com/callback",
		Scope:               "openid profile",
		State:               state,
		CodeChallenge:       s256Challenge(verifier),
		CodeChallengeMethod: "S256",
	}, oidcErrorLoginRequired)
	require.NoError(t, err)
	errorCallback, err := url.Parse(errorRedirect)
	require.NoError(t, err)
	assert.Equal(t, state, errorCallback.Query().Get("state"))

	logoutRedirect, hasRedirect, err := service.EndSessionRedirect(ctx, LogoutRequest{
		ClientID:              gateway.app.ClientID,
		PostLogoutRedirectURI: "https://client.example.com/callback",
		State:                 state,
	})
	require.NoError(t, err)
	require.True(t, hasRedirect)
	logoutCallback, err := url.Parse(logoutRedirect)
	require.NoError(t, err)
	assert.Equal(t, state, logoutCallback.Query().Get("state"))
}

func TestBeginPromptConsentForcesConsent(t *testing.T) {
	ctx := context.Background()
	gateway := newFakeIdentityOpenPlatform()
	service, _ := newIdentityTestService(t, gateway)
	verifier := validTestPKCEVerifier()

	redirectURL, err := service.Begin(ctx, AuthorizeRequest{
		ResponseType:        "code",
		ClientID:            gateway.app.ClientID,
		RedirectURI:         "https://client.example.com/callback",
		Scope:               "openid profile",
		State:               "state-consent",
		CodeChallenge:       s256Challenge(verifier),
		CodeChallengeMethod: "S256",
		Prompt:              "consent",
	}, 42)

	require.NoError(t, err)
	assert.Contains(t, redirectURL, "code=")
	assert.True(t, gateway.authorizationRequest.ForceConsent)
	assert.False(t, gateway.authorizationRequest.PromptNone)
	assert.Equal(t, []string{"openid", "profile"}, gateway.authorizationRequest.Scopes)
}

func TestNewServiceUsesConfiguredRefreshTokenTTL(t *testing.T) {
	gateway := newFakeIdentityOpenPlatform()
	fixture := redisfixture.Start(t)
	signer, err := NewSigner("https://id.example.com", "identity-test-key", "")
	require.NoError(t, err)

	service, err := NewService(
		gateway,
		fixture.Client,
		signer,
		"https://id.example.com",
		5*time.Minute,
		15*time.Minute,
		2*time.Hour,
	)

	require.NoError(t, err)
	assert.Equal(t, 2*time.Hour, service.refreshTTL)
}

func TestBeginPromptNoneReturnsInteractionRequiredErrorRedirect(t *testing.T) {
	ctx := context.Background()
	gateway := newFakeIdentityOpenPlatform()
	gateway.authorizationDecision = &openplatform.AuthorizationDecision{
		App:                 gateway.app,
		InteractionRequired: true,
		InteractionError:    "consent_required",
	}
	service, fixture := newIdentityTestService(t, gateway)
	verifier := validTestPKCEVerifier()

	redirectURL, err := service.Begin(ctx, AuthorizeRequest{
		ResponseType:        "code",
		ClientID:            gateway.app.ClientID,
		RedirectURI:         "https://client.example.com/callback",
		Scope:               "openid email",
		State:               "state-silent-consent",
		CodeChallenge:       s256Challenge(verifier),
		CodeChallengeMethod: "S256",
		Prompt:              "none",
	}, 42)

	require.NoError(t, err)
	assert.True(t, gateway.authorizationRequest.PromptNone)
	callback, err := url.Parse(redirectURL)
	require.NoError(t, err)
	assert.Equal(t, "https://client.example.com/callback", callback.Scheme+"://"+callback.Host+callback.Path)
	assert.Equal(t, "consent_required", callback.Query().Get("error"))
	assert.Equal(t, "https://id.example.com", callback.Query().Get("iss"))
	assert.Equal(t, "state-silent-consent", callback.Query().Get("state"))
	assert.Empty(t, callback.Query().Get("code"))
	keys, err := fixture.Client.Keys(ctx, authCodeRedisPrefix+"*").Result()
	require.NoError(t, err)
	assert.Empty(t, keys)
}

func TestAuthorizationCodeExchangeUserInfoIntrospectAndRevoke(t *testing.T) {
	ctx := context.Background()
	gateway := newFakeIdentityOpenPlatform()
	service, _ := newIdentityTestService(t, gateway)
	verifier := validTestPKCEVerifier()
	var auditEvents []audit.Event
	service.audit = func(_ context.Context, event audit.Event) {
		auditEvents = append(auditEvents, event)
	}
	authTime := time.Date(2026, 5, 24, 9, 15, 0, 0, time.UTC)

	redirectURL, err := service.Begin(ctx, AuthorizeRequest{
		ResponseType:        "code",
		ClientID:            gateway.app.ClientID,
		RedirectURI:         "https://client.example.com/callback",
		Scope:               "openid profile email",
		State:               "state-2",
		CodeChallenge:       s256Challenge(verifier),
		CodeChallengeMethod: "S256",
		Nonce:               "nonce-1",
		AuthTime:            authTime,
	}, 42)
	require.NoError(t, err)

	callback, err := url.Parse(redirectURL)
	require.NoError(t, err)
	assert.Equal(t, "https://client.example.com/callback", callback.Scheme+"://"+callback.Host+callback.Path)
	assert.Equal(t, "https://id.example.com", callback.Query().Get("iss"))
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
	idTokenClaims := parseIdentityTestToken(t, service, idToken)
	assert.Equal(t, "stuhelper:42", idTokenClaims["sub"])
	assert.Equal(t, gateway.app.ClientID, idTokenClaims["aud"])
	assert.Equal(t, "id_token", idTokenClaims["typ"])
	assert.Equal(t, gateway.app.ClientID, idTokenClaims["azp"])
	assert.Equal(t, "alice", idTokenClaims["preferred_username"])
	assert.Equal(t, "alice", idTokenClaims["name"])
	assert.Equal(t, "alice@example.com", idTokenClaims["email"])
	assert.Equal(t, true, idTokenClaims["email_verified"])
	assert.Equal(t, float64(authTime.Unix()), idTokenClaims["auth_time"])
	assert.NotContains(t, idTokenClaims, "phone")
	assert.NotContains(t, idTokenClaims, "identityVerified")
	assert.NotContains(t, idTokenClaims, "stuhelper_identity_type")
	assert.Equal(t, "Bearer", tokenSet["token_type"])
	assert.Equal(t, "openid profile email", tokenSet["scope"])
	assert.NotContains(t, tokenSet, "refresh_token")

	claims, err := service.signer.VerifyAccessToken(accessToken)
	require.NoError(t, err)
	assert.Equal(t, "stuhelper:42", claims.Subject)
	assert.Equal(t, gateway.app.ClientID, claims.ClientID)
	assert.Equal(t, int64(42), claims.UserID)
	assert.Equal(t, []string{"openid", "profile", "email"}, claims.Scopes)

	userInfo, err := service.UserInfo(ctx, accessToken)
	require.NoError(t, err)
	assert.Equal(t, "stuhelper:42", userInfo["sub"])
	assert.Equal(t, "alice", userInfo["preferred_username"])
	assert.Equal(t, "alice", userInfo["name"])
	assert.Equal(t, "alice@example.com", userInfo["email"])
	assert.Equal(t, true, userInfo["email_verified"])

	active := service.Introspect(ctx, accessToken, gateway.app.ClientID)
	assert.Equal(t, true, active["active"])
	assert.Equal(t, gateway.app.ClientID, active["client_id"])
	assert.Equal(t, "stuhelper:42", active["sub"])
	assert.Equal(t, gateway.app.ClientID, active["aud"])
	assert.Equal(t, "Bearer", active["token_type"])
	assert.Equal(t, "access_token", active["token_kind"])
	assert.Equal(t, "openid profile email", active["scope"])

	crossClient := service.Introspect(ctx, accessToken, gateway.otherApp.ClientID)
	assert.Equal(t, false, crossClient["active"])
	require.NoError(t, service.Revoke(ctx, accessToken, gateway.otherApp.ClientID))
	assert.Empty(t, auditEvents)
	stillActive := service.Introspect(ctx, accessToken, gateway.app.ClientID)
	assert.Equal(t, true, stillActive["active"])

	require.NoError(t, service.Revoke(ctx, accessToken, gateway.app.ClientID, "refresh_token"))
	require.Len(t, auditEvents, 1)
	assert.Equal(t, audit.EventTokenRevoked, auditEvents[0].Type)
	assert.Equal(t, "client", auditEvents[0].ActorType)
	assert.Equal(t, "42", auditEvents[0].UserID)
	assert.Equal(t, "identity.access_token", auditEvents[0].ResourceType)
	assert.Equal(t, claims.JTI, auditEvents[0].ResourceID)
	assert.Equal(t, "identity_access_token_revoked", auditEvents[0].Action)
	assert.Equal(t, "success", auditEvents[0].Result)
	assert.Equal(t, gateway.app.ClientID, auditEvents[0].Details["clientID"])
	assert.Equal(t, "access_token", auditEvents[0].Details["tokenType"])
	assert.NotEqual(t, accessToken, auditEvents[0].ResourceID)
	assert.NotContains(t, auditEvents[0].Details, "token")
	assert.NotContains(t, auditEvents[0].Details, "accessToken")
	revoked := service.Introspect(ctx, accessToken, gateway.app.ClientID)
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

func TestAuthorizationCodeExchangeWithoutOpenIDOmitsIDTokenAndUserInfo(t *testing.T) {
	ctx := context.Background()
	gateway := newFakeIdentityOpenPlatform()
	service, _ := newIdentityTestService(t, gateway)
	verifier := validTestPKCEVerifier()

	redirectURL, err := service.Begin(ctx, AuthorizeRequest{
		ResponseType:        "code",
		ClientID:            gateway.app.ClientID,
		RedirectURI:         "https://client.example.com/callback",
		Scope:               openplatform.ScopeResourceRead + " " + openplatform.ScopeResourceWrite,
		State:               "state-resource-only",
		CodeChallenge:       s256Challenge(verifier),
		CodeChallengeMethod: "S256",
	}, 42)
	require.NoError(t, err)
	assert.Equal(t, []string{openplatform.ScopeResourceRead, openplatform.ScopeResourceWrite}, gateway.authorizationRequest.Scopes)

	callback, err := url.Parse(redirectURL)
	require.NoError(t, err)
	tokenSet, err := service.ExchangeCode(ctx, TokenRequest{
		GrantType:    "authorization_code",
		Code:         callback.Query().Get("code"),
		RedirectURI:  "https://client.example.com/callback",
		CodeVerifier: verifier,
		ClientID:     gateway.app.ClientID,
		ClientSecret: gateway.clientSecret,
	})
	require.NoError(t, err)

	accessToken, ok := tokenSet["access_token"].(string)
	require.True(t, ok)
	require.NotEmpty(t, accessToken)
	assert.Equal(t, "Bearer", tokenSet["token_type"])
	assert.Equal(t, openplatform.ScopeResourceRead+" "+openplatform.ScopeResourceWrite, tokenSet["scope"])
	assert.NotContains(t, tokenSet, "id_token")
	assert.NotContains(t, tokenSet, "refresh_token")
	assert.Zero(t, gateway.userInfoCallCount)
	assert.Empty(t, gateway.userInfoScopes)

	claims, err := service.signer.VerifyAccessToken(accessToken)
	require.NoError(t, err)
	assert.Equal(t, "stuhelper:42", claims.Subject)
	assert.Equal(t, gateway.app.ClientID, claims.ClientID)
	assert.Equal(t, int64(42), claims.UserID)
	assert.Equal(t, []string{openplatform.ScopeResourceRead, openplatform.ScopeResourceWrite}, claims.Scopes)

	_, err = service.UserInfo(ctx, accessToken)
	require.Error(t, err)
	assert.Zero(t, gateway.userInfoCallCount)

	active := service.Introspect(ctx, accessToken, gateway.app.ClientID)
	assert.Equal(t, true, active["active"])
	assert.Equal(t, openplatform.ScopeResourceRead+" "+openplatform.ScopeResourceWrite, active["scope"])
	assert.Equal(t, []string{openplatform.ScopeResourceRead, openplatform.ScopeResourceWrite}, gateway.identityAccessTokenScopes)
}

func TestClientCredentialsGrantIssuesAppOnlyTokenAndHonorsCurrentScopeApproval(t *testing.T) {
	ctx := context.Background()
	gateway := newFakeIdentityOpenPlatform()
	service, _ := newIdentityTestService(t, gateway)
	var auditEvents []audit.Event
	service.audit = func(_ context.Context, event audit.Event) {
		auditEvents = append(auditEvents, event)
	}

	tokenSet, err := service.Token(ctx, TokenRequest{
		GrantType:    "client_credentials",
		Scope:        openplatform.ScopeResourceRead + " " + openplatform.ScopeResourceWrite,
		ClientID:     gateway.app.ClientID,
		ClientSecret: gateway.clientSecret,
	})
	require.NoError(t, err)

	accessToken, ok := tokenSet["access_token"].(string)
	require.True(t, ok)
	require.NotEmpty(t, accessToken)
	assert.Equal(t, "Bearer", tokenSet["token_type"])
	assert.Equal(t, "resource.read resource.write", tokenSet["scope"])
	assert.NotContains(t, tokenSet, "id_token")
	assert.NotContains(t, tokenSet, "refresh_token")
	assert.Equal(t, gateway.app.ClientID, gateway.clientCredentialsClientID)
	assert.Equal(t, []string{openplatform.ScopeResourceRead, openplatform.ScopeResourceWrite}, gateway.clientCredentialsScopes)

	claims, err := service.signer.VerifyAccessToken(accessToken)
	require.NoError(t, err)
	assert.Equal(t, "client:"+gateway.app.ClientID, claims.Subject)
	assert.Equal(t, gateway.app.ClientID, claims.ClientID)
	assert.Zero(t, claims.UserID)
	assert.Equal(t, []string{openplatform.ScopeResourceRead, openplatform.ScopeResourceWrite}, claims.Scopes)
	assert.Equal(t, "client_credentials", claims.GrantType)

	rawClaims := parseIdentityTestToken(t, service, accessToken)
	assert.Equal(t, "client_credentials", rawClaims["grant_type"])
	assert.Equal(t, float64(0), rawClaims["stuhelper_id"])

	resourceToken, err := service.VerifyOpenPlatformResourceAccessToken(ctx, accessToken)
	require.NoError(t, err)
	assert.Equal(t, gateway.app.ClientID, resourceToken.ClientID)
	assert.Equal(t, []string{openplatform.ScopeResourceRead, openplatform.ScopeResourceWrite}, resourceToken.Scopes)

	_, err = service.UserInfo(ctx, accessToken)
	require.Error(t, err)

	active := service.Introspect(ctx, accessToken, gateway.app.ClientID)
	assert.Equal(t, true, active["active"])
	assert.Equal(t, gateway.app.ClientID, active["client_id"])
	assert.Equal(t, "client:"+gateway.app.ClientID, active["sub"])
	assert.Equal(t, "resource.read resource.write", active["scope"])
	assert.Equal(t, "Bearer", active["token_type"])
	assert.Equal(t, "access_token", active["token_kind"])
	assert.Equal(t, "client_credentials", active["grant_type"])

	gateway.clientCredentialsTokenActive = false
	inactiveAfterScopeWithdrawal := service.Introspect(ctx, accessToken, gateway.app.ClientID)
	assert.Equal(t, false, inactiveAfterScopeWithdrawal["active"])
	_, err = service.VerifyOpenPlatformResourceAccessToken(ctx, accessToken)
	require.ErrorIs(t, err, ErrInvalidGrant)
	gateway.clientCredentialsTokenActive = true

	crossClient := service.Introspect(ctx, accessToken, gateway.otherApp.ClientID)
	assert.Equal(t, false, crossClient["active"])
	require.NoError(t, service.Revoke(ctx, accessToken, gateway.otherApp.ClientID))
	assert.Empty(t, auditEvents)

	require.NoError(t, service.Revoke(ctx, accessToken, gateway.app.ClientID))
	require.Len(t, auditEvents, 1)
	assert.Equal(t, audit.EventTokenRevoked, auditEvents[0].Type)
	assert.Equal(t, "client", auditEvents[0].ActorType)
	assert.Equal(t, "0", auditEvents[0].UserID)
	assert.Equal(t, gateway.app.ClientID, auditEvents[0].Details["clientID"])
	assert.Equal(t, "access_token", auditEvents[0].Details["tokenType"])
	assert.NotEqual(t, accessToken, auditEvents[0].ResourceID)
	assert.NotContains(t, auditEvents[0].Details, "token")
	revoked := service.Introspect(ctx, accessToken, gateway.app.ClientID)
	assert.Equal(t, false, revoked["active"])
}

func TestClientCredentialsGrantRejectsUserAndUnapprovedScopes(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name       string
		scope      string
		active     bool
		wantErr    error
		wantClient string
	}{
		{name: "missing scope", scope: "", active: true, wantErr: ErrInvalidScope},
		{name: "openid scope", scope: "openid", active: true, wantErr: ErrInvalidScope},
		{name: "profile scope", scope: "profile", active: true, wantErr: ErrInvalidScope},
		{name: "offline access scope", scope: openplatform.ScopeOfflineAccess, active: true, wantErr: ErrInvalidScope},
		{name: "approved app does not have requested resource scope", scope: openplatform.ScopeResourceRead, active: false, wantErr: ErrInvalidScope, wantClient: "client-1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gateway := newFakeIdentityOpenPlatform()
			gateway.clientCredentialsTokenActive = tt.active
			service, _ := newIdentityTestService(t, gateway)

			_, err := service.Token(ctx, TokenRequest{
				GrantType:    "client_credentials",
				Scope:        tt.scope,
				ClientID:     gateway.app.ClientID,
				ClientSecret: gateway.clientSecret,
			})

			require.ErrorIs(t, err, tt.wantErr)
			if tt.wantClient != "" {
				assert.Equal(t, tt.wantClient, gateway.clientCredentialsClientID)
			}
		})
	}
}

func TestOfflineAccessRefreshTokenRotatesAndHonorsCurrentAuthorization(t *testing.T) {
	ctx := context.Background()
	gateway := newFakeIdentityOpenPlatform()
	service, fixture := newIdentityTestService(t, gateway)
	verifier := validTestPKCEVerifier()
	var auditEvents []audit.Event
	service.audit = func(_ context.Context, event audit.Event) {
		auditEvents = append(auditEvents, event)
	}

	redirectURL, err := service.Begin(ctx, AuthorizeRequest{
		ResponseType:        "code",
		ClientID:            gateway.app.ClientID,
		RedirectURI:         "https://client.example.com/callback",
		Scope:               "openid profile offline_access",
		State:               "state-refresh",
		CodeChallenge:       s256Challenge(verifier),
		CodeChallengeMethod: "S256",
	}, 42)
	require.NoError(t, err)
	callback, err := url.Parse(redirectURL)
	require.NoError(t, err)

	tokenSet, err := service.ExchangeCode(ctx, TokenRequest{
		GrantType:    "authorization_code",
		Code:         callback.Query().Get("code"),
		RedirectURI:  "https://client.example.com/callback",
		CodeVerifier: verifier,
		ClientID:     gateway.app.ClientID,
		ClientSecret: gateway.clientSecret,
	})
	require.NoError(t, err)
	refreshToken, ok := tokenSet["refresh_token"].(string)
	require.True(t, ok)
	require.NotEmpty(t, refreshToken)
	assert.Equal(t, "openid profile offline_access", tokenSet["scope"])
	refreshKeys, err := fixture.Client.Keys(ctx, refreshTokenRedisPrefix+"*").Result()
	require.NoError(t, err)
	require.Len(t, refreshKeys, 1)
	assert.NotEqual(t, refreshTokenRedisPrefix+refreshToken, refreshKeys[0])
	refreshIntrospection := service.Introspect(ctx, refreshToken, gateway.app.ClientID)
	assert.Equal(t, true, refreshIntrospection["active"])
	assert.Equal(t, "refresh_token", refreshIntrospection["token_type"])
	assert.Equal(t, "refresh_token", refreshIntrospection["token_kind"])
	assert.Equal(t, gateway.app.ClientID, refreshIntrospection["client_id"])
	assert.Equal(t, "openid profile offline_access", refreshIntrospection["scope"])
	crossClientRefreshIntrospection := service.Introspect(ctx, refreshToken, gateway.otherApp.ClientID)
	assert.Equal(t, false, crossClientRefreshIntrospection["active"])

	refreshed, err := service.Refresh(ctx, TokenRequest{
		GrantType:    "refresh_token",
		RefreshToken: refreshToken,
		ClientID:     gateway.app.ClientID,
		ClientSecret: gateway.clientSecret,
	})
	require.NoError(t, err)
	rotatedRefreshToken, ok := refreshed["refresh_token"].(string)
	require.True(t, ok)
	require.NotEmpty(t, rotatedRefreshToken)
	assert.NotEqual(t, refreshToken, rotatedRefreshToken)
	assert.NotEmpty(t, refreshed["access_token"])
	assert.NotEmpty(t, refreshed["id_token"])
	assert.Equal(t, "openid profile offline_access", refreshed["scope"])
	assert.Equal(t, []string{"openid", "profile", openplatform.ScopeOfflineAccess}, gateway.identityAccessTokenScopes)
	usedRefreshIntrospection := service.Introspect(ctx, refreshToken, gateway.app.ClientID)
	assert.Equal(t, false, usedRefreshIntrospection["active"])
	rotatedRefreshIntrospection := service.Introspect(ctx, rotatedRefreshToken, gateway.app.ClientID)
	assert.Equal(t, true, rotatedRefreshIntrospection["active"])
	usedRefreshToken, err := service.loadUsedRefreshToken(ctx, refreshToken)
	require.NoError(t, err)
	assert.Empty(t, auditEvents)

	_, err = service.Refresh(ctx, TokenRequest{
		GrantType:    "refresh_token",
		RefreshToken: rotatedRefreshToken,
		ClientID:     gateway.otherApp.ClientID,
		ClientSecret: gateway.otherSecret,
	})
	require.ErrorIs(t, err, ErrInvalidGrant)

	require.NoError(t, service.Revoke(ctx, rotatedRefreshToken, gateway.otherApp.ClientID))
	assert.Empty(t, auditEvents)
	stillUsable, err := service.Refresh(ctx, TokenRequest{
		GrantType:    "refresh_token",
		RefreshToken: rotatedRefreshToken,
		ClientID:     gateway.app.ClientID,
		ClientSecret: gateway.clientSecret,
	})
	require.NoError(t, err)
	nextRefreshToken := stillUsable["refresh_token"].(string)

	_, err = service.Refresh(ctx, TokenRequest{
		GrantType:    "refresh_token",
		RefreshToken: refreshToken,
		ClientID:     gateway.app.ClientID,
		ClientSecret: gateway.clientSecret,
	})
	require.ErrorIs(t, err, ErrInvalidGrant)
	require.Len(t, auditEvents, 1)
	assert.Equal(t, audit.EventTokenRevoked, auditEvents[0].Type)
	assert.Equal(t, "system", auditEvents[0].ActorType)
	assert.Equal(t, "42", auditEvents[0].UserID)
	assert.Equal(t, "identity.refresh_token_family", auditEvents[0].ResourceType)
	assert.Equal(t, "identity_refresh_reuse_detected", auditEvents[0].Action)
	assert.Equal(t, "failure", auditEvents[0].Result)
	assert.Equal(t, refreshTokenFamilyAuditID(usedRefreshToken.FamilyID), auditEvents[0].ResourceID)
	assert.NotEqual(t, usedRefreshToken.FamilyID, auditEvents[0].ResourceID)
	assert.NotEqual(t, refreshToken, auditEvents[0].ResourceID)
	assert.NotEqual(t, nextRefreshToken, auditEvents[0].ResourceID)
	assert.Equal(t, gateway.app.ClientID, auditEvents[0].Details["clientID"])
	assert.Equal(t, refreshTokenFamilyAuditID(usedRefreshToken.FamilyID), auditEvents[0].Details["familyHash"])
	assert.Equal(t, usedRefreshToken.Generation, auditEvents[0].Details["generation"])
	assert.Equal(t, len(usedRefreshToken.Scopes), auditEvents[0].Details["scopeCount"])
	assert.NotContains(t, auditEvents[0].Details, "refreshToken")
	assert.NotContains(t, auditEvents[0].Details, "familyID")
	_, err = service.Refresh(ctx, TokenRequest{
		GrantType:    "refresh_token",
		RefreshToken: nextRefreshToken,
		ClientID:     gateway.app.ClientID,
		ClientSecret: gateway.clientSecret,
	})
	require.ErrorIs(t, err, ErrInvalidGrant)
	afterReplayIntrospection := service.Introspect(ctx, nextRefreshToken, gateway.app.ClientID)
	assert.Equal(t, false, afterReplayIntrospection["active"])

	redirectURL, err = service.Begin(ctx, AuthorizeRequest{
		ResponseType:        "code",
		ClientID:            gateway.app.ClientID,
		RedirectURI:         "https://client.example.com/callback",
		Scope:               "openid profile offline_access",
		CodeChallenge:       s256Challenge(verifier),
		CodeChallengeMethod: "S256",
	}, 42)
	require.NoError(t, err)
	callback, err = url.Parse(redirectURL)
	require.NoError(t, err)
	tokenSet, err = service.ExchangeCode(ctx, TokenRequest{
		GrantType:    "authorization_code",
		Code:         callback.Query().Get("code"),
		RedirectURI:  "https://client.example.com/callback",
		CodeVerifier: verifier,
		ClientID:     gateway.app.ClientID,
		ClientSecret: gateway.clientSecret,
	})
	require.NoError(t, err)
	explicitRevokedRefreshToken := tokenSet["refresh_token"].(string)
	explicitRevokedPayload, err := service.loadRefreshToken(ctx, explicitRevokedRefreshToken)
	require.NoError(t, err)
	require.NoError(t, service.Revoke(ctx, explicitRevokedRefreshToken, gateway.app.ClientID))
	require.Len(t, auditEvents, 2)
	assert.Equal(t, audit.EventTokenRevoked, auditEvents[1].Type)
	assert.Equal(t, "client", auditEvents[1].ActorType)
	assert.Equal(t, "42", auditEvents[1].UserID)
	assert.Equal(t, "identity.refresh_token_family", auditEvents[1].ResourceType)
	assert.Equal(t, refreshTokenFamilyAuditID(explicitRevokedPayload.FamilyID), auditEvents[1].ResourceID)
	assert.Equal(t, "identity_refresh_token_revoked", auditEvents[1].Action)
	assert.Equal(t, "success", auditEvents[1].Result)
	assert.Equal(t, gateway.app.ClientID, auditEvents[1].Details["clientID"])
	assert.Equal(t, "refresh_token", auditEvents[1].Details["tokenType"])
	assert.Equal(t, refreshTokenFamilyAuditID(explicitRevokedPayload.FamilyID), auditEvents[1].Details["familyHash"])
	assert.NotEqual(t, explicitRevokedPayload.FamilyID, auditEvents[1].ResourceID)
	assert.NotEqual(t, explicitRevokedRefreshToken, auditEvents[1].ResourceID)
	assert.NotContains(t, auditEvents[1].Details, "refreshToken")
	assert.NotContains(t, auditEvents[1].Details, "familyID")
	_, err = service.Refresh(ctx, TokenRequest{
		GrantType:    "refresh_token",
		RefreshToken: explicitRevokedRefreshToken,
		ClientID:     gateway.app.ClientID,
		ClientSecret: gateway.clientSecret,
	})
	require.ErrorIs(t, err, ErrInvalidGrant)
	revokedRefreshIntrospection := service.Introspect(ctx, explicitRevokedRefreshToken, gateway.app.ClientID)
	assert.Equal(t, false, revokedRefreshIntrospection["active"])

	redirectURL, err = service.Begin(ctx, AuthorizeRequest{
		ResponseType:        "code",
		ClientID:            gateway.app.ClientID,
		RedirectURI:         "https://client.example.com/callback",
		Scope:               "openid profile offline_access",
		CodeChallenge:       s256Challenge(verifier),
		CodeChallengeMethod: "S256",
	}, 42)
	require.NoError(t, err)
	callback, err = url.Parse(redirectURL)
	require.NoError(t, err)
	tokenSet, err = service.ExchangeCode(ctx, TokenRequest{
		GrantType:    "authorization_code",
		Code:         callback.Query().Get("code"),
		RedirectURI:  "https://client.example.com/callback",
		CodeVerifier: verifier,
		ClientID:     gateway.app.ClientID,
		ClientSecret: gateway.clientSecret,
	})
	require.NoError(t, err)
	gateway.identityAccessTokenActive = false
	inactiveByAuthorization := service.Introspect(ctx, tokenSet["refresh_token"].(string), gateway.app.ClientID)
	assert.Equal(t, false, inactiveByAuthorization["active"])
	_, err = service.Refresh(ctx, TokenRequest{
		GrantType:    "refresh_token",
		RefreshToken: tokenSet["refresh_token"].(string),
		ClientID:     gateway.app.ClientID,
		ClientSecret: gateway.clientSecret,
	})
	require.ErrorIs(t, err, ErrInvalidGrant)
}

func TestRefreshTokenWithoutOpenIDIsInactive(t *testing.T) {
	ctx := context.Background()
	gateway := newFakeIdentityOpenPlatform()
	service, _ := newIdentityTestService(t, gateway)

	refreshToken, err := service.issueRefreshToken(ctx, RefreshToken{
		ClientID:                 gateway.app.ClientID,
		AuthorizationFingerprint: gateway.identityAuthorizationHash,
		Scopes:                   []string{openplatform.ScopeOfflineAccess},
		UserID:                   42,
		Subject:                  identitySubject(42),
	})
	require.NoError(t, err)

	introspection := service.Introspect(ctx, refreshToken, gateway.app.ClientID)
	assert.Equal(t, false, introspection["active"])

	_, err = service.Refresh(ctx, TokenRequest{
		GrantType:    "refresh_token",
		RefreshToken: refreshToken,
		ClientID:     gateway.app.ClientID,
		ClientSecret: gateway.clientSecret,
	})
	require.ErrorIs(t, err, ErrInvalidGrant)
	assert.Empty(t, gateway.identityAccessTokenScopes)
}

func TestRevokeUsedRefreshTokenRevokesCurrentFamily(t *testing.T) {
	ctx := context.Background()
	gateway := newFakeIdentityOpenPlatform()
	service, _ := newIdentityTestService(t, gateway)
	verifier := validTestPKCEVerifier()
	var auditEvents []audit.Event
	service.audit = func(_ context.Context, event audit.Event) {
		auditEvents = append(auditEvents, event)
	}

	redirectURL, err := service.Begin(ctx, AuthorizeRequest{
		ResponseType:        "code",
		ClientID:            gateway.app.ClientID,
		RedirectURI:         "https://client.example.com/callback",
		Scope:               "openid profile offline_access",
		State:               "state-used-revoke",
		CodeChallenge:       s256Challenge(verifier),
		CodeChallengeMethod: "S256",
	}, 42)
	require.NoError(t, err)
	callback, err := url.Parse(redirectURL)
	require.NoError(t, err)

	tokenSet, err := service.ExchangeCode(ctx, TokenRequest{
		GrantType:    "authorization_code",
		Code:         callback.Query().Get("code"),
		RedirectURI:  "https://client.example.com/callback",
		CodeVerifier: verifier,
		ClientID:     gateway.app.ClientID,
		ClientSecret: gateway.clientSecret,
	})
	require.NoError(t, err)
	refreshToken := tokenSet["refresh_token"].(string)

	refreshed, err := service.Refresh(ctx, TokenRequest{
		GrantType:    "refresh_token",
		RefreshToken: refreshToken,
		ClientID:     gateway.app.ClientID,
		ClientSecret: gateway.clientSecret,
	})
	require.NoError(t, err)
	rotatedRefreshToken := refreshed["refresh_token"].(string)
	rotatedRefreshIntrospection := service.Introspect(ctx, rotatedRefreshToken, gateway.app.ClientID)
	assert.Equal(t, true, rotatedRefreshIntrospection["active"])
	usedRefreshToken, err := service.loadUsedRefreshToken(ctx, refreshToken)
	require.NoError(t, err)

	require.NoError(t, service.Revoke(ctx, refreshToken, gateway.otherApp.ClientID))
	assert.Empty(t, auditEvents)
	stillActive := service.Introspect(ctx, rotatedRefreshToken, gateway.app.ClientID)
	assert.Equal(t, true, stillActive["active"])
	_, err = service.loadUsedRefreshToken(ctx, refreshToken)
	require.NoError(t, err)

	require.NoError(t, service.Revoke(ctx, refreshToken, gateway.app.ClientID, "access_token"))
	require.Len(t, auditEvents, 1)
	assert.Equal(t, audit.EventTokenRevoked, auditEvents[0].Type)
	assert.Equal(t, "client", auditEvents[0].ActorType)
	assert.Equal(t, "42", auditEvents[0].UserID)
	assert.Equal(t, "identity.refresh_token_family", auditEvents[0].ResourceType)
	assert.Equal(t, refreshTokenFamilyAuditID(usedRefreshToken.FamilyID), auditEvents[0].ResourceID)
	assert.Equal(t, "identity_refresh_token_revoked", auditEvents[0].Action)
	assert.Equal(t, "success", auditEvents[0].Result)
	assert.Equal(t, gateway.app.ClientID, auditEvents[0].Details["clientID"])
	assert.Equal(t, "refresh_token", auditEvents[0].Details["tokenType"])
	assert.Equal(t, refreshTokenFamilyAuditID(usedRefreshToken.FamilyID), auditEvents[0].Details["familyHash"])
	assert.Equal(t, usedRefreshToken.Generation, auditEvents[0].Details["generation"])
	assert.Equal(t, len(usedRefreshToken.Scopes), auditEvents[0].Details["scopeCount"])
	assert.NotEqual(t, usedRefreshToken.FamilyID, auditEvents[0].ResourceID)
	assert.NotEqual(t, refreshToken, auditEvents[0].ResourceID)
	assert.NotEqual(t, rotatedRefreshToken, auditEvents[0].ResourceID)
	assert.NotContains(t, auditEvents[0].Details, "refreshToken")
	assert.NotContains(t, auditEvents[0].Details, "familyID")

	afterRevoke := service.Introspect(ctx, rotatedRefreshToken, gateway.app.ClientID)
	assert.Equal(t, false, afterRevoke["active"])
	_, err = service.Refresh(ctx, TokenRequest{
		GrantType:    "refresh_token",
		RefreshToken: rotatedRefreshToken,
		ClientID:     gateway.app.ClientID,
		ClientSecret: gateway.clientSecret,
	})
	require.ErrorIs(t, err, ErrInvalidGrant)
	_, err = service.loadUsedRefreshToken(ctx, refreshToken)
	require.ErrorIs(t, err, ErrInvalidGrant)

	require.NoError(t, service.Revoke(ctx, refreshToken, gateway.app.ClientID))
	require.Len(t, auditEvents, 1)
}

func TestAccessTokenRejectsStaleAuthorizationFingerprint(t *testing.T) {
	ctx := context.Background()
	gateway := newFakeIdentityOpenPlatform()
	service, _ := newIdentityTestService(t, gateway)
	verifier := validTestPKCEVerifier()

	redirectURL, err := service.Begin(ctx, AuthorizeRequest{
		ResponseType:        "code",
		ClientID:            gateway.app.ClientID,
		RedirectURI:         "https://client.example.com/callback",
		Scope:               "openid profile email",
		CodeChallenge:       s256Challenge(verifier),
		CodeChallengeMethod: "S256",
	}, 42)
	require.NoError(t, err)
	callback, err := url.Parse(redirectURL)
	require.NoError(t, err)

	tokenSet, err := service.ExchangeCode(ctx, TokenRequest{
		GrantType:    "authorization_code",
		Code:         callback.Query().Get("code"),
		RedirectURI:  "https://client.example.com/callback",
		CodeVerifier: verifier,
		ClientID:     gateway.app.ClientID,
		ClientSecret: gateway.clientSecret,
	})
	require.NoError(t, err)
	accessToken := tokenSet["access_token"].(string)
	claims := parseIdentityTestToken(t, service, accessToken)
	assert.Equal(t, "identity-authorization-fingerprint-1", claims["stuhelper_authz"])

	introspection := service.Introspect(ctx, accessToken, gateway.app.ClientID)
	assert.Equal(t, true, introspection["active"])
	_, err = service.UserInfo(ctx, accessToken)
	require.NoError(t, err)

	gateway.identityAuthorizationHash = "identity-authorization-fingerprint-after-reconsent"
	staleIntrospection := service.Introspect(ctx, accessToken, gateway.app.ClientID)
	assert.Equal(t, false, staleIntrospection["active"])
	userInfoCalls := gateway.userInfoCallCount
	_, err = service.UserInfo(ctx, accessToken)
	require.Error(t, err)
	assert.Equal(t, userInfoCalls, gateway.userInfoCallCount)
}

func TestRefreshTokenGrantRejectsStaleAuthorizationFingerprintAfterReconsent(t *testing.T) {
	ctx := context.Background()
	gateway := newFakeIdentityOpenPlatform()
	service, _ := newIdentityTestService(t, gateway)
	verifier := validTestPKCEVerifier()

	redirectURL, err := service.Begin(ctx, AuthorizeRequest{
		ResponseType:        "code",
		ClientID:            gateway.app.ClientID,
		RedirectURI:         "https://client.example.com/callback",
		Scope:               "openid profile offline_access",
		CodeChallenge:       s256Challenge(verifier),
		CodeChallengeMethod: "S256",
	}, 42)
	require.NoError(t, err)
	callback, err := url.Parse(redirectURL)
	require.NoError(t, err)

	tokenSet, err := service.ExchangeCode(ctx, TokenRequest{
		GrantType:    "authorization_code",
		Code:         callback.Query().Get("code"),
		RedirectURI:  "https://client.example.com/callback",
		CodeVerifier: verifier,
		ClientID:     gateway.app.ClientID,
		ClientSecret: gateway.clientSecret,
	})
	require.NoError(t, err)
	refreshToken := tokenSet["refresh_token"].(string)
	payload, err := service.loadRefreshToken(ctx, refreshToken)
	require.NoError(t, err)
	assert.Equal(t, "identity-authorization-fingerprint-1", payload.AuthorizationFingerprint)

	gateway.identityAuthorizationHash = "identity-authorization-fingerprint-after-reconsent"
	staleIntrospection := service.Introspect(ctx, refreshToken, gateway.app.ClientID)
	assert.Equal(t, false, staleIntrospection["active"])
	userInfoCalls := gateway.userInfoCallCount
	_, err = service.Refresh(ctx, TokenRequest{
		GrantType:    "refresh_token",
		RefreshToken: refreshToken,
		ClientID:     gateway.app.ClientID,
		ClientSecret: gateway.clientSecret,
	})
	require.ErrorIs(t, err, ErrInvalidGrant)
	assert.Equal(t, userInfoCalls, gateway.userInfoCallCount)
}

func TestRefreshTokenGrantNarrowsScopesAndRejectsExpansion(t *testing.T) {
	ctx := context.Background()
	gateway := newFakeIdentityOpenPlatform()
	service, _ := newIdentityTestService(t, gateway)
	verifier := validTestPKCEVerifier()

	redirectURL, err := service.Begin(ctx, AuthorizeRequest{
		ResponseType:        "code",
		ClientID:            gateway.app.ClientID,
		RedirectURI:         "https://client.example.com/callback",
		Scope:               "openid profile email offline_access",
		State:               "state-refresh-narrow",
		CodeChallenge:       s256Challenge(verifier),
		CodeChallengeMethod: "S256",
	}, 42)
	require.NoError(t, err)
	callback, err := url.Parse(redirectURL)
	require.NoError(t, err)

	tokenSet, err := service.ExchangeCode(ctx, TokenRequest{
		GrantType:    "authorization_code",
		Code:         callback.Query().Get("code"),
		RedirectURI:  "https://client.example.com/callback",
		CodeVerifier: verifier,
		ClientID:     gateway.app.ClientID,
		ClientSecret: gateway.clientSecret,
	})
	require.NoError(t, err)
	refreshToken := tokenSet["refresh_token"].(string)

	_, err = service.Refresh(ctx, TokenRequest{
		GrantType:    "refresh_token",
		RefreshToken: refreshToken,
		Scope:        "openid profile email phone offline_access",
		ClientID:     gateway.app.ClientID,
		ClientSecret: gateway.clientSecret,
	})
	require.ErrorIs(t, err, ErrInvalidScope)
	stillActive := service.Introspect(ctx, refreshToken, gateway.app.ClientID)
	assert.Equal(t, true, stillActive["active"])
	assert.Equal(t, "openid profile email offline_access", stillActive["scope"])

	_, err = service.Refresh(ctx, TokenRequest{
		GrantType:    "refresh_token",
		RefreshToken: refreshToken,
		Scope:        "openid profile",
		ClientID:     gateway.app.ClientID,
		ClientSecret: gateway.clientSecret,
	})
	require.ErrorIs(t, err, ErrInvalidScope)

	narrowed, err := service.Refresh(ctx, TokenRequest{
		GrantType:    "refresh_token",
		RefreshToken: refreshToken,
		Scope:        "openid profile offline_access",
		ClientID:     gateway.app.ClientID,
		ClientSecret: gateway.clientSecret,
	})
	require.NoError(t, err)
	assert.Equal(t, "openid profile offline_access", narrowed["scope"])
	assert.Equal(t, []string{"openid", "profile", openplatform.ScopeOfflineAccess}, gateway.identityAccessTokenScopes)
	assert.Equal(t, []string{"openid", "profile", openplatform.ScopeOfflineAccess}, gateway.userInfoScopes)

	accessClaims, err := service.signer.VerifyAccessToken(narrowed["access_token"].(string))
	require.NoError(t, err)
	assert.Equal(t, []string{"openid", "profile", openplatform.ScopeOfflineAccess}, accessClaims.Scopes)

	rotatedRefreshToken := narrowed["refresh_token"].(string)
	rotatedIntrospection := service.Introspect(ctx, rotatedRefreshToken, gateway.app.ClientID)
	assert.Equal(t, true, rotatedIntrospection["active"])
	assert.Equal(t, "openid profile offline_access", rotatedIntrospection["scope"])

	refreshedAgain, err := service.Refresh(ctx, TokenRequest{
		GrantType:    "refresh_token",
		RefreshToken: rotatedRefreshToken,
		ClientID:     gateway.app.ClientID,
		ClientSecret: gateway.clientSecret,
	})
	require.NoError(t, err)
	assert.Equal(t, "openid profile offline_access", refreshedAgain["scope"])
}

func TestRefreshTokenGrantRequiresCurrentFamilyToken(t *testing.T) {
	ctx := context.Background()
	gateway := newFakeIdentityOpenPlatform()
	service, fixture := newIdentityTestService(t, gateway)
	verifier := validTestPKCEVerifier()

	redirectURL, err := service.Begin(ctx, AuthorizeRequest{
		ResponseType:        "code",
		ClientID:            gateway.app.ClientID,
		RedirectURI:         "https://client.example.com/callback",
		Scope:               "openid profile offline_access",
		State:               "state-refresh-family",
		CodeChallenge:       s256Challenge(verifier),
		CodeChallengeMethod: "S256",
	}, 42)
	require.NoError(t, err)
	callback, err := url.Parse(redirectURL)
	require.NoError(t, err)

	tokenSet, err := service.ExchangeCode(ctx, TokenRequest{
		GrantType:    "authorization_code",
		Code:         callback.Query().Get("code"),
		RedirectURI:  "https://client.example.com/callback",
		CodeVerifier: verifier,
		ClientID:     gateway.app.ClientID,
		ClientSecret: gateway.clientSecret,
	})
	require.NoError(t, err)
	refreshToken := tokenSet["refresh_token"].(string)
	payload, err := service.loadRefreshToken(ctx, refreshToken)
	require.NoError(t, err)
	callsBefore := gateway.userInfoCallCount
	gateway.identityAccessTokenClientID = ""

	rawFamily, err := json.Marshal(refreshTokenFamily{
		ClientID:   payload.ClientID,
		CurrentKey: refreshTokenKey("different-current-refresh-token"),
	})
	require.NoError(t, err)
	require.NoError(t, fixture.Client.Set(ctx, refreshTokenFamilyKey(payload.FamilyID), rawFamily, time.Hour).Err())

	staleIntrospection := service.Introspect(ctx, refreshToken, gateway.app.ClientID)
	assert.Equal(t, false, staleIntrospection["active"])
	_, err = service.Refresh(ctx, TokenRequest{
		GrantType:    "refresh_token",
		RefreshToken: refreshToken,
		ClientID:     gateway.app.ClientID,
		ClientSecret: gateway.clientSecret,
	})
	require.ErrorIs(t, err, ErrInvalidGrant)
	assert.Empty(t, gateway.identityAccessTokenClientID)
	assert.Equal(t, callsBefore, gateway.userInfoCallCount)

	gateway.identityAccessTokenClientID = ""
	require.NoError(t, fixture.Client.Del(ctx, refreshTokenFamilyKey(payload.FamilyID)).Err())
	revokedFamilyIntrospection := service.Introspect(ctx, refreshToken, gateway.app.ClientID)
	assert.Equal(t, false, revokedFamilyIntrospection["active"])
	_, err = service.Refresh(ctx, TokenRequest{
		GrantType:    "refresh_token",
		RefreshToken: refreshToken,
		ClientID:     gateway.app.ClientID,
		ClientSecret: gateway.clientSecret,
	})
	require.ErrorIs(t, err, ErrInvalidGrant)
	assert.Empty(t, gateway.identityAccessTokenClientID)
	assert.Equal(t, callsBefore, gateway.userInfoCallCount)
}

func TestRefreshTokenGrantDoesNotDiscloseWhenTokenIsConsumedBeforeRotation(t *testing.T) {
	ctx := context.Background()
	gateway := newFakeIdentityOpenPlatform()
	service, _ := newIdentityTestService(t, gateway)
	verifier := validTestPKCEVerifier()

	redirectURL, err := service.Begin(ctx, AuthorizeRequest{
		ResponseType:        "code",
		ClientID:            gateway.app.ClientID,
		RedirectURI:         "https://client.example.com/callback",
		Scope:               "openid profile offline_access",
		State:               "state-refresh-race",
		CodeChallenge:       s256Challenge(verifier),
		CodeChallengeMethod: "S256",
	}, 42)
	require.NoError(t, err)
	callback, err := url.Parse(redirectURL)
	require.NoError(t, err)

	tokenSet, err := service.ExchangeCode(ctx, TokenRequest{
		GrantType:    "authorization_code",
		Code:         callback.Query().Get("code"),
		RedirectURI:  "https://client.example.com/callback",
		CodeVerifier: verifier,
		ClientID:     gateway.app.ClientID,
		ClientSecret: gateway.clientSecret,
	})
	require.NoError(t, err)
	refreshToken := tokenSet["refresh_token"].(string)
	callsBefore := gateway.userInfoCallCount
	var hookErr error
	gateway.identityAccessTokenHook = func() {
		gateway.identityAccessTokenHook = nil
		_, hookErr = service.consumeRefreshToken(ctx, refreshToken)
	}

	_, err = service.Refresh(ctx, TokenRequest{
		GrantType:    "refresh_token",
		RefreshToken: refreshToken,
		ClientID:     gateway.app.ClientID,
		ClientSecret: gateway.clientSecret,
	})
	require.ErrorIs(t, err, ErrInvalidGrant)
	require.NoError(t, hookErr)
	assert.Equal(t, callsBefore, gateway.userInfoCallCount)
}

func TestBeginUsesGrantedOAuthScopesForTokenContract(t *testing.T) {
	ctx := context.Background()
	gateway := newFakeIdentityOpenPlatform()
	gateway.authorizationScopes = []string{openplatform.ScopeProfileBasicRead, openplatform.ScopeEmailRead, openplatform.ScopeResourceRead}
	gateway.authorizationOAuthScopes = []string{"openid", "profile", "email", openplatform.ScopeResourceRead}
	service, _ := newIdentityTestService(t, gateway)
	verifier := validTestPKCEVerifier()

	redirectURL, err := service.Begin(ctx, AuthorizeRequest{
		ResponseType:        "code",
		ClientID:            gateway.app.ClientID,
		RedirectURI:         "https://client.example.com/callback",
		Scope:               "openid profile email " + openplatform.ScopeResourceRead,
		State:               "state-oauth-scope",
		CodeChallenge:       s256Challenge(verifier),
		CodeChallengeMethod: "S256",
	}, 42)
	require.NoError(t, err)

	callback, err := url.Parse(redirectURL)
	require.NoError(t, err)
	tokenSet, err := service.ExchangeCode(ctx, TokenRequest{
		GrantType:    "authorization_code",
		Code:         callback.Query().Get("code"),
		RedirectURI:  "https://client.example.com/callback",
		CodeVerifier: verifier,
		ClientID:     gateway.app.ClientID,
		ClientSecret: gateway.clientSecret,
	})
	require.NoError(t, err)
	assert.Equal(t, "openid profile email "+openplatform.ScopeResourceRead, tokenSet["scope"])
	assert.Equal(t, []string{"openid", "profile", "email", openplatform.ScopeResourceRead}, gateway.userInfoScopes)

	accessToken := tokenSet["access_token"].(string)
	claims, err := service.signer.VerifyAccessToken(accessToken)
	require.NoError(t, err)
	assert.Equal(t, []string{"openid", "profile", "email", openplatform.ScopeResourceRead}, claims.Scopes)

	active := service.Introspect(ctx, accessToken, gateway.app.ClientID)
	assert.Equal(t, true, active["active"])
	assert.Equal(t, "openid profile email "+openplatform.ScopeResourceRead, active["scope"])
	assert.Equal(t, []string{"openid", "profile", "email", openplatform.ScopeResourceRead}, gateway.identityAccessTokenScopes)
}

func TestIntrospectRequiresCurrentOpenPlatformAuthorization(t *testing.T) {
	ctx := context.Background()
	gateway := newFakeIdentityOpenPlatform()
	service, _ := newIdentityTestService(t, gateway)

	accessToken, _, err := service.signer.SignAccessToken(AccessTokenInput{
		Subject:                  "stuhelper:42",
		ClientID:                 gateway.app.ClientID,
		UserID:                   42,
		Scopes:                   []string{"openid", "profile", "email"},
		AuthorizationFingerprint: gateway.identityAuthorizationHash,
		TTL:                      15 * time.Minute,
	})
	require.NoError(t, err)

	active := service.Introspect(ctx, accessToken, gateway.app.ClientID)
	assert.Equal(t, true, active["active"])
	assert.Equal(t, gateway.app.ClientID, gateway.identityAccessTokenClientID)
	assert.Equal(t, int64(42), gateway.identityAccessTokenUserID)
	assert.Equal(t, []string{"openid", "profile", "email"}, gateway.identityAccessTokenScopes)

	gateway.identityAccessTokenActive = false
	inactiveAfterConsentRevoke := service.Introspect(ctx, accessToken, gateway.app.ClientID)
	assert.Equal(t, false, inactiveAfterConsentRevoke["active"])

	gateway.identityAccessTokenActive = true
	gateway.identityAccessTokenErr = errors.New("authorization lookup unavailable")
	inactiveOnLookupError := service.Introspect(ctx, accessToken, gateway.app.ClientID)
	assert.Equal(t, false, inactiveOnLookupError["active"])
}

func TestExchangeCodeRejectsPKCEMismatch(t *testing.T) {
	ctx := context.Background()
	gateway := newFakeIdentityOpenPlatform()
	service, _ := newIdentityTestService(t, gateway)
	verifier := validTestPKCEVerifier()

	redirectURL, err := service.Begin(ctx, AuthorizeRequest{
		ResponseType:        "code",
		ClientID:            gateway.app.ClientID,
		RedirectURI:         "https://client.example.com/callback",
		Scope:               "openid profile",
		CodeChallenge:       s256Challenge(verifier),
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

func TestExchangeCodeRejectsOfflineAccessCodeWithoutOpenID(t *testing.T) {
	ctx := context.Background()
	gateway := newFakeIdentityOpenPlatform()
	service, _ := newIdentityTestService(t, gateway)
	verifier := validTestPKCEVerifier()

	redirectURL, err := service.issueCodeRedirect(ctx, gateway.app.ClientID, "https://client.example.com/callback", []string{openplatform.ScopeResourceRead, openplatform.ScopeOfflineAccess}, 42, identitySubject(42), AuthorizeRequest{
		CodeChallenge:       s256Challenge(verifier),
		CodeChallengeMethod: "S256",
	})
	require.NoError(t, err)
	callback, err := url.Parse(redirectURL)
	require.NoError(t, err)

	_, err = service.ExchangeCode(ctx, TokenRequest{
		GrantType:    "authorization_code",
		Code:         callback.Query().Get("code"),
		RedirectURI:  "https://client.example.com/callback",
		CodeVerifier: verifier,
		ClientID:     gateway.app.ClientID,
		ClientSecret: gateway.clientSecret,
	})
	require.ErrorIs(t, err, ErrInvalidGrant)
}

func TestIDTokenProfileClaimsUseAuthorizedClaimNamesAndOIDCAliases(t *testing.T) {
	claims := idTokenProfileClaims(map[string]any{
		"sub":                "must-not-override-token-sub",
		"username":           "alice",
		"displayName":        "Alice Zhang",
		"avatar":             "https://cdn.example.com/alice.png",
		"email":              "alice@example.com",
		"phone":              "13800138000",
		"phoneMasked":        "138****8000",
		"phoneVerified":      true,
		"identityVerified":   true,
		"identityType":       "student",
		"studentVerified":    true,
		"school":             map[string]any{"id": float64(1), "name": "Test University"},
		"unapprovedClaim":    "must-not-leak",
		"stuhelper_id":       int64(42),
		"preferred_username": "must-be-overwritten",
	})

	assert.Equal(t, "alice", claims["username"])
	assert.Equal(t, "Alice Zhang", claims["displayName"])
	assert.Equal(t, "https://cdn.example.com/alice.png", claims["avatar"])
	assert.Equal(t, "alice@example.com", claims["email"])
	assert.Equal(t, "13800138000", claims["phone"])
	assert.Equal(t, "13800138000", claims["phone_number"])
	assert.Equal(t, "138****8000", claims["phoneMasked"])
	assert.Equal(t, true, claims["phoneVerified"])
	assert.Equal(t, true, claims["phone_number_verified"])
	assert.Equal(t, true, claims["identityVerified"])
	assert.Equal(t, "student", claims["identityType"])
	assert.Equal(t, true, claims["studentVerified"])
	assert.Equal(t, map[string]any{"id": float64(1), "name": "Test University"}, claims["school"])
	assert.Equal(t, "alice", claims["preferred_username"])
	assert.Equal(t, "Alice Zhang", claims["name"])
	assert.Equal(t, "https://cdn.example.com/alice.png", claims["picture"])
	assert.Equal(t, true, claims["email_verified"])
	assert.NotContains(t, claims, "sub")
	assert.NotContains(t, claims, "unapprovedClaim")
	assert.NotContains(t, claims, "stuhelper_id")
	assert.NotContains(t, claims, "stuhelper_identity_type")
	assert.NotContains(t, claims, "stuhelper_student_verified")
}

func TestExchangeCodeRequiresExactRedirectURI(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		redirectURI string
	}{
		{name: "missing", redirectURI: ""},
		{name: "mismatched", redirectURI: "https://client.example.com/other-callback"},
		{name: "trailing whitespace", redirectURI: "https://client.example.com/callback "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gateway := newFakeIdentityOpenPlatform()
			service, _ := newIdentityTestService(t, gateway)
			verifier := validTestPKCEVerifier()

			redirectURL, err := service.Begin(ctx, AuthorizeRequest{
				ResponseType:        "code",
				ClientID:            gateway.app.ClientID,
				RedirectURI:         "https://client.example.com/callback",
				Scope:               "openid profile",
				CodeChallenge:       s256Challenge(verifier),
				CodeChallengeMethod: "S256",
			}, 42)
			require.NoError(t, err)
			callback, err := url.Parse(redirectURL)
			require.NoError(t, err)

			_, err = service.ExchangeCode(ctx, TokenRequest{
				GrantType:    "authorization_code",
				Code:         callback.Query().Get("code"),
				RedirectURI:  tt.redirectURI,
				CodeVerifier: verifier,
				ClientID:     gateway.app.ClientID,
				ClientSecret: gateway.clientSecret,
			})
			require.ErrorIs(t, err, ErrInvalidGrant)
		})
	}
}

func TestExchangeCodeRejectsInvalidPKCEVerifier(t *testing.T) {
	ctx := context.Background()
	gateway := newFakeIdentityOpenPlatform()
	service, _ := newIdentityTestService(t, gateway)
	verifier := validTestPKCEVerifier()

	redirectURL, err := service.Begin(ctx, AuthorizeRequest{
		ResponseType:        "code",
		ClientID:            gateway.app.ClientID,
		RedirectURI:         "https://client.example.com/callback",
		Scope:               "openid profile",
		CodeChallenge:       s256Challenge(verifier),
		CodeChallengeMethod: "S256",
	}, 42)
	require.NoError(t, err)
	callback, err := url.Parse(redirectURL)
	require.NoError(t, err)

	_, err = service.ExchangeCode(ctx, TokenRequest{
		GrantType:    "authorization_code",
		Code:         callback.Query().Get("code"),
		RedirectURI:  "https://client.example.com/callback",
		CodeVerifier: "too-short",
		ClientID:     gateway.app.ClientID,
		ClientSecret: gateway.clientSecret,
	})
	require.ErrorIs(t, err, ErrInvalidGrant)
}

func TestExchangeCodeRejectsWhitespacePKCEVerifier(t *testing.T) {
	ctx := context.Background()
	gateway := newFakeIdentityOpenPlatform()
	service, _ := newIdentityTestService(t, gateway)
	verifier := validTestPKCEVerifier()

	redirectURL, err := service.Begin(ctx, AuthorizeRequest{
		ResponseType:        "code",
		ClientID:            gateway.app.ClientID,
		RedirectURI:         "https://client.example.com/callback",
		Scope:               "openid profile",
		CodeChallenge:       s256Challenge(verifier),
		CodeChallengeMethod: "S256",
	}, 42)
	require.NoError(t, err)
	callback, err := url.Parse(redirectURL)
	require.NoError(t, err)

	_, err = service.ExchangeCode(ctx, TokenRequest{
		GrantType:    "authorization_code",
		Code:         callback.Query().Get("code"),
		RedirectURI:  "https://client.example.com/callback",
		CodeVerifier: verifier + " ",
		ClientID:     gateway.app.ClientID,
		ClientSecret: gateway.clientSecret,
	})
	require.ErrorIs(t, err, ErrInvalidGrant)
}

func TestIssueCodeFromConsentChallengeRequiresPKCE(t *testing.T) {
	ctx := context.Background()
	gateway := newFakeIdentityOpenPlatform()
	service, fixture := newIdentityTestService(t, gateway)
	gateway.consentChallenge = &openplatform.ConsentChallenge{
		Token:       "consent-token",
		AppID:       gateway.app.ID,
		UserID:      42,
		Scopes:      []string{"openid", "profile"},
		RedirectURI: "https://client.example.com/callback",
		Flow:        openplatform.AuthorizeFlowIdentity,
	}

	redirectURL, err := service.IssueCodeFromConsentChallenge(ctx, "consent-token", 42, time.Time{})

	require.ErrorIs(t, err, ErrInvalidAuthorizeRequest)
	assert.Empty(t, redirectURL)
	assert.Empty(t, gateway.deletedChallenge)
	keys, err := fixture.Client.Keys(ctx, authCodeRedisPrefix+"*").Result()
	require.NoError(t, err)
	assert.Empty(t, keys)
}

func TestIssueCodeFromConsentChallengeRejectsRedirectURIDrift(t *testing.T) {
	ctx := context.Background()
	gateway := newFakeIdentityOpenPlatform()
	service, fixture := newIdentityTestService(t, gateway)
	verifier := validTestPKCEVerifier()
	gateway.app.RedirectURIs = []string{"https://new-client.example.com/callback"}
	gateway.consentChallenge = &openplatform.ConsentChallenge{
		Token:               "consent-token",
		AppID:               gateway.app.ID,
		UserID:              42,
		Scopes:              []string{openplatform.ScopeEmailRead},
		OAuthScopes:         []string{"openid", "email"},
		RedirectURI:         "https://client.example.com/callback",
		Flow:                openplatform.AuthorizeFlowIdentity,
		CodeChallenge:       s256Challenge(verifier),
		CodeChallengeMethod: "S256",
	}

	redirectURL, err := service.IssueCodeFromConsentChallenge(ctx, "consent-token", 42, time.Time{})

	require.ErrorIs(t, err, openplatform.ErrRedirectURINotAllowed)
	assert.Empty(t, redirectURL)
	assert.Empty(t, gateway.deletedChallenge)
	assert.Zero(t, gateway.userInfoCallCount)
	assert.Empty(t, gateway.identityAccessTokenClientID)
	keys, err := fixture.Client.Keys(ctx, authCodeRedisPrefix+"*").Result()
	require.NoError(t, err)
	assert.Empty(t, keys)
}

func TestIssueCodeFromConsentChallengePreservesPKCE(t *testing.T) {
	ctx := context.Background()
	gateway := newFakeIdentityOpenPlatform()
	service, _ := newIdentityTestService(t, gateway)
	verifier := validTestPKCEVerifier()
	gateway.consentChallenge = &openplatform.ConsentChallenge{
		Token:               "consent-token",
		AppID:               gateway.app.ID,
		UserID:              42,
		Scopes:              []string{openplatform.ScopeEmailRead},
		OAuthScopes:         []string{"openid", "email"},
		RedirectURI:         "https://client.example.com/callback",
		State:               "state-consent",
		Flow:                openplatform.AuthorizeFlowIdentity,
		CodeChallenge:       s256Challenge(verifier),
		CodeChallengeMethod: "S256",
		Nonce:               "nonce-consent",
	}

	authTime := time.Date(2026, 5, 24, 10, 30, 0, 0, time.UTC)
	redirectURL, err := service.IssueCodeFromConsentChallenge(ctx, "consent-token", 42, authTime)
	require.NoError(t, err)
	assert.Equal(t, "consent-token", gateway.deletedChallenge)
	assert.Zero(t, gateway.userInfoCallCount)
	assert.Equal(t, gateway.app.ClientID, gateway.identityAccessTokenClientID)
	assert.Equal(t, int64(42), gateway.identityAccessTokenUserID)
	assert.Equal(t, []string{"openid", "email"}, gateway.identityAccessTokenScopes)
	callback, err := url.Parse(redirectURL)
	require.NoError(t, err)
	assert.Equal(t, "https://id.example.com", callback.Query().Get("iss"))
	assert.Equal(t, "state-consent", callback.Query().Get("state"))

	tokenSet, err := service.ExchangeCode(ctx, TokenRequest{
		GrantType:    "authorization_code",
		Code:         callback.Query().Get("code"),
		RedirectURI:  "https://client.example.com/callback",
		CodeVerifier: verifier,
		ClientID:     gateway.app.ClientID,
		ClientSecret: gateway.clientSecret,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, tokenSet["access_token"])
	assert.Equal(t, "openid email", tokenSet["scope"])
	assert.Equal(t, 1, gateway.userInfoCallCount)
	assert.Equal(t, []string{"openid", "email"}, gateway.userInfoScopes)
	idTokenClaims := parseIdentityTestToken(t, service, tokenSet["id_token"].(string))
	assert.Equal(t, "nonce-consent", idTokenClaims["nonce"])
	assert.Equal(t, float64(authTime.Unix()), idTokenClaims["auth_time"])
}

func newIdentityTestService(t *testing.T, gateway *fakeIdentityOpenPlatform) (*Service, *redisfixture.Fixture) {
	t.Helper()
	fixture := redisfixture.Start(t)
	signer, err := NewSigner("https://id.example.com", "identity-test-key", "")
	require.NoError(t, err)
	service, err := NewService(
		gateway,
		fixture.Client,
		signer,
		"https://id.example.com",
		5*time.Minute,
		15*time.Minute,
		defaultRefreshTokenTTL,
	)
	require.NoError(t, err)
	return service, fixture
}

func parseIdentityTestToken(t *testing.T, service *Service, raw string) jwt.MapClaims {
	t.Helper()
	token, err := jwt.Parse(raw, func(token *jwt.Token) (any, error) {
		require.Equal(t, jwt.SigningMethodRS256, token.Method)
		return &service.signer.privateKey.PublicKey, nil
	})
	require.NoError(t, err)
	require.True(t, token.Valid)
	claims, ok := token.Claims.(jwt.MapClaims)
	require.True(t, ok)
	return claims
}

func s256Challenge(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func validTestPKCEVerifier() string {
	return "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-._~"
}

type fakeIdentityOpenPlatform struct {
	app                          *openplatform.App
	otherApp                     *openplatform.App
	clientSecret                 string
	otherSecret                  string
	projection                   *openplatform.UserProjection
	profile                      map[string]any
	authorizationScopes          []string
	authorizationOAuthScopes     []string
	authorizationRequest         openplatform.AuthorizeRequest
	authorizationDecision        *openplatform.AuthorizationDecision
	userInfoScopes               []string
	userInfoCallCount            int
	consentChallenge             *openplatform.ConsentChallenge
	deletedChallenge             string
	identityAccessTokenActive    bool
	identityAccessTokenErr       error
	identityAccessTokenClientID  string
	identityAccessTokenUserID    int64
	identityAccessTokenScopes    []string
	identityAuthorizationHash    string
	identityAccessTokenHook      func()
	clientCredentialsTokenActive bool
	clientCredentialsTokenErr    error
	clientCredentialsClientID    string
	clientCredentialsScopes      []string
}

func newFakeIdentityOpenPlatform() *fakeIdentityOpenPlatform {
	return &fakeIdentityOpenPlatform{
		app: &openplatform.App{
			ID:           7,
			ClientID:     "client-1",
			RedirectURIs: []string{"https://client.example.com/callback"},
			Status:       openplatform.AppStatusApproved,
		},
		otherApp: &openplatform.App{
			ID:           8,
			ClientID:     "client-2",
			RedirectURIs: []string{"https://other-client.example.com/callback"},
			Status:       openplatform.AppStatusApproved,
		},
		clientSecret:                 "secret-1",
		otherSecret:                  "secret-2",
		identityAccessTokenActive:    true,
		identityAuthorizationHash:    "identity-authorization-fingerprint-1",
		clientCredentialsTokenActive: true,
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
	f.authorizationRequest = req
	if f.authorizationDecision != nil {
		decision := *f.authorizationDecision
		if decision.UserID == 0 {
			decision.UserID = userID
		}
		return &decision, nil
	}
	scopes := append([]string(nil), req.Scopes...)
	if f.authorizationScopes != nil {
		scopes = append([]string(nil), f.authorizationScopes...)
	}
	oauthScopes := append([]string(nil), req.Scopes...)
	if f.authorizationOAuthScopes != nil {
		oauthScopes = append([]string(nil), f.authorizationOAuthScopes...)
	}
	return &openplatform.AuthorizationDecision{
		App:         f.app,
		UserID:      userID,
		Scopes:      scopes,
		OAuthScopes: oauthScopes,
	}, nil
}

func (f *fakeIdentityOpenPlatform) AuthorizeAppByClientID(_ context.Context, clientID string) (*openplatform.App, error) {
	if clientID == f.app.ClientID && f.app.Status == openplatform.AppStatusApproved {
		return f.app, nil
	}
	if clientID == f.otherApp.ClientID && f.otherApp.Status == openplatform.AppStatusApproved {
		return f.otherApp, nil
	}
	return nil, openplatform.ErrAppNotFound
}

func (f *fakeIdentityOpenPlatform) LoadConsentChallenge(_ context.Context, token string) (*openplatform.ConsentChallenge, error) {
	if f.consentChallenge != nil && f.consentChallenge.Token == token {
		challenge := *f.consentChallenge
		return &challenge, nil
	}
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

func (f *fakeIdentityOpenPlatform) UserInfoForIdentityToken(_ context.Context, clientID string, userID int64, subject string, scopes []string) (map[string]any, error) {
	if clientID != f.app.ClientID || userID != 42 || subject != identitySubject(userID) {
		return nil, openplatform.ErrDisclosureUnavailable
	}
	f.userInfoCallCount++
	f.userInfoScopes = append([]string(nil), scopes...)
	payload := make(map[string]any, len(f.profile))
	for key, value := range f.profile {
		payload[key] = value
	}
	return payload, nil
}

func (f *fakeIdentityOpenPlatform) IdentityAccessTokenActive(_ context.Context, clientID string, userID int64, scopes []string) (bool, error) {
	_, active, err := f.IdentityAuthorizationFingerprint(context.Background(), clientID, userID, scopes)
	return active, err
}

func (f *fakeIdentityOpenPlatform) IdentityAuthorizationFingerprint(_ context.Context, clientID string, userID int64, scopes []string) (string, bool, error) {
	f.identityAccessTokenClientID = clientID
	f.identityAccessTokenUserID = userID
	f.identityAccessTokenScopes = append([]string(nil), scopes...)
	if f.identityAccessTokenHook != nil {
		f.identityAccessTokenHook()
	}
	if f.identityAccessTokenErr != nil {
		return "", false, f.identityAccessTokenErr
	}
	if !f.identityAccessTokenActive {
		return "", false, nil
	}
	normalized, err := openplatform.NormalizeAuthorizationScopes(scopes)
	if err != nil {
		return "", false, err
	}
	if len(openplatform.UserConsentScopes(normalized)) == 0 {
		return "", true, nil
	}
	return f.identityAuthorizationHash, true, nil
}

func (f *fakeIdentityOpenPlatform) IdentityClientCredentialsTokenActive(_ context.Context, clientID string, scopes []string) (bool, error) {
	f.clientCredentialsClientID = clientID
	f.clientCredentialsScopes = append([]string(nil), scopes...)
	if f.clientCredentialsTokenErr != nil {
		return false, f.clientCredentialsTokenErr
	}
	return f.clientCredentialsTokenActive, nil
}

func (f *fakeIdentityOpenPlatform) DeleteConsentChallenge(_ context.Context, token string) error {
	f.deletedChallenge = token
	return nil
}

func (f *fakeIdentityOpenPlatform) VerifyClientSecret(_ context.Context, clientID, clientSecret string) (*openplatform.App, error) {
	if clientID == f.app.ClientID && clientSecret == f.clientSecret {
		return f.app, nil
	}
	if clientID == f.otherApp.ClientID && clientSecret == f.otherSecret {
		return f.otherApp, nil
	}
	return nil, openplatform.ErrAppNotFound
}
