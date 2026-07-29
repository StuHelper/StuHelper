package redis

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"

	"github.com/StuHelper/StuHelper/server/internal/pkg/config"
)

// TestClientACLIntegration exercises the production TLS/ACL contract against a
// real Redis instance. It is opt-in because unit-test environments use
// miniredis and do not have the generated CA or ACL users.
func TestClientACLIntegration(t *testing.T) {
	if os.Getenv("STUHELPER_REDIS_INTEGRATION") != "1" {
		t.Skip("set STUHELPER_REDIS_INTEGRATION=1 to run the real Redis TLS/ACL test")
	}

	host := requiredIntegrationEnv(t, "STUHELPER_REDIS_INTEGRATION_HOST")
	port := requiredIntegrationEnv(t, "STUHELPER_REDIS_INTEGRATION_PORT")
	username := requiredIntegrationEnv(t, "STUHELPER_REDIS_INTEGRATION_USERNAME")
	password := requiredIntegrationEnv(t, "STUHELPER_REDIS_INTEGRATION_PASSWORD")
	caFile := requiredIntegrationEnv(t, "STUHELPER_REDIS_INTEGRATION_CA_FILE")

	client, err := NewClient(config.RedisConfig{
		Host:         host,
		Port:         port,
		Username:     username,
		Password:     password,
		DB:           0,
		PoolSize:     4,
		MinIdleConns: 1,
		TLSEnabled:   true,
		TLSCAFile:    caFile,
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, client.Close())
	})

	rdb := client.GetClient()
	require.NotNil(t, rdb)
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()

	keyPrefix := fmt.Sprintf("integration:redis-acl:%d", time.Now().UnixNano())
	valueKey := keyPrefix + ":value"
	setKey := keyPrefix + ":set"
	sortedSetKey := keyPrefix + ":zset"
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cleanupCancel()
		_ = rdb.Del(cleanupCtx, valueKey, setKey, sortedSetKey).Err()
	})

	require.NoError(t, rdb.Set(ctx, valueKey, "1", time.Minute).Err())
	value, err := rdb.Get(ctx, valueKey).Result()
	require.NoError(t, err)
	require.Equal(t, "1", value)
	require.EqualValues(t, 2, rdb.Incr(ctx, valueKey).Val())
	require.NoError(t, rdb.Expire(ctx, valueKey, time.Minute).Err())
	require.EqualValues(t, 1, rdb.Exists(ctx, valueKey).Val())

	require.EqualValues(t, 2, rdb.SAdd(ctx, setKey, "one", "two").Val())
	members, err := rdb.SMembers(ctx, setKey).Result()
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"one", "two"}, members)
	require.EqualValues(t, 1, rdb.SRem(ctx, setKey, "one").Val())

	require.EqualValues(t, 2, rdb.ZAdd(ctx, sortedSetKey,
		goredis.Z{Score: 1, Member: "one"},
		goredis.Z{Score: 2, Member: "two"},
	).Val())
	require.EqualValues(t, 2, rdb.ZCard(ctx, sortedSetKey).Val())
	require.EqualValues(t, 1, rdb.ZRemRangeByScore(ctx, sortedSetKey, "0", "1").Val())

	script := goredis.NewScript(`
local value = redis.call("GET", KEYS[1])
if value then
  redis.call("PEXPIRE", KEYS[1], ARGV[1])
end
return value
`)
	for range 2 {
		result, runErr := script.Run(ctx, rdb, []string{valueKey}, 60000).Text()
		require.NoError(t, runErr)
		require.Equal(t, "2", result)
	}

	pubsub := rdb.PSubscribe(ctx, "notify:*")
	t.Cleanup(func() {
		require.NoError(t, pubsub.Close())
	})
	_, err = pubsub.Receive(ctx)
	require.NoError(t, err)
	require.GreaterOrEqual(t, rdb.Publish(ctx, "notify:1", "acl-integration").Val(), int64(1))
	message, err := pubsub.ReceiveMessage(ctx)
	require.NoError(t, err)
	require.Equal(t, "notify:1", message.Channel)
	require.Equal(t, "acl-integration", message.Payload)

	_, err = rdb.ConfigGet(ctx, "maxmemory").Result()
	require.Error(t, err, "application ACL must deny administrative CONFIG access")

	deleted, err := rdb.GetDel(ctx, valueKey).Result()
	require.NoError(t, err)
	require.Equal(t, "2", deleted)
}

func requiredIntegrationEnv(t *testing.T, key string) string {
	t.Helper()
	value := os.Getenv(key)
	require.NotEmpty(t, value, "%s is required", key)
	return value
}
