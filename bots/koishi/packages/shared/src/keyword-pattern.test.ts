import assert from 'node:assert/strict'
import test from 'node:test'

import {
  UnsafeKeywordPatternError,
  assertSafeKeywordRegex,
  testSafeKeywordRegex,
} from './keyword-pattern.ts'

test('testSafeKeywordRegex matches valid bounded keyword patterns', () => {
  assert.equal(testSafeKeywordRegex('bad\\s+word', 'this has a bad word'), true)
  assert.equal(testSafeKeywordRegex('bad\\s+word', 'clean content'), false)
})

test('assertSafeKeywordRegex rejects nested quantified patterns', () => {
  assert.throws(
    () => assertSafeKeywordRegex('(a+)+$'),
    UnsafeKeywordPatternError,
  )
})

test('assertSafeKeywordRegex rejects backreferences and invalid regex syntax', () => {
  assert.throws(
    () => assertSafeKeywordRegex('(a)\\1'),
    UnsafeKeywordPatternError,
  )
  assert.throws(
    () => assertSafeKeywordRegex('('),
    UnsafeKeywordPatternError,
  )
})
