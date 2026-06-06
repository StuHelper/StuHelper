import { afterEach, describe, expect, it, vi } from 'vitest'

async function loadObservability() {
  vi.resetModules()
  const { observabilityTestInternals } = await import('../observability')
  const sendBeaconJSON = observabilityTestInternals?.sendBeaconJSON
  expect(sendBeaconJSON).toBeTypeOf('function')
  return sendBeaconJSON as NonNullable<typeof sendBeaconJSON>
}

function stubBrowserGlobals(options: {
  cookie?: string
  fetch?: ReturnType<typeof vi.fn>
  origin?: string
  sendBeacon?: ReturnType<typeof vi.fn>
} = {}) {
  const documentStub = {
    cookie: options.cookie ?? '',
    createElement: vi.fn((tagName: string) => ({
      tagName: tagName.toUpperCase(),
      style: {},
      setAttribute: vi.fn(),
      removeAttribute: vi.fn(),
    })),
    createElementNS: vi.fn((_namespace: string, tagName: string) => ({
      tagName: tagName.toUpperCase(),
      style: {},
      setAttribute: vi.fn(),
      removeAttribute: vi.fn(),
    })),
  }
  const windowStub = {
    document: documentStub,
    location: { origin: options.origin ?? 'http://localhost:3000' },
    clearTimeout,
    setTimeout,
  }

  vi.stubGlobal('document', documentStub)
  vi.stubGlobal('window', windowStub)
  vi.stubGlobal('navigator', {
    onLine: true,
    ...(options.sendBeacon ? { sendBeacon: options.sendBeacon } : {}),
  })
  vi.stubGlobal('fetch', options.fetch ?? vi.fn())
}

describe('observability transport', () => {
  afterEach(() => {
    vi.restoreAllMocks()
    vi.unstubAllGlobals()
    vi.unstubAllEnvs()
  })

  it('prefers navigator.sendBeacon for metrics payloads', async () => {
    const fetchMock = vi.fn()
    const sendBeaconMock = vi.fn().mockReturnValue(true)
    stubBrowserGlobals({ fetch: fetchMock, sendBeacon: sendBeaconMock })
    const sendBeaconJSON = await loadObservability()

    sendBeaconJSON('/api/v1/metrics/vitals', {
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

  it('resolves metrics URLs with the configured API origin', async () => {
    const fetchMock = vi.fn()
    const sendBeaconMock = vi.fn().mockReturnValue(true)
    vi.stubEnv('VITE_API_URL', 'https://api.example.test/api')
    stubBrowserGlobals({ fetch: fetchMock, sendBeacon: sendBeaconMock })
    const sendBeaconJSON = await loadObservability()

    sendBeaconJSON('/api/v1/metrics/vitals', { name: 'CLS', value: 0 })

    expect(sendBeaconMock).toHaveBeenCalledTimes(1)
    expect(sendBeaconMock.mock.calls[0]?.[0]).toBe(
      'https://api.example.test/api/v1/metrics/vitals',
    )
    expect(fetchMock).not.toHaveBeenCalled()
  })

  it('falls back to keepalive fetch with browser cookies and CSRF header', async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(null, { status: 204 }))
    stubBrowserGlobals({
      cookie: 'csrf_token=test-csrf',
      fetch: fetchMock,
    })
    const sendBeaconJSON = await loadObservability()

    sendBeaconJSON('/api/v1/metrics/vitals', {
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
    stubBrowserGlobals({ fetch: fetchMock, sendBeacon: sendBeaconMock })
    const sendBeaconJSON = await loadObservability()

    sendBeaconJSON('/api/v1/metrics/vitals', {
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
