import type { ApiClient } from './client'

export const NOTIFICATION_STREAM_PATH = '/api/v1/course/review/user/notifications/stream' as const

export const createNotificationApi = (client: ApiClient) => ({
  getNotifications: (page = 1, pageSize = 10) =>
    client.GET('/api/v1/course/review/user/notifications', { params: { query: { page, pageSize } } }),

  markAsRead: (id: string) =>
    client.PUT('/api/v1/course/review/user/notifications/{notificationID}/read', { params: { path: { notificationID: id } } }),

  markAllAsRead: () =>
    client.PUT('/api/v1/course/review/user/notifications/read-all'),

  getStreamPath: () => NOTIFICATION_STREAM_PATH,

  getUnreadCount: () =>
    client.GET('/api/v1/course/review/user/notifications/unread-count')
})
