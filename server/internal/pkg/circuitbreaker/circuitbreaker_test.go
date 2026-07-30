package circuitbreaker

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCircuitBreakerTransitionsAndSerializesHalfOpenProbes(t *testing.T) {
	cb := NewNamed("transition_test", Config{
		FailureThreshold: 2,
		SuccessThreshold: 2,
		Timeout:          10 * time.Millisecond,
	})

	require.True(t, cb.Allow())
	cb.RecordFailure()
	assert.Equal(t, StateClosed, cb.State())

	require.True(t, cb.Allow())
	cb.RecordFailure()
	assert.Equal(t, StateOpen, cb.State())
	assert.False(t, cb.Allow())

	time.Sleep(15 * time.Millisecond)
	assert.Equal(t, StateHalfOpen, cb.State())
	require.True(t, cb.Allow())
	assert.False(t, cb.Allow(), "only one recovery probe may be in flight")
	cb.RecordSuccess()

	assert.Equal(t, StateHalfOpen, cb.State())
	require.True(t, cb.Allow())
	cb.RecordSuccess()
	assert.Equal(t, StateClosed, cb.State())
	require.True(t, cb.Allow())
}

func TestCircuitBreakerNormalizesInvalidConfig(t *testing.T) {
	cb := NewNamed("", Config{})

	for range DefaultConfig().FailureThreshold - 1 {
		require.True(t, cb.Allow())
		cb.RecordFailure()
	}

	assert.Equal(t, StateClosed, cb.State())
	require.True(t, cb.Allow())
	cb.RecordFailure()
	assert.Equal(t, StateOpen, cb.State())
}

func TestCircuitBreakerAllowsOneConcurrentHalfOpenProbe(t *testing.T) {
	cb := NewNamed("concurrent_probe_test", Config{
		FailureThreshold: 1,
		SuccessThreshold: 1,
		Timeout:          5 * time.Millisecond,
	})
	require.True(t, cb.Allow())
	cb.RecordFailure()
	time.Sleep(10 * time.Millisecond)

	var allowed atomic.Int32
	start := make(chan struct{})
	var wg sync.WaitGroup
	for range 32 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			if cb.Allow() {
				allowed.Add(1)
			}
		}()
	}
	close(start)
	wg.Wait()

	assert.Equal(t, int32(1), allowed.Load())
}

func TestCircuitBreakerNeutralOutcomePreservesHealthAndReleasesProbe(t *testing.T) {
	cb := NewNamed("neutral_outcome_test", Config{
		FailureThreshold: 2,
		SuccessThreshold: 1,
		Timeout:          time.Nanosecond,
	})

	require.True(t, cb.Allow())
	cb.RecordFailure()
	assert.Equal(t, StateClosed, cb.State())
	cb.RecordNeutral()
	assert.Equal(t, 1, cb.Metrics()["failures"])

	require.True(t, cb.Allow())
	cb.RecordFailure()
	assert.Equal(t, StateHalfOpen, cb.State())
	require.True(t, cb.Allow())
	assert.False(t, cb.Allow(), "half-open probe must be reserved")

	cb.RecordNeutral()
	metrics := cb.Metrics()
	assert.Equal(t, StateHalfOpen.String(), metrics["state"])
	assert.Equal(t, 0, metrics["failures"])
	assert.Equal(t, false, metrics["probe_in_flight"])
	assert.True(t, cb.Allow(), "neutral outcome must release the half-open probe")
}
