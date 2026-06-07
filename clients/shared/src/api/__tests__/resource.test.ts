import { describe, expect, it, vi } from 'vitest'

import { createResourceApi } from '../resource'
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

describe('createResourceApi', () => {
  it('lists resources with search filters', () => {
    const client = createMockClient()
    const api = createResourceApi(client)

    api.listResources({
      page: 2,
      pageSize: 12,
      query: '高数',
      tag: '期末',
      bindingType: 'course',
      bindingValue: '8',
    })

    expect(client.GET).toHaveBeenCalledWith('/api/v1/resources', {
      params: {
        query: {
          page: 2,
          pageSize: 12,
          query: '高数',
          tag: '期末',
          bindingType: 'course',
          bindingValue: '8',
        },
      },
    })
  })

  it('loads detail and download URLs with path params', () => {
    const client = createMockClient()
    const api = createResourceApi(client)

    api.getResource(42)
    api.getDownloadURL(42)

    expect(client.GET).toHaveBeenNthCalledWith(
      1,
      '/api/v1/resources/{resourceID}',
      { params: { path: { resourceID: 42 } } },
    )
    expect(client.GET).toHaveBeenNthCalledWith(
      2,
      '/api/v1/resources/{resourceID}/download-url',
      { params: { path: { resourceID: 42 } } },
    )
  })

  it('passes write payloads through the generated resource contract', () => {
    const client = createMockClient()
    const api = createResourceApi(client)
    const payload = {
      title: '高等数学A 期末复习讲义',
      description: '覆盖极限、导数和积分。',
      category: '讲义',
      visibility: 'public' as const,
      tags: ['期末', '高数'],
      bindings: [{ type: 'course', value: '8' }],
      filename: 'math-final.pdf',
      contentType: 'application/pdf',
      dataBase64: 'ZHVtbXk=',
    }

    api.createResource(payload)
    api.updateResource(42, {
      title: '高等数学A 期末复习讲义 v2',
      visibility: 'public',
      tags: ['期末'],
      bindings: [{ type: 'course', value: '8' }],
    })
    api.deleteResource(42)

    expect(client.POST).toHaveBeenCalledWith('/api/v1/resources', {
      body: payload,
    })
    expect(client.PATCH).toHaveBeenCalledWith(
      '/api/v1/resources/{resourceID}',
      {
        params: { path: { resourceID: 42 } },
        body: {
          title: '高等数学A 期末复习讲义 v2',
          visibility: 'public',
          tags: ['期末'],
          bindings: [{ type: 'course', value: '8' }],
        },
      },
    )
    expect(client.DELETE).toHaveBeenCalledWith(
      '/api/v1/resources/{resourceID}',
      { params: { path: { resourceID: 42 } } },
    )
  })
})
