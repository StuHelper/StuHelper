import { describe, expect, it } from 'vitest'

import { isNonArrayRecord } from '../typeGuards'

describe('isNonArrayRecord', () => {
  it('accepts plain objects and rejects arrays, null, and primitives', () => {
    expect(isNonArrayRecord({ value: 1 })).toBe(true)
    expect(isNonArrayRecord([])).toBe(false)
    expect(isNonArrayRecord(null)).toBe(false)
    expect(isNonArrayRecord('value')).toBe(false)
  })
})
