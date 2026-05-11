import { afterEach, describe, expect, it, vi } from 'vitest'

import { observabilityTestInternals } from '../observability'

describe('observability transport', () => {
  afterEach(() => {
    vi.restoreAllMocks()
    vi.unstubAllGlobals()
  })

  it('sends metrics with browser cookies and CSRF header', async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(null, { status: 204 }))
    vi.stubGlobal('document', { cookie: 'csrf_token=test-csrf' })
    vi.stubGlobal('window', { location: { origin: 'http://localhost:3000' } })
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
})
