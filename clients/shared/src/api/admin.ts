import type { ApiClient } from './client'
import type { components } from '../types/api.gen'

type AdminUpdateReviewRequest = components['schemas']['AdminUpdateReviewRequest']
type ProcessReportRequest = components['schemas']['ProcessReportRequest']
type BatchUpdateReviewsRequest = components['schemas']['BatchUpdateReviewsRequest']
type AdminEditReviewRequest = components['schemas']['AdminEditReviewRequest']
type CreateTeacherRequest = components['schemas']['CreateTeacherRequest']
type UpdateTeacherRequest = components['schemas']['UpdateTeacherRequest']
type CreateSensitiveWordRequest = components['schemas']['CreateSensitiveWordRequest']
type UpdateSensitiveWordRequest = components['schemas']['UpdateSensitiveWordRequest']

export const createAdminApi = (client: ApiClient) => ({
  getStats: () =>
    client.GET('/api/v1/course/review/admin/stats'),

  getReviews: (params?: { status?: 'published' | 'hidden' | 'deleted' | 'all'; page?: number; pageSize?: number }) =>
    client.GET('/api/v1/course/review/admin/reviews', { params: { query: params } }),

  updateReview: (id: string, data: AdminUpdateReviewRequest) =>
    client.PUT('/api/v1/course/review/admin/reviews/{reviewID}', { params: { path: { reviewID: id } }, body: data }),

  editReview: (id: string, data: AdminEditReviewRequest) =>
    client.POST('/api/v1/course/review/admin/reviews/{reviewID}/edit', { params: { path: { reviewID: id } }, body: data }),

  batchUpdateReviews: (data: BatchUpdateReviewsRequest) =>
    client.PATCH('/api/v1/course/review/admin/reviews/batch', { body: data }),

  getReports: (params?: { status?: 'pending' | 'resolved' | 'rejected' | 'all'; page?: number; pageSize?: number }) =>
    client.GET('/api/v1/course/review/admin/reports', { params: { query: params } }),

  processReport: (id: string, data: ProcessReportRequest) =>
    client.PUT('/api/v1/course/review/admin/reports/{reportID}', { params: { path: { reportID: id } }, body: data }),

  getLogs: (params?: { page?: number; pageSize?: number }) =>
    client.GET('/api/v1/course/review/admin/logs', { params: { query: params } }),

  exportReviews: (params?: { format?: 'json' | 'ndjson' | 'csv'; status?: 'all' | 'published' | 'hidden' | 'deleted' }) =>
    client.GET('/api/v1/course/review/admin/export', { params: { query: params } }),

  getTeachers: (params?: { page?: number; pageSize?: number; search?: string; departmentID?: number }) =>
    client.GET('/api/v1/course/review/admin/teachers', { params: { query: params } }),

  createTeacher: (data: CreateTeacherRequest) =>
    client.POST('/api/v1/course/review/admin/teachers', { body: data }),

  updateTeacher: (id: number, data: UpdateTeacherRequest) =>
    client.PUT('/api/v1/course/review/admin/teachers/{teacherID}', { params: { path: { teacherID: id } }, body: data }),

  deleteTeacher: (id: number) =>
    client.DELETE('/api/v1/course/review/admin/teachers/{teacherID}', { params: { path: { teacherID: id } } }),

  getSensitiveWords: (params?: { page?: number; pageSize?: number; category?: string; level?: 'block' | 'warn' | 'review' }) =>
    client.GET('/api/v1/course/review/admin/sensitive-words', { params: { query: params } }),

  getContentFlags: (params?: { page?: number; pageSize?: number }) =>
    client.GET('/api/v1/course/review/admin/content-flags', { params: { query: params } }),

  clearContentFlag: (id: string) =>
    client.PUT('/api/v1/course/review/admin/content-flags/{reviewID}/clear', { params: { path: { reviewID: id } } }),

  createSensitiveWord: (data: CreateSensitiveWordRequest) =>
    client.POST('/api/v1/course/review/admin/sensitive-words', { body: data }),

  updateSensitiveWord: (id: string, data: UpdateSensitiveWordRequest) =>
    client.PUT('/api/v1/course/review/admin/sensitive-words/{sensitiveWordID}', { params: { path: { sensitiveWordID: id } }, body: data }),

  deleteSensitiveWord: (id: string) =>
    client.DELETE('/api/v1/course/review/admin/sensitive-words/{sensitiveWordID}', { params: { path: { sensitiveWordID: id } } })
})
