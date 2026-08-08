import assert from 'node:assert/strict'
import test from 'node:test'

import { opaqueLogReference } from './log-reference'

test('opaque log references correlate in-process without exposing raw platform ids', () => {
  const raw = '123456789012'
  const first = opaqueLogReference('member', raw)
  const second = opaqueLogReference('member', raw)
  assert.equal(first, second)
  assert.match(first ?? '', /^member_[a-f0-9]{16}$/)
  assert.doesNotMatch(first ?? '', new RegExp(raw))
  assert.notEqual(first, opaqueLogReference('guild', raw))
})
