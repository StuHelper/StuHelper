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
