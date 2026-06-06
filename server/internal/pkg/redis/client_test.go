package redis

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClientGetClientIsNilSafe(t *testing.T) {
	var client *Client

	assert.Nil(t, client.GetClient())
	assert.Nil(t, (&Client{}).GetClient())
}

func TestClientCloseIsNilAndZeroValueSafe(t *testing.T) {
	var nilClient *Client
	require.NoError(t, nilClient.Close())

	zeroClient := &Client{}
	require.NoError(t, zeroClient.Close())
	require.NoError(t, zeroClient.Close())
}
