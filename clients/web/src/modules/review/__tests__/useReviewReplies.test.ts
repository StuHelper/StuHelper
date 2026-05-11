import { beforeEach, describe, expect, it, vi } from 'vitest'

const mockCreateReply = vi.fn()
const mockToastSuccess = vi.fn()
const mockToastError = vi.fn()

vi.mock('@/api', () => ({
  api: {
    reply: {
      createReply: mockCreateReply,
      getReplies: vi.fn(),
      deleteReply: vi.fn(),
    },
  },
}))

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
    }),
  }
})

vi.mock('@/composables/useToast', () => ({
  useToast: () => ({
    success: mockToastSuccess,
    error: mockToastError,
  }),
}))

const { useReviewReplies } = await import('../useReviewReplies')

describe('useReviewReplies', () => {
  beforeEach(() => {
    mockCreateReply.mockReset()
    mockToastSuccess.mockReset()
    mockToastError.mockReset()
  })

  it('clears the active reply form when Vue exposes v-for refs as an array', async () => {
    const clear = vi.fn()
    mockCreateReply.mockResolvedValue({
      data: {
        data: {
          id: 'reply-1',
          content: 'saved reply',
          createdAt: '2026-05-11T00:00:00Z',
        },
      },
    })

    const replies = useReviewReplies()
    replies.replyFormRef.value = [{ clear }] as never

    await replies.handleReplySubmit('review-1', 'saved reply')

    expect(mockCreateReply).toHaveBeenCalledWith('review-1', {
      content: 'saved reply',
    })
    expect(clear).toHaveBeenCalledTimes(1)
    expect(mockToastSuccess).toHaveBeenCalledWith('review.review.replySuccess')
    expect(mockToastError).not.toHaveBeenCalled()
    expect(replies.replyCountMap['review-1']).toBe(1)
  })
})
