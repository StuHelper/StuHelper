/**
 * 通知状态管理
 */
import { computed, onScopeDispose, ref } from 'vue'
import { defineStore } from 'pinia'
import type { Notification as AppNotification } from '@/types/notification'
import { api, NOTIFICATION_STREAM_PATH } from '@/api'

const POLL_INTERVAL_MS = 30_000
const MAX_POLL_FAILURES = 5
const SSE_INITIAL_RECONNECT_MS = 1_000
const SSE_MAX_RECONNECT_MS = 30_000

export const useNotificationStore = defineStore('notification', () => {
  const notifications = ref<AppNotification[]>([])
  const total = ref(0)
  const loading = ref(false)
  const hasMore = ref(true)
  const fetchError = ref<Error | null>(null)
  const unreadCount = ref(0)

  let pollTimer: ReturnType<typeof setInterval> | null = null
  let reconnectTimer: ReturnType<typeof setTimeout> | null = null
  let eventSource: EventSource | null = null
  let consecutiveFailures = 0
  let monitoringActive = false
  let pollingFallbackActive = false
  let reconnectDelay = SSE_INITIAL_RECONNECT_MS

  const hasUnread = computed(() => unreadCount.value > 0)

  const clearPollTimer = () => {
    if (pollTimer) {
      clearInterval(pollTimer)
      pollTimer = null
    }
  }

  const clearReconnectTimer = () => {
    if (reconnectTimer) {
      clearTimeout(reconnectTimer)
      reconnectTimer = null
    }
  }

  const closeEventSource = (source?: EventSource) => {
    const target = source ?? eventSource
    if (!target) {
      return
    }

    target.close()
    if (!source || eventSource === source) {
      eventSource = null
    }
  }

  const stopPollingFallback = () => {
    pollingFallbackActive = false
    clearPollTimer()
  }

  const fetchNotifications = async (page = 1, pageSize = 20) => {
    if (page < 1) page = 1
    if (pageSize < 1) pageSize = 20

    loading.value = true
    fetchError.value = null
    try {
      const res = await api.notification.getNotifications(page, pageSize)
      const list = res.data?.data?.list || []

      if (page === 1) {
        notifications.value = list
      } else {
        const existingIDs = new Set(notifications.value.map((n: AppNotification) => n.id))
        const newItems = list.filter((n: AppNotification) => !existingIDs.has(n.id))
        notifications.value = [...notifications.value, ...newItems]
      }

      total.value = res.data?.data?.total || 0
      hasMore.value = notifications.value.length < total.value
    } catch (err) {
      fetchError.value = err instanceof Error ? err : new Error(String(err))
      throw err
    } finally {
      loading.value = false
    }
  }

  const fetchUnreadCount = async () => {
    try {
      const res = await api.notification.getUnreadCount()
      unreadCount.value = res.data?.data?.count || 0
      consecutiveFailures = 0
    } catch {
      consecutiveFailures++
      if (consecutiveFailures >= MAX_POLL_FAILURES && pollingFallbackActive) {
        if (import.meta.env.DEV) {
          console.warn(`[Notification] ${MAX_POLL_FAILURES} consecutive poll failures, stopping polling fallback`)
        }
        stopPollingFallback()
      }
    }
  }

  const markAsRead = async (id: string) => {
    await api.notification.markAsRead(id)
    const target = notifications.value.find((notification) => notification.id === id)
    if (target && !target.isRead) {
      notifications.value = notifications.value.map((notification) => (
        notification.id === id ? { ...notification, isRead: true } : notification
      ))
      unreadCount.value = Math.max(0, unreadCount.value - 1)
    }
  }

  const markAllAsRead = async () => {
    await api.notification.markAllAsRead()
    notifications.value = notifications.value.map((notification) => ({ ...notification, isRead: true }))
    unreadCount.value = 0
  }

  const startPollingFallback = (interval = POLL_INTERVAL_MS) => {
    stopPollingFallback()
    pollingFallbackActive = true
    consecutiveFailures = 0
    void fetchUnreadCount()

    pollTimer = setInterval(() => {
      if (!monitoringActive || !pollingFallbackActive) {
        stopPollingFallback()
        return
      }
      void fetchUnreadCount()
    }, interval)
  }

  const scheduleReconnect = () => {
    if (!monitoringActive || reconnectTimer) {
      return
    }

    const delay = reconnectDelay
    reconnectDelay = Math.min(reconnectDelay * 2, SSE_MAX_RECONNECT_MS)

    reconnectTimer = setTimeout(() => {
      reconnectTimer = null
      if (!monitoringActive) {
        return
      }
      connectSSE()
    }, delay)
  }

  const startPolling = (interval = POLL_INTERVAL_MS) => {
    monitoringActive = true
    clearReconnectTimer()
    closeEventSource()
    reconnectDelay = SSE_INITIAL_RECONNECT_MS
    startPollingFallback(interval)
  }

  const stopPolling = () => {
    monitoringActive = false
    stopPollingFallback()
    clearReconnectTimer()
    closeEventSource()
    reconnectDelay = SSE_INITIAL_RECONNECT_MS
    consecutiveFailures = 0
  }

  const connectSSE = () => {
    monitoringActive = true
    clearReconnectTimer()

    if (typeof EventSource === 'undefined') {
      startPollingFallback()
      return
    }

    if (!pollingFallbackActive) {
      void fetchUnreadCount()
    }

    closeEventSource()

    const source = new EventSource(NOTIFICATION_STREAM_PATH, { withCredentials: true })
    eventSource = source

    source.onopen = () => {
      if (eventSource !== source) {
        return
      }
      reconnectDelay = SSE_INITIAL_RECONNECT_MS
      consecutiveFailures = 0
      stopPollingFallback()
    }

    source.addEventListener('unread_count', (event) => {
      try {
        const data = JSON.parse(event.data)
        if (typeof data.count === 'number') {
          unreadCount.value = data.count
          consecutiveFailures = 0
        }
      } catch {
        // ignore malformed SSE data
      }
    })

    source.addEventListener('notification', () => {
      void fetchUnreadCount()
    })

    source.onerror = () => {
      closeEventSource(source)
      if (!monitoringActive) {
        return
      }
      startPollingFallback()
      scheduleReconnect()
    }
  }

  onScopeDispose(stopPolling)

  const reset = () => {
    stopPolling()
    notifications.value = []
    total.value = 0
    loading.value = false
    hasMore.value = true
    unreadCount.value = 0
    fetchError.value = null
  }

  return {
    notifications,
    total,
    loading,
    hasMore,
    unreadCount,
    hasUnread,
    fetchError,
    fetchNotifications,
    fetchUnreadCount,
    markAsRead,
    markAllAsRead,
    startPolling,
    stopPolling,
    connectSSE,
    reset,
  }
})
