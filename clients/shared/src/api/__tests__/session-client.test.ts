import { describe, expect, it, vi } from 'vitest'

import {
  executeSessionRefresh,
  extractRefreshSessionData,
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

  it('maps 401/403 responses to unauthorized', async () => {
    const unauthorized401 = await executeSessionRefresh({
      request: vi.fn().mockResolvedValue({
        error: { code: 'UNAUTHORIZED' },
        response: { status: 401 },
      }),
    })
    const unauthorized403 = await executeSessionRefresh({
      request: vi.fn().mockResolvedValue({
        error: { code: 'CSRF_INVALID' },
        response: { status: 403 },
      }),
    })

    expect(unauthorized401).toEqual({ kind: 'unauthorized', status: 401 })
    expect(unauthorized403).toEqual({ kind: 'unauthorized', status: 403 })
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
