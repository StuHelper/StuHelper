import { beforeEach, describe, expect, it, vi } from 'vitest'

import type { Reply } from '@stuhelper/shared/reply'
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

function makeReply(id: string, overrides: Partial<Reply> = {}): Reply {
  return {
    id,
    reviewID: 'review-1',
    parentID: null,
    content: `reply-${id}`,
    likeCount: 0,
    status: 'published',
    isOwner: false,
    createdAt: '2026-05-24T04:05:00Z',
    updatedAt: '2026-05-24T04:05:00Z',
    ...overrides,
  }
}

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
          list: [makeReply('reply-1', { content: '已有回复' })],
          total: 1,
        },
      },
    })

    const reply = useReviewReply(() => review, (key) => key)

    await reply.loadReplies()

    expect(reply.replies.value).toEqual([
      makeReply('reply-1', { content: '已有回复' }),
    ])
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

  it('fails closed when a reply list item is malformed', async () => {
    mocks.getReplies.mockResolvedValue({
      data: {
        data: {
          list: [
            {
              id: 'reply-1',
              reviewID: 'review-1',
              content: '已有回复',
              likeCount: 0,
              status: 'published',
              isOwner: 'yes',
              createdAt: '2026-05-24T04:05:00Z',
              updatedAt: '2026-05-24T04:05:00Z',
            },
          ],
          total: 1,
        },
      },
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
