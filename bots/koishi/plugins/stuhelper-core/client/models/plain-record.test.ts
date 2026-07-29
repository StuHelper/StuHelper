import assert from 'node:assert/strict'
import test from 'node:test'

import { deepMerge, isPlainRecord } from './plain-record'

test('deepMerge merges own plain-record fields without mutating prototypes', () => {
  const target = { nested: { enabled: false }, label: 'default' }
  const source = JSON.parse(
    '{"nested":{"enabled":true},"label":"saved","__proto__":{"polluted":true},"constructor":{"prototype":{"polluted":true}}}',
  )

  assert.deepEqual(deepMerge(target, source), {
    nested: { enabled: true },
    label: 'saved',
  })
  assert.equal(({} as { polluted?: boolean }).polluted, undefined)
})

test('isPlainRecord rejects arrays and class instances', () => {
  class Example {}

  assert.equal(isPlainRecord({}), true)
  assert.equal(isPlainRecord(Object.create(null)), true)
  assert.equal(isPlainRecord([]), false)
  assert.equal(isPlainRecord(new Example()), false)
})
