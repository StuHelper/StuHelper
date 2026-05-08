import assert from 'node:assert/strict'
import test from 'node:test'

import { parseViolationInfo } from './report-commands.ts'
import { ViolationLevel } from './report-types.ts'

test('parseViolationInfo rejects moderation JSON with an out-of-range violation level', () => {
  assert.throws(() => parseViolationInfo(JSON.stringify({
    level: 99,
    reason: 'invalid',
    action: [],
  })), /level/)
})

test('parseViolationInfo rejects actions missing required numeric fields', () => {
  assert.throws(() => parseViolationInfo(JSON.stringify({
    level: ViolationLevel.MEDIUM,
    reason: 'missing ban time',
    action: [{ type: 'ban' }],
  })), /ban time/)
})

test('parseViolationInfo accepts a bounded moderation action payload', () => {
  const result = parseViolationInfo(JSON.stringify({
    level: ViolationLevel.MEDIUM,
    reason: 'validated',
    action: [{ type: 'ban', time: 600 }, { type: 'warn', count: 1 }],
    reporterPenalty: { shouldLimit: true, duration: 30, reason: 'abuse' },
  }))

  assert.equal(result.level, ViolationLevel.MEDIUM)
  assert.deepEqual(result.action.map((item) => item.type), ['ban', 'warn'])
})
