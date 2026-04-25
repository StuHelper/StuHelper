import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const authApi = {
  exchangeNative: vi.fn(),
  logout: vi.fn(),
  me: vi.fn(),
  requestPhoneOTP: vi.fn(),
  verifyPhoneOTP: vi.fn(),
}

vi.mock('@/api', () => ({
  api: {
    auth: authApi,
  },
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
    setActivePinia(createPinia())
  })

  it('persists session id returned by exchange-native', async () => {
    const setStorageSync = vi.fn()
    const getStorageSync = vi.fn(() => '')

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
      removeStorageSync: vi.fn(),
      setStorageSync,
    })

    const { useAuthStore } = await import('./auth')
    const store = useAuthStore()

    await store.exchangeNativeCode('code-1', 'state-1')

    expect(setStorageSync).toHaveBeenCalledWith(
      'stuhelper:native-tokens',
      expect.stringContaining('"sessionID":"sid-native-1"'),
    )
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
      getStorageSync: vi.fn(() => ''),
      removeStorageSync: vi.fn(),
      setStorageSync: vi.fn(() => {
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
      getStorageSync: vi.fn(() => storedTokens),
      removeStorageSync: vi.fn(),
      setStorageSync: vi.fn(),
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
      getStorageSync: vi.fn(() => {
        throw new Error('bridge unavailable')
      }),
      removeStorageSync: vi.fn(),
      setStorageSync: vi.fn(),
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
      getStorageSync: vi.fn(() => {
        throw new Error('bridge unavailable')
      }),
      navigateTo,
      removeStorageSync: vi.fn(),
      setStorageSync: vi.fn(),
      showToast,
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
