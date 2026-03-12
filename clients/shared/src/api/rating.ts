import type { ApiClient } from './client'

export const createRatingApi = (client: ApiClient) => ({
  getDimensions: () =>
    client.GET('/api/v1/course/review/rating-dimensions'),

  getCourseStats: (courseId: number) =>
    client.GET('/api/v1/course/review/courses/{courseID}/rating-stats', {
      params: { path: { courseID: courseId } }
    }),

  getRatingTrend: (courseId: number) =>
    client.GET('/api/v1/course/review/courses/{courseID}/rating-trend', {
      params: { path: { courseID: courseId } }
    }),

  getCourseTeachers: (courseId: number) =>
    client.GET('/api/v1/course/review/courses/{courseID}/teachers', {
      params: { path: { courseID: courseId } }
    }),

  getTeacherStats: (teacherId: number) =>
    client.GET('/api/v1/course/review/teachers/{teacherID}/stats', {
      params: { path: { teacherID: teacherId } }
    })
})
