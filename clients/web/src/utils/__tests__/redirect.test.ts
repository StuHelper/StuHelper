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
    vi.stubEnv('VITE_WEB_URL', 'http://stuhelper.com')
  })

  afterEach(() => {
    vi.unstubAllEnvs()
  })

  it('allows the configured web origin from the login page', () => {
    expect(sanitizePostLoginRedirect('http://stuhelper.com/courses?tab=latest')).toBe(
      'http://stuhelper.com/courses?tab=latest',
    )
    expect(resolvePostLoginRedirectTarget('http://stuhelper.com/courses?tab=latest')).toBe(
      'http://stuhelper.com/courses?tab=latest',
    )
  })

  it('moves account relative redirects to the configured web origin', () => {
    expect(sanitizePostLoginRedirect('/developers/apps')).toBe('/developers/apps')
    expect(resolvePostLoginRedirectTarget('/developers/apps')).toBe(
      'http://stuhelper.com/developers/apps',
    )
  })

  it('rejects external and scheme-relative redirects', () => {
    expect(sanitizePostLoginRedirect('https://evil.example/path')).toBeUndefined()
    expect(sanitizePostLoginRedirect('//evil.example/path')).toBeUndefined()
  })

  it('keeps configured HTTP origins on HTTP pages', () => {
    expect(
      normalizeConfiguredHTTPOrigin('http://sso.stuhelper.com', 'http://stuhelper.com'),
    ).toBe('http://sso.stuhelper.com')
    expect(
      normalizeConfiguredHTTPOrigin('http://stuhelper.com', 'http://sso.stuhelper.com'),
    ).toBe('http://stuhelper.com')
  })

  it('does not downgrade configured origins from an HTTPS page', () => {
    expect(
      normalizeConfiguredHTTPOrigin('http://sso.stuhelper.com', 'https://stuhelper.com'),
    ).toBe('https://sso.stuhelper.com')
    expect(
      normalizeConfiguredHTTPOrigin('http://stuhelper.com', 'https://sso.stuhelper.com'),
    ).toBe('https://stuhelper.com')
  })

  it('builds login return targets on the configured web origin before entering the identity host', () => {
    expect(
      absoluteURLOnPreferredOrigin('/courses?tab=latest', 'http://stuhelper.com'),
    ).toBe('http://stuhelper.com/courses?tab=latest')
  })

  it('moves current-origin return targets onto the configured web origin', () => {
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

  it('keeps admission verification redirects on the current business origin', () => {
    vi.stubEnv('VITE_WEB_URL', 'https://stuhelper.com')

    expect(
      resolvePostLoginRedirectTarget(
        `${window.location.origin}/verify/ADMIT-LOGIN?qq=123456`,
      ),
    ).toBe(`${window.location.origin}/verify/ADMIT-LOGIN?qq=123456`)
  })

  it('falls back to the current origin when no preferred origin is configured', () => {
    expect(absoluteURLOnPreferredOrigin('/identity', null)).toBe(
      `${window.location.origin}/identity`,
    )
  })

  it('builds account URLs on the configured web origin', () => {
    expect(identityPortalURL('/user/student-verification')).toBe(
      'http://stuhelper.com/user/student-verification',
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

  it('resolves relative account hrefs to the configured web origin', () => {
    vi.stubEnv('VITE_WEB_URL', 'https://stuhelper.com')

    expect(identityPortalURLForHref('/user/student-verification?next=1#form')).toBe(
      'https://stuhelper.com/user/student-verification?next=1#form',
    )
    expect(identityPortalURLForHref('/courses/1/reviews')).toBeNull()
    expect(identityPortalURLForHref('https://evil.example/user/student-verification')).toBeNull()
  })

  it('resolves account post-login redirects to the web origin', () => {
    vi.stubEnv('VITE_WEB_URL', 'https://stuhelper.com')

    expect(resolvePostLoginRedirectTarget('/developers/apps')).toBe(
      'https://stuhelper.com/developers/apps',
    )
    expect(resolvePostLoginRedirectTarget('/user/authorized-apps')).toBe(
      'https://stuhelper.com/user/authorized-apps',
    )
    expect(resolvePostLoginRedirectTarget('/courses')).toBe(
      'https://stuhelper.com/courses',
    )
  })
})
