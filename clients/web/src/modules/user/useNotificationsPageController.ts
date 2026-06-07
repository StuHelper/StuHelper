import { computed, ref, type Ref } from 'vue'
import { resolveNotificationHref } from '@stuhelper/shared/notification'
import { useNotificationSSESync, type NotificationFilter } from '@/composables/useNotificationSSESync'
import type { SSENotificationEvent } from '@/stores/notification'
import type { Notification } from '@stuhelper/shared/notification'
import i18n from '@/i18n'
import { getErrorMessage } from '@/api/errors'
import { useToast } from '@/composables/useToast'
import { accountCenterURLForHref, navigateToExternalURL } from '@/utils/redirect'

interface NotificationsPageControllerOptions {
  pageNotifications: Ref<Notification[]>
  pageTotal: Ref<number>
  pageLoading: Ref<boolean>
  pageHasMore: Ref<boolean>
  unreadCount: Ref<number>
  hasUnread: Ref<boolean>
  lastSSEEvent: Ref<SSENotificationEvent>
  fetchPageNotifications: (page?: number) => Promise<void>
  markAllAsRead: () => Promise<void>
  markAsRead: (id: string) => Promise<void>
  push: (href: string) => Promise<unknown>
  t: (key: string, params?: Record<string, unknown>) => string
}

export function useNotificationsPageController({
  pageNotifications,
  pageTotal,
  pageLoading,
  pageHasMore,
  unreadCount,
  hasUnread,
  lastSSEEvent,
  fetchPageNotifications,
  markAllAsRead,
  markAsRead,
  push,
  t,
}: NotificationsPageControllerOptions) {
  const toast = useToast()
  const activeFilter = ref<NotificationFilter>('all')
  const page = ref(1)

  useNotificationSSESync(pageNotifications, pageTotal, activeFilter, lastSSEEvent)

  const filterOptions = computed(() => [
    { value: 'all' as const, label: t('user.notification.filterAll') },
    { value: 'unread' as const, label: t('user.notification.filterUnread') },
    { value: 'read' as const, label: t('user.notification.filterRead') },
  ])

  const visibleNotifications = computed(() => {
    switch (activeFilter.value) {
      case 'unread':
        return pageNotifications.value.filter(notification => !notification.isRead)
      case 'read':
        return pageNotifications.value.filter(notification => notification.isRead)
      default:
        return pageNotifications.value
    }
  })

  const loading = computed(() => pageLoading.value)
  const hasMore = computed(() => {
    if (!pageHasMore.value) return false
    switch (activeFilter.value) {
      case 'unread':
        return visibleNotifications.value.length < unreadCount.value
      case 'read':
        return true
      default:
        return true
    }
  })

  const init = async () => {
    page.value = 1
    try {
      await fetchPageNotifications(1)
    } catch (err) {
      toast.error(getErrorMessage(err, i18n.global.t('common.loadFailed')))
    }
  }

  const loadMore = async () => {
    if (loading.value || !hasMore.value) return
    page.value += 1
    try {
      await fetchPageNotifications(page.value)
    } catch (err) {
      page.value = Math.max(1, page.value - 1)
      toast.error(getErrorMessage(err, i18n.global.t('common.loadFailed')))
    }
  }

  const handleMarkAllRead = () => {
    void markAllAsRead().catch((err) => {
      toast.error(getErrorMessage(err, i18n.global.t('common.actions.operationFailed')))
    })
  }

  const navigateNotification = async (notification: Notification) => {
    const href = resolveNotificationHref(notification)
    if (!href) return

    const accountCenterURL = accountCenterURLForHref(href)
    if (accountCenterURL) {
      navigateToExternalURL(accountCenterURL)
      return
    }
    await push(href)
  }

  const handleClick = async (notification: Notification) => {
    try {
      if (!notification.isRead) {
        await markAsRead(notification.id)
      }
    } catch (err) {
      toast.error(getErrorMessage(err, i18n.global.t('common.actions.operationFailed')))
    }

    try {
      await navigateNotification(notification)
    } catch (err) {
      toast.error(getErrorMessage(err, i18n.global.t('common.actions.operationFailed')))
    }
  }

  return {
    activeFilter,
    filterOptions,
    visibleNotifications,
    loading,
    hasMore,
    unreadCount,
    hasUnread,
    init,
    loadMore,
    handleMarkAllRead,
    handleClick,
  }
}
