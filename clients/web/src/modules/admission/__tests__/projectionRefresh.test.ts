import { describe, expect, it, vi } from 'vitest'

import { ApiError } from '@/api/errors'

import { waitForAdmissionProjection } from '../projectionRefresh'

describe('waitForAdmissionProjection', () => {
  it('refreshes auth state with five bounded exponential-backoff attempts', async () => {
    const waits: number[] = []
    const refreshAuth = vi.fn().mockResolvedValue({ capabilities: [] })

    const ready = await waitForAdmissionProjection({
      hasProjectedCapability: (user) =>
        user.capabilities?.includes('review:create') === true,
      refreshAuth,
      wait: async (delay) => {
        waits.push(delay)
      },
    })

    expect(ready).toBe(false)
    expect(refreshAuth).toHaveBeenCalledTimes(5)
    expect(waits).toEqual([1000, 2000, 4000, 8000, 16_000])
  })

  it('stops polling as soon as the projected capability is visible', async () => {
    const waits: number[] = []
    const refreshAuth = vi
      .fn()
      .mockResolvedValueOnce({ capabilities: [] })
      .mockResolvedValueOnce({ capabilities: ['review:create'] })

    const ready = await waitForAdmissionProjection({
      hasProjectedCapability: (user) =>
        user.capabilities?.includes('review:create') === true,
      refreshAuth,
      wait: async (delay) => {
        waits.push(delay)
      },
    })

    expect(ready).toBe(true)
    expect(refreshAuth).toHaveBeenCalledTimes(2)
    expect(waits).toEqual([1000, 2000])
  })

  it('continues bounded polling after a transient auth refresh failure', async () => {
    const waits: number[] = []
    const refreshAuth = vi
      .fn()
      .mockRejectedValueOnce(new ApiError({
        code: 'B0000001',
        message: 'temporarily unavailable',
        status: 503,
      }))
      .mockResolvedValueOnce({ capabilities: ['review:create'] })

    const ready = await waitForAdmissionProjection({
      refreshAuth,
      wait: async (delay) => {
        waits.push(delay)
      },
    })

    expect(ready).toBe(true)
    expect(refreshAuth).toHaveBeenCalledTimes(2)
    expect(waits).toEqual([1000, 2000])
  })

  it('returns a recoverable timeout after all transient refresh attempts fail', async () => {
    const refreshAuth = vi.fn().mockRejectedValue(new ApiError({
      code: 'NETWORK_ERROR',
      message: 'network unavailable',
      status: 0,
    }))

    await expect(waitForAdmissionProjection({
      refreshAuth,
      wait: async () => {},
    })).resolves.toBe(false)
    expect(refreshAuth).toHaveBeenCalledTimes(5)
  })

  it('stops immediately when auth/me explicitly rejects the session', async () => {
    const refreshAuth = vi.fn().mockRejectedValue(new ApiError({
      code: 'A0010001',
      message: 'unauthorized',
      status: 401,
    }))

    await expect(waitForAdmissionProjection({
      refreshAuth,
      wait: async () => {},
    })).rejects.toMatchObject({ status: 401 })
    expect(refreshAuth).toHaveBeenCalledTimes(1)
  })

  it('does not swallow an abort raised by the auth refresh request', async () => {
    const refreshAuth = vi.fn().mockRejectedValue(
      new DOMException('request aborted', 'AbortError'),
    )

    await expect(waitForAdmissionProjection({
      refreshAuth,
      wait: async () => {},
    })).rejects.toMatchObject({ name: 'AbortError' })
    expect(refreshAuth).toHaveBeenCalledTimes(1)
  })

  it('aborts before the next projection refresh request', async () => {
    const controller = new AbortController()
    const refreshAuth = vi.fn()
    controller.abort()

    await expect(waitForAdmissionProjection({
      refreshAuth,
      signal: controller.signal,
      wait: async () => {},
    })).rejects.toMatchObject({ name: 'AbortError' })
    expect(refreshAuth).not.toHaveBeenCalled()
  })
})
