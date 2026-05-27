import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const authApi = {
  exchangeNative: vi.fn(),
  logout: vi.fn(),
  me: vi.fn(),
}
const hasStoredH5SessionHint = vi.fn()

vi.mock('@/api', () => ({
  api: {
    auth: authApi,
  },
}))

vi.mock('@/api/shared-client', () => ({
  hasStoredH5SessionHint,
}))

vi.mock('@/api/result', () => ({
  assertMutationSuccess: vi.fn(),
  unwrapData: vi.fn((result: { data?: { data?: unknown } }) => result.data?.data),
  unwrapOptionalData: vi.fn((result: { data?: { data?: unknown } }) => result.data?.data ?? null),
}))

vi.mock('@/i18n', () => ({
  translate: (key: string) => key,
}))

describe('useAuthStore', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.resetModules()
    vi.unstubAllGlobals()
    hasStoredH5SessionHint.mockReturnValue(false)
    setActivePinia(createPinia())
  })

  it('skips H5 session bootstrap when no local browser session hint exists', async () => {
    const { useAuthStore } = await import('./auth')
    const store = useAuthStore()

    await store.bootstrapSession(true)

    expect(hasStoredH5SessionHint).toHaveBeenCalledTimes(1)
    expect(authApi.me).not.toHaveBeenCalled()
    expect(store.initialized).toBe(true)
    expect(store.isAuthenticated).toBe(false)
  })

  it('probes H5 session bootstrap when a local browser session hint exists', async () => {
    hasStoredH5SessionHint.mockReturnValue(true)
    authApi.me.mockResolvedValue({
      data: {
        data: {
          id: 'h5-user-1',
          displayName: 'H5 User',
        },
      },
    })

    const { useAuthStore } = await import('./auth')
    const store = useAuthStore()

    await store.bootstrapSession(true)

    expect(authApi.me).toHaveBeenCalledTimes(1)
    expect(store.isAuthenticated).toBe(true)
    expect(store.user?.displayName).toBe('H5 User')
  })

  it('persists session id returned by exchange-native', async () => {
    const getStorageSync = vi.fn((key: string) => {
      if (key === 'stuhelper:sso-state') {
        return 'state-1'
      }
      return ''
    })
    const removeStorageSync = vi.fn()
    const secureStorage = {
      getItem: vi.fn(),
      removeItem: vi.fn(),
      setItem: vi.fn(),
    }

    authApi.exchangeNative.mockResolvedValue({
      data: {
        data: {
          accessToken: 'native-access',
          refreshToken: 'native-refresh',
          sessionID: 'sid-native-1',
          expiresIn: 600,
        },
      },
    })
    authApi.me.mockResolvedValue({
      data: {
        data: {
          id: 'user-1',
          displayName: 'Native User',
        },
      },
    })

    vi.stubGlobal('plus', {})
    vi.stubGlobal('uni', {
      getStorageSync,
      removeStorageSync,
    })
    vi.stubGlobal('stuhelperSecureStorage', secureStorage)

    const { useAuthStore } = await import('./auth')
    const store = useAuthStore()

    await store.exchangeNativeCode('code-1', 'state-1')

    expect(secureStorage.setItem).toHaveBeenCalledWith(
      'stuhelper.native-session',
      'stuhelper:native-tokens',
      expect.stringContaining('"sessionID":"sid-native-1"'),
    )
    expect(removeStorageSync).toHaveBeenCalledWith('stuhelper:native-tokens')
    expect(removeStorageSync).toHaveBeenCalledWith('stuhelper:sso-state')
  })

  it('fails closed when persisting native session tokens fails', async () => {
    authApi.exchangeNative.mockResolvedValue({
      data: {
        data: {
          accessToken: 'native-access',
          refreshToken: 'native-refresh',
          sessionID: 'sid-native-write-fail',
          expiresIn: 600,
        },
      },
    })

    vi.stubGlobal('plus', {})
    vi.stubGlobal('uni', {
      getStorageSync: vi.fn((key: string) => {
        if (key === 'stuhelper:sso-state') {
          return 'state-write-fail'
        }
        return ''
      }),
      removeStorageSync: vi.fn(),
    })
    vi.stubGlobal('stuhelperSecureStorage', {
      getItem: vi.fn(),
      removeItem: vi.fn(),
      setItem: vi.fn(() => {
        throw new Error('disk full')
      }),
    })

    const { useAuthStore } = await import('./auth')
    const store = useAuthStore()

    await expect(store.exchangeNativeCode('code-write-fail', 'state-write-fail')).rejects.toThrow(
      'failed to persist native session tokens',
    )
    expect(authApi.me).not.toHaveBeenCalled()
  })

  it('fails closed when the callback state does not match the stored native state', async () => {
    vi.stubGlobal('plus', {})
    vi.stubGlobal('uni', {
      getStorageSync: vi.fn((key: string) => {
        if (key === 'stuhelper:sso-state') {
          return 'saved-state'
        }
        return ''
      }),
      removeStorageSync: vi.fn(),
      setStorageSync: vi.fn(),
    })

    const { useAuthStore } = await import('./auth')
    const store = useAuthStore()

    await expect(store.exchangeNativeCode('code-1', 'other-state')).rejects.toThrow(
      'invalid native SSO state',
    )
    expect(authApi.exchangeNative).not.toHaveBeenCalled()
  })

  it('always clears the stored native state after exchange failures', async () => {
    const removeStorageSync = vi.fn()

    authApi.exchangeNative.mockRejectedValue(new Error('exchange failed'))

    vi.stubGlobal('plus', {})
    vi.stubGlobal('uni', {
      getStorageSync: vi.fn((key: string) => {
        if (key === 'stuhelper:sso-state') {
          return 'state-1'
        }
        return ''
      }),
      removeStorageSync,
      setStorageSync: vi.fn(),
    })

    const { useAuthStore } = await import('./auth')
    const store = useAuthStore()

    await expect(store.exchangeNativeCode('code-1', 'state-1')).rejects.toThrow(
      'exchange failed',
    )
    expect(removeStorageSync).toHaveBeenCalledWith('stuhelper:sso-state')
  })

  it('bootstraps native session even when access token is expired but refresh session still exists', async () => {
    const storedTokens = JSON.stringify({
      accessToken: 'expired-access',
      refreshToken: 'valid-refresh',
      sessionID: 'sid-native-2',
      expiresAt: Date.now() - 1_000,
    })

    authApi.me.mockResolvedValue({
      data: {
        data: {
          id: 'user-2',
          displayName: 'Recovered User',
        },
      },
    })

    vi.stubGlobal('plus', {})
    vi.stubGlobal('uni', {
      getStorageSync: vi.fn(),
      removeStorageSync: vi.fn(),
    })
    vi.stubGlobal('stuhelperSecureStorage', {
      getItem: vi.fn(() => storedTokens),
      removeItem: vi.fn(),
      setItem: vi.fn(),
    })

    const { useAuthStore } = await import('./auth')
    const store = useAuthStore()

    await store.bootstrapSession(true)

    expect(authApi.me).toHaveBeenCalledTimes(1)
    expect(store.isAuthenticated).toBe(true)
    expect(store.user?.displayName).toBe('Recovered User')
  })

  it('surfaces native session storage read failures during bootstrap', async () => {
    vi.stubGlobal('plus', {})
    vi.stubGlobal('uni', {
      removeStorageSync: vi.fn(),
    })
    vi.stubGlobal('stuhelperSecureStorage', {
      getItem: vi.fn(() => {
        throw new Error('bridge unavailable')
      }),
      removeItem: vi.fn(),
      setItem: vi.fn(),
    })

    const { useAuthStore } = await import('./auth')
    const store = useAuthStore()

    await expect(store.bootstrapSession(true)).rejects.toThrow('failed to read native session tokens')
    expect(store.initialized).toBe(false)
    expect(authApi.me).not.toHaveBeenCalled()
  })

  it('does not redirect to login when bootstrap fails unexpectedly', async () => {
    const navigateTo = vi.fn()
    const showToast = vi.fn()

    vi.stubGlobal('plus', {})
    vi.stubGlobal('uni', {
      navigateTo,
      removeStorageSync: vi.fn(),
      showToast,
    })
    vi.stubGlobal('stuhelperSecureStorage', {
      getItem: vi.fn(() => {
        throw new Error('bridge unavailable')
      }),
      removeItem: vi.fn(),
      setItem: vi.fn(),
    })

    const { useAuthStore } = await import('./auth')
    const store = useAuthStore()

    await expect(store.requireAuth()).resolves.toBe(false)
    expect(showToast).toHaveBeenCalledWith({
      title: 'failed to read native session tokens',
      icon: 'none',
    })
    expect(navigateTo).not.toHaveBeenCalled()
  })
})
