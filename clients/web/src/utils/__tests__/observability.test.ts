import { afterEach, describe, expect, it, vi } from 'vitest'

import { observabilityTestInternals } from '../observability'

describe('observability transport', () => {
  afterEach(() => {
    vi.restoreAllMocks()
    vi.unstubAllGlobals()
  })

  it('prefers navigator.sendBeacon for metrics payloads', async () => {
    const fetchMock = vi.fn()
    const sendBeaconMock = vi.fn().mockReturnValue(true)
    vi.stubGlobal('document', { cookie: '' })
    vi.stubGlobal('window', { location: { origin: 'http://localhost:3000' } })
    vi.stubGlobal('navigator', { sendBeacon: sendBeaconMock })
    vi.stubGlobal('fetch', fetchMock)

    observabilityTestInternals?.sendBeaconJSON('/api/v1/metrics/vitals', {
      name: 'LCP',
      value: 42,
      rating: 'good',
    })

    expect(sendBeaconMock).toHaveBeenCalledTimes(1)
    const [url, body] = sendBeaconMock.mock.calls[0] as [string, Blob]
    expect(url).toBe('http://localhost:3000/api/v1/metrics/vitals')
    expect(body.type).toBe('application/json')
    await expect(body.text()).resolves.toBe(
      JSON.stringify({ name: 'LCP', value: 42, rating: 'good' }),
    )
    expect(fetchMock).not.toHaveBeenCalled()
  })

  it('falls back to keepalive fetch with browser cookies and CSRF header', async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(null, { status: 204 }))
    vi.stubGlobal('document', { cookie: 'csrf_token=test-csrf' })
    vi.stubGlobal('window', { location: { origin: 'http://localhost:3000' } })
    vi.stubGlobal('navigator', {})
    vi.stubGlobal('fetch', fetchMock)

    observabilityTestInternals?.sendBeaconJSON('/api/v1/metrics/vitals', {
      name: 'LCP',
      value: 42,
    })
    await Promise.resolve()

    expect(fetchMock).toHaveBeenCalledWith(
      'http://localhost:3000/api/v1/metrics/vitals',
      expect.objectContaining({
        credentials: 'include',
        headers: {
          'Content-Type': 'application/json',
          'X-CSRF-Token': 'test-csrf',
        },
        keepalive: true,
        method: 'POST',
      }),
    )
  })

  it('falls back to keepalive fetch when sendBeacon rejects the payload', async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(null, { status: 204 }))
    const sendBeaconMock = vi.fn().mockReturnValue(false)
    vi.stubGlobal('document', { cookie: '' })
    vi.stubGlobal('window', { location: { origin: 'http://localhost:3000' } })
    vi.stubGlobal('navigator', { sendBeacon: sendBeaconMock })
    vi.stubGlobal('fetch', fetchMock)

    observabilityTestInternals?.sendBeaconJSON('/api/v1/metrics/vitals', {
      name: 'TTFB',
      value: 12,
      rating: 'good',
    })
    await Promise.resolve()

    expect(sendBeaconMock).toHaveBeenCalledTimes(1)
    expect(fetchMock).toHaveBeenCalledWith(
      'http://localhost:3000/api/v1/metrics/vitals',
      expect.objectContaining({
        credentials: 'include',
        headers: {
          'Content-Type': 'application/json',
        },
        keepalive: true,
        method: 'POST',
      }),
    )
  })
})
