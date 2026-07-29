package app

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRuntimeShutdownAndCleanupAreOrderedAndIdempotent(t *testing.T) {
	bgCtx, bgCancel := context.WithCancel(context.Background())
	rt := &Runtime{bgCancel: bgCancel}

	shutdownCalls := 0
	cleanupCalls := 0
	rt.addShutdownHook(func() {
		shutdownCalls++
	})
	rt.addCleanup(func() {
		cleanupCalls++
	})

	rt.beginShutdown()
	rt.beginShutdown()

	assert.ErrorIs(t, bgCtx.Err(), context.Canceled)
	assert.Equal(t, 1, shutdownCalls)
	assert.Zero(t, cleanupCalls)

	rt.runCleanups()
	rt.runCleanups()

	assert.Equal(t, 1, shutdownCalls)
	assert.Equal(t, 1, cleanupCalls)
	assert.Nil(t, rt.cleanups)
	assert.Nil(t, rt.shutdownHooks)
}
