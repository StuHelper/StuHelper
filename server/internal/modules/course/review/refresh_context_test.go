package review

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetachedRefreshContextIgnoresParentCancellation(t *testing.T) {
	parent, cancelParent := context.WithCancel(context.Background())
	cancelParent()

	ctx, cancel := detachedRefreshContext(parent, time.Second)
	defer cancel()

	require.NoError(t, ctx.Err())
	deadline, ok := ctx.Deadline()
	require.True(t, ok)
	assert.WithinDuration(t, time.Now().Add(time.Second), deadline, 100*time.Millisecond)
}

func TestCacheRefreshContextUsesServiceLifecycleContext(t *testing.T) {
	lifecycleCtx, cancelLifecycle := context.WithCancel(context.Background())
	cancelLifecycle()

	service := &Service{asyncCtx: lifecycleCtx}
	ctx, cancel := service.cacheRefreshContext(context.Background(), time.Second)
	defer cancel()

	require.ErrorIs(t, ctx.Err(), context.Canceled)
}
