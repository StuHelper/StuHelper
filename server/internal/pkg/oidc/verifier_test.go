package oidc

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewProviderUnavailableKeySetDefaultsNilContext(t *testing.T) {
	var parent context.Context

	require.NotPanics(t, func() {
		keySet := newProviderUnavailableKeySet(parent, "https://sso.example.com/.well-known/jwks")
		require.NotNil(t, keySet.inner)
	})
}
