import { describe, expect, it, vi } from 'vitest'

import { createAuthApi, NATIVE_SESSION_ID_HEADER } from '../auth'
import type { ApiClient } from '../client'

function createMockClient(): ApiClient {
  return {
    DELETE: vi.fn(),
    GET: vi.fn(),
    PATCH: vi.fn(),
    POST: vi.fn(),
    PUT: vi.fn(),
  } as unknown as ApiClient
}

describe('createAuthApi', () => {
  it('passes login reauthentication query parameters', () => {
    const client = createMockClient()
    const api = createAuthApi(client)

    api.login('/oauth2/authorize?client_id=client-1', undefined, 'web', {
      prompt: 'login',
      maxAge: 0,
    })

    expect(client.GET).toHaveBeenCalledWith('/api/v1/auth/login', {
      params: {
        query: {
          app: 'web',
          max_age: 0,
          prompt: 'login',
          redirect: '/oauth2/authorize?client_id=client-1',
        },
      },
    })
  })

  it('passes native session header for refresh and logout when provided', () => {
    const client = createMockClient()
    const api = createAuthApi(client)

    api.refresh({ sessionID: 'sid-native-1' })
    api.logout({ sessionID: 'sid-native-1' })

    expect(client.POST).toHaveBeenNthCalledWith(1, '/api/v1/auth/refresh', {
      params: {
        header: {
          [NATIVE_SESSION_ID_HEADER]: 'sid-native-1',
        },
      },
    })
    expect(client.POST).toHaveBeenNthCalledWith(2, '/api/v1/auth/logout', {
      params: {
        header: {
          [NATIVE_SESSION_ID_HEADER]: 'sid-native-1',
        },
      },
    })
  })

  it('omits native session header when session id is absent', () => {
    const client = createMockClient()
    const api = createAuthApi(client)

    api.refresh()
    api.logout()

    expect(client.POST).toHaveBeenNthCalledWith(1, '/api/v1/auth/refresh', undefined)
    expect(client.POST).toHaveBeenNthCalledWith(2, '/api/v1/auth/logout', undefined)
  })

  it('passes step-up redirect and platform query parameters', () => {
    const client = createMockClient()
    const api = createAuthApi(client)

    api.stepUp('/admin/content', 'web')

    expect(client.GET).toHaveBeenCalledWith('/api/v1/auth/step-up', {
      params: {
        query: {
          platform: 'web',
          redirect: '/admin/content',
        },
      },
    })
  })
})
