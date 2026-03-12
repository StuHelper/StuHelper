import type { ApiClient } from './client'

export const createUserApi = (client: ApiClient) => ({
  getMyReviews: (page = 1, pageSize = 10) =>
    client.GET('/api/v1/course/review/user/reviews', { params: { query: { page, pageSize } } }),

  getMyVotes: (page = 1, pageSize = 10, voteType: 'like' | 'dislike' = 'like') =>
    client.GET('/api/v1/course/review/user/votes', { params: { query: { page, pageSize, voteType } } }),

  getMyFavorites: (page = 1, pageSize = 10) =>
    client.GET('/api/v1/course/review/user/favorites', { params: { query: { page, pageSize } } }),

  addFavorite: (courseId: number) =>
    client.POST('/api/v1/course/review/courses/{courseID}/favorites', { params: { path: { courseID: courseId } } }),

  removeFavorite: (courseId: number) =>
    client.DELETE('/api/v1/course/review/courses/{courseID}/favorites', { params: { path: { courseID: courseId } } })
})
