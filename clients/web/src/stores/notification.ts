/**
 * 通知状态管理
 */
import { defineStore } from 'pinia'
import { ref, computed, onScopeDispose } from 'vue'
import type { Notification } from '@/types/notification'
import * as notificationApi from '@/api/notification'

export const useNotificationStore = defineStore('notification', () => {
  // 通知列表
  const notifications = ref<Notification[]>([])
  const total = ref(0)
  const loading = ref(false)
  const hasMore = ref(true)

  // 未读数量
  const unreadCount = ref(0)

  // 轮询定时器
  let pollTimer: ReturnType<typeof setInterval> | null = null

  // 是否有未读通知
  const hasUnread = computed(() => unreadCount.value > 0)

  // 获取通知列表
  const fetchNotifications = async (page = 1, pageSize = 20) => {
    loading.value = true
    try {
      const res = await notificationApi.getNotifications(page, pageSize)
      const list = res.data?.list || []

      if (page === 1) {
        notifications.value = list
      } else {
        notifications.value.push(...list)
      }

      total.value = res.data?.total || 0
      hasMore.value = notifications.value.length < total.value
    } finally {
      loading.value = false
    }
  }

  // 获取未读数量
  const fetchUnreadCount = async () => {
    try {
      const res = await notificationApi.getUnreadCount()
      unreadCount.value = res.data?.count || 0
    } catch {
      // 轮询失败静默忽略，下次轮询会重试
    }
  }

  // 标记单条已读
  const markAsRead = async (id: string) => {
    await notificationApi.markAsRead(id)
    // 更新本地状态
    const notification = notifications.value.find(n => n.id === id)
    if (notification && !notification.isRead) {
      notification.isRead = true
      unreadCount.value = Math.max(0, unreadCount.value - 1)
    }
  }

  // 全部标记已读
  const markAllAsRead = async () => {
    await notificationApi.markAllAsRead()
    // 更新本地状态
    notifications.value.forEach(n => {
      n.isRead = true
    })
    unreadCount.value = 0
  }

  // 开始轮询
  const startPolling = (interval = 30000) => {
    stopPolling()
    fetchUnreadCount()
    pollTimer = setInterval(fetchUnreadCount, interval)
  }

  // 停止轮询
  const stopPolling = () => {
    if (pollTimer) {
      clearInterval(pollTimer)
      pollTimer = null
    }
  }

  // store 作用域销毁时自动停止轮询
  onScopeDispose(stopPolling)

  return {
    notifications,
    total,
    loading,
    hasMore,
    unreadCount,
    hasUnread,
    fetchNotifications,
    fetchUnreadCount,
    markAsRead,
    markAllAsRead,
    startPolling,
    stopPolling
  }
})
