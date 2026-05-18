import type { ApiClient } from './client'
import type { operations } from '../types/api.gen'

type StepUpURLResponse = operations['getStepUpURL']['responses'][200]['content']['application/json']['data']
type NativeSessionHeader = NonNullable<operations['refreshToken']['parameters']['header']>

export const NATIVE_SESSION_ID_HEADER = 'X-Stuhelper-Session-ID' as const

export interface NativeSessionRequestOptions {
  sessionID?: NativeSessionHeader[typeof NATIVE_SESSION_ID_HEADER]
}

function withNativeSessionHeader(sessionID?: NativeSessionRequestOptions['sessionID']) {
  if (!sessionID) return undefined

  return {
    params: {
      header: {
        [NATIVE_SESSION_ID_HEADER]: sessionID,
      },
    },
  }
}

export type FirstPartyOIDCApp = 'admin' | 'uniapp' | 'web'

export const createAuthApi = (client: ApiClient) => ({
  login: (redirect?: string, platform?: string, app?: FirstPartyOIDCApp) => {
    const query: Record<string, string> = {}
    if (redirect) query.redirect = redirect
    if (platform) query.platform = platform
    if (app) query.app = app
    return client.GET('/api/v1/auth/login', Object.keys(query).length > 0 ? { params: { query } } : undefined)
  },

  signup: (redirect?: string, platform?: string, app?: FirstPartyOIDCApp) => {
    const query: Record<string, string> = {}
    if (redirect) query.redirect = redirect
    if (platform) query.platform = platform
    if (app) query.app = app
    return client.GET('/api/v1/auth/signup', Object.keys(query).length > 0 ? { params: { query } } : undefined)
  },

  stepUp: (redirect?: string, platform?: 'native' | 'web') => {
    const query: Record<string, string> = {}
    if (redirect) query.redirect = redirect
    if (platform) query.platform = platform
    return client.GET('/api/v1/auth/step-up', Object.keys(query).length > 0 ? { params: { query } } : undefined)
  },

  refresh: (options?: NativeSessionRequestOptions) =>
    client.POST('/api/v1/auth/refresh', withNativeSessionHeader(options?.sessionID)),

  me: () =>
    client.GET('/api/v1/auth/me'),

  logout: (options?: NativeSessionRequestOptions) =>
    client.POST('/api/v1/auth/logout', withNativeSessionHeader(options?.sessionID)),

  logoutAll: () =>
    client.POST('/api/v1/auth/logout-all'),

  exchangeNative: (code: string, state: string) =>
    client.POST('/api/v1/auth/exchange-native', { body: { code, state } }),
})

export type {
  StepUpURLResponse,
}
