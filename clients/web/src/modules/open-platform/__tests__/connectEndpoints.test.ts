import { describe, expect, it } from 'vitest'

import enUSDeveloper from '@/i18n/locales/en-US/developer'
import zhCNDeveloper from '@/i18n/locales/zh-CN/developer'
import { DEFAULT_IDENTITY_ISSUER, buildConnectEndpoints, normalizeIdentityIssuer } from '../connectEndpoints'

describe('StuHelper ID Connect endpoint helpers', () => {
  it('normalizes configured identity origins to issuer origins', () => {
    expect(normalizeIdentityIssuer('https://id.stuhelper.com/path')).toBe('https://id.stuhelper.com')
    expect(normalizeIdentityIssuer(' http://id.stuhelper.com ')).toBe('http://id.stuhelper.com')
  })

  it('falls back to the current origin and then the production identity issuer', () => {
    expect(normalizeIdentityIssuer('', 'https://local-id.stuhelper.test/identity')).toBe(
      'https://local-id.stuhelper.test',
    )
    expect(normalizeIdentityIssuer('not a url', 'also bad')).toBe(DEFAULT_IDENTITY_ISSUER)
  })

  it('builds the public OIDC and OAuth endpoint baseline from the issuer', () => {
    expect(buildConnectEndpoints('https://id.stuhelper.com')).toEqual([
      { key: 'issuer', value: 'https://id.stuhelper.com' },
      {
        key: 'discovery',
        value: 'https://id.stuhelper.com/.well-known/openid-configuration',
      },
      {
        key: 'authorization',
        value: 'https://id.stuhelper.com/oauth2/authorize',
      },
      { key: 'token', value: 'https://id.stuhelper.com/oauth2/token' },
      {
        key: 'userinfo',
        value: 'https://id.stuhelper.com/oidc/userinfo',
      },
      {
        key: 'jwks',
        value: 'https://id.stuhelper.com/.well-known/jwks.json',
      },
      {
        key: 'introspection',
        value: 'https://id.stuhelper.com/oauth2/introspect',
      },
      {
        key: 'revocation',
        value: 'https://id.stuhelper.com/oauth2/revoke',
      },
      { key: 'logout', value: 'https://id.stuhelper.com/oauth2/logout' },
    ])
  })

  it('keeps every Connect endpoint label present in Chinese and English locales', () => {
    for (const endpoint of buildConnectEndpoints('https://id.stuhelper.com')) {
      expect(zhCNDeveloper.apps.connect.endpoints[endpoint.key], `zh-CN ${endpoint.key}`).toEqual(expect.any(String))
      expect(enUSDeveloper.apps.connect.endpoints[endpoint.key], `en-US ${endpoint.key}`).toEqual(expect.any(String))
    }
  })

  it('keeps the public Connect copy centered on the identity issuer only', () => {
    const publicCopy = [
      zhCNDeveloper.connect.subtitle,
      zhCNDeveloper.apps.connect.subtitle,
      enUSDeveloper.connect.subtitle,
      enUSDeveloper.apps.connect.subtitle,
    ].join('\n')

    expect(publicCopy).toContain('id.stuhelper.com')
    expect(publicCopy).not.toContain('sso.stuhelper.com')
    expect(publicCopy).not.toContain('StuHelper SSO')
  })
})
