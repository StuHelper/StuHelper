import { beforeEach, describe, expect, it, vi } from 'vitest'
import { nextTick } from 'vue'

const mockVoteReview = vi.fn()
const mockToastError = vi.fn()

vi.mock('@/api', () => ({
  api: {
    review: {
      voteReview: mockVoteReview,
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
    error: mockToastError,
  }),
}))

const { useReviewVoting } = await import('../useReviewVoting')

describe('useReviewVoting', () => {
  beforeEach(() => {
    mockVoteReview.mockReset()
    mockToastError.mockReset()
  })

  it('optimistically updates vote state on success', async () => {
    mockVoteReview.mockResolvedValue(undefined)
    const review = { id: 'review-1', likeCount: 10, dislikeCount: 2 }
    const voting = useReviewVoting()

    expect(voting.displayLikeCount(review as never)).toBe(10)

    await voting.handleVote(review as never, 'like')

    expect(mockVoteReview).toHaveBeenCalledWith('review-1', { voteType: 'like' })
    expect(voting.reviewVotes['review-1']).toBe('like')
    expect(voting.displayLikeCount(review as never)).toBe(11)
    expect(voting.displayDislikeCount(review as never)).toBe(2)
  })

  it('rolls back optimistic vote state on failure', async () => {
    mockVoteReview.mockRejectedValue(new Error('boom'))
    const review = { id: 'review-2', likeCount: 3, dislikeCount: 1 }
    const voting = useReviewVoting()

    await voting.handleVote(review as never, 'dislike')
    await nextTick()

    expect(voting.reviewVotes['review-2']).toBeNull()
    expect(voting.displayLikeCount(review as never)).toBe(3)
    expect(voting.displayDislikeCount(review as never)).toBe(1)
    expect(mockToastError).toHaveBeenCalledWith('review.review.voteFailed')
  })
})
