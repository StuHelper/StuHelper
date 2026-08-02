import { beforeEach, describe, expect, it, vi } from 'vitest'

import type { Review } from '@stuhelper/shared/review'

const mocks = vi.hoisted(() => ({
  updateReview: vi.fn(),
  toastSuccess: vi.fn(),
  toastError: vi.fn(),
}))

vi.mock('@/api', () => ({
  api: {
    review: {
      updateReview: mocks.updateReview,
    },
  },
}))

vi.mock('@/composables/useToast', () => ({
  useToast: () => ({
    success: mocks.toastSuccess,
    error: mocks.toastError,
  }),
}))

const { useReviewEdit } = await import('../useReviewEdit')

const review: Review = {
  id: 'review-1',
  courseID: 42,
  title: '课程评价',
  content: '原始评课内容足够长',
  ratings: { recommendation: 4 },
  likeCount: 0,
  dislikeCount: 0,
  replyCount: 0,
  status: 'published',
  createdAt: '2026-08-02T00:00:00Z',
}

const translate = (key: string, params?: Record<string, string | number>) =>
  params ? `${key}:${JSON.stringify(params)}` : key

describe('useReviewEdit validation', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mocks.updateReview.mockResolvedValue({})
  })

  it('rejects trimmed content shorter than the API minimum', async () => {
    const edit = useReviewEdit(() => review, translate, vi.fn())
    edit.editContent.value = '123456789'

    await edit.handleSaveEdit()

    expect(mocks.updateReview).not.toHaveBeenCalled()
    expect(mocks.toastError).toHaveBeenCalledWith(
      'review.validation.contentTooShort:{"min":10}',
    )
  })

  it('rejects content longer than the API maximum', async () => {
    const edit = useReviewEdit(() => review, translate, vi.fn())
    edit.editContent.value = 'a'.repeat(5001)

    await edit.handleSaveEdit()

    expect(mocks.updateReview).not.toHaveBeenCalled()
    expect(mocks.toastError).toHaveBeenCalledWith(
      'review.validation.contentTooLong:{"max":5000}',
    )
  })

  it('submits content at the API minimum', async () => {
    const onUpdated = vi.fn()
    const edit = useReviewEdit(() => review, translate, onUpdated)
    edit.editContent.value = '  1234567890  '

    await edit.handleSaveEdit()

    expect(mocks.updateReview).toHaveBeenCalledWith(review.id, {
      content: '1234567890',
      ratings: review.ratings,
    })
    expect(onUpdated).toHaveBeenCalledWith(review.id, '1234567890')
    expect(mocks.toastSuccess).toHaveBeenCalledWith('review.review.editSuccess')
  })
})
