/**
 * 通知相关类型定义 — 直接别名到 OpenAPI 生成类型
 */
import type { components } from '../api.gen'

export type NotificationType = components['schemas']['NotificationType']
export type NotificationPayload = components['schemas']['NotificationPayload']
export type Notification = components['schemas']['Notification']

/** 未读数量响应 */
export interface UnreadCountResponse {
  count: number
}
