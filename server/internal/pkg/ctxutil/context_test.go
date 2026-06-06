package ctxutil

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type testContextKey string

func TestNormalizeDefaultsNilContext(t *testing.T) {
	var input context.Context

	ctx := Normalize(input)

	require.NotNil(t, ctx)
	require.NoError(t, ctx.Err())
}

func TestWithoutCancelIgnoresParentCancellation(t *testing.T) {
	parent, cancel := context.WithCancel(context.Background())
	cancel()

	ctx := WithoutCancel(parent)

	require.NoError(t, ctx.Err())
}

func TestTimeoutDefaultsNilContext(t *testing.T) {
	var input context.Context

	ctx, cancel := Timeout(input, time.Second)
	defer cancel()

	require.NotNil(t, ctx)
	require.NoError(t, ctx.Err())
}

func TestDetachedTimeoutIgnoresParentCancellation(t *testing.T) {
	key := testContextKey("request_id")
	parent := context.WithValue(context.Background(), key, "req-1")
	parent, cancelParent := context.WithCancel(parent)
	cancelParent()

	ctx, cancel := DetachedTimeout(parent, time.Second)
	defer cancel()

	require.NoError(t, ctx.Err())
	require.Equal(t, "req-1", ctx.Value(key))
	_, ok := ctx.Deadline()
	require.True(t, ok)
}
