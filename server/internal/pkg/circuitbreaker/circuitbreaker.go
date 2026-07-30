package circuitbreaker

import (
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Prometheus 指标
var (
	cbStateGauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "circuit_breaker_state",
			Help: "Current circuit breaker state (0=closed, 1=open, 2=half-open)",
		},
		[]string{"name"},
	)
	cbFailuresGauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "circuit_breaker_failures",
			Help: "Current failure count of circuit breaker",
		},
		[]string{"name"},
	)
	cbSuccessesGauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "circuit_breaker_successes",
			Help: "Current success count of circuit breaker (half-open state)",
		},
		[]string{"name"},
	)
	cbTransitionsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "circuit_breaker_transitions_total",
			Help: "Total number of circuit breaker state transitions",
		},
		[]string{"name", "from", "to"},
	)
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
	name             string
	state            State
	failures         int
	successes        int
	lastFailureTime  time.Time
	halfOpenInFlight bool
	mu               sync.RWMutex
}

// New 创建熔断器
func New(cfg Config) *CircuitBreaker {
	return NewNamed("default", cfg)
}

// NewNamed 创建带名称的熔断器（名称用于 Prometheus 指标标签）
func NewNamed(name string, cfg Config) *CircuitBreaker {
	cfg = normalizeConfig(cfg)
	name = strings.TrimSpace(name)
	if name == "" {
		name = "default"
	}
	cb := &CircuitBreaker{
		config: cfg,
		name:   name,
		state:  StateClosed,
	}
	cbStateGauge.WithLabelValues(name).Set(float64(StateClosed))
	return cb
}

func normalizeConfig(cfg Config) Config {
	defaults := DefaultConfig()
	if cfg.FailureThreshold <= 0 {
		cfg.FailureThreshold = defaults.FailureThreshold
	}
	if cfg.SuccessThreshold <= 0 {
		cfg.SuccessThreshold = defaults.SuccessThreshold
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = defaults.Timeout
	}
	return cfg
}

// State 获取当前状态
func (cb *CircuitBreaker) State() State {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.currentState()
}

// currentState 获取当前逻辑状态（内部使用，需要持有读锁或写锁）
// 注意：此方法仅返回逻辑状态，不修改 cb.state 字段
func (cb *CircuitBreaker) currentState() State {
	if cb.state == StateOpen {
		if time.Since(cb.lastFailureTime) >= cb.config.Timeout {
			return StateHalfOpen
		}
	}
	return cb.state
}

// ensureStateTransition 在持有写锁时，将 Open→HalfOpen 的隐式转换持久化
// 所有需要修改状态的方法（Allow/RecordSuccess/RecordFailure）统一调用此方法
func (cb *CircuitBreaker) ensureStateTransition() State {
	logical := cb.currentState()
	if logical == StateHalfOpen && cb.state == StateOpen {
		cb.transition(StateHalfOpen)
		cb.failures = 0
		cb.successes = 0
		cb.halfOpenInFlight = false
	}
	return cb.state
}

// transition 记录状态转换并更新 Prometheus 指标（需持有写锁）
func (cb *CircuitBreaker) transition(to State) {
	from := cb.state
	if from == to {
		return
	}
	cbTransitionsTotal.WithLabelValues(cb.name, from.String(), to.String()).Inc()
	cbStateGauge.WithLabelValues(cb.name).Set(float64(to))
	cb.state = to
	if to != StateHalfOpen {
		cb.halfOpenInFlight = false
	}
}

// Allow 检查是否允许执行操作
func (cb *CircuitBreaker) Allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	// 统一通过 ensureStateTransition 处理 Open→HalfOpen 转换
	state := cb.ensureStateTransition()
	switch state {
	case StateClosed:
		return true
	case StateHalfOpen:
		if cb.halfOpenInFlight {
			return false
		}
		cb.halfOpenInFlight = true
		return true
	default:
		return false
	}
}

// RecordSuccess 记录成功
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	// 统一通过 ensureStateTransition 处理 Open→HalfOpen 转换
	cb.ensureStateTransition()

	switch cb.state {
	case StateClosed:
		cb.failures = 0
	case StateHalfOpen:
		cb.halfOpenInFlight = false
		cb.successes++
		if cb.successes >= cb.config.SuccessThreshold {
			cb.transition(StateClosed)
			cb.failures = 0
			cb.successes = 0
		}
	}
	cb.updateGauges()
}

// RecordFailure 记录失败
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	// 统一通过 ensureStateTransition 处理 Open→HalfOpen 转换
	cb.ensureStateTransition()

	cb.failures++
	cb.lastFailureTime = time.Now()

	switch cb.state {
	case StateClosed:
		if cb.failures >= cb.config.FailureThreshold {
			cb.transition(StateOpen)
		}
	case StateHalfOpen:
		cb.halfOpenInFlight = false
		cb.transition(StateOpen)
		cb.successes = 0
	}
	cb.updateGauges()
}

// RecordNeutral completes an allowed operation without treating its outcome as
// backend success or failure. This is used for caller-driven cancellation: it
// must not affect health counters, but a half-open probe reservation still has
// to be released so a later recovery probe can run.
func (cb *CircuitBreaker) RecordNeutral() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.ensureStateTransition()
	if cb.state == StateHalfOpen {
		cb.halfOpenInFlight = false
	}
	cb.updateGauges()
}

// updateGauges 更新 Prometheus failure/success gauges（需持有锁）
func (cb *CircuitBreaker) updateGauges() {
	cbFailuresGauge.WithLabelValues(cb.name).Set(float64(cb.failures))
	cbSuccessesGauge.WithLabelValues(cb.name).Set(float64(cb.successes))
}

// Metrics 获取熔断器指标
func (cb *CircuitBreaker) Metrics() map[string]any {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	return map[string]any{
		"state":           cb.currentState().String(),
		"failures":        cb.failures,
		"successes":       cb.successes,
		"last_failure":    cb.lastFailureTime,
		"probe_in_flight": cb.halfOpenInFlight,
	}
}
