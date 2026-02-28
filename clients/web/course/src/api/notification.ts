/**
 * 通知 API
 */
import api from './index'
import type { Notification, UnreadCountResponse } from '@/types/notification'
import type { PaginatedResponse } from '@/types/api'

// 获取通知列表
export function getNotifications(page = 1, pageSize = 20) {
  return api.get<PaginatedResponse<Notification>>('/notifications', {
    params: { page, pageSize }
  })
}

// 获取未读数量
export function getUnreadCount() {
  return api.get<UnreadCountResponse>('/notifications/unread-count')
}

// 标记单条已读
export function markAsRead(notificationID: string) {
  return api.put<{ message: string }>(`/notifications/${notificationID}/read`)
}

// 全部标记已读
export function markAllAsRead() {
  return api.put<{ message: string }>('/notifications/read-all')
}
