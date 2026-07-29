package httputil

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/StuHelper/StuHelper/server/internal/pkg/crypto"
)

func TestSanitizeCacheKey(t *testing.T) {
	require.NoError(t, crypto.InitHMACKey("test-httputil-hmac-secret-32-bytes", false))

	assert.Equal(t, emptyCacheKey, SanitizeCacheKey(""))

	first := SanitizeCacheKey("course=query")
	second := SanitizeCacheKey("course=query")

	assert.Len(t, first, cacheKeyHashLen)
	assert.Equal(t, first, second)
	assert.NotEqual(t, "course=query", first)
}

func TestPaginationBounds(t *testing.T) {
	assert.Equal(t, DefaultPageSize, ClampPageSize(0))
	assert.Equal(t, MaxPageSize, ClampPageSize(MaxPageSize+1))
	assert.Equal(t, 0, SafeOffset(-1, -1))
	assert.Equal(t, (MaxPage-1)*MaxPageSize, SafeOffset(MaxPage+1, MaxPageSize+1))
	assert.Equal(t, 0, ClampOffset(-1))
	assert.Equal(t, (MaxPage-1)*MaxPageSize, ClampOffset(MaxPage*MaxPageSize))
}
