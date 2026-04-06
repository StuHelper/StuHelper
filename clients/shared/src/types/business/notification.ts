/**
 * 通知相关类型定义
 */

// 通知类型
export type NotificationType = 'reply' | 'vote' | 'system'

// 通知
export interface Notification {
  id: string
  type: NotificationType
  title: string
  content?: string
  relatedType?: string
  relatedID?: string
  courseID?: number
  isRead: boolean
  createdAt: string
}

// 未读数量响应
export interface UnreadCountResponse {
  count: number
}
