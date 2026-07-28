/**
 * 测评 presentation 层
 *
 * 包含 wire → view-model 的 normalizer、类型守卫、以及纯 UI 接口。
 * 业务别名（Review, ReviewRatings 等）仍在 types/business/review.ts。
 */
import type { components } from '../types/api.gen'
import type { ReviewRatings } from '../types/business/review'

// ---- 局部 wire 别名（仅 normalizer 内部使用） ----

type ApiContentCheckResult = components['schemas']['ContentCheckResult']
export type VoteType = components['schemas']['VoteRequest']['voteType']

// ---- 纯 UI 接口 ----

export interface PaginatedResult<T> {
  list: T[]
  total: number
}

export interface ReviewContentCheck {
  isValid: boolean
  level?: 'allow' | 'warn' | 'review' | 'block'
  reason?: string
}

export interface ReviewVoteState {
  userVote: VoteType | null
  likeOffset: number
  dislikeOffset: number
}

// ---- 类型守卫 ----

export function isValidRatings(
  ratings: unknown,
  requiredKeys: string[]
): ratings is ReviewRatings {
  if (!ratings || typeof ratings !== 'object') return false
  const obj = ratings as Record<string, unknown>
  return requiredKeys.every(
    key => key in obj && typeof obj[key] === 'number' && obj[key]! >= 1 && obj[key]! <= 5
  )
}

export function normalizeContentCheck(payload?: ApiContentCheckResult | null): ReviewContentCheck {
  if (!payload || typeof payload !== 'object') {
    throw new Error('Invalid content check response')
  }

  const { isValid, level, matchCount } = payload
  if (
    typeof isValid !== 'boolean' ||
    (level !== undefined && level !== 'block' && level !== 'warn') ||
    (!isValid && level === undefined) ||
    (
      matchCount !== undefined &&
      (
        typeof matchCount !== 'number' ||
        !Number.isInteger(matchCount) ||
        matchCount < 0
      )
    )
  ) {
    throw new Error('Invalid content check response')
  }

  return {
    isValid,
    level: level as ReviewContentCheck['level'],
  }
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
