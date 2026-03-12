import type { ApiClient } from './client'

export const createAuthApi = (client: ApiClient) => ({
  login: () =>
    client.GET('/api/v1/auth/login'),

  signup: () =>
    client.GET('/api/v1/auth/signup'),

  callback: (code: string, state: string) =>
    client.GET('/api/v1/auth/callback', { params: { query: { code, state } } }),

  refresh: () =>
    client.POST('/api/v1/auth/refresh'),

  me: () =>
    client.GET('/api/v1/auth/me'),

  logout: () =>
    client.POST('/api/v1/auth/logout'),

  logoutAll: () =>
    client.POST('/api/v1/auth/logout-all')
})
