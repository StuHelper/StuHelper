import type { ApiClient } from './client'

export const createNotificationApi = (client: ApiClient) => ({
  getNotifications: (page = 1, pageSize = 10) =>
    client.GET('/api/v1/course/review/user/notifications', { params: { query: { page, pageSize } } }),

  markAsRead: (id: string) =>
    client.PUT('/api/v1/course/review/user/notifications/{notificationID}/read', { params: { path: { notificationID: id } } }),

  markAllAsRead: () =>
    client.PUT('/api/v1/course/review/user/notifications/read-all'),

  getUnreadCount: () =>
    client.GET('/api/v1/course/review/user/notifications/unread-count')
})
