import { describe, expect, it } from 'vitest'

import {
  applyOptimisticVote,
  createReviewVoteState,
  getDisplayVoteCount,
} from '../review'

describe('review vote state', () => {
  it('adds a like from the neutral state', () => {
    const next = applyOptimisticVote(createReviewVoteState(), 'like')

    expect(next).toEqual({
      userVote: 'like',
      likeOffset: 1,
      dislikeOffset: 0,
    })
  })

  it('toggles the same vote back off', () => {
    const current = createReviewVoteState({ userVote: 'like', likeOffset: 1 })
    const next = applyOptimisticVote(current, 'like')

    expect(next).toEqual({
      userVote: null,
      likeOffset: 0,
      dislikeOffset: 0,
    })
  })

  it('switches from like to dislike without mutating the rollback snapshot', () => {
    const previous = createReviewVoteState({ userVote: 'like', likeOffset: 1 })
    const next = applyOptimisticVote(previous, 'dislike')

    expect(previous).toEqual({
      userVote: 'like',
      likeOffset: 1,
      dislikeOffset: 0,
    })
    expect(next).toEqual({
      userVote: 'dislike',
      likeOffset: 0,
      dislikeOffset: 1,
    })
  })

  it('clamps display counts at zero', () => {
    expect(getDisplayVoteCount(0, -2)).toBe(0)
    expect(getDisplayVoteCount(4, -1)).toBe(3)
  })
})
