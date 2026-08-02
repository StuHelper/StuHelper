import { afterEach, describe, expect, it, vi } from 'vitest'

import {
  clearStoredSSORedirect,
  clearStoredSSOState,
  consumeStoredSSORedirect,
  DEFAULT_SSO_REDIRECT,
  normalizeRedirectOption,
  persistSSORedirect,
  readStoredSSORedirect,
  SSO_REDIRECT_STORAGE_KEY,
  SSO_STATE_STORAGE_KEY,
  persistSSOState,
  readStoredSSOState,
  validateStoredSSOState,
} from '@/auth/sso-state'

describe('validateStoredSSOState', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
  })

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

  it('rejects states with an unexpected character set', () => {
    expect(validateStoredSSOState('saved-state', 'saved state')).toEqual({
      ok: false,
      reason: 'missing_saved_state',
    })
  })

  it('rejects states that exceed the allowed length', () => {
    expect(validateStoredSSOState('a'.repeat(257), 'a'.repeat(257))).toEqual({
      ok: false,
      reason: 'missing_saved_state',
    })
  })

  it('persists native sso state before redirecting', () => {
    const setStorageSync = vi.fn()

    vi.stubGlobal('uni', {
      setStorageSync,
    })

    persistSSOState('saved-state')

    expect(setStorageSync).toHaveBeenCalledWith(SSO_STATE_STORAGE_KEY, 'saved-state')
  })

  it('fails closed when native sso state cannot be persisted', () => {
    vi.stubGlobal('uni', {
      setStorageSync: vi.fn(() => {
        throw new Error('disk full')
      }),
    })

    expect(() => persistSSOState('saved-state')).toThrow('failed to persist native SSO state')
  })

  it('reads the stored native sso state', () => {
    vi.stubGlobal('uni', {
      getStorageSync: vi.fn(() => 'saved-state'),
    })

    expect(readStoredSSOState()).toBe('saved-state')
  })

  it('surfaces native sso state storage read failures', () => {
    vi.stubGlobal('uni', {
      getStorageSync: vi.fn(() => {
        throw new Error('bridge unavailable')
      }),
    })

    expect(() => readStoredSSOState()).toThrow('failed to read native SSO state')
  })

  it('clears the stored native sso state after callback handling', () => {
    const removeStorageSync = vi.fn()

    vi.stubGlobal('uni', {
      removeStorageSync,
    })

    clearStoredSSOState()

    expect(removeStorageSync).toHaveBeenCalledWith(SSO_STATE_STORAGE_KEY)
  })

  it('surfaces native sso state storage clear failures', () => {
    vi.stubGlobal('uni', {
      removeStorageSync: vi.fn(() => {
        throw new Error('bridge unavailable')
      }),
    })

    expect(() => clearStoredSSOState()).toThrow('failed to clear native SSO state')
  })
})

describe('native SSO redirect storage', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('normalizes an encoded internal page and rejects external or over-encoded targets', () => {
    expect(normalizeRedirectOption('%2Fpages%2Fcourse%2Fdetail%3Fid%3D42')).toBe(
      '/pages/course/detail?id=42',
    )
    expect(normalizeRedirectOption('https://evil.example/pages/user/index')).toBe(
      DEFAULT_SSO_REDIRECT,
    )
    const overEncoded = encodeURIComponent(
      encodeURIComponent(encodeURIComponent('/pages/review/index')),
    )
    expect(normalizeRedirectOption(overEncoded)).toBe(DEFAULT_SSO_REDIRECT)
  })

  it('persists a validated redirect without making storage failure fatal', () => {
    const setStorageSync = vi.fn()
    vi.stubGlobal('uni', { setStorageSync })

    persistSSORedirect('/pages/course/detail?id=42')
    expect(setStorageSync).toHaveBeenCalledWith(
      SSO_REDIRECT_STORAGE_KEY,
      '/pages/course/detail?id=42',
    )

    setStorageSync.mockImplementationOnce(() => {
      throw new Error('storage unavailable')
    })
    expect(() => persistSSORedirect('/pages/review/index')).not.toThrow()
  })

  it('consumes the stored redirect once and falls back when it is missing', () => {
    const removeStorageSync = vi.fn()
    vi.stubGlobal('uni', {
      getStorageSync: vi.fn(() => '/pages/course/detail?id=42'),
      removeStorageSync,
    })

    expect(consumeStoredSSORedirect()).toBe('/pages/course/detail?id=42')
    expect(removeStorageSync).toHaveBeenCalledWith(SSO_REDIRECT_STORAGE_KEY)

    vi.stubGlobal('uni', {
      getStorageSync: vi.fn(() => ''),
      removeStorageSync,
    })
    expect(consumeStoredSSORedirect()).toBe(DEFAULT_SSO_REDIRECT)
  })

  it('treats redirect read and clear failures as a non-fatal fallback', () => {
    vi.stubGlobal('uni', {
      getStorageSync: vi.fn(() => {
        throw new Error('bridge unavailable')
      }),
      removeStorageSync: vi.fn(() => {
        throw new Error('bridge unavailable')
      }),
    })

    expect(readStoredSSORedirect()).toBeNull()
    expect(() => clearStoredSSORedirect()).not.toThrow()
    expect(consumeStoredSSORedirect()).toBe(DEFAULT_SSO_REDIRECT)
  })
})
