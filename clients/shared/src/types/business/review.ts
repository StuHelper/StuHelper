/**
 * 测评相关类型定义
 *
 * 核心实体直接别名到 OpenAPI 生成类型。
 * normalizer 函数负责 wire → view-model 转换。
 */
import type { components } from '../api.gen'

// ---- OpenAPI 别名 ----

export type Review = components['schemas']['Review']
export type PostReviewRequest = components['schemas']['PostReviewRequest']

// ---- 前端补充类型 ----

/** 动态评分（维度 key → 分值），与 OpenAPI ReviewRatings 一致 */
export type ReviewRatings = components['schemas']['ReviewRatings']

export interface PaginatedResult<T> {
  list: T[]
  total: number
}

export interface ReviewContentCheck {
  isValid: boolean
  level?: 'allow' | 'warn' | 'review' | 'block'
  reason?: string
}

// ---- 守卫 ----

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

// ---- wire → view-model 转换 ----

type ApiReview = components['schemas']['Review']
type ApiReviewListPayload = {
  list?: ApiReview[] | null
  total?: number | null
}
type ApiContentCheckResult = components['schemas']['ContentCheckResult']

export function normalizeReview(apiReview: ApiReview): Review {
  return { ...apiReview }
}

export function normalizeReviews(items?: ApiReview[] | null): Review[] {
  return (items ?? []).map(normalizeReview)
}

export function normalizeReviewList(payload?: ApiReviewListPayload | null): PaginatedResult<Review> {
  return {
    list: (payload?.list ?? []).map(normalizeReview),
    total: payload?.total ?? 0,
  }
}

export function normalizeContentCheck(payload?: ApiContentCheckResult | null): ReviewContentCheck {
  return {
    isValid: payload?.isValid ?? true,
    level: payload?.level as ReviewContentCheck['level'],
  }
}
