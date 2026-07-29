import assert from 'node:assert/strict'
import test from 'node:test'

import { secureRandomInt, secureRandomUnit } from './secure-random'

test('secureRandomUnit stays inside the half-open unit interval', () => {
  for (let index = 0; index < 128; index += 1) {
    const value = secureRandomUnit()
    assert.ok(value >= 0)
    assert.ok(value < 1)
  }
})

test('secureRandomInt validates bounds and handles a collapsed range', () => {
  assert.equal(secureRandomInt(42, 42), 42)
  assert.throws(() => secureRandomInt(2, 1), RangeError)
  assert.throws(() => secureRandomInt(0.5, 2), RangeError)

  for (let index = 0; index < 128; index += 1) {
    const value = secureRandomInt(5, 11)
    assert.ok(value >= 5)
    assert.ok(value < 11)
  }
})
