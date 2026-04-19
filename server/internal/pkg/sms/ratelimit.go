package sms

import (
	"sync"
	"time"
)

// rateLimiter 提供进程内限流，包含单键和全局两个维度。
// 该实现面向单实例内部服务，不依赖外部组件。
type rateLimiter struct {
	mu sync.Mutex

	// 按手机号记录上次允许请求的时间。
	perKey    map[string]time.Time
	keyWindow time.Duration // 同一手机号两次请求的最小间隔

	// 全局滑动窗口使用环形缓冲区记录近期请求时间。
	globalBuf  []time.Time // 长度为 globalMax 的环形缓冲区
	globalHead int         // 下一次写入位置
	globalLen  int         // 当前已记录条数
	globalMax  int         // globalWindow 内允许的最大请求数
	globalWin  time.Duration
}

// newRateLimiter 创建限流器。
//   - keyWindow: 同一手机号的最小请求间隔，例如 60s
//   - globalMax: globalWindow 内允许的总请求上限
//   - globalWindow: 全局滑动窗口时长，例如 60s
func newRateLimiter(keyWindow time.Duration, globalMax int, globalWindow time.Duration) *rateLimiter {
	return &rateLimiter{
		perKey:    make(map[string]time.Time),
		keyWindow: keyWindow,
		globalBuf: make([]time.Time, globalMax),
		globalMax: globalMax,
		globalWin: globalWindow,
	}
}

// allow 判断给定 key 的请求是否允许通过。
// 成功时返回 ("", true)，被限流时返回 (reason, false)。
func (rl *rateLimiter) allow(key string, now time.Time) (reason string, ok bool) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// 1. 单键检查：同一 key 在 keyWindow 内最多允许一次请求。
	if last, exists := rl.perKey[key]; exists {
		if elapsed := now.Sub(last); elapsed < rl.keyWindow {
			remaining := rl.keyWindow - elapsed
			return "rate limited: retry after " + remaining.Truncate(time.Second).String(), false
		}
	}

	// 2. 全局检查：统计滑动窗口内的请求数。
	globalCount := rl.countGlobal(now)
	if globalCount >= rl.globalMax {
		return "rate limited: too many requests, retry later", false
	}

	// 两个维度都通过后记录本次请求。
	rl.perKey[key] = now
	rl.recordGlobal(now)

	// 惰性清理过期 key，避免 perKey 无界增长。
	// 这里按 globalMax 次请求触发一次清理，成本较低。
	if (rl.globalLen % rl.globalMax) == 0 {
		rl.pruneKeys(now)
	}

	return "", true
}

// countGlobal 统计环形缓冲区中仍落在窗口内的请求数。
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

// recordGlobal 把时间戳写入全局环形缓冲区。
func (rl *rateLimiter) recordGlobal(now time.Time) {
	rl.globalBuf[rl.globalHead] = now
	rl.globalHead = (rl.globalHead + 1) % rl.globalMax
	if rl.globalLen < rl.globalMax {
		rl.globalLen++
	}
}

// pruneKeys 删除早于 keyWindow 的单键记录。
func (rl *rateLimiter) pruneKeys(now time.Time) {
	cutoff := now.Add(-rl.keyWindow)
	for k, ts := range rl.perKey {
		if ts.Before(cutoff) {
			delete(rl.perKey, k)
		}
	}
}
