/**
 * 通知状态管理
 */
import { defineStore } from 'pinia'
import { ref, computed, onScopeDispose } from 'vue'
import type { Notification } from '@/types/notification'
import * as notificationApi from '@/api/notification'

// L-31: 提取魔法数字为命名常量
const POLL_INTERVAL_MS = 30_000
const MAX_POLL_FAILURES = 5

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
  // 连续失败计数器
  let consecutiveFailures = 0
  // M-76: 轮询是否应继续的标志
  let pollingActive = false

  // 是否有未读通知
  const hasUnread = computed(() => unreadCount.value > 0)

  // L-55: 参数验证
  const fetchNotifications = async (page = 1, pageSize = 20) => {
    if (page < 1) page = 1
    if (pageSize < 1) pageSize = 20

    loading.value = true
    try {
      const res = await notificationApi.getNotifications(page, pageSize)
      const list = res.data?.list || []

      if (page === 1) {
        notifications.value = list
      } else {
        // 按 ID 去重，防止两次请求间有新通知导致重复
        const existingIDs = new Set(notifications.value.map(n => n.id))
        const newItems = list.filter(n => !existingIDs.has(n.id))
        notifications.value.push(...newItems)
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
      consecutiveFailures = 0
    } catch {
      consecutiveFailures++
      // M-76: 仅在轮询仍活跃时才因失败停止
      if (consecutiveFailures >= MAX_POLL_FAILURES && pollingActive) {
        if (import.meta.env.DEV) {
          console.warn(`[Notification] ${MAX_POLL_FAILURES} consecutive poll failures, stopping`)
        }
        stopPolling()
      }
    }
  }

  // M-01: 标记单条已读 - API 失败时异常自然上抛，不更新本地状态，保持前后端一致
  const markAsRead = async (id: string) => {
    await notificationApi.markAsRead(id)
    const notification = notifications.value.find(n => n.id === id)
    if (notification && !notification.isRead) {
      notification.isRead = true
      unreadCount.value = Math.max(0, unreadCount.value - 1)
    }
  }

  // 全部标记已读
  const markAllAsRead = async () => {
    await notificationApi.markAllAsRead()
    notifications.value.forEach(n => {
      n.isRead = true
    })
    unreadCount.value = 0
  }

  // M-09: 开始轮询，支持最大轮询次数
  const startPolling = (interval = POLL_INTERVAL_MS) => {
    stopPolling()
    pollingActive = true
    consecutiveFailures = 0
    fetchUnreadCount()
    pollTimer = setInterval(() => {
      if (!pollingActive) {
        stopPolling()
        return
      }
      fetchUnreadCount()
    }, interval)
  }

  // 停止轮询
  const stopPolling = () => {
    pollingActive = false
    if (pollTimer) {
      clearInterval(pollTimer)
      pollTimer = null
    }
  }

  // store 作用域销毁时自动停止轮询
  onScopeDispose(stopPolling)

  // 重置状态（setup store 不支持 $reset）
  const reset = () => {
    notifications.value = []
    total.value = 0
    loading.value = false
    hasMore.value = true
    unreadCount.value = 0
    consecutiveFailures = 0
  }

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
    stopPolling,
    reset
  }
})
