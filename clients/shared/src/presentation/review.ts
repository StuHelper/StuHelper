/**
 * 测评 presentation 层
 *
 * 包含 wire → view-model 的 normalizer、类型守卫、以及纯 UI 接口。
 * 业务别名（Review, ReviewRatings 等）仍在 types/business/review.ts。
 */
import type { components } from '../types/api.gen'
import type { Review, ReviewRatings } from '../types/business/review'

// ---- 局部 wire 别名（仅 normalizer 内部使用） ----

type ApiReview = components['schemas']['Review']
type ApiReviewListPayload = {
  list?: ApiReview[] | null
  total?: number | null
}
type ApiContentCheckResult = components['schemas']['ContentCheckResult']

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

export function normalizeReviews(items?: ApiReview[] | null): Review[] {
  return (items ?? []) as Review[]
}

export function normalizeReviewList(payload?: ApiReviewListPayload | null): PaginatedResult<Review> {
  if (!payload || typeof payload !== 'object') {
    throw new Error('Invalid review list response')
  }

  const { list, total } = payload
  if (!Array.isArray(list) || typeof total !== 'number' || !Number.isFinite(total) || total < 0) {
    throw new Error('Invalid review list response')
  }

  return {
    list: list as Review[],
    total,
  }
}

export function normalizeContentCheck(payload?: ApiContentCheckResult | null): ReviewContentCheck {
  if (!payload || typeof payload !== 'object') {
    throw new Error('Invalid content check response')
  }

  const { isValid, level } = payload
  if (
    typeof isValid !== 'boolean' ||
    (level !== undefined && level !== 'block' && level !== 'warn') ||
    (!isValid && level === undefined)
  ) {
    throw new Error('Invalid content check response')
  }

  return {
    isValid,
    level: level as ReviewContentCheck['level'],
  }
}
