package singleflightx

import (
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/singleflight"
)

func TestDoValue(t *testing.T) {
	var group singleflight.Group
	var calls atomic.Int32

	value, err := DoValue(&group, "key", func() (string, error) {
		calls.Add(1)
		return "ok", nil
	})
	require.NoError(t, err)
	assert.Equal(t, "ok", value)
	assert.Equal(t, int32(1), calls.Load())
}

func TestDo(t *testing.T) {
	var group singleflight.Group
	var calls atomic.Int32

	err := Do(&group, "key", func() error {
		calls.Add(1)
		return nil
	})
	require.NoError(t, err)
	assert.Equal(t, int32(1), calls.Load())
}
