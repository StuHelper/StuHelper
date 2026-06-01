import { beforeEach, describe, expect, it, vi } from 'vitest'
import { nextTick, ref } from 'vue'
import type { Notification } from '@stuhelper/shared/notification'
import type { SSENotificationEvent } from '@/stores/notification'
import { useNotificationsPageController } from '../useNotificationsPageController'

const mockToastError = vi.fn()
const redirectMocks = vi.hoisted(() => ({
  identityPortalURLForHref: vi.fn((href: string) =>
    href.startsWith('/user/') || href.startsWith('/developers/')
      ? `https://stuhelper.com${href}`
      : null,
  ),
  navigateToExternalURL: vi.fn(),
}))

vi.mock('@/composables/useToast', () => ({
  useToast: () => ({
    error: mockToastError,
  }),
}))

vi.mock('@/api/errors', () => ({
  getErrorMessage: () => 'notification-error',
}))

vi.mock('@/i18n', () => ({
  default: {
    global: {
      t: (key: string) => key,
    },
  },
}))

vi.mock('@/utils/redirect', () => ({
  identityPortalURLForHref: redirectMocks.identityPortalURLForHref,
  navigateToExternalURL: redirectMocks.navigateToExternalURL,
}))

function makeNotification(id: string, isRead: boolean, overrides: Partial<Notification> = {}): Notification {
  return {
    id,
    type: 'reply',
    title: `notification-${id}`,
    isRead,
    createdAt: '2026-04-08T00:00:00Z',
    ...overrides,
  }
}

function setupController(options: {
  initialNotifications?: Notification[]
  total?: number
  unreadCount?: number
}) {
  const pageNotifications = ref(options.initialNotifications ?? [])
  const pageTotal = ref(options.total ?? pageNotifications.value.length)
  const pageLoading = ref(false)
  const pageHasMore = ref(false)
  const unreadCount = ref(options.unreadCount ?? 0)
  const hasUnread = ref(unreadCount.value > 0)
  const lastSSEEvent = ref<SSENotificationEvent>({ seq: 0, type: 'notification', data: null })
  const push = vi.fn().mockResolvedValue(undefined)
  const fetchPageNotifications = vi.fn().mockImplementation(async (page = 1) => {
    if (page === 1 && pageNotifications.value.length === 0) {
      pageNotifications.value = [makeNotification('2', false)]
      pageTotal.value = 1
    }
  })
  const markAllAsRead = vi.fn().mockImplementation(async () => {
    pageNotifications.value = pageNotifications.value.map(notification => ({ ...notification, isRead: true }))
    unreadCount.value = 0
    hasUnread.value = false
  })
  const markAsRead = vi.fn().mockImplementation(async (id: string) => {
    const wasUnread = pageNotifications.value.some(notification => notification.id === id && !notification.isRead)
    pageNotifications.value = pageNotifications.value.map(notification =>
      notification.id === id ? { ...notification, isRead: true } : notification,
    )
    if (wasUnread) {
      unreadCount.value = Math.max(0, unreadCount.value - 1)
      hasUnread.value = unreadCount.value > 0
    }
  })

  const controller = useNotificationsPageController({
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
    t: (key) => key,
  })

  let seq = 0
  const emit = async (type: SSENotificationEvent['type'], data: unknown) => {
    lastSSEEvent.value = { seq: ++seq, type, data }
    await nextTick()
  }

  return {
    controller,
    pageNotifications,
    pageTotal,
    unreadCount,
    hasUnread,
    fetchPageNotifications,
    markAllAsRead,
    markAsRead,
    push,
    emit,
  }
}

