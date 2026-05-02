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
	authURL, verifier := client.GetAuthURL("state-123")
	assert.Contains(t, authURL, "https://sso.example.com/authorize")
	assert.Contains(t, authURL, "state-123")
	assert.NotEmpty(t, verifier)

	stepUpURL, stepUpVerifier := client.GetStepUpAuthURL("step-up-state")
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
