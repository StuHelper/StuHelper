package middleware

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/StuHelper/StuHelper/server/internal/testutil/redisfixture"
)

type failingEntropyReader struct{}

func (failingEntropyReader) Read(_ []byte) (int, error) {
	return 0, errors.New("entropy unavailable")
}

func TestRedisRateLimiter_AllowFailsClosedWhenEntropyUnavailable(t *testing.T) {
	fixture := redisfixture.Start(t)
	limiter := NewRedisRateLimiter(fixture.Client, 1, time.Minute)

	originalReader := rateLimitEntropyReader
	rateLimitEntropyReader = failingEntropyReader{}
	t.Cleanup(func() {
		rateLimitEntropyReader = originalReader
	})

	allowed, err := limiter.Allow(t.Context(), "rl:test:entropy")
	require.Error(t, err)
	assert.False(t, allowed)
	assert.Contains(t, err.Error(), "generate unique id")
}

func TestRedisRateLimiter_AllowFailsClosedWithoutRedisClient(t *testing.T) {
	limiter := NewRedisRateLimiter(nil, 1, time.Minute)

	allowed, err := limiter.Allow(t.Context(), "rl:test:nil-redis")

	require.ErrorIs(t, err, errRateLimiterUnavailable)
	assert.False(t, allowed)
}

func TestRedisRateLimiter_AllowFailsClosedOnNilReceiver(t *testing.T) {
	var limiter *RedisRateLimiter

	allowed, err := limiter.Allow(t.Context(), "rl:test:nil-limiter")

	require.ErrorIs(t, err, errRateLimiterUnavailable)
	assert.False(t, allowed)
}

func TestRateLimitMiddleware_FailsClosedWithoutLimiter(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(RateLimitMiddleware(nil))
	router.GET("/", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

var _ io.Reader = failingEntropyReader{}
