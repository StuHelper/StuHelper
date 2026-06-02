package oidc

import (
	"testing"

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

	stepUpURL, stepUpVerifier, err := client.GetStepUpAuthURLForApplication(ApplicationWeb, "step-up-state")
	require.NoError(t, err)
	assert.Contains(t, stepUpURL, "prompt=login")
	assert.Contains(t, stepUpURL, "max_age=0")
	assert.Contains(t, stepUpURL, "acr_values=mfa")
	assert.Contains(t, stepUpURL, "step-up-state")
	assert.NotEmpty(t, stepUpVerifier)

	tokenWithID := (&oauth2.Token{}).WithExtra(map[string]any{"id_token": "id-token-value"})
	assert.Equal(t, "id-token-value", ExtractIDToken(tokenWithID))
	assert.Empty(t, ExtractIDToken((&oauth2.Token{}).WithExtra(map[string]any{"id_token": 123})))
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
