package oidc

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNewProviderUnavailableKeySetDefaultsNilContext(t *testing.T) {
	var parent context.Context

	keySet := newProviderUnavailableKeySet(parent, "https://sso.example.com/.well-known/jwks", time.Minute)

	require.NotNil(t, keySet.ctx)
	require.NoError(t, keySet.ctx.Err())
}
