import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

describe('native session secure storage', () => {
  beforeEach(() => {
    vi.resetModules()
    vi.unstubAllGlobals()
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('persists native tokens only through the secure storage bridge', async () => {
    const removeStorageSync = vi.fn()
    const setStorageSync = vi.fn()
    const secureStorage = {
      getItem: vi.fn(),
      removeItem: vi.fn(),
      setItem: vi.fn(),
    }

    vi.stubGlobal('plus', {})
    vi.stubGlobal('uni', {
      removeStorageSync,
      setStorageSync,
    })
    vi.stubGlobal('stuhelperSecureStorage', secureStorage)

    const {
      NATIVE_TOKEN_SECURE_STORAGE_SERVICE,
      NATIVE_TOKEN_STORAGE_KEY,
      writeNativeTokens,
    } = await import('../native-session')

    writeNativeTokens({
      accessToken: 'access-token',
      refreshToken: 'refresh-token',
      sessionID: 'sid-1',
      expiresAt: 123,
    })

    expect(removeStorageSync).toHaveBeenCalledWith(NATIVE_TOKEN_STORAGE_KEY)
    expect(setStorageSync).not.toHaveBeenCalled()
    expect(secureStorage.setItem).toHaveBeenCalledWith(
      NATIVE_TOKEN_SECURE_STORAGE_SERVICE,
      NATIVE_TOKEN_STORAGE_KEY,
      expect.stringContaining('"refreshToken":"refresh-token"'),
    )
  })

  it('reads native tokens from secure storage and ignores legacy storage payloads', async () => {
    const removeStorageSync = vi.fn()
    const getStorageSync = vi.fn(() => JSON.stringify({
      accessToken: 'legacy-access',
      refreshToken: 'legacy-refresh',
      sessionID: 'legacy-sid',
      expiresAt: Date.now() + 60_000,
    }))
    const securePayload = JSON.stringify({
      accessToken: 'secure-access',
      refreshToken: 'secure-refresh',
      sessionID: 'secure-sid',
      expiresAt: Date.now() + 60_000,
    })
    const secureStorage = {
      getItem: vi.fn(() => securePayload),
      removeItem: vi.fn(),
      setItem: vi.fn(),
    }

    vi.stubGlobal('plus', {})
    vi.stubGlobal('uni', {
      getStorageSync,
      removeStorageSync,
    })
    vi.stubGlobal('stuhelperSecureStorage', secureStorage)

    const { NATIVE_TOKEN_STORAGE_KEY, readNativeTokens } = await import('../native-session')

    expect(readNativeTokens()).toMatchObject({
      accessToken: 'secure-access',
      refreshToken: 'secure-refresh',
      sessionID: 'secure-sid',
    })
    expect(getStorageSync).not.toHaveBeenCalled()
    expect(removeStorageSync).toHaveBeenCalledWith(NATIVE_TOKEN_STORAGE_KEY)
  })

  it('fails closed when the native secure storage bridge is unavailable', async () => {
    vi.stubGlobal('plus', {})
    vi.stubGlobal('uni', {
      removeStorageSync: vi.fn(),
    })

    const { readNativeTokens, writeNativeTokens } = await import('../native-session')

    expect(() => readNativeTokens()).toThrow('failed to read native session tokens')
    expect(() => writeNativeTokens({
      accessToken: 'access-token',
      refreshToken: 'refresh-token',
      sessionID: 'sid-1',
      expiresAt: Date.now() + 60_000,
    })).toThrow('failed to persist native session tokens')
  })

  it('clears secure storage and legacy storage on logout', async () => {
    const removeStorageSync = vi.fn()
    const secureStorage = {
      getItem: vi.fn(),
      removeItem: vi.fn(),
      setItem: vi.fn(),
    }

    vi.stubGlobal('plus', {})
    vi.stubGlobal('uni', {
      removeStorageSync,
    })
    vi.stubGlobal('stuhelperSecureStorage', secureStorage)

    const {
      NATIVE_TOKEN_SECURE_STORAGE_SERVICE,
      NATIVE_TOKEN_STORAGE_KEY,
      clearNativeTokens,
    } = await import('../native-session')

    clearNativeTokens()

    expect(removeStorageSync).toHaveBeenCalledWith(NATIVE_TOKEN_STORAGE_KEY)
    expect(secureStorage.removeItem).toHaveBeenCalledWith(
      NATIVE_TOKEN_SECURE_STORAGE_SERVICE,
      NATIVE_TOKEN_STORAGE_KEY,
    )
  })
})
