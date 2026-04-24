import { describe, expect, it, vi } from 'vitest'

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
