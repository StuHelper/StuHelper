/**
 * 测评相关类型定义
 */
import type { components } from '../api.gen'
import type { RatingValue } from './course'

// 动态评分类型
export type ReviewRatings = Record<string, RatingValue>

export interface PaginatedResult<T> {
  list: T[]
  total: number
}

export interface ReviewContentCheck {
  isValid: boolean
  level?: 'allow' | 'warn' | 'review' | 'block'
  reason?: string
}

// 测评
export interface Review {
  id: string
  courseID: number
  courseName?: string
  teacherID?: number | null
  teacherName?: string
  termID: string
  termName?: string
  title: string
  content: string
  grade?: string
  ratings: ReviewRatings
  likeCount: number
  dislikeCount: number
  replyCount: number
  status: 'published' | 'pending_review' | 'hidden' | 'deleted'
  contentFlag?: 'warn' | 'review' | 'cleared' | null
  moderationReason?: string | null
  createdAt: string
  updatedAt?: string
}

// 发布测评参数
export interface PostReviewParams {
  courseID: number
  teacherID?: number
  termID?: string
  title?: string
  content: string
  grade?: string
  ratings: ReviewRatings
}

// 类型守卫：检查是否为有效的评分对象
export function isValidRatings(
  ratings: unknown,
  requiredKeys: string[]
): ratings is ReviewRatings {
  if (!ratings || typeof ratings !== 'object') return false
  const obj = ratings as Record<string, unknown>
  return requiredKeys.every(
    key => key in obj && typeof obj[key] === 'number' && obj[key] >= 1 && obj[key] <= 5
  )
}

type ApiReview = components['schemas']['Review']
type ApiReviewListPayload = {
  list?: ApiReview[] | null
  total?: number | null
}
type ApiContentCheckResult = components['schemas']['ContentCheckResult']

export function normalizeReview(apiReview: ApiReview): Review {
  return {
    ...apiReview,
    ratings: apiReview.ratings as ReviewRatings,
    status: apiReview.status as Review['status'],
    contentFlag: (apiReview.contentFlag ?? null) as Review['contentFlag'],
  }
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
