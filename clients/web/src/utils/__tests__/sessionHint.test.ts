import { beforeEach, describe, expect, it, vi } from 'vitest'

const mockTokenExpiryGet = vi.fn()

vi.mock('../auth', () => ({
  tokenExpiry: {
    get: mockTokenExpiryGet,
  },
}))

describe('session hint utilities', () => {
  beforeEach(() => {
    mockTokenExpiryGet.mockReset()
    mockTokenExpiryGet.mockReturnValue(null)
    Object.defineProperty(globalThis, 'document', {
      configurable: true,
      value: {
        cookie: '',
      },
    })
  })

  it('reads encoded cookie values', async () => {
    Object.defineProperty(globalThis, 'document', {
      configurable: true,
      value: {
        cookie: 'csrf_token=hello%20world',
      },
    })

    const { readCookie } = await import('../sessionHint')

    expect(readCookie('csrf_token')).toBe('hello world')
  })

  it('detects a browser session hint from CSRF cookie or token expiry', async () => {
    const { hasStoredSessionHint } = await import('../sessionHint')

    expect(hasStoredSessionHint()).toBe(false)

    Object.defineProperty(globalThis, 'document', {
      configurable: true,
      value: {
        cookie: 'csrf_token=test-csrf',
      },
    })
    expect(hasStoredSessionHint()).toBe(true)

    Object.defineProperty(globalThis, 'document', {
      configurable: true,
      value: {
        cookie: '',
      },
    })
    mockTokenExpiryGet.mockReturnValue(Date.now() + 60_000)
    expect(hasStoredSessionHint()).toBe(true)
  })
})
