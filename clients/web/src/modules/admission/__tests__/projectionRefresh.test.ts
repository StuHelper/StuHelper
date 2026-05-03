import { describe, expect, it, vi } from 'vitest'

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
})
