package sms

import (
	"sync"
	"time"
)

// rateLimiter provides in-memory rate limiting with per-key and global dimensions.
// Designed for a single-instance internal service — no external dependencies.
type rateLimiter struct {
	mu sync.Mutex

	// Per-key (phone number) tracking: stores the last allowed timestamp.
	perKey    map[string]time.Time
	keyWindow time.Duration // minimum interval between requests for the same key

	// Global sliding window: circular buffer of recent request timestamps.
	globalBuf  []time.Time // ring buffer of size globalMax
	globalHead int         // next write position
	globalLen  int         // current number of entries
	globalMax  int         // max requests within globalWindow
	globalWin  time.Duration
}

// newRateLimiter creates a rate limiter.
//   - keyWindow:  minimum interval between requests for the same key (e.g. 60s)
//   - globalMax:  maximum total requests within globalWindow
//   - globalWindow: sliding window duration for global limit (e.g. 60s)
func newRateLimiter(keyWindow time.Duration, globalMax int, globalWindow time.Duration) *rateLimiter {
	return &rateLimiter{
		perKey:    make(map[string]time.Time),
		keyWindow: keyWindow,
		globalBuf: make([]time.Time, globalMax),
		globalMax: globalMax,
		globalWin: globalWindow,
	}
}

// allow checks whether a request for the given key is permitted.
// Returns ("", true) on success, or (reason, false) when rate limited.
func (rl *rateLimiter) allow(key string, now time.Time) (reason string, ok bool) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// 1. Per-key check: at most one request per keyWindow.
	if last, exists := rl.perKey[key]; exists {
		if elapsed := now.Sub(last); elapsed < rl.keyWindow {
			remaining := rl.keyWindow - elapsed
			return "rate limited: retry after " + remaining.Truncate(time.Second).String(), false
		}
	}

	// 2. Global check: count requests within the sliding window.
	globalCount := rl.countGlobal(now)
	if globalCount >= rl.globalMax {
		return "rate limited: too many requests, retry later", false
	}

	// Both checks passed — record the request.
	rl.perKey[key] = now
	rl.recordGlobal(now)

	// Lazy cleanup: periodically prune stale per-key entries to prevent unbounded growth.
	// Run every globalMax requests (cheap heuristic).
	if (rl.globalLen % rl.globalMax) == 0 {
		rl.pruneKeys(now)
	}

	return "", true
}

// countGlobal returns how many entries in the ring buffer fall within the window.
func (rl *rateLimiter) countGlobal(now time.Time) int {
	cutoff := now.Add(-rl.globalWin)
	count := 0
	for i := 0; i < rl.globalLen && i < rl.globalMax; i++ {
		idx := (rl.globalHead - 1 - i + rl.globalMax) % rl.globalMax
		if !rl.globalBuf[idx].Before(cutoff) {
			count++
		}
	}
	return count
}

// recordGlobal appends a timestamp to the ring buffer.
func (rl *rateLimiter) recordGlobal(now time.Time) {
	rl.globalBuf[rl.globalHead] = now
	rl.globalHead = (rl.globalHead + 1) % rl.globalMax
	if rl.globalLen < rl.globalMax {
		rl.globalLen++
	}
}

// pruneKeys removes per-key entries older than keyWindow.
func (rl *rateLimiter) pruneKeys(now time.Time) {
	cutoff := now.Add(-rl.keyWindow)
	for k, ts := range rl.perKey {
		if ts.Before(cutoff) {
			delete(rl.perKey, k)
		}
	}
}
