import assert from 'node:assert/strict'
import test from 'node:test'

import {
  BindingAttemptGuard,
  BINDING_ATTEMPT_MIN_INTERVAL_MS,
  BINDING_FAILURE_LOCKOUT_MS,
  BINDING_MAX_CONSECUTIVE_FAILURES,
} from './attempt-guard.ts'

const BASE = 1_000_000

test('限速：最小间隔内的第二次尝试被拒绝并给出重试等待时间', () => {
  const guard = new BindingAttemptGuard()

  assert.deepEqual(guard.check('qq:10001', BASE), { allowed: true })
  const denied = guard.check('qq:10001', BASE + 1_000)
  assert.equal(denied.allowed, false)
  assert.equal(denied.allowed === false && denied.reason, 'rate_limited')
  assert.equal(
    denied.allowed === false && denied.retryAfterMs,
    BINDING_ATTEMPT_MIN_INTERVAL_MS - 1_000,
  )

  const allowedAgain = guard.check('qq:10001', BASE + BINDING_ATTEMPT_MIN_INTERVAL_MS)
  assert.deepEqual(allowedAgain, { allowed: true })
})

test('限速按用户隔离：另一个用户不受影响', () => {
  const guard = new BindingAttemptGuard()

  assert.deepEqual(guard.check('qq:10001', BASE), { allowed: true })
  assert.deepEqual(guard.check('qq:10002', BASE + 1), { allowed: true })
})

test('连续失败达到上限后锁定，期间尝试被拒绝', () => {
  const guard = new BindingAttemptGuard()
  let now = BASE

  for (let attempt = 0; attempt < BINDING_MAX_CONSECUTIVE_FAILURES; attempt += 1) {
    assert.deepEqual(guard.check('qq:10001', now), { allowed: true })
    guard.markFailure('qq:10001', now)
    now += BINDING_ATTEMPT_MIN_INTERVAL_MS
  }

  const locked = guard.check('qq:10001', now)
  assert.equal(locked.allowed, false)
  assert.equal(locked.allowed === false && locked.reason, 'locked_out')
  assert.ok(locked.allowed === false && locked.retryAfterMs > 0)
  assert.ok(locked.allowed === false && locked.retryAfterMs <= BINDING_FAILURE_LOCKOUT_MS)
})

test('锁定到期后允许重试且失败计数清零', () => {
  const guard = new BindingAttemptGuard()
  let now = BASE

  for (let attempt = 0; attempt < BINDING_MAX_CONSECUTIVE_FAILURES; attempt += 1) {
    guard.check('qq:10001', now)
    guard.markFailure('qq:10001', now)
    now += BINDING_ATTEMPT_MIN_INTERVAL_MS
  }
  assert.equal(guard.check('qq:10001', now).allowed, false)

  const afterLockout = now + BINDING_FAILURE_LOCKOUT_MS
  assert.deepEqual(guard.check('qq:10001', afterLockout), { allowed: true })

  // 计数已清零：再失败一次不会立即重新锁定
  guard.markFailure('qq:10001', afterLockout)
  assert.deepEqual(
    guard.check('qq:10001', afterLockout + BINDING_ATTEMPT_MIN_INTERVAL_MS),
    { allowed: true },
  )
})

test('绑定成功后失败计数清零', () => {
  const guard = new BindingAttemptGuard()
  let now = BASE

  for (let attempt = 0; attempt < BINDING_MAX_CONSECUTIVE_FAILURES - 1; attempt += 1) {
    guard.check('qq:10001', now)
    guard.markFailure('qq:10001', now)
    now += BINDING_ATTEMPT_MIN_INTERVAL_MS
  }
  guard.check('qq:10001', now)
  guard.markSuccess('qq:10001')
  now += BINDING_ATTEMPT_MIN_INTERVAL_MS

  // 成功后再失败一次不触发锁定
  guard.check('qq:10001', now)
  guard.markFailure('qq:10001', now)
  assert.deepEqual(
    guard.check('qq:10001', now + BINDING_ATTEMPT_MIN_INTERVAL_MS),
    { allowed: true },
  )
})

test('状态表超出容量时清理过期条目，不影响新条目登记', () => {
  const guard = new BindingAttemptGuard({ capacity: 3 })
  let now = BASE

  guard.check('stale-1', now)
  guard.check('stale-2', now + 1)
  guard.check('stale-3', now + 2)

  // 远超 lockout 窗口后插入新用户，触发容量清理
  now += BINDING_FAILURE_LOCKOUT_MS * 2
  assert.deepEqual(guard.check('fresh', now), { allowed: true })
  // 新用户的限速立即生效，证明状态被正常登记
  assert.equal(guard.check('fresh', now + 1).allowed, false)
})
