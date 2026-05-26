import { beforeEach, describe, expect, it, vi } from 'vitest'

import type { Review } from '@stuhelper/shared/review'

const mocks = vi.hoisted(() => ({
  getReplies: vi.fn(),
  createReply: vi.fn(),
  deleteReply: vi.fn(),
  toastSuccess: vi.fn(),
  toastError: vi.fn(),
}))

vi.mock('@/api', () => ({
  api: {
    reply: {
      getReplies: mocks.getReplies,
      createReply: mocks.createReply,
      deleteReply: mocks.deleteReply,
    },
  },
}))

vi.mock('@/composables/useToast', () => ({
  useToast: () => ({
    success: mocks.toastSuccess,
    error: mocks.toastError,
  }),
}))

const { useReviewReply } = await import('../useReviewReply')

const review: Review = {
  id: 'review-1',
  courseID: 42,
  courseName: '操作系统',
  title: '课程评价',
  content: '评价内容',
  ratings: { recommendation: 4 },
  likeCount: 0,
  dislikeCount: 0,
  replyCount: 3,
  status: 'published',
  createdAt: '2026-05-24T04:00:00Z',
}

describe('useReviewReply', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('loads valid replies and replaces the server reply count', async () => {
    mocks.getReplies.mockResolvedValue({
      data: {
        data: {
          list: [{ id: 'reply-1', content: '已有回复' }],
          total: 1,
        },
      },
    })

    const reply = useReviewReply(() => review, (key) => key)

    await reply.loadReplies()

    expect(reply.replies.value).toEqual([{ id: 'reply-1', content: '已有回复' }])
    expect(reply.replyCount.value).toBe(1)
    expect(reply.repliesError.value).toBe(false)
  })

  it('fails closed when reply list response is malformed', async () => {
    mocks.getReplies.mockResolvedValue({
      data: { data: null },
    })

    const reply = useReviewReply(() => review, (key) => key)

    await reply.loadReplies()

    expect(reply.replies.value).toEqual([])
    expect(reply.replyCount.value).toBe(3)
    expect(reply.repliesError.value).toBe(true)
  })

  it('surfaces malformed create-reply success responses as submit failures', async () => {
    mocks.createReply.mockResolvedValue({
      data: { data: null },
    })

    const reply = useReviewReply(() => review, (key) => key)

    await reply.handleReplySubmit('新增回复')

    expect(reply.replies.value).toEqual([])
    expect(reply.replyCount.value).toBe(3)
    expect(mocks.toastSuccess).not.toHaveBeenCalled()
    expect(mocks.toastError).toHaveBeenCalledWith('review.review.replyFailed')
  })
})
