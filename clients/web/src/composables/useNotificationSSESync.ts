import { watch, type Ref } from 'vue'
import type { SSENotificationEvent } from '@/stores/notification'

export interface PageNotification {
  id: string
  isRead: boolean
  [key: string]: unknown
}

export type NotificationFilter = 'all' | 'unread' | 'read'

/**
 * 响应 SSE 实时推送，幂等更新本页通知列表。
 *
 * 幂等保证：本地操作（markAsRead/delete）已先于 SSE 事件更新列表，
 * SSE watcher 到达时若 item 已不在列表中则跳过，避免 pageTotal 双减。
 */
export function useNotificationSSESync(
  pageNotifications: Ref<PageNotification[]>,
  pageTotal: Ref<number>,
  activeFilter: Ref<NotificationFilter>,
  lastSSEEvent: Ref<SSENotificationEvent>,
) {
  watch(lastSSEEvent, (event: SSENotificationEvent) => {
    if (!event || event.seq === 0) return

    switch (event.type) {
      case 'notification': {
        const n = event.data as PageNotification
        if (activeFilter.value === 'read') break
        const exists = pageNotifications.value.some(item => item.id === n.id)
        if (!exists) {
          pageNotifications.value = [n, ...pageNotifications.value]
          pageTotal.value++
        }
        break
      }
      case 'notification_read': {
        const { id } = event.data as { id: string }
        if (activeFilter.value === 'unread') {
          const found = pageNotifications.value.some(item => item.id === id)
          if (found) {
            pageNotifications.value = pageNotifications.value.filter(item => item.id !== id)
            pageTotal.value = Math.max(0, pageTotal.value - 1)
          }
        } else {
          pageNotifications.value = pageNotifications.value.map(item =>
            item.id === id ? { ...item, isRead: true } : item,
          )
        }
        break
      }
      case 'notification_read_all': {
        if (activeFilter.value === 'unread') {
          if (pageNotifications.value.length > 0) {
            pageNotifications.value = []
            pageTotal.value = 0
          }
        } else {
          pageNotifications.value = pageNotifications.value.map(n => ({ ...n, isRead: true }))
        }
        break
      }
      case 'notification_deleted': {
        const { id } = event.data as { id: string }
        const found = pageNotifications.value.some(item => item.id === id)
        if (found) {
          pageNotifications.value = pageNotifications.value.filter(item => item.id !== id)
          pageTotal.value = Math.max(0, pageTotal.value - 1)
        }
        break
      }
    }
  })
}
