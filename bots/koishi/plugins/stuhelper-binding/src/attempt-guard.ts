/**
 * 绑定码消费的进程内防爆破护栏（F055）。
 *
 * 两层限制（均按用户维度，内存实现）：
 * 1. 每用户限速：两次消费尝试之间至少间隔 BINDING_ATTEMPT_MIN_INTERVAL_MS；
 * 2. 连续失败上限：绑定码连续无效 BINDING_MAX_CONSECUTIVE_FAILURES 次后
 *    锁定 BINDING_FAILURE_LOCKOUT_MS，期间一律拒绝。
 */

/** 同一用户两次绑定码消费尝试的最小间隔 */
export const BINDING_ATTEMPT_MIN_INTERVAL_MS = 3_000
/** 触发锁定前允许的连续无效绑定码次数 */
export const BINDING_MAX_CONSECUTIVE_FAILURES = 5
/** 连续失败达到上限后的锁定时长 */
export const BINDING_FAILURE_LOCKOUT_MS = 10 * 60 * 1000
/** 内存状态表容量上限，超出时优先清理过期条目 */
export const BINDING_ATTEMPT_STATE_CAPACITY = 10_000

export type BindingAttemptRejection = 'rate_limited' | 'locked_out'

export type BindingAttemptDecision =
  | { readonly allowed: true }
  | {
      readonly allowed: false
      readonly reason: BindingAttemptRejection
      readonly retryAfterMs: number
    }

export interface BindingAttemptGuardOptions {
  readonly minIntervalMs?: number
  readonly maxConsecutiveFailures?: number
  readonly lockoutMs?: number
  readonly capacity?: number
}

interface BindingAttemptState {
  readonly lastAttemptAt: number
  readonly consecutiveFailures: number
  readonly lockedUntil: number
}

export class BindingAttemptGuard {
  private readonly states = new Map<string, BindingAttemptState>()
  private readonly minIntervalMs: number
  private readonly maxConsecutiveFailures: number
  private readonly lockoutMs: number
  private readonly capacity: number

  constructor(options: BindingAttemptGuardOptions = {}) {
    this.minIntervalMs = options.minIntervalMs ?? BINDING_ATTEMPT_MIN_INTERVAL_MS
    this.maxConsecutiveFailures = options.maxConsecutiveFailures ?? BINDING_MAX_CONSECUTIVE_FAILURES
    this.lockoutMs = options.lockoutMs ?? BINDING_FAILURE_LOCKOUT_MS
    this.capacity = options.capacity ?? BINDING_ATTEMPT_STATE_CAPACITY
  }

  /** 检查并登记一次消费尝试；拒绝时不更新状态。 */
  check(key: string, now = Date.now()): BindingAttemptDecision {
    const state = this.states.get(key)
    if (state && state.lockedUntil > now) {
      return { allowed: false, reason: 'locked_out', retryAfterMs: state.lockedUntil - now }
    }
    if (state && now - state.lastAttemptAt < this.minIntervalMs) {
      return {
        allowed: false,
        reason: 'rate_limited',
        retryAfterMs: state.lastAttemptAt + this.minIntervalMs - now,
      }
    }
    this.prune(now)
    const lockoutExpired = Boolean(state && state.lockedUntil > 0 && state.lockedUntil <= now)
    this.states.set(key, {
      lastAttemptAt: now,
      consecutiveFailures: lockoutExpired ? 0 : state?.consecutiveFailures ?? 0,
      lockedUntil: 0,
    })
    return { allowed: true }
  }

  /** 记录一次绑定码无效；达到上限后进入锁定。 */
  markFailure(key: string, now = Date.now()): void {
    const state = this.states.get(key)
    const failures = (state?.consecutiveFailures ?? 0) + 1
    if (failures >= this.maxConsecutiveFailures) {
      this.states.set(key, {
        lastAttemptAt: now,
        consecutiveFailures: 0,
        lockedUntil: now + this.lockoutMs,
      })
      return
    }
    this.states.set(key, {
      lastAttemptAt: state?.lastAttemptAt ?? now,
      consecutiveFailures: failures,
      lockedUntil: 0,
    })
  }

  /** 绑定成功后清零失败计数（保留限速窗口）。 */
  markSuccess(key: string): void {
    const state = this.states.get(key)
    if (!state) return
    this.states.set(key, { ...state, consecutiveFailures: 0, lockedUntil: 0 })
  }

  private prune(now: number): void {
    if (this.states.size < this.capacity) return
    for (const [key, state] of this.states) {
      if (state.lockedUntil <= now && now - state.lastAttemptAt >= this.lockoutMs) {
        this.states.delete(key)
      }
    }
    while (this.states.size >= this.capacity) {
      const oldest = this.states.keys().next()
      if (oldest.done) break
      this.states.delete(oldest.value)
    }
  }
}
