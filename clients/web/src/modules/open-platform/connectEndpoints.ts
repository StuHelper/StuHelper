export const DEFAULT_SSO_ISSUER = 'https://sso.stuhelper.com'

export type ConnectEndpointKey =
  | 'issuer'
  | 'discovery'
  | 'authorization'
  | 'token'
  | 'userinfo'
  | 'jwks'
  | 'introspection'
  | 'revocation'
  | 'logout'

export interface ConnectEndpoint {
  key: ConnectEndpointKey
  value: string
}

const connectEndpointPaths: Array<{
  key: Exclude<ConnectEndpointKey, 'issuer'>
  path: string
}> = [
  { key: 'discovery', path: '/.well-known/openid-configuration' },
  { key: 'authorization', path: '/login/oauth/authorize' },
  { key: 'token', path: '/api/login/oauth/access_token' },
  { key: 'userinfo', path: '/api/userinfo' },
  { key: 'jwks', path: '/.well-known/jwks' },
  { key: 'introspection', path: '/api/login/oauth/introspect' },
  { key: 'revocation', path: '/api/login/oauth/revoke' },
  { key: 'logout', path: '/logout' },
]

export function normalizeSsoIssuer(configuredOrigin?: string | null, currentOrigin?: string | null) {
  const candidate = configuredOrigin?.trim() || currentOrigin?.trim() || DEFAULT_SSO_ISSUER
  try {
    const url = new URL(candidate)
    return url.origin
  } catch {
    return DEFAULT_SSO_ISSUER
  }
}

export function buildConnectEndpoints(issuer: string): ConnectEndpoint[] {
  const normalizedIssuer = normalizeSsoIssuer(issuer)
  return [
    { key: 'issuer', value: normalizedIssuer },
    ...connectEndpointPaths.map(({ key, path }) => ({
      key,
      value: new URL(path, normalizedIssuer).toString(),
    })),
  ]
}
