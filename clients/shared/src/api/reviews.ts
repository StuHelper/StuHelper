import type { ApiClient } from './client'
import type { components } from '../types/api.gen'
import type { PostReviewParams, ReviewRatings } from '../types/business/review'

type PostReviewRequest = components['schemas']['PostReviewRequest']
type UpdateReviewRequest = components['schemas']['UpdateReviewRequest']
type VoteRequest = components['schemas']['VoteRequest']
type ReportReviewRequest = components['schemas']['ReportReviewRequest']
type ContentCheckRequest = components['schemas']['ContentCheckRequest']
type ReviewGrade = NonNullable<PostReviewRequest['grade']>

const REVIEW_GRADES: readonly ReviewGrade[] = ['A+', 'A', 'A-', 'B+', 'B', 'B-', 'C+', 'C', 'C-', 'D', 'F'] as const

function normalizeReviewGrade(grade?: string): ReviewGrade | undefined {
  if (!grade) return undefined
  const trimmed = grade.trim()
  return REVIEW_GRADES.includes(trimmed as ReviewGrade) ? (trimmed as ReviewGrade) : undefined
}

function toPostReviewRequest(data: PostReviewParams): PostReviewRequest {
  return {
    courseID: data.courseID,
    content: data.content,
    ratings: data.ratings,
    termID: data.termID ?? '',
    title: data.title ?? '',
    ...(data.teacherID !== undefined && { teacherID: data.teacherID }),
    ...(normalizeReviewGrade(data.grade) !== undefined && { grade: normalizeReviewGrade(data.grade) }),
  }
}

function toUpdateReviewRequest(data: { title?: string; content: string; grade?: string; ratings: ReviewRatings }): UpdateReviewRequest {
  return {
    content: data.content,
    ratings: data.ratings,
    ...(data.title !== undefined && { title: data.title }),
    ...(normalizeReviewGrade(data.grade) !== undefined && { grade: normalizeReviewGrade(data.grade) }),
  }
}

export const createReviewApi = (client: ApiClient) => ({
  getReviewStats: () =>
    client.GET('/api/v1/course/review/stats'),

  getHotCourses: (params?: { period?: 'week' | 'month' | 'all'; limit?: number }) =>
    client.GET('/api/v1/course/review/rankings/hot', { params: { query: params } }),

  getReviews: (courseId: number, params?: { page?: number; pageSize?: number; sort?: 'time' | 'likes' | 'rating'; termID?: string; teacherID?: number }) =>
    client.GET('/api/v1/course/review/courses/{courseID}/reviews', {
      params: { path: { courseID: courseId }, query: params }
    }),

  getLatestReviews: (params?: { page?: number; pageSize?: number; sort?: 'time' | 'likes' | 'rating' }) =>
    client.GET('/api/v1/course/review/reviews/latest', { params: { query: params } }),

  getBatchCourseReviews: (courseIDs: number[], params?: { pageSize?: number; sort?: 'time' | 'likes' | 'rating' }) =>
    client.GET('/api/v1/course/review/reviews/batch', {
      params: {
        query: {
          courseIDs,
          ...params,
        },
      },
    }),

  searchReviews: (
    params?: { q?: string; departmentID?: number; teacherName?: string; termID?: string; page?: number; pageSize?: number; sort?: 'time' | 'likes' | 'rating' },
    options?: { signal?: AbortSignal }
  ) =>
    client.GET('/api/v1/course/review/reviews/search', {
      params: { query: params },
      signal: options?.signal,
    }),

  createReview: (data: PostReviewParams) =>
    client.POST('/api/v1/course/review/reviews', { body: toPostReviewRequest(data) }),

  updateReview: (id: string, data: { title?: string; content: string; grade?: string; ratings: ReviewRatings }) =>
    client.PUT('/api/v1/course/review/reviews/{reviewID}', {
      params: { path: { reviewID: id } },
      body: toUpdateReviewRequest(data)
    }),

  deleteReview: (id: string) =>
    client.DELETE('/api/v1/course/review/reviews/{reviewID}', {
      params: { path: { reviewID: id } }
    }),

  voteReview: (id: string, data: VoteRequest) =>
    client.POST('/api/v1/course/review/reviews/{reviewID}/votes', {
      params: { path: { reviewID: id } },
      body: data
    }),

  reportReview: (id: string, data: ReportReviewRequest) =>
    client.POST('/api/v1/course/review/reviews/{reviewID}/reports', {
      params: { path: { reviewID: id } },
      body: data
    }),

  checkContent: (data: ContentCheckRequest) =>
    client.POST('/api/v1/course/review/content/check', { body: data })
})
