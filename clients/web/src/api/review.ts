/**
 * 测评相关 API
 */
import api from './index'
import type { Review, PostReviewParams } from '@/types/review'
import type { Reply, PostReplyParams } from '@/types/reply'
import type { PaginatedResponse } from '@/types/api'

// 排序类型
export type SortType = 'time' | 'likes' | 'rating'

// 获取课程测评查询参数
export interface GetCourseReviewsParams {
  page?: number
  pageSize?: number
  sort?: SortType
  termID?: string
  teacherID?: number
}

// 获取课程测评
export function getCourseReviews(
  courseID: number,
  params: GetCourseReviewsParams = {}
) {
  const { page = 1, pageSize = 10, sort, termID, teacherID } = params
  return api.get<PaginatedResponse<Review>>(`/courses/${courseID}/reviews`, {
    params: { page, pageSize, sort, termID, teacherID }
  })
}

// 获取最新测评
export function getLatestReviews(page = 1, pageSize = 10, sort?: string) {
  return api.get<PaginatedResponse<Review>>('/reviews/latest', {
    params: { page, pageSize, sort }
  })
}

// 发布测评
export function postReview(data: PostReviewParams) {
  return api.post<Review>('/reviews', {
    courseID: data.courseID,
    teacherID: data.teacherID,
    termID: data.termID,
    title: data.title,
    content: data.content,
    grade: data.grade,
    ratings: data.ratings
  })
}

// 投票类型
export type VoteType = 'like' | 'dislike'

// 投票
export function voteReview(reviewID: string, voteType: VoteType) {
  return api.post<{ success: boolean }>(`/reviews/${reviewID}/vote`, {
    voteType
  })
}

// ========== 回复相关 API ==========

// 获取回复列表
export function getReplies(reviewID: string, page = 1, pageSize = 20) {
  return api.get<PaginatedResponse<Reply>>(`/reviews/${reviewID}/replies`, {
    params: { page, pageSize }
  })
}

// 发表回复
export function postReply(params: PostReplyParams) {
  return api.post<Reply>(`/reviews/${params.reviewID}/replies`, {
    parentID: params.parentID,
    content: params.content
  })
}

// 删除回复
export function deleteReply(replyID: string) {
  return api.delete<{ success: boolean }>(`/replies/${replyID}`)
}

// ========== 测评管理 API ==========

// 更新测评
export function updateReview(reviewID: string, data: Partial<PostReviewParams>) {
  return api.put<Review>(`/reviews/${reviewID}`, data)
}

// 删除测评
export function deleteReview(reviewID: string) {
  return api.delete<{ message: string }>(`/reviews/${reviewID}`)
}

// 举报测评
export function reportReview(reviewID: string, reason: string, description?: string) {
  return api.post<{ message: string }>(`/reviews/${reviewID}/report`, { reason, description })
}
