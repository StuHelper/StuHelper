import { describe, expect, it } from 'vitest'

import enUSDeveloper from '@/i18n/locales/en-US/developer'
import zhCNDeveloper from '@/i18n/locales/zh-CN/developer'
import { DEFAULT_SSO_ISSUER, buildConnectEndpoints, normalizeSsoIssuer } from '../connectEndpoints'

describe('StuHelper Connect endpoint helpers', () => {
  it('normalizes configured SSO origins to issuer origins', () => {
    expect(normalizeSsoIssuer('https://sso.stuhelper.com/path')).toBe('https://sso.stuhelper.com')
    expect(normalizeSsoIssuer(' http://sso.stuhelper.com ')).toBe('http://sso.stuhelper.com')
  })

  it('falls back to the current origin and then the production SSO issuer', () => {
    expect(normalizeSsoIssuer('', 'https://local-sso.stuhelper.test/login')).toBe(
      'https://local-sso.stuhelper.test',
    )
    expect(normalizeSsoIssuer('not a url', 'also bad')).toBe(DEFAULT_SSO_ISSUER)
  })

  it('builds the public OIDC and OAuth endpoint baseline from the issuer', () => {
    expect(buildConnectEndpoints('https://sso.stuhelper.com')).toEqual([
      { key: 'issuer', value: 'https://sso.stuhelper.com' },
      {
        key: 'discovery',
        value: 'https://sso.stuhelper.com/.well-known/openid-configuration',
      },
      {
        key: 'authorization',
        value: 'https://sso.stuhelper.com/login/oauth/authorize',
      },
      { key: 'token', value: 'https://sso.stuhelper.com/api/login/oauth/access_token' },
      {
        key: 'userinfo',
        value: 'https://sso.stuhelper.com/api/userinfo',
      },
      {
        key: 'jwks',
        value: 'https://sso.stuhelper.com/.well-known/jwks',
      },
      {
        key: 'introspection',
        value: 'https://sso.stuhelper.com/api/login/oauth/introspect',
      },
      {
        key: 'revocation',
        value: 'https://sso.stuhelper.com/api/login/oauth/revoke',
      },
      { key: 'logout', value: 'https://sso.stuhelper.com/logout' },
    ])
  })

  it('keeps every Connect endpoint label present in Chinese and English locales', () => {
    for (const endpoint of buildConnectEndpoints('https://sso.stuhelper.com')) {
      expect(zhCNDeveloper.apps.connect.endpoints[endpoint.key], `zh-CN ${endpoint.key}`).toEqual(expect.any(String))
      expect(enUSDeveloper.apps.connect.endpoints[endpoint.key], `en-US ${endpoint.key}`).toEqual(expect.any(String))
    }
  })

  it('keeps the public Connect copy centered on the Casdoor SSO issuer and Open API data boundary', () => {
    const publicCopy = [
      zhCNDeveloper.connect.subtitle,
      zhCNDeveloper.apps.connect.subtitle,
      enUSDeveloper.connect.subtitle,
      enUSDeveloper.apps.connect.subtitle,
    ].join('\n')

    expect(publicCopy).toContain('sso.stuhelper.com')
    expect(publicCopy).toContain('StuHelper Open API')
    expect(publicCopy).not.toContain('StuHelper SSO')
  })
})
