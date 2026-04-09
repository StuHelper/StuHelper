/**
 * 通知相关类型定义
 */

// 通知类型
export type NotificationType =
  | 'reply'
  | 'like'
  | 'vote'
  | 'review_hidden'
  | 'review_restored'
  | 'report_resolved'
  | 'identity_approved'
  | 'identity_rejected'
  | 'student_approved'
  | 'student_rejected'
  | 'system'

export type NotificationPayload = Record<string, unknown>

// 通知
export interface Notification {
  id: string
  type: NotificationType
  title: string
  content?: string
  payload?: NotificationPayload
  sourceModule?: string
  sourceId?: string
  sourceUrl?: string
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
