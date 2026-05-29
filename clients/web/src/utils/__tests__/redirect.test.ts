// @vitest-environment jsdom

import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import {
  absoluteURLOnPreferredOrigin,
  identityPortalURL,
  identityPortalURLForHref,
  isIdentityPortalPath,
  normalizeConfiguredHTTPOrigin,
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
      'http://localhost:3000/developers/apps',
    )
  })

  it('rejects external and scheme-relative redirects', () => {
    expect(sanitizePostLoginRedirect('https://evil.example/path')).toBeUndefined()
    expect(sanitizePostLoginRedirect('//evil.example/path')).toBeUndefined()
  })

  it('keeps configured HTTP origins on HTTP pages', () => {
    expect(
      normalizeConfiguredHTTPOrigin('http://id.stuhelper.com', 'http://stuhelper.com'),
    ).toBe('http://id.stuhelper.com')
    expect(
      normalizeConfiguredHTTPOrigin('http://stuhelper.com', 'http://id.stuhelper.com'),
    ).toBe('http://stuhelper.com')
  })

  it('does not downgrade configured origins from an HTTPS page', () => {
    expect(
      normalizeConfiguredHTTPOrigin('http://id.stuhelper.com', 'https://stuhelper.com'),
    ).toBe('https://id.stuhelper.com')
    expect(
      normalizeConfiguredHTTPOrigin('http://stuhelper.com', 'https://id.stuhelper.com'),
    ).toBe('https://stuhelper.com')
  })

  it('builds login return targets on the configured web origin before entering the identity host', () => {
    expect(
      absoluteURLOnPreferredOrigin('/courses?tab=latest', 'http://stuhelper.com'),
    ).toBe('http://stuhelper.com/courses?tab=latest')
  })

  it('moves current-origin return targets onto the configured web origin', () => {
    vi.stubEnv('VITE_IDENTITY_URL', 'http://id.stuhelper.com')

    expect(
      absoluteURLOnPreferredOrigin(
        `${window.location.origin}/courses?tab=latest`,
        'http://stuhelper.com',
      ),
    ).toBe('http://stuhelper.com/courses?tab=latest')
    expect(
      resolvePostLoginRedirectTarget(
        `${window.location.origin}/courses?tab=latest`,
      ),
    ).toBe('http://stuhelper.com/courses?tab=latest')
  })

  it('falls back to the current origin when no preferred origin is configured', () => {
    expect(absoluteURLOnPreferredOrigin('/identity', null)).toBe(
      `${window.location.origin}/identity`,
    )
  })

  it('builds identity portal URLs on the configured identity origin', () => {
    vi.stubEnv('VITE_IDENTITY_URL', 'http://id.stuhelper.com')

    expect(identityPortalURL('/user/student-verification')).toBe(
      'http://id.stuhelper.com/user/student-verification',
    )
  })

  it('detects identity portal paths', () => {
    expect(isIdentityPortalPath('/identity')).toBe(true)
    expect(isIdentityPortalPath('/account/profile')).toBe(true)
    expect(isIdentityPortalPath('/developers/apps')).toBe(true)
    expect(isIdentityPortalPath('/user/identity-verification')).toBe(true)
    expect(isIdentityPortalPath('/user/student-verification')).toBe(true)
    expect(isIdentityPortalPath('/courses')).toBe(false)
    expect(isIdentityPortalPath('/user/reviews')).toBe(false)
  })

  it('resolves relative identity hrefs to the configured identity origin', () => {
    vi.stubEnv('VITE_IDENTITY_URL', 'https://id.stuhelper.com')

    expect(identityPortalURLForHref('/user/student-verification?next=1#form')).toBe(
      'https://id.stuhelper.com/user/student-verification?next=1#form',
    )
    expect(identityPortalURLForHref('/courses/1/reviews')).toBeNull()
    expect(identityPortalURLForHref('https://evil.example/user/student-verification')).toBeNull()
  })

  it('resolves identity post-login redirects to the identity origin', () => {
    vi.stubEnv('VITE_WEB_URL', 'https://stuhelper.com')
    vi.stubEnv('VITE_IDENTITY_URL', 'https://id.stuhelper.com')

    expect(resolvePostLoginRedirectTarget('/developers/apps')).toBe(
      'https://id.stuhelper.com/developers/apps',
    )
    expect(resolvePostLoginRedirectTarget('/user/authorized-apps')).toBe(
      'https://id.stuhelper.com/user/authorized-apps',
    )
    expect(resolvePostLoginRedirectTarget('/courses')).toBe(
      'https://stuhelper.com/courses',
    )
  })
})
