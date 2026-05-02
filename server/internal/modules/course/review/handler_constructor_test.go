package review

import (
	"testing"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/cache"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/config"
)

func TestNewHandler_PanicsOnMissingAuthorizationProvider(t *testing.T) {
	rdb := redis.NewClient(&redis.Options{Addr: "127.0.0.1:0"})
	t.Cleanup(func() { assert.NoError(t, rdb.Close()) })

	assert.Panics(t, func() {
		NewHandler(cache.NewHelper(rdb), &Service{}, rdb, config.ReviewRateLimitConfig{}, nil)
	})
}
