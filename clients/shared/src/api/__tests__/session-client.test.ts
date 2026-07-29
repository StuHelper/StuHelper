import { describe, expect, it, vi } from 'vitest'

import {
  createSessionApiClient,
  executeSessionRefresh,
  extractRefreshSessionData,
  normalizeSchemaPath,
  normalizeRequestHeaders,
  serializePath,
} from '../session-client'

describe('extractRefreshSessionData', () => {
  it('returns typed refresh payload when expiresIn is present', () => {
    expect(
      extractRefreshSessionData({
        data: {
          data: {
            expiresIn: 900,
            accessToken: 'access-token',
            refreshToken: 'refresh-token',
          },
        },
        response: { status: 200 },
      }),
    ).toEqual({
      expiresIn: 900,
      accessToken: 'access-token',
      refreshToken: 'refresh-token',
    })
  })

  it('returns null when payload is missing refresh metadata', () => {
    const missingMetadataResult =
      {
        data: { data: { message: 'ok' } },
        response: { status: 200 },
      } as unknown as Parameters<typeof extractRefreshSessionData>[0]

    expect(
      extractRefreshSessionData(missingMetadataResult),
    ).toBeNull()
  })
})

describe('executeSessionRefresh', () => {
  it('normalizes successful refresh and calls onSuccess with payload', async () => {
    const request = vi.fn().mockResolvedValue({
      data: {
        data: {
          expiresIn: 600,
          accessToken: 'new-access',
        },
      },
      response: { status: 200 },
    })
    const onSuccess = vi.fn()

    await expect(
      executeSessionRefresh({
        body: { refreshToken: 'refresh-token' },
        onSuccess,
        request,
      }),
    ).resolves.toEqual({ kind: 'ok' })

    expect(request).toHaveBeenCalledWith({
      body: { refreshToken: 'refresh-token' },
    })
    expect(onSuccess).toHaveBeenCalledWith({
      expiresIn: 600,
      accessToken: 'new-access',
    })
  })

  it('rejects successful refresh responses without required metadata', async () => {
    const onSuccess = vi.fn()

    const result = await executeSessionRefresh({
      onSuccess,
      request: vi.fn().mockResolvedValue({
        data: {
          data: {
            message: 'token refreshed successfully',
          },
        },
        response: { status: 200 },
      }),
    })

    expect(result).toMatchObject({
      kind: 'error',
      status: 200,
    })
    expect(result.kind === 'error' ? result.error : undefined).toBeInstanceOf(Error)
    expect(result.kind === 'error' ? (result.error as Error).message : '').toBe(
      'invalid refresh response',
    )
    expect(onSuccess).not.toHaveBeenCalled()
  })

  it('maps 401 responses to unauthorized', async () => {
    const unauthorized401 = await executeSessionRefresh({
      request: vi.fn().mockResolvedValue({
        error: { code: 'UNAUTHORIZED' },
        response: { status: 401 },
      }),
    })

    expect(unauthorized401).toEqual({ kind: 'unauthorized', status: 401 })
  })

  it('keeps refresh 403 failures visible unless explicitly configured as unauthorized', async () => {
    const forbidden = await executeSessionRefresh({
      request: vi.fn().mockResolvedValue({
        error: { code: 'CSRF_INVALID' },
        response: { status: 403 },
      }),
    })
    const configuredUnauthorized = await executeSessionRefresh({
      request: vi.fn().mockResolvedValue({
        error: { code: 'TOKEN_EXPIRED' },
        response: { status: 403 },
      }),
      unauthorizedStatuses: [401, 403],
    })

    expect(forbidden).toEqual({
      kind: 'error',
      error: { code: 'CSRF_INVALID' },
      status: 403,
    })
    expect(configuredUnauthorized).toEqual({
      kind: 'unauthorized',
      status: 403,
    })
  })

  it('returns error result for non-auth refresh failures', async () => {
    const error = await executeSessionRefresh({
      request: vi.fn().mockResolvedValue({
        error: { code: 'INTERNAL_ERROR' },
        response: { status: 500 },
      }),
    })

    expect(error).toEqual({
      kind: 'error',
      error: { code: 'INTERNAL_ERROR' },
      status: 500,
    })
  })

  it('captures thrown transport errors', async () => {
    const networkError = new Error('network down')
    await expect(
      executeSessionRefresh({
        request: vi.fn().mockRejectedValue(networkError),
      }),
    ).resolves.toEqual({
      kind: 'error',
      error: networkError,
    })
  })
})

