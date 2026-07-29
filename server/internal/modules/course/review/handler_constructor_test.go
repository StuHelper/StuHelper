package review

import (
	"context"
	"testing"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"

	"github.com/StuHelper/StuHelper/server/internal/pkg/cache"
	"github.com/StuHelper/StuHelper/server/internal/pkg/config"
)

func TestNewHandler_PanicsOnMissingAuthorizationProvider(t *testing.T) {
	rdb := redis.NewClient(&redis.Options{Addr: "127.0.0.1:0"})
	t.Cleanup(func() { assert.NoError(t, rdb.Close()) })

	assert.Panics(t, func() {
		NewHandler(HandlerConfig{
			CacheHelper:            cache.NewHelper(rdb),
			Service:                &Service{},
			Redis:                  rdb,
			RateLimit:              config.ReviewRateLimitConfig{},
			Authorizer:             nil,
			InternalUserIDResolver: func(context.Context, string) (int64, error) { return 1, nil },
		})
	})
}
