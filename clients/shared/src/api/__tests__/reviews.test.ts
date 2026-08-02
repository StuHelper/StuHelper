import { describe, expect, it, vi } from 'vitest'

import { createApiClient } from '../client'
import { createReviewApi } from '../reviews'

describe('createReviewApi.updateReview', () => {
  it('allows partial updates without forcing content or ratings', async () => {
    const PUT = vi.fn(async () => ({ data: undefined }))
    const api = createReviewApi({ PUT } as never)

    await api.updateReview('review-1', { title: '只改标题' })

    expect(PUT).toHaveBeenCalledWith(
      '/api/v1/course/review/reviews/{reviewID}',
      {
        params: { path: { reviewID: 'review-1' } },
        body: {
          title: '只改标题',
        },
      },
    )
  })
})

describe('createReviewApi.getBatchCourseReviews', () => {
  it('serializes course IDs as repeated form query parameters', async () => {
    let capturedRequest: Request | undefined
    const fetchMock: typeof fetch = async (input, init) => {
      capturedRequest = input instanceof Request ? input : new Request(input, init)
      return new Response(JSON.stringify({ data: {}, success: true }), {
        headers: { 'content-type': 'application/json' },
        status: 200,
      })
    }
    const client = createApiClient({
      baseUrl: 'https://stuhelper.test',
      fetch: fetchMock,
    })

    await createReviewApi(client).getBatchCourseReviews([11, 12, 13])

    expect(capturedRequest).toBeInstanceOf(Request)
    const url = new URL(capturedRequest?.url ?? '')
    expect(url.searchParams.getAll('courseIDs')).toEqual(['11', '12', '13'])
  })
})
