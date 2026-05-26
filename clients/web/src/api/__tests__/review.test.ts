import { describe, expect, it, vi } from 'vitest'

import { createReviewAppApi } from '../review'

describe('createReviewAppApi', () => {
  it('keeps wire calls in the transport layer and normalizes paginated responses in the app adapter', async () => {
    const client = {
      GET: vi.fn().mockResolvedValue({
        data: {
          data: {
            list: [{ id: 'r1', content: 'hello' }],
            total: 1,
          },
        },
      }),
      POST: vi.fn().mockResolvedValue({
        data: {
          data: {
            isValid: false,
            level: 'warn',
          },
        },
      }),
      PUT: vi.fn(),
      PATCH: vi.fn(),
      DELETE: vi.fn(),
    }

    const api = createReviewAppApi(client as never)

    await expect(api.getReviewsPage(42, { page: 2, pageSize: 10 })).resolves.toEqual({
      list: [{ id: 'r1', content: 'hello' }],
      total: 1,
    })
    await expect(api.checkContentResult({ content: 'hello' })).resolves.toEqual({
      isValid: false,
      level: 'warn',
    })

    expect(client.GET).toHaveBeenCalledWith(
      '/api/v1/course/review/courses/{courseID}/reviews',
      {
        params: {
          path: { courseID: 42 },
          query: { page: 2, pageSize: 10 },
        },
      },
    )
    expect(client.POST).toHaveBeenCalledWith(
      '/api/v1/course/review/content/check',
      {
        body: { content: 'hello' },
      },
    )
  })

  it('fails closed when a paginated review response is malformed', async () => {
    const client = {
      GET: vi.fn().mockResolvedValue({
        data: { data: null },
      }),
      POST: vi.fn(),
      PUT: vi.fn(),
      PATCH: vi.fn(),
      DELETE: vi.fn(),
    }

    const api = createReviewAppApi(client as never)

    await expect(api.getLatestReviewsPage()).rejects.toThrow(
      'Invalid review list response',
    )
  })

  it('fails closed when a content check response is malformed', async () => {
    const client = {
      GET: vi.fn(),
      POST: vi.fn().mockResolvedValue({
        data: { data: null },
      }),
      PUT: vi.fn(),
      PATCH: vi.fn(),
      DELETE: vi.fn(),
    }

    const api = createReviewAppApi(client as never)

    await expect(api.checkContentResult({ content: 'hello' })).rejects.toThrow(
      'Invalid content check response',
    )
  })
})
