package circuitbreaker

import (
	"sync"
	"time"
)

// State 熔断器状态
type State int

const (
	StateClosed   State = iota // 关闭状态，正常工作
	StateOpen                  // 打开状态，快速失败
	StateHalfOpen              // 半开状态，尝试恢复
)

func (s State) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// Config 熔断器配置
type Config struct {
	FailureThreshold int           // 触发熔断的失败次数阈值
	SuccessThreshold int           // 半开状态下恢复所需的成功次数
	Timeout          time.Duration // 熔断器打开后的超时时间
}

// DefaultConfig 默认配置
func DefaultConfig() Config {
	return Config{
		FailureThreshold: 5,
		SuccessThreshold: 2,
		Timeout:          30 * time.Second,
	}
}

// CircuitBreaker 熔断器
type CircuitBreaker struct {
	config           Config
	state            State
	failures         int
	successes        int
	lastFailureTime  time.Time
	mu               sync.RWMutex
}

// New 创建熔断器
func New(cfg Config) *CircuitBreaker {
	return &CircuitBreaker{
		config: cfg,
		state:  StateClosed,
	}
}

// State 获取当前状态
func (cb *CircuitBreaker) State() State {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.currentState()
}

// currentState 获取当前状态（内部使用，需要持有锁）
func (cb *CircuitBreaker) currentState() State {
	if cb.state == StateOpen {
		// 检查是否应该转换到半开状态
		if time.Since(cb.lastFailureTime) >= cb.config.Timeout {
			return StateHalfOpen
		}
	}
	return cb.state
}

// Allow 检查是否允许执行操作
func (cb *CircuitBreaker) Allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	state := cb.currentState()
	switch state {
	case StateClosed:
		return true
	case StateOpen:
		return false
	case StateHalfOpen:
		// 半开状态允许尝试
		return true
	default:
		return false
	}
}

// RecordSuccess 记录成功
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	state := cb.currentState()
	switch state {
	case StateClosed:
		cb.failures = 0
	case StateHalfOpen:
		cb.successes++
		if cb.successes >= cb.config.SuccessThreshold {
			cb.state = StateClosed
			cb.failures = 0
			cb.successes = 0
		}
	case StateOpen:
		// 更新状态为半开
		cb.state = StateHalfOpen
		cb.successes = 1
	}
}

// RecordFailure 记录失败
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures++
	cb.lastFailureTime = time.Now()

	state := cb.currentState()
	switch state {
	case StateClosed:
		if cb.failures >= cb.config.FailureThreshold {
			cb.state = StateOpen
		}
	case StateHalfOpen:
		cb.state = StateOpen
		cb.successes = 0
	}
}

// Metrics 获取熔断器指标
func (cb *CircuitBreaker) Metrics() map[string]interface{} {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	return map[string]interface{}{
		"state":            cb.currentState().String(),
		"failures":         cb.failures,
		"successes":        cb.successes,
		"last_failure":     cb.lastFailureTime,
	}
}
