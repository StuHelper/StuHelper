package oidc

import (
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

func TestVerifyIDTokenAudienceForApplication(t *testing.T) {
	client := &Client{
		oauth2Configs: map[string]oauth2.Config{
			ApplicationWeb:   {ClientID: "web-client"},
			ApplicationAdmin: {ClientID: "admin-client"},
		},
	}

	require.NoError(t, client.verifyIDTokenAudience([]string{"web-client"}, "web-client"))
	require.ErrorIs(t, client.verifyIDTokenAudience([]string{"web-client"}, "admin-client"), ErrInvalidAudience)
	require.NoError(t, client.verifyIDTokenAudience([]string{"admin-client"}, ""))
	require.ErrorIs(t, client.verifyIDTokenAudience([]string{"unknown-client"}, ""), ErrInvalidAudience)
}
