package middleware

import (
	"errors"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type failingEntropyReader struct{}

func (failingEntropyReader) Read(_ []byte) (int, error) {
	return 0, errors.New("entropy unavailable")
}

func TestRedisRateLimiter_AllowFailsClosedWhenEntropyUnavailable(t *testing.T) {
	limiter := NewRedisRateLimiter(nil, 1, time.Minute)

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

var _ io.Reader = failingEntropyReader{}
