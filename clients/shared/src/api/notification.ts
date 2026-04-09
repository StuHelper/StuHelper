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
    client.GET('/api/v1/course/review/user/notifications/unread-count'),

  // 后端尚未提供通知偏好 API，前端默认全部启用。
  getPreferences: async () =>
    ({ data: { data: { preferences: [] } } }) as {
      data: { data: { preferences: Array<{ type: string; enabled: boolean }> } }
    },

  // 后端尚未提供通知偏好 API，先做 no-op 保持前端交互可用。
  updatePreference: async (_type: string, _enabled: boolean) =>
    ({ data: { success: true } }) as { data: { success: true } }
})
