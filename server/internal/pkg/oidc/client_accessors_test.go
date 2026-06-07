package oidc

import (
	"net/url"
	"testing"

	gooidc "github.com/coreos/go-oidc/v3/oidc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

func TestClaimsAccessorsAndStubClientHelpers(t *testing.T) {
	claims := &Claims{
		Sub:               "user-1",
		Name:              "Display Name",
		PreferredUsername: "preferred-user",
		Email:             "user@example.com",
		Picture:           "https://cdn.example.com/avatar.png",
	}

	assert.Equal(t, "user-1", claims.GetUserID())
	assert.Equal(t, "preferred-user", claims.GetUsername())
	assert.Equal(t, "Display Name", claims.GetDisplayName())
	assert.Equal(t, "user@example.com", claims.GetEmail())
	require.NotNil(t, claims.GetAvatar())
	assert.Equal(t, "https://cdn.example.com/avatar.png", *claims.GetAvatar())

	claims.PreferredUsername = ""
	assert.Equal(t, "Display Name", claims.GetUsername())
	claims.Picture = ""
	assert.Nil(t, claims.GetAvatar())

	client := NewStubClient("https://sso.example.com/authorize")
	authURL, verifier, err := client.GetAuthURLForApplication(ApplicationWeb, "state-123")
	require.NoError(t, err)
	assert.Contains(t, authURL, "https://sso.example.com/authorize")
	assert.Contains(t, authURL, "state-123")
	assert.NotEmpty(t, verifier)

	reauthURL, reauthVerifier, err := client.GetReauthURLForApplication(ApplicationWeb, "reauth-state")
	require.NoError(t, err)
	assert.Contains(t, reauthURL, "prompt=login")
	assert.Contains(t, reauthURL, "max_age=0")
	assert.NotContains(t, reauthURL, "acr_values=mfa")
	assert.Contains(t, reauthURL, "reauth-state")
	assert.NotEmpty(t, reauthVerifier)

	stepUpURL, stepUpVerifier, err := client.GetStepUpAuthURLForApplication(ApplicationWeb, "step-up-state")
	require.NoError(t, err)
	assert.Contains(t, stepUpURL, "prompt=login")
	assert.Contains(t, stepUpURL, "max_age=0")
	assert.Contains(t, stepUpURL, "acr_values=mfa")
	assert.Contains(t, stepUpURL, "step-up-state")
	assert.NotEmpty(t, stepUpVerifier)

	tokenWithID := (&oauth2.Token{}).WithExtra(map[string]any{"id_token": "id-token-value"})
	tokenWithSpacedID := (&oauth2.Token{}).WithExtra(map[string]any{"id_token": " \tid-token-value\n "})
	tokenWithBlankID := (&oauth2.Token{}).WithExtra(map[string]any{"id_token": " \t\n "})
	tokenWithNonStringID := (&oauth2.Token{}).WithExtra(map[string]any{"id_token": 123})
	assert.Equal(t, "id-token-value", ExtractIDToken(tokenWithID))
	assert.Equal(t, "id-token-value", ExtractIDToken(tokenWithSpacedID))
	assert.Empty(t, ExtractIDToken(tokenWithBlankID))
	assert.Empty(t, ExtractIDToken(tokenWithNonStringID))
	assert.Empty(t, ExtractIDToken(&oauth2.Token{}))
}

func TestGetSignupURLForApplicationUsesCasdoorSignupAuthorizePath(t *testing.T) {
	client := NewStubClient("https://sso.example.com/login/oauth/authorize")

	signupURL, verifier, err := client.GetSignupURLForApplication(ApplicationWeb, "signup-state")
	require.NoError(t, err)
	assert.NotEmpty(t, verifier)
	assert.Contains(t, signupURL, "https://sso.example.com/signup/oauth/authorize")
	assert.Contains(t, signupURL, "state=signup-state")
	assert.NotContains(t, signupURL, "/login/oauth/authorize")
}

func TestGetAuthURLForApplicationWithRedirectURIUsesOverride(t *testing.T) {
	client := NewStubClient("https://sso.example.com/login/oauth/authorize")

	authURL, verifier, err := client.GetAuthURLForApplicationWithRedirectURI(
		ApplicationWeb,
		"state-join",
		" http://join.localhost:3000/api/v1/auth/callback ",
	)
	require.NoError(t, err)
	assert.NotEmpty(t, verifier)

	parsed, err := url.Parse(authURL)
	require.NoError(t, err)
	assert.Equal(t, "http://join.localhost:3000/api/v1/auth/callback", parsed.Query().Get("redirect_uri"))
	assert.Equal(t, "state-join", parsed.Query().Get("state"))
}

func TestGetAuthURLForApplicationRewritesBrowserAuthBaseURL(t *testing.T) {
	client := NewStubClient("https://casdoor.internal/oauth2/authorize")
	client.publicAuthBaseURL = "https://sso.example.com"

	authURL, verifier, err := client.GetAuthURLForApplication(ApplicationWeb, "state-123")
	require.NoError(t, err)
	assert.NotEmpty(t, verifier)
	assert.Contains(t, authURL, "https://sso.example.com/oauth2/authorize")
	assert.Contains(t, authURL, "state-123")
	assert.NotContains(t, authURL, "https://casdoor.internal")

	stepUpURL, _, err := client.GetStepUpAuthURLForApplication(ApplicationWeb, "step-up-state")
	require.NoError(t, err)
	assert.Contains(t, stepUpURL, "https://sso.example.com/oauth2/authorize")
	assert.Contains(t, stepUpURL, "prompt=login")
}

func TestGetAuthURLForApplicationDoesNotRewriteCasdoorLoginOAuthToDifferentHost(t *testing.T) {
	client := NewStubClient("https://casdoor.internal/login/oauth/authorize")
	client.publicAuthBaseURL = "https://sso.example.com"

	authURL, verifier, err := client.GetAuthURLForApplication(ApplicationWeb, "state-123")
	require.NoError(t, err)
	assert.NotEmpty(t, verifier)
	assert.Contains(t, authURL, "https://casdoor.internal/login/oauth/authorize")
	assert.NotContains(t, authURL, "https://sso.example.com/login/oauth/authorize")

	stepUpURL, _, err := client.GetStepUpAuthURLForApplication(ApplicationWeb, "step-up-state")
	require.NoError(t, err)
	assert.Contains(t, stepUpURL, "https://casdoor.internal/login/oauth/authorize")
	assert.NotContains(t, stepUpURL, "https://sso.example.com/login/oauth/authorize")
	assert.Contains(t, stepUpURL, "prompt=login")
}

func TestApplicationLookupNormalizesInputs(t *testing.T) {
	client := NewStubClient("https://sso.example.com/login/oauth/authorize")

	authURL, verifier, err := client.GetAuthURLForApplication(" \t"+ApplicationAdmin+"\n ", "state-admin")
	require.NoError(t, err)
	assert.NotEmpty(t, verifier)
	assert.Contains(t, authURL, "client_id=test-admin-client-id")

	authURL, verifier, err = client.GetAuthURLForApplication(" \t\n ", "state-web")
	require.NoError(t, err)
	assert.NotEmpty(t, verifier)
	assert.Contains(t, authURL, "client_id=test-client-id")

	assert.Equal(t, ApplicationAdmin, client.ApplicationKeyForClientID(" \ttest-admin-client-id\n "))
	assert.Empty(t, client.ApplicationKeyForClientID(" \t\n "))
}

func TestOAuth2ConfigFromInputNormalizesConfiguredValues(t *testing.T) {
	endpoint := oauth2.Endpoint{AuthURL: "https://sso.example.com/authorize", TokenURL: "https://sso.example.com/token"}

	cfg, ok, err := oauth2ConfigFromInput(endpoint, oauth2ApplicationInput{
		appKey:       ApplicationAdmin,
		clientID:     " \tadmin-client\n ",
		clientSecret: " \tadmin-secret\n ",
		redirectURI:  " \thttps://admin.example.com/callback\n ",
		scopes:       []string{" openid ", "profile"},
	})
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, "admin-client", cfg.ClientID)
	assert.Equal(t, "admin-secret", cfg.ClientSecret)
	assert.Equal(t, "https://admin.example.com/callback", cfg.RedirectURL)
	assert.Equal(t, endpoint, cfg.Endpoint)
	assert.Equal(t, []string{"openid", "profile"}, cfg.Scopes)

	cfg, ok, err = oauth2ConfigFromInput(endpoint, oauth2ApplicationInput{
		appKey:       ApplicationAdmin,
		clientID:     " \t\n ",
		clientSecret: " \t\n ",
		redirectURI:  " \t\n ",
	})
	require.NoError(t, err)
	assert.False(t, ok)
	assert.Empty(t, cfg.ClientID)

	_, ok, err = oauth2ConfigFromInput(endpoint, oauth2ApplicationInput{
		appKey:       ApplicationWeb,
		clientID:     " \t\n ",
		clientSecret: "web-secret",
		redirectURI:  "https://web.example.com/callback",
		required:     true,
	})
	assert.False(t, ok)
	require.ErrorIs(t, err, ErrApplicationNotConfigured)
}

func TestOAuth2ScopesNormalizeConfiguredValues(t *testing.T) {
	assert.Equal(t,
		[]string{"openid", "profile", "email"},
		oauth2Scopes([]string{" openid ", "", "profile", "openid", " email "}),
	)
	assert.Equal(t,
		[]string{gooidc.ScopeOpenID, "profile", "email", "offline_access"},
		oauth2Scopes([]string{" ", ""}),
	)
}
