import assert from 'node:assert/strict'
import test from 'node:test'

import {
  DEFAULT_POLICY_CATEGORY_ID,
  POLICY_CATEGORIES,
  isPolicyCategoryId,
  resolvePolicyCategoryId,
} from './categories'

test('POLICY_CATEGORIES keeps the fixed policy center order', () => {
  assert.deepEqual(
    POLICY_CATEGORIES.map((item) => item.id),
    [
      'keyword-rules',
      'command-policies',
      'member-roles',
      'guard-templates',
      'guard-bindings',
    ],
  )
})

test('resolvePolicyCategoryId falls back to the default category for unknown values', () => {
  assert.equal(DEFAULT_POLICY_CATEGORY_ID, 'keyword-rules')
  assert.equal(resolvePolicyCategoryId('member-roles'), 'member-roles')
  assert.equal(resolvePolicyCategoryId('unknown'), DEFAULT_POLICY_CATEGORY_ID)
  assert.equal(resolvePolicyCategoryId(null), DEFAULT_POLICY_CATEGORY_ID)
})

test('isPolicyCategoryId only accepts fixed category ids', () => {
  assert.equal(isPolicyCategoryId('guard-bindings'), true)
  assert.equal(isPolicyCategoryId('review'), false)
  assert.equal(isPolicyCategoryId(''), false)
})
