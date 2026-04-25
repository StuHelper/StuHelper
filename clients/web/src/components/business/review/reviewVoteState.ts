import type { components } from '@stuhelper/shared/types'

export type VoteType = components['schemas']['VoteRequest']['voteType']

export interface ReviewVoteState {
  userVote: VoteType | null
  likeOffset: number
  dislikeOffset: number
}

export function createReviewVoteState(
  state: Partial<ReviewVoteState> = {},
): ReviewVoteState {
  return {
    userVote: state.userVote ?? null,
    likeOffset: state.likeOffset ?? 0,
    dislikeOffset: state.dislikeOffset ?? 0,
  }
}

export function applyOptimisticVote(
  state: ReviewVoteState,
  nextVote: VoteType,
): ReviewVoteState {
  const nextState = createReviewVoteState(state)

  if (nextState.userVote === nextVote) {
    nextState.userVote = null
    if (nextVote === 'like') nextState.likeOffset -= 1
    else nextState.dislikeOffset -= 1
    return nextState
  }

  if (nextState.userVote === 'like') nextState.likeOffset -= 1
  if (nextState.userVote === 'dislike') nextState.dislikeOffset -= 1

  nextState.userVote = nextVote
  if (nextVote === 'like') nextState.likeOffset += 1
  else nextState.dislikeOffset += 1

  return nextState
}

export function getDisplayVoteCount(baseCount: number, offset: number): number {
  return Math.max(0, baseCount + offset)
}
