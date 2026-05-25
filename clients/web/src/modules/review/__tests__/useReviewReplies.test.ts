import { beforeEach, describe, expect, it, vi } from 'vitest'

const mockCreateReply = vi.fn()
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

describe('useReviewReplies', () => {
  beforeEach(() => {
    mockCreateReply.mockReset()
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
