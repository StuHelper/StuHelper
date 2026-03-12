import type { ApiClient } from './client'

export const createReplyApi = (client: ApiClient) => ({
  getReplies: (reviewId: string, params?: { page?: number; pageSize?: number }) =>
    client.GET('/api/v1/course/review/reviews/{reviewID}/replies', {
      params: { path: { reviewID: reviewId }, query: params }
    }),

  createReply: (reviewId: string, data: { content: string }) =>
    client.POST('/api/v1/course/review/reviews/{reviewID}/replies', {
      params: { path: { reviewID: reviewId } },
      body: data
    }),

  deleteReply: (replyId: string) =>
    client.DELETE('/api/v1/course/review/replies/{replyID}', {
      params: { path: { replyID: replyId } }
    })
})
