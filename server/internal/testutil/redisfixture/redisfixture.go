package redisfixture

import (
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

type Fixture struct {
	Server *miniredis.Miniredis
	Client *redis.Client
}

func Start(t *testing.T) *Fixture {
	t.Helper()

	server, err := miniredis.Run()
	require.NoError(t, err)

	client := redis.NewClient(&redis.Options{Addr: server.Addr()})

	t.Cleanup(func() {
		require.NoError(t, client.Close())
		server.Close()
	})

	return &Fixture{
		Server: server,
		Client: client,
	}
}
