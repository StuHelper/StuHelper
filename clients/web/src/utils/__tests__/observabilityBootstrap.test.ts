import { describe, expect, it } from 'vitest'

import { shouldInitObservability } from '../observabilityBootstrap'

describe('shouldInitObservability', () => {
  it('skips observability when the e2e API stub is enabled', () => {
    expect(
      shouldInitObservability({
        e2eApiStub: '1',
        hostname: 'localhost',
      }),
    ).toBe(false)
  })

  it('skips unconfigured join admission hosts', () => {
    expect(
      shouldInitObservability({
        apiBaseUrl: '',
        hostname: 'join.localhost',
      }),
    ).toBe(false)
  })

  it('keeps observability on configured join admission hosts', () => {
    expect(
      shouldInitObservability({
        apiBaseUrl: 'https://api.example.test',
        hostname: 'join.stuhelper.com',
      }),
    ).toBe(true)
  })

  it('keeps the existing main-site fallback behavior', () => {
    expect(
      shouldInitObservability({
        apiBaseUrl: '',
        hostname: 'localhost',
      }),
    ).toBe(true)
  })
})
