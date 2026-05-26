import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { Reply } from '@stuhelper/shared/reply'

const mockCreateReply = vi.fn()
const mockGetReplies = vi.fn()
const mockDeleteReply = vi.fn()
const mockToastSuccess = vi.fn()
const mockToastError = vi.fn()
const mockRouterPush = vi.fn()
const mockBootstrapSession = vi.fn()
const mockAuthStore = {
  bootstrapCompleted: true,
  isAuthenticated: true,
  bootstrapSession: mockBootstrapSession,
}

vi.mock('@/api', () => ({
  api: {
    reply: {
      createReply: mockCreateReply,
      getReplies: mockGetReplies,
      deleteReply: mockDeleteReply,
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

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => mockAuthStore,
}))

vi.mock('vue-router', () => ({
  useRouter: () => ({
    currentRoute: {
      value: {
        fullPath: '/courses/42/reviews',
      },
    },
    push: mockRouterPush,
  }),
}))

const { useReviewReplies } = await import('../useReviewReplies')

function makeReply(id: string, overrides: Partial<Reply> = {}): Reply {
  return {
    id,
    reviewID: 'review-1',
    parentID: null,
    content: `reply-${id}`,
    likeCount: 0,
    status: 'published',
    isOwner: true,
    createdAt: '2026-05-11T00:00:00Z',
    updatedAt: '2026-05-11T00:00:00Z',
    ...overrides,
  }
}

describe('useReviewReplies', () => {
  beforeEach(() => {
    mockCreateReply.mockReset()
    mockGetReplies.mockReset()
    mockDeleteReply.mockReset()
    mockToastSuccess.mockReset()
    mockToastError.mockReset()
    mockRouterPush.mockReset()
    mockBootstrapSession.mockReset()
    mockAuthStore.bootstrapCompleted = true
    mockAuthStore.isAuthenticated = true
  })

  it('clears the active reply form when Vue exposes v-for refs as an array', async () => {
    const clear = vi.fn()
    mockCreateReply.mockResolvedValue({
      data: {
        data: makeReply('reply-1', {
          content: 'saved reply',
        }),
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

  it('fails closed when reply list response is missing page data', async () => {
    mockGetReplies.mockResolvedValue({
      data: { data: null },
    })

    const replies = useReviewReplies()

    await replies.toggleExpand('review-1')

    expect(mockGetReplies).toHaveBeenCalledWith('review-1')
    expect(replies.replies.value).toEqual([])
    expect(replies.repliesError.value).toBe(true)
    expect(replies.replyCountMap['review-1']).toBeUndefined()
  })

  it('fails closed when a reply list item is malformed', async () => {
    mockGetReplies.mockResolvedValue({
      data: {
        data: {
          list: [
            {
              id: 'reply-bad',
              reviewID: 'review-1',
              content: 'bad reply',
              likeCount: -1,
              status: 'published',
              isOwner: true,
              createdAt: '2026-05-11T00:00:00Z',
              updatedAt: '2026-05-11T00:00:00Z',
            },
          ],
          total: 1,
        },
      },
    })

    const replies = useReviewReplies()

    await replies.toggleExpand('review-1')

    expect(replies.replies.value).toEqual([])
    expect(replies.repliesError.value).toBe(true)
    expect(replies.replyCountMap['review-1']).toBeUndefined()
  })

  it('surfaces malformed create-reply success responses as submit failures', async () => {
    mockCreateReply.mockResolvedValue({
      data: { data: null },
    })

    const replies = useReviewReplies()

    await replies.handleReplySubmit('review-1', 'saved reply')

    expect(replies.replies.value).toEqual([])
    expect(replies.replyCountMap['review-1']).toBeUndefined()
    expect(mockToastSuccess).not.toHaveBeenCalled()
    expect(mockToastError).toHaveBeenCalledWith('review.review.replyFailed')
  })

  it('redirects unauthenticated users before submitting replies', async () => {
    mockAuthStore.isAuthenticated = false
    const replies = useReviewReplies()

    await replies.handleReplySubmit('review-guest', 'guest reply')

    expect(mockCreateReply).not.toHaveBeenCalled()
    expect(replies.replySubmitting.value).toBe(false)
    expect(mockToastSuccess).not.toHaveBeenCalled()
    expect(mockToastError).not.toHaveBeenCalled()
    expect(mockRouterPush).toHaveBeenCalledWith({
      name: 'login',
      query: { redirect: '/courses/42/reviews' },
    })
  })
})
