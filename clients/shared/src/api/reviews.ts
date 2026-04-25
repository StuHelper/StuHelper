import type { ApiClient } from './client'
import type { components } from '../types/api.gen'
import type { PostReviewRequest, ReviewRatings } from '../types/business/review'
import { normalizeReviewGrade } from '../constants/review'

type UpdateReviewRequest = components['schemas']['UpdateReviewRequest']
type VoteRequest = components['schemas']['VoteRequest']
type ReportReviewRequest = components['schemas']['ReportReviewRequest']
type ContentCheckRequest = components['schemas']['ContentCheckRequest']

type ReviewUpdateInput = Omit<UpdateReviewRequest, 'grade' | 'ratings'> & {
  grade?: string
  ratings?: ReviewRatings
}

function toUpdateReviewRequest(data: ReviewUpdateInput): UpdateReviewRequest {
  const grade = normalizeReviewGrade(data.grade)

  return {
    ...(data.title !== undefined && { title: data.title }),
    ...(data.content !== undefined && { content: data.content }),
    ...(grade !== undefined && { grade }),
    ...(data.ratings !== undefined && { ratings: data.ratings }),
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

  createReview: (data: PostReviewRequest) =>
    client.POST('/api/v1/course/review/reviews', { body: data }),

  updateReview: (id: string, data: ReviewUpdateInput) =>
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
    client.POST('/api/v1/course/review/content/check', { body: data }),
})
