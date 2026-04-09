package sms

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRateLimiter_PerKeyAllowsFirstRequest(t *testing.T) {
	rl := newRateLimiter(60*time.Second, 100, 60*time.Second)
	now := time.Now()

	reason, ok := rl.allow("+8613800138000", now)
	assert.True(t, ok)
	assert.Empty(t, reason)
}

func TestRateLimiter_PerKeyBlocksSecondRequestWithinWindow(t *testing.T) {
	rl := newRateLimiter(60*time.Second, 100, 60*time.Second)
	now := time.Now()

	_, ok := rl.allow("+8613800138000", now)
	assert.True(t, ok)

	reason, ok := rl.allow("+8613800138000", now.Add(30*time.Second))
	assert.False(t, ok)
	assert.Contains(t, reason, "rate limited")
	assert.Contains(t, reason, "retry after")
}

func TestRateLimiter_PerKeyAllowsAfterWindowExpires(t *testing.T) {
	rl := newRateLimiter(60*time.Second, 100, 60*time.Second)
	now := time.Now()

	_, ok := rl.allow("+8613800138000", now)
	assert.True(t, ok)

	reason, ok := rl.allow("+8613800138000", now.Add(61*time.Second))
	assert.True(t, ok)
	assert.Empty(t, reason)
}

func TestRateLimiter_DifferentKeysAreIndependent(t *testing.T) {
	rl := newRateLimiter(60*time.Second, 100, 60*time.Second)
	now := time.Now()

	_, ok := rl.allow("+8613800138000", now)
	assert.True(t, ok)

	reason, ok := rl.allow("+8613900139000", now)
	assert.True(t, ok)
	assert.Empty(t, reason)
}

func TestRateLimiter_GlobalLimitBlocksWhenExceeded(t *testing.T) {
	globalMax := 5
	rl := newRateLimiter(1*time.Second, globalMax, 60*time.Second)
	now := time.Now()

	// Fill up the global limit with distinct phone numbers.
	for i := 0; i < globalMax; i++ {
		phone := "+861380013" + string(rune('0'+i)) + "000"
		_, ok := rl.allow(phone, now.Add(time.Duration(i)*2*time.Second))
		assert.True(t, ok, "request %d should be allowed", i)
	}

	// Next request should be blocked by global limit.
	reason, ok := rl.allow("+8619900199000", now.Add(time.Duration(globalMax)*2*time.Second))
	assert.False(t, ok)
	assert.Contains(t, reason, "too many requests")
}

func TestRateLimiter_GlobalLimitRecoverAfterWindowExpires(t *testing.T) {
	globalMax := 3
	rl := newRateLimiter(1*time.Second, globalMax, 10*time.Second)
	now := time.Now()

	// Fill global limit.
	for i := 0; i < globalMax; i++ {
		phone := "+861380013" + string(rune('0'+i)) + "000"
		_, ok := rl.allow(phone, now.Add(time.Duration(i)*2*time.Second))
		assert.True(t, ok)
	}

	// Blocked immediately after.
	_, ok := rl.allow("+8619900199000", now.Add(time.Duration(globalMax)*2*time.Second))
	assert.False(t, ok)

	// Allowed after the window expires (all old entries fall out).
	reason, ok := rl.allow("+8619900199000", now.Add(15*time.Second))
	assert.True(t, ok)
	assert.Empty(t, reason)
}

func TestRateLimiter_PruneRemovesStaleKeys(t *testing.T) {
	rl := newRateLimiter(60*time.Second, 100, 60*time.Second)
	now := time.Now()

	rl.mu.Lock()
	rl.perKey["+8613800138000"] = now.Add(-120 * time.Second) // stale
	rl.perKey["+8613900139000"] = now                         // fresh
	rl.pruneKeys(now)
	rl.mu.Unlock()

	assert.NotContains(t, rl.perKey, "+8613800138000")
	assert.Contains(t, rl.perKey, "+8613900139000")
}
