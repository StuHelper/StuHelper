package openplatform

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAppendOAuthErrorPreservesOpaqueState(t *testing.T) {
	state := " state with surrounding spaces "

	redirectURL := appendOAuthError("https://client.example.com/callback?existing=1", "access_denied", state)

	callback, err := url.Parse(redirectURL)
	require.NoError(t, err)
	assert.Equal(t, "access_denied", callback.Query().Get("error"))
	assert.Equal(t, state, callback.Query().Get("state"))
	assert.Equal(t, "1", callback.Query().Get("existing"))
}
