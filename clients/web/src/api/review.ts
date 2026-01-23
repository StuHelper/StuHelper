/**
 * 测评相关 API
 */
import api from './index'
import type { Review, PostReviewParams } from '@/types/review'

// 分页响应类型
interface PaginatedResponse<T> {
  list: T[]
  total: number
}

// 获取课程测评
export function getCourseReviews(courseId: number, page = 1, pageSize = 10) {
  return api.get<PaginatedResponse<Review>>(`/courses/${courseId}/reviews`, {
    params: { page, pageSize }
  })
}

// 获取最新测评
export function getLatestReviews(page = 1, pageSize = 10) {
  return api.get<PaginatedResponse<Review>>('/reviews/latest', {
    params: { page, pageSize }
  })
}

// 发布测评
export function postReview(data: PostReviewParams) {
  return api.post<Review>('/reviews', {
    courseId: data.courseId,
    teacherId: data.teacherId,
    termId: data.termId,
    title: data.title,
    content: data.content,
    grade: data.grade,
    ratings: data.ratings
  })
}

// 投票类型
export type VoteType = 'like' | 'dislike'

// 投票
export function voteReview(reviewId: number, voteType: VoteType) {
  return api.post<{ success: boolean }>(`/reviews/${reviewId}/vote`, {
    voteType
  })
}
