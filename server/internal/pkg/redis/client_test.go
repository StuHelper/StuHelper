package redis

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/config"
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

func TestNewClientReturnsErrorWhenPingFails(t *testing.T) {
	client, err := NewClient(config.RedisConfig{
		Host: "127.0.0.1",
		Port: "0",
	})

	require.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "failed to connect to redis")
}
