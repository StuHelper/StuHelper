import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

describe('uniappx api client transport', () => {
  beforeEach(() => {
    vi.resetModules()
    vi.unstubAllGlobals()
    vi.unstubAllEnvs()
  })

  afterEach(() => {
    vi.unstubAllGlobals()
    vi.unstubAllEnvs()
  })

  it('injects csrf headers for unsafe requests in browser environments', async () => {
    const request = vi.fn((options: any) => {
      options.success?.({
        statusCode: 200,
        data: {
          data: { ok: true },
        },
        header: {
          'content-type': 'application/json',
        },
      })
      return { abort: vi.fn() }
    })

    vi.stubEnv('VITE_API_URL', '/api')
    vi.stubGlobal('window', {
      location: {
        origin: 'https://app.example',
      },
    })
    vi.stubGlobal('document', {
      cookie: 'foo=bar; csrf_token=test-csrf-token',
    })
    vi.stubGlobal('uni', {
      request,
    })

    const { apiClient } = await import('../shared-client')
    const result = await apiClient.POST('/api/v1/review' as never, {
      body: {
        hello: 'world',
      },
    } as never)

    expect(request).toHaveBeenCalledTimes(1)
    expect(request.mock.calls[0][0]).toMatchObject({
      url: 'https://app.example/api/v1/review',
      method: 'POST',
      withCredentials: true,
      header: {
        'Content-Type': 'application/json',
        'X-CSRF-Token': 'test-csrf-token',
      },
    })
    expect(result.response?.status).toBe(200)
    expect(result.data).toEqual({ data: { ok: true } })
  })

  it('preserves top-level request headers in uni request transport', async () => {
    const request = vi.fn((options: any) => {
      options.success?.({
        statusCode: 200,
        data: {
          data: { ok: true },
        },
        header: {
          'content-type': 'application/json',
        },
      })
      return { abort: vi.fn() }
    })

    vi.stubEnv('VITE_API_URL', '/api')
    vi.stubGlobal('window', {
      location: {
        origin: 'https://app.example',
      },
    })
    vi.stubGlobal('document', {
      cookie: 'csrf_token=test-csrf-token',
    })
    vi.stubGlobal('uni', {
      request,
    })

    const { apiClient } = await import('../shared-client')
    const result = await apiClient.POST('/api/v1/open-platform/resources/access/check' as never, {
      body: {
        action: 'read',
        resourceID: 'resource-42',
        resourceType: 'resource_item',
      },
      headers: {
        Authorization: 'Bearer resource-access-token',
      },
    } as never)

    expect(request).toHaveBeenCalledTimes(1)
    expect(request.mock.calls[0][0]).toMatchObject({
      header: {
        Authorization: 'Bearer resource-access-token',
        'Content-Type': 'application/json',
        'X-CSRF-Token': 'test-csrf-token',
      },
      method: 'POST',
      url: 'https://app.example/api/v1/open-platform/resources/access/check',
    })
    expect(result.response?.status).toBe(200)
  })

  it('detects H5 session hints from browser csrf state only', async () => {
    vi.stubEnv('VITE_API_URL', '/api')
    vi.stubGlobal('window', {
      location: {
        origin: 'https://app.example',
      },
    })
    vi.stubGlobal('document', {
      cookie: '',
    })
    vi.stubGlobal('localStorage', {
      getItem: vi.fn(() => null),
      setItem: vi.fn(),
    })

    const { hasStoredH5SessionHint } = await import('../shared-client')

    expect(hasStoredH5SessionHint()).toBe(false)

    vi.stubGlobal('document', {
      cookie: 'foo=bar; csrf_token=test-csrf-token',
    })
    expect(hasStoredH5SessionHint()).toBe(true)

    vi.stubGlobal('plus', {})
    expect(hasStoredH5SessionHint()).toBe(false)
  })

  it('reuses and refreshes csrf tokens from storage for non-h5 builds', async () => {
    const request = vi.fn((options: any) => {
      options.success?.({
        statusCode: 200,
        data: {
          data: { ok: true },
        },
        header: {
          'X-CSRF-Token': 'rotated-csrf-token',
        },
      })
      return { abort: vi.fn() }
    })
    const getStorageSync = vi.fn(() => 'stored-csrf-token')
    const setStorageSync = vi.fn()

    vi.stubEnv('VITE_API_URL', 'https://api.example.com')
    vi.stubGlobal('uni', {
      request,
      getStorageSync,
      setStorageSync,
    })

    const { apiClient } = await import('../shared-client')
    const result = await apiClient.POST('/api/v1/review' as never, {
      body: {
        hello: 'world',
      },
    } as never)

    expect(request).toHaveBeenCalledTimes(1)
    expect(request.mock.calls[0][0]).toMatchObject({
      url: 'https://api.example.com/api/v1/review',
      method: 'POST',
      withCredentials: true,
      header: {
        'Content-Type': 'application/json',
        'X-CSRF-Token': 'stored-csrf-token',
      },
    })
    expect(setStorageSync).toHaveBeenCalledWith(
      'stuhelper:csrf-token',
      'rotated-csrf-token',
    )
    expect(result.response?.status).toBe(200)
  })

  it('fails closed when csrf token storage cannot be read in non-h5 builds', async () => {
    const request = vi.fn()

    vi.stubEnv('VITE_API_URL', 'https://api.example.com')
    vi.stubGlobal('uni', {
      request,
      getStorageSync: vi.fn(() => {
        throw new Error('storage unavailable')
      }),
      setStorageSync: vi.fn(),
    })

    const { apiClient } = await import('../shared-client')

    await expect(apiClient.POST('/api/v1/review' as never, {
      body: { hello: 'world' },
    } as never)).rejects.toThrow('failed to read csrf token')
    expect(request).not.toHaveBeenCalled()
  })

  it('fails closed when rotated csrf token cannot be persisted', async () => {
    const request = vi.fn((options: any) => {
      options.success?.({
        statusCode: 200,
        data: {
          data: { ok: true },
        },
        header: {
          'X-CSRF-Token': 'rotated-csrf-token',
        },
      })
      return { abort: vi.fn() }
    })

    vi.stubEnv('VITE_API_URL', 'https://api.example.com')
    vi.stubGlobal('uni', {
      request,
      getStorageSync: vi.fn(() => 'stored-csrf-token'),
      setStorageSync: vi.fn(() => {
        throw new Error('disk full')
      }),
    })

    const { apiClient } = await import('../shared-client')
    const result: any = await apiClient.POST('/api/v1/review' as never, {
      body: { hello: 'world' },
    } as never)

    const error: any = result.error
    if (
      !error
      || typeof error !== 'object'
      || !('message' in error)
      || typeof error.message !== 'string'
    ) {
      throw new Error('expected csrf storage persistence error')
    }
    expect(error.message).toContain('failed to persist csrf token')
    expect(result.response?.status).toBe(0)
  })

  it('hard-fails when a non-h5 build has only a relative api base', async () => {
    const request = vi.fn()
    vi.stubEnv('VITE_API_URL', '/api')
    vi.stubGlobal('uni', {
      request,
    })

    const { apiClient } = await import('../shared-client')

    await expect(apiClient.POST('/api/v1/review' as never, { body: {} } as never)).rejects.toThrow(
      'VITE_API_URL must be an absolute http(s) URL for non-H5 uniappx builds'
    )
    expect(request).not.toHaveBeenCalled()
  })

  it('refreshes session once and retries the original request after a 401', async () => {
    const request = vi.fn((options: any) => {
      if (options.url.endsWith('/api/v1/auth/refresh')) {
        options.success?.({
          statusCode: 200,
          data: { success: true },
          header: {},
        })
        return { abort: vi.fn() }
      }

      if (request.mock.calls.length === 1) {
        options.success?.({
          statusCode: 401,
          data: { error: { message: 'expired' } },
          header: {},
        })
        return { abort: vi.fn() }
      }

      options.success?.({
        statusCode: 200,
        data: { data: { ok: true } },
        header: {},
      })
      return { abort: vi.fn() }
    })

    vi.stubEnv('VITE_API_URL', 'https://api.example.com')
    vi.stubGlobal('uni', {
      request,
      getStorageSync: vi.fn(() => 'stored-csrf-token'),
      setStorageSync: vi.fn(),
    })

    const { apiClient } = await import('../shared-client')
    const result = await apiClient.GET('/api/v1/auth/me' as never)

    expect(request).toHaveBeenCalledTimes(3)
    expect(request.mock.calls[1][0]).toMatchObject({
      header: {
        'X-CSRF-Token': 'stored-csrf-token',
      },
      method: 'POST',
      url: 'https://api.example.com/api/v1/auth/refresh',
    })
    expect(result.response?.status).toBe(200)
    expect(result.data).toEqual({ data: { ok: true } })
  })

  it('returns refresh csrf failures without clearing the recoverable error', async () => {
    const csrfError = {
      error: {
        code: 'A0010202',
        message: 'csrf invalid',
      },
    }
    const request = vi.fn((options: any) => {
      if (options.url.endsWith('/api/v1/auth/refresh')) {
        options.success?.({
          statusCode: 403,
          data: csrfError,
          header: {
            'X-CSRF-Token': 'rotated-csrf-token',
          },
        })
        return { abort: vi.fn() }
      }

      options.success?.({
        statusCode: 401,
        data: { error: { code: 'A0010100', message: 'expired' } },
        header: {},
      })
      return { abort: vi.fn() }
    })
    const setStorageSync = vi.fn()

    vi.stubEnv('VITE_API_URL', 'https://api.example.com')
    vi.stubGlobal('uni', {
      request,
      getStorageSync: vi.fn(() => 'stored-csrf-token'),
      setStorageSync,
    })

    const { apiClient } = await import('../shared-client')
    const result = await apiClient.GET('/api/v1/auth/me' as never)

    expect(request).toHaveBeenCalledTimes(2)
    expect(request.mock.calls[1][0]).toMatchObject({
      header: {
        'X-CSRF-Token': 'stored-csrf-token',
      },
      method: 'POST',
      url: 'https://api.example.com/api/v1/auth/refresh',
    })
    expect(setStorageSync).toHaveBeenCalledWith(
      'stuhelper:csrf-token',
      'rotated-csrf-token',
    )
    expect(result.response?.status).toBe(403)
    expect(result.error).toEqual(csrfError)
  })

  it('does not refresh an anonymous 401 response without a session hint', async () => {
    const request = vi.fn((options: any) => {
      options.success?.({
        statusCode: 401,
        data: { error: { message: 'missing authentication token' } },
        header: {},
      })
      return { abort: vi.fn() }
    })

    vi.stubEnv('VITE_API_URL', 'https://api.example.com')
    vi.stubGlobal('uni', {
      request,
      getStorageSync: vi.fn(() => null),
      setStorageSync: vi.fn(),
    })

    const { apiClient } = await import('../shared-client')
    const result = await apiClient.GET('/api/v1/auth/me' as never)

    expect(request).toHaveBeenCalledTimes(1)
    expect(request.mock.calls[0][0].url).toBe('https://api.example.com/api/v1/auth/me')
    expect(result.response?.status).toBe(401)
  })

  it('injects native session header into refresh requests', async () => {
    const storedTokens = JSON.stringify({
      accessToken: 'native-access-token',
      refreshToken: 'native-refresh-token',
      sessionID: 'sid-native-refresh',
      expiresAt: Date.now() + 60_000,
    })
    const request = vi.fn((options: any) => {
      if (options.url.endsWith('/api/v1/auth/refresh')) {
        options.success?.({
          statusCode: 200,
          data: {
            data: {
              expiresIn: 600,
              accessToken: 'rotated-access-token',
              refreshToken: 'rotated-refresh-token',
            },
          },
          header: {},
        })
        return { abort: vi.fn() }
      }

      if (request.mock.calls.length === 1) {
        options.success?.({
          statusCode: 401,
          data: { error: { message: 'expired' } },
          header: {},
        })
        return { abort: vi.fn() }
      }

      options.success?.({
        statusCode: 200,
        data: { data: { ok: true } },
        header: {},
      })
      return { abort: vi.fn() }
    })

    let secureTokenPayload = storedTokens
    const secureStorage = {
      getItem: vi.fn(() => secureTokenPayload),
      removeItem: vi.fn(() => {
        secureTokenPayload = ''
      }),
      setItem: vi.fn((_service: string, _key: string, value: string) => {
        secureTokenPayload = value
      }),
    }

    vi.stubEnv('VITE_API_URL', 'https://api.example.com')
    vi.stubGlobal('plus', {})
    vi.stubGlobal('uni', {
      request,
      removeStorageSync: vi.fn(),
    })
    vi.stubGlobal('stuhelperSecureStorage', secureStorage)

    const { apiClient } = await import('../shared-client')
    const { NATIVE_SESSION_ID_HEADER } = await import('@stuhelper/shared/api')
    await apiClient.GET('/api/v1/auth/me' as never)

    expect(request).toHaveBeenCalledTimes(3)
    expect(request.mock.calls[1][0]).toMatchObject({
      data: {
        refreshToken: 'native-refresh-token',
      },
      header: {
        [NATIVE_SESSION_ID_HEADER]: 'sid-native-refresh',
      },
      method: 'POST',
      url: 'https://api.example.com/api/v1/auth/refresh',
    })
    expect(secureStorage.setItem).toHaveBeenCalledWith(
      'stuhelper.native-session',
      'stuhelper:native-tokens',
      expect.stringContaining('"refreshToken":"rotated-refresh-token"'),
    )
  })

  it('injects native session header into logout requests', async () => {
    const storedTokens = JSON.stringify({
      accessToken: 'native-access-token',
      refreshToken: 'native-refresh-token',
      sessionID: 'sid-native-logout',
      expiresAt: Date.now() + 60_000,
    })
    const request = vi.fn((options: any) => {
      options.success?.({
        statusCode: 200,
        data: { data: { message: 'logout successful' } },
        header: {},
      })
      return { abort: vi.fn() }
    })

    vi.stubEnv('VITE_API_URL', 'https://api.example.com')
    vi.stubGlobal('plus', {})
    vi.stubGlobal('uni', {
      request,
      removeStorageSync: vi.fn(),
    })
    vi.stubGlobal('stuhelperSecureStorage', {
      getItem: vi.fn(() => storedTokens),
      removeItem: vi.fn(),
      setItem: vi.fn(),
    })

    const { apiClient } = await import('../shared-client')
    const { createAuthApi, NATIVE_SESSION_ID_HEADER } = await import('@stuhelper/shared/api')
    const authApi = createAuthApi(apiClient)

    await authApi.logout()

    expect(request).toHaveBeenCalledTimes(1)
    expect(request.mock.calls[0][0]).toMatchObject({
      header: {
        Authorization: 'Bearer native-access-token',
        [NATIVE_SESSION_ID_HEADER]: 'sid-native-logout',
      },
      method: 'POST',
      url: 'https://api.example.com/api/v1/auth/logout',
    })
  })
})
