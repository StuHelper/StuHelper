// @vitest-environment jsdom

import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import {
  resolvePostLoginRedirectTarget,
  sanitizePostLoginRedirect,
} from '../redirect'

describe('post-login redirect helpers', () => {
  beforeEach(() => {
    vi.stubEnv('VITE_IDENTITY_URL', window.location.origin)
    vi.stubEnv('VITE_WEB_URL', 'http://stuhelper.com')
  })

  afterEach(() => {
    vi.unstubAllEnvs()
  })

  it('allows the configured web origin from the identity login page', () => {
    expect(sanitizePostLoginRedirect('http://stuhelper.com/courses?tab=latest')).toBe(
      'http://stuhelper.com/courses?tab=latest',
    )
    expect(resolvePostLoginRedirectTarget('http://stuhelper.com/courses?tab=latest')).toBe(
      'http://stuhelper.com/courses?tab=latest',
    )
  })

  it('allows identity-local relative redirects', () => {
    expect(sanitizePostLoginRedirect('/developers/apps')).toBe('/developers/apps')
    expect(resolvePostLoginRedirectTarget('/developers/apps')).toBe(
      `${window.location.origin}/developers/apps`,
    )
  })

  it('rejects external and scheme-relative redirects', () => {
    expect(sanitizePostLoginRedirect('https://evil.example/path')).toBeUndefined()
    expect(sanitizePostLoginRedirect('//evil.example/path')).toBeUndefined()
  })
})
