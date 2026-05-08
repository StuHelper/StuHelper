import assert from 'node:assert/strict'
import test from 'node:test'

import {
  MAX_DURATION,
  MIN_DURATION,
  evaluateExpression,
  formatDuration,
  parseTimeString,
} from './index'

test('evaluateExpression supports arithmetic, exponentiation, sqrt, and x multiplication alias', () => {
  assert.equal(evaluateExpression('2x3+sqrt(9)+2^3'), 17)
})

test('parseTimeString parses single and combined time expressions', () => {
  assert.equal(parseTimeString('1h30m'), 90 * 60 * 1000)
  assert.equal(parseTimeString('2*3m'), 6 * 60 * 1000)
})

test('parseTimeString clamps durations to configured bounds', () => {
  assert.equal(parseTimeString('0.1s'), MIN_DURATION)
  assert.equal(parseTimeString('99d'), MAX_DURATION)
})

test('formatDuration renders non-zero duration parts', () => {
  assert.equal(formatDuration(90_061_000), '1天1小时1分钟1秒')
})
