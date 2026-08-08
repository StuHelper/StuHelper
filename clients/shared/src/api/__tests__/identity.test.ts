import { describe, expect, it, vi } from 'vitest'

import { createIdentityApi } from '../identity'
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

describe('createIdentityApi', () => {
  it('exposes only the current account surface and QQ binding endpoints', () => {
    const client = createMockClient()
    const api = createIdentityApi(client)

    api.getUserSurface()
    api.getQQBinding()
    api.createQQBindingCode()

    expect(client.GET).toHaveBeenNthCalledWith(1, '/api/v1/user/me')
    expect(client.GET).toHaveBeenNthCalledWith(2, '/api/v1/user/qq-binding')
    expect(client.POST).toHaveBeenCalledWith('/api/v1/user/qq-binding/code')
    expect(Object.keys(api).sort()).toEqual([
      'createQQBindingCode',
      'getQQBinding',
      'getUserSurface',
    ])
  })
})
