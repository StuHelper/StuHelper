import assert from 'node:assert/strict'
import test from 'node:test'

import { evaluateThresholdExpression } from './threshold-expression'

test('阈值表达式支持比较与逻辑运算', () => {
  assert.equal(
    evaluateThresholdExpression('warnings >= 3 && repeats >= 2', {
      warnings: 3,
      repeats: 2,
      reports: 0,
    }),
    true,
  )

  assert.equal(
    evaluateThresholdExpression('warnings >= 4 || reports >= 1', {
      warnings: 3,
      repeats: 0,
      reports: 1,
    }),
    true,
  )

  assert.equal(
    evaluateThresholdExpression('warnings >= 4 && reports >= 1', {
      warnings: 3,
      repeats: 0,
      reports: 1,
    }),
    false,
  )
})

test('非法阈值表达式抛出明确错误', () => {
  const metrics = { warnings: 0, repeats: 0, reports: 0 }

  assert.throws(() => evaluateThresholdExpression('warnings => 3', metrics))
  assert.throws(() => evaluateThresholdExpression('warnings >=', metrics), /unexpected end of expression/)
  assert.throws(() => evaluateThresholdExpression('warnings ~ 3', metrics), /invalid expression near/)
  assert.throws(() => evaluateThresholdExpression('(warnings >= 3', metrics), /expected \)/)
})
