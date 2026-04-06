import { beforeEach, describe, expect, it, vi } from 'vitest'

const clearSession = vi.fn()

vi.mock('@/i18n', () => ({
  translate: (key: string) => key,
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({
    clearSession,
  }),
}))

import {
  assertMutationSuccess,
  extractErrorMessage,
  unwrapData,
  unwrapOptionalData,
} from '../result'

describe('uniappx api result helpers', () => {
  beforeEach(() => {
    clearSession.mockClear()
  })

  it('extracts structured backend error messages', () => {
    expect(
      extractErrorMessage({
        data: {
          error: {
            code: 'COURSE_NOT_FOUND',
            message: 'course not found',
            details: { courseID: 1 },
          },
        },
      })
    ).toBe('course not found')
  })

  it('falls back to nested error objects', () => {
    expect(
      extractErrorMessage({
        error: {
          error: {
            message: 'permission denied',
          },
        },
      })
    ).toBe('permission denied')
  })

  it('uses session-expired copy for 401 responses', () => {
    expect(extractErrorMessage({ response: { status: 401 } })).toBe('common.sessionExpired')
  })

  it('unwraps payload data and optional payloads', () => {
    expect(unwrapData({ data: { data: { id: 1 } }, response: { status: 200 } })).toEqual({ id: 1 })
    expect(unwrapOptionalData({ data: { data: null }, response: { status: 200 } })).toBeNull()
  })

  it('clears session for 401 optional lookups', () => {
    expect(unwrapOptionalData({ response: { status: 401 } })).toBeNull()
    expect(clearSession).toHaveBeenCalledTimes(1)
  })

  it('throws on failed mutations even when the response is otherwise present', () => {
    expect(() =>
      assertMutationSuccess({
        data: {
          success: false,
          message: 'save failed',
        },
        response: { status: 200 },
      })
    ).toThrow('save failed')
  })
})
