import { computed, ref, type Ref } from 'vue'
import type { Router } from 'vue-router'
import { resolveNotificationHref } from '@stuhelper/shared/notification'
import type { Notification } from '@stuhelper/shared/notification'
import i18n from '@/i18n'
import { getErrorMessage } from '@/api/errors'
import { useToast } from '@/composables/useToast'

export interface NotificationBellStore {
  bellNotifications: Notification[]
  unreadCount: number
  hasUnread: boolean
  bellLoading: boolean
  connectSSE: () => void
  stopPolling: () => void
  fetchBellNotifications: (page?: number, pageSize?: number) => Promise<void>
  markAllAsRead: () => Promise<void>
  markAsRead: (id: string) => Promise<void>
}

interface NotificationBellControllerOptions {
  rootRef: Ref<HTMLElement | null>
  store: NotificationBellStore
  router: Pick<Router, 'push'>
}

export function useNotificationBellController({
  rootRef,
  store,
  router,
}: NotificationBellControllerOptions) {
  const toast = useToast()
  const showPanel = ref(false)
  const hasLoadedHistory = ref(false)

  const notifications = computed(() => store.bellNotifications.slice(0, 5))
  const unreadCount = computed(() => store.unreadCount)
  const hasUnread = computed(() => store.hasUnread)
  const loading = computed(() => store.bellLoading)

  const togglePanel = async () => {
    showPanel.value = !showPanel.value
    if (showPanel.value && !hasLoadedHistory.value) {
      try {
        await store.fetchBellNotifications(1, 5)
        hasLoadedHistory.value = true
      } catch (err) {
        toast.error(getErrorMessage(err, i18n.global.t('common.loadFailed')))
      }
    }
  }

  const handleMarkAllRead = async () => {
    try {
      await store.markAllAsRead()
    } catch (err) {
      toast.error(getErrorMessage(err, i18n.global.t('common.actions.operationFailed')))
    }
  }

  const handleNotificationClick = async (payload: string | Notification) => {
    const notification = typeof payload === 'string'
      ? notifications.value.find(item => item.id === payload)
      : payload

    if (!notification) return

    try {
      await store.markAsRead(notification.id)
      showPanel.value = false
      const href = resolveNotificationHref(notification)
      if (href) {
        await router.push(href)
      }
    } catch (err) {
      toast.error(getErrorMessage(err, i18n.global.t('common.actions.operationFailed')))
    }
  }

  const handleDocumentClick = (event: MouseEvent) => {
    const target = event.target
    if (!(target instanceof Node)) {
      showPanel.value = false
      return
    }
    if (!rootRef.value?.contains(target)) {
      showPanel.value = false
    }
  }

  const start = () => {
    store.connectSSE()
  }

  const stop = () => {
    store.stopPolling()
  }

  return {
    showPanel,
    notifications,
    unreadCount,
    hasUnread,
    loading,
    togglePanel,
    handleMarkAllRead,
    handleNotificationClick,
    handleDocumentClick,
    start,
    stop,
  }
}
