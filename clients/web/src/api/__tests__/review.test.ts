import { describe, expect, it, vi } from 'vitest'

import { createReviewAppApi } from '../review'

describe('createReviewAppApi', () => {
  it('keeps wire calls in the transport layer and normalizes paginated responses in the app adapter', async () => {
    const review = {
      id: 'r1',
      courseID: 42,
      courseName: '操作系统',
      termID: '2026-spring',
      title: 'hello',
      content: 'hello',
      ratings: { recommendation: 5 },
      likeCount: 1,
      dislikeCount: 0,
      replyCount: 0,
      status: 'published',
      createdAt: '2026-04-01T00:00:00Z',
    }
    const client = {
      GET: vi.fn().mockResolvedValue({
        data: {
          data: {
            list: [review],
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
      list: [review],
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
        data: {
          data: {
            list: [
              {
                id: 'r1',
                courseID: 42,
                courseName: '操作系统',
                termID: '2026-spring',
                title: 'hello',
                content: 'hello',
                ratings: { recommendation: 6 },
                likeCount: 1,
                dislikeCount: 0,
                replyCount: 0,
                status: 'published',
                createdAt: '2026-04-01T00:00:00Z',
              },
            ],
            total: 1,
          },
        },
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
