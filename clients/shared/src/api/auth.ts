import type { ApiClient } from './client'
import type { operations } from '../types/api.gen'

type RequestPhoneOTPResponse = operations['requestPhoneOTP']['responses'][200]['content']['application/json']['data']
type VerifyPhoneOTPResponse = operations['verifyPhoneOTP']['responses'][200]['content']['application/json']['data']

export const createAuthApi = (client: ApiClient) => ({
  login: (redirect?: string, platform?: string) => {
    const query: Record<string, string> = {}
    if (redirect) query.redirect = redirect
    if (platform) query.platform = platform
    return client.GET('/api/v1/auth/login', Object.keys(query).length > 0 ? { params: { query } } : undefined)
  },

  signup: (redirect?: string) =>
    client.GET('/api/v1/auth/signup', redirect ? { params: { query: { redirect } } } : undefined),

  refresh: () =>
    client.POST('/api/v1/auth/refresh'),

  me: () =>
    client.GET('/api/v1/auth/me'),

  logout: () =>
    client.POST('/api/v1/auth/logout'),

  logoutAll: () =>
    client.POST('/api/v1/auth/logout-all'),

  requestPhoneOTP: (phone: string) =>
    client.POST('/api/v1/auth/phone/request-otp', { body: { phone } }),

  verifyPhoneOTP: (phone: string, code: string) =>
    client.POST('/api/v1/auth/phone/verify-otp', { body: { phone, code } }),

  exchangeNative: (code: string, state: string, codeVerifier: string) =>
    client.POST('/api/v1/auth/exchange-native', { body: { code, state, code_verifier: codeVerifier } }),
})

export type {
  RequestPhoneOTPResponse,
  VerifyPhoneOTPResponse,
}