describe('normalizeRequestHeaders', () => {
  it('merges top-level headers with params headers', () => {
    expect(
      normalizeRequestHeaders({
        headers: {
          Authorization: 'Bearer top-level-token',
          'X-Null': null,
        },
        params: {
          header: {
            Authorization: 'Bearer params-token',
            'X-Trace-ID': 'trace-1',
          },
        },
      }),
    ).toEqual({
      Authorization: 'Bearer params-token',
      'X-Trace-ID': 'trace-1',
    })
  })

  it('accepts Headers and tuple header inputs', () => {
    expect(
      normalizeRequestHeaders({
        headers: new Headers([['Authorization', 'Bearer top-level-token']]),
        params: {
          header: [['X-Trace-ID', 'trace-1']],
        },
      }),
    ).toEqual({
      authorization: 'Bearer top-level-token',
      'X-Trace-ID': 'trace-1',
    })
  })
})

describe('normalizeSchemaPath', () => {
  it('removes duplicated API version path when base URL already includes it', () => {
    expect(
      normalizeSchemaPath('https://example.com/api/v1', '/api/v1/user/me'),
    ).toBe('/user/me')
  })

  it('removes duplicated API base path when base URL is /api', () => {
    expect(
      normalizeSchemaPath('https://example.com/api', '/api/v1/user/me'),
    ).toBe('/v1/user/me')
  })

  it('trims long trailing slash runs without a regular expression', () => {
    expect(
      normalizeSchemaPath(`https://example.com/api/v1${'/'.repeat(10_000)}`, '/api/v1/user/me'),
    ).toBe('/user/me')
  })
})

describe('serializePath', () => {
  it('serializes repeated placeholders and preserves unmatched braces', () => {
    expect(
      serializePath('/schools/{school}/users/{user}', {
        school: '北航/沙河',
        user: 42,
      }),
    ).toBe('/schools/%E5%8C%97%E8%88%AA%2F%E6%B2%99%E6%B2%B3/users/42')
    expect(serializePath('/literal/{}', {})).toBe('/literal/{}')
    expect(serializePath('/literal/{missing', {})).toBe('/literal/{missing')
  })

  it('handles many placeholders with bounded linear scanning', () => {
    const schemaPath = Array.from({ length: 2_000 }, () => '/{id}').join('')
    expect(serializePath(schemaPath, { id: 'x' })).toBe('/x'.repeat(2_000))
  })
})

describe('createSessionApiClient', () => {
  it('reauthenticates when refresh returns unauthorized', async () => {
    const onUnauthorized = vi.fn()
    const request = vi
      .fn()
      .mockResolvedValueOnce({
        error: { code: 'TOKEN_EXPIRED' },
        response: { status: 401 },
      })
      .mockResolvedValueOnce({
        data: { data: { message: 'csrf invalid' } },
        response: { status: 401 },
      })

    const client = createSessionApiClient(
      {
        onUnauthorized,
        refresh: vi.fn().mockResolvedValue({ kind: 'unauthorized', status: 401 }),
        request,
      },
      { reauthenticateOnUnauthorized: true },
    )

    await client.GET('/api/v1/user/me')

    expect(onUnauthorized).toHaveBeenCalledTimes(1)
    expect(request).toHaveBeenCalledTimes(1)
  })

  it('does not reauthenticate on direct 403 responses', async () => {
    const onUnauthorized = vi.fn()
    const request = vi.fn().mockResolvedValue({
      error: { code: 'FORBIDDEN' },
      response: { status: 403 },
    })

    const client = createSessionApiClient(
      {
        onUnauthorized,
        refresh: vi.fn(),
        request,
        shouldRefresh: () => false,
      },
      { reauthenticateOnUnauthorized: true },
    )

    await client.GET('/api/v1/admin/resource' as never, undefined as never)

    expect(onUnauthorized).not.toHaveBeenCalled()
  })

  it('does not expose semantic refresh failures as HTTP 200 request results', async () => {
    const request = vi.fn().mockResolvedValue({
      error: { code: 'TOKEN_EXPIRED' },
      response: { status: 401 },
    })

    const client = createSessionApiClient({
      refresh: vi.fn().mockResolvedValue({
        error: new Error('invalid refresh response'),
        kind: 'error',
        status: 200,
      }),
      request,
    })

    const result = await client.GET('/api/v1/user/me' as never)

    expect(result.error).toBeInstanceOf(Error)
    expect(result.response?.status).toBe(401)
    expect(request).toHaveBeenCalledTimes(1)
  })

  it('preserves refresh failure statuses when they are real HTTP errors', async () => {
    const client = createSessionApiClient({
      refresh: vi.fn().mockResolvedValue({
        error: { code: 'SERVICE_UNAVAILABLE' },
        kind: 'error',
        status: 503,
      }),
      request: vi.fn().mockResolvedValue({
        error: { code: 'TOKEN_EXPIRED' },
        response: { status: 401 },
      }),
    })

    const result = await client.GET('/api/v1/user/me' as never)

    expect(result.error).toEqual({ code: 'SERVICE_UNAVAILABLE' })
    expect(result.response?.status).toBe(503)
  })
})
