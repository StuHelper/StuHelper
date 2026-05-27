export const DEFAULT_IDENTITY_ISSUER = 'https://id.stuhelper.com'

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
  { key: 'authorization', path: '/oauth2/authorize' },
  { key: 'token', path: '/oauth2/token' },
  { key: 'userinfo', path: '/oidc/userinfo' },
  { key: 'jwks', path: '/.well-known/jwks.json' },
  { key: 'introspection', path: '/oauth2/introspect' },
  { key: 'revocation', path: '/oauth2/revoke' },
  { key: 'logout', path: '/oauth2/logout' },
]

export function normalizeIdentityIssuer(configuredOrigin?: string | null, currentOrigin?: string | null) {
  const candidate = configuredOrigin?.trim() || currentOrigin?.trim() || DEFAULT_IDENTITY_ISSUER
  try {
    const url = new URL(candidate)
    return url.origin
  } catch {
    return DEFAULT_IDENTITY_ISSUER
  }
}

export function buildConnectEndpoints(issuer: string): ConnectEndpoint[] {
  const normalizedIssuer = normalizeIdentityIssuer(issuer)
  return [
    { key: 'issuer', value: normalizedIssuer },
    ...connectEndpointPaths.map(({ key, path }) => ({
      key,
      value: new URL(path, normalizedIssuer).toString(),
    })),
  ]
}
