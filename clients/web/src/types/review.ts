/**
 * 测评相关类型定义
 */
import type { RatingValue } from './course'

// 动态评分类型
export type ReviewRatings = Record<string, RatingValue>

// 测评
export interface Review {
  id: string
  courseId: number
  courseName?: string
  teacherName?: string
  termId?: string
  termName?: string
  title: string
  content: string
  grade?: string
  ratings: ReviewRatings
  likeCount: number
  dislikeCount: number
  createdAt: string
}

// 发布测评参数
export interface PostReviewParams {
  courseId: number
  teacherId?: number
  termId?: string
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
