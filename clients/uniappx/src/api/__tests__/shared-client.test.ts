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
    expect(request.mock.calls[1][0].url).toBe('https://api.example.com/api/v1/auth/refresh')
    expect(result.response?.status).toBe(200)
    expect(result.data).toEqual({ data: { ok: true } })
  })
})
