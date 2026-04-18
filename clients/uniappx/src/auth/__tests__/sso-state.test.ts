import { describe, expect, it } from 'vitest'

import { validateStoredSSOState } from '@/auth/sso-state'

describe('validateStoredSSOState', () => {
  it('fails closed when the saved state is missing', () => {
    expect(validateStoredSSOState('', 'returned-state')).toEqual({
      ok: false,
      reason: 'missing_saved_state',
    })
  })

  it('fails closed when the callback state is missing', () => {
    expect(validateStoredSSOState('saved-state', '')).toEqual({
      ok: false,
      reason: 'missing_saved_state',
    })
  })

  it('rejects a mismatched state value', () => {
    expect(validateStoredSSOState('saved-state', 'other-state')).toEqual({
      ok: false,
      reason: 'mismatch',
    })
  })

  it('accepts only an exact state match', () => {
    expect(validateStoredSSOState('saved-state', 'saved-state')).toEqual({
      ok: true,
    })
  })
})
