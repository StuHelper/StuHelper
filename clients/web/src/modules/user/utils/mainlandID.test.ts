import { describe, expect, it } from 'vitest'

import { isValidMainlandIDNumber, normalizeMainlandIDNumber } from './mainlandID'

describe('mainlandID utils', () => {
  it('normalizes whitespace and lowercase checksum X', () => {
    expect(normalizeMainlandIDNumber(' 11010519491231002x ')).toBe(
      '11010519491231002X',
    )
  })

  it('accepts valid Mainland China ID numbers', () => {
    expect(isValidMainlandIDNumber('110101200001010010')).toBe(true)
    expect(isValidMainlandIDNumber('11010519491231002x')).toBe(true)
  })

  it('rejects invalid format, date and checksum', () => {
    expect(isValidMainlandIDNumber('not-an-id-card')).toBe(false)
    expect(isValidMainlandIDNumber('110101200102290010')).toBe(false)
    expect(isValidMainlandIDNumber('110101200001010011')).toBe(false)
  })
})