describe('useNotificationsPageController', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('loads first page on init and exposes visible items', async () => {
    const { controller, fetchPageNotifications } = setupController({ initialNotifications: [] })

    await controller.init()

    expect(fetchPageNotifications).toHaveBeenCalledWith(1)
    expect(controller.visibleNotifications.value).toHaveLength(1)
    expect(controller.visibleNotifications.value[0]?.id).toBe('2')
  })

  it('keeps read filter stable when unread SSE notification arrives', async () => {
    const readNotification = makeNotification('3', true)
    const incomingUnread = makeNotification('1', false)
    const { controller, emit } = setupController({
      initialNotifications: [readNotification],
      total: 1,
      unreadCount: 0,
    })

    controller.activeFilter.value = 'read'
    await nextTick()
    await emit('notification', incomingUnread)

    expect(controller.visibleNotifications.value).toHaveLength(1)
    expect(controller.visibleNotifications.value[0]?.id).toBe('3')
  })

  it('keeps unread-filter state coherent when local mark-read is followed by duplicate SSE', async () => {
    const first = makeNotification('1', false, { sourceUrl: '/notifications/1' })
    const second = makeNotification('2', false)
    const { controller, pageNotifications, pageTotal, markAsRead, push, emit } = setupController({
      initialNotifications: [first, second],
      total: 2,
      unreadCount: 2,
    })

    controller.activeFilter.value = 'unread'
    await nextTick()

    await controller.handleClick(first)
    await nextTick()

    expect(markAsRead).toHaveBeenCalledWith('1')
    expect(push).toHaveBeenCalledWith('/notifications/1')
    expect(controller.visibleNotifications.value.map(notification => notification.id)).toEqual(['2'])
    expect(pageTotal.value).toBe(2)

    await emit('notification_read', { id: '1' })

    expect(controller.visibleNotifications.value.map(notification => notification.id)).toEqual(['2'])
    expect(pageNotifications.value.map(notification => notification.id)).toEqual(['2'])
    expect(pageTotal.value).toBe(1)
  })

  it('opens account notification hrefs on the account center', async () => {
    const notification = makeNotification('1', false, { type: 'identity_rejected' })
    const { controller, markAsRead, push } = setupController({
      initialNotifications: [notification],
      total: 1,
      unreadCount: 1,
    })

    await controller.handleClick(notification)

    expect(markAsRead).toHaveBeenCalledWith('1')
    expect(redirectMocks.identityPortalURLForHref).toHaveBeenCalledWith('/user/identity-verification')
    expect(redirectMocks.navigateToExternalURL).toHaveBeenCalledWith(
      'https://stuhelper.com/user/identity-verification',
    )
    expect(push).not.toHaveBeenCalled()
  })

  it('opens account source URLs on the account center', async () => {
    const notification = makeNotification('1', false, { sourceUrl: '/developers/apps' })
    const { controller, push } = setupController({
      initialNotifications: [notification],
      total: 1,
      unreadCount: 1,
    })

    await controller.handleClick(notification)

    expect(redirectMocks.identityPortalURLForHref).toHaveBeenCalledWith('/developers/apps')
    expect(redirectMocks.navigateToExternalURL).toHaveBeenCalledWith(
      'https://stuhelper.com/developers/apps',
    )
    expect(push).not.toHaveBeenCalled()
  })

  it('surfaces init/load-more/interaction failures via toast', async () => {
    const pageNotifications = ref([makeNotification('1', false, { sourceUrl: '/target' })])
    const pageTotal = ref(2)
    const pageLoading = ref(false)
    const pageHasMore = ref(true)
    const unreadCount = ref(1)
    const hasUnread = ref(true)
    const lastSSEEvent = ref<SSENotificationEvent>({ seq: 0, type: 'notification', data: null })
    const fetchPageNotifications = vi.fn()
      .mockRejectedValueOnce(new Error('init failed'))
      .mockRejectedValueOnce(new Error('load more failed'))
    const markAllAsRead = vi.fn().mockRejectedValue(new Error('mark all failed'))
    const markAsRead = vi.fn().mockRejectedValue(new Error('mark failed'))
    const push = vi.fn().mockResolvedValue(undefined)

    const controller = useNotificationsPageController({
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
      t: (key) => key,
    })

    await controller.init()
    await controller.loadMore()
    controller.handleMarkAllRead()
    await Promise.resolve()
    await controller.handleClick(pageNotifications.value[0]!)

    expect(mockToastError).toHaveBeenCalledTimes(4)
    expect(push).not.toHaveBeenCalled()
  })
})
