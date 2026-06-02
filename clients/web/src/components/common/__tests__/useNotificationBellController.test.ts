// @vitest-environment jsdom

import { beforeEach, describe, expect, it, vi } from 'vitest'
import { reactive, ref } from 'vue'
import type { Notification } from '@stuhelper/shared/notification'
import { useNotificationBellController, type NotificationBellStore } from '../useNotificationBellController'

const mockToastError = vi.fn()
const redirectMocks = vi.hoisted(() => ({
  accountCenterURLForHref: vi.fn((href: string) =>
    href.startsWith('/user/') ? `https://stuhelper.com${href}` : null,
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
  accountCenterURLForHref: redirectMocks.accountCenterURLForHref,
  navigateToExternalURL: redirectMocks.navigateToExternalURL,
}))

function makeNotification(id: string, overrides: Partial<Notification> = {}): Notification {
  return {
    id,
    type: 'reply',
    title: `notification-${id}`,
    isRead: false,
    createdAt: '2026-04-08T00:00:00Z',
    ...overrides,
  }
}

function makeStore(overrides: Partial<NotificationBellStore> = {}) {
  return reactive({
    bellNotifications: [] as Notification[],
    unreadCount: 0,
    hasUnread: false,
    bellLoading: false,
    connectSSE: vi.fn(),
    stopPolling: vi.fn(),
    fetchBellNotifications: vi.fn().mockResolvedValue(undefined),
    markAllAsRead: vi.fn().mockResolvedValue(undefined),
    markAsRead: vi.fn().mockResolvedValue(undefined),
    ...overrides,
  }) as NotificationBellStore
}

describe('useNotificationBellController', () => {
  beforeEach(() => {
    document.body.innerHTML = ''
    vi.clearAllMocks()
  })

  it('loads bell history on first open only once even if SSE already inserted items', async () => {
    const store = makeStore({
      bellNotifications: [makeNotification('sse-1')],
      unreadCount: 1,
      hasUnread: true,
    })
    const root = document.createElement('div')
    document.body.appendChild(root)
    const rootRef = ref<HTMLElement | null>(root)

    const controller = useNotificationBellController({
      rootRef,
      store,
      router: { push: vi.fn() },
    })

    await controller.togglePanel()
    expect(controller.showPanel.value).toBe(true)
    expect(store.fetchBellNotifications).toHaveBeenCalledTimes(1)
    expect(store.fetchBellNotifications).toHaveBeenCalledWith(1, 5)
    expect(controller.notifications.value[0]?.id).toBe('sse-1')

    await controller.togglePanel()
    await controller.togglePanel()
    expect(store.fetchBellNotifications).toHaveBeenCalledTimes(1)
  })

  it('marks clicked notification as read, closes panel, and routes when href exists', async () => {
    const push = vi.fn().mockResolvedValue(undefined)
    const store = makeStore({
      bellNotifications: [makeNotification('1', { sourceUrl: '/target' })],
      unreadCount: 1,
      hasUnread: true,
    })
    const rootRef = ref<HTMLElement | null>(document.createElement('div'))

    const controller = useNotificationBellController({
      rootRef,
      store,
      router: { push },
    })

    controller.showPanel.value = true
    await controller.handleNotificationClick(store.bellNotifications[0]!)

    expect(store.markAsRead).toHaveBeenCalledWith('1')
    expect(push).toHaveBeenCalledWith('/target')
    expect(controller.showPanel.value).toBe(false)
  })

  it('opens account notification hrefs on the account center', async () => {
    const push = vi.fn().mockResolvedValue(undefined)
    const store = makeStore({
      bellNotifications: [makeNotification('1', { type: 'student_rejected' })],
      unreadCount: 1,
      hasUnread: true,
    })
    const rootRef = ref<HTMLElement | null>(document.createElement('div'))

    const controller = useNotificationBellController({
      rootRef,
      store,
      router: { push },
    })

    controller.showPanel.value = true
    await controller.handleNotificationClick(store.bellNotifications[0]!)

    expect(store.markAsRead).toHaveBeenCalledWith('1')
    expect(redirectMocks.accountCenterURLForHref).toHaveBeenCalledWith('/user/student-verification')
    expect(redirectMocks.navigateToExternalURL).toHaveBeenCalledWith(
      'https://stuhelper.com/user/student-verification',
    )
    expect(push).not.toHaveBeenCalled()
    expect(controller.showPanel.value).toBe(false)
  })

  it('delegates mark-all-read to the store', async () => {
    const store = makeStore({
      unreadCount: 2,
      hasUnread: true,
    })

    const controller = useNotificationBellController({
      rootRef: ref(document.createElement('div')),
      store,
      router: { push: vi.fn() },
    })

    await controller.handleMarkAllRead()
    expect(store.markAllAsRead).toHaveBeenCalledTimes(1)
  })

  it('surfaces bell interactions failures via toast', async () => {
    const store = makeStore({
      fetchBellNotifications: vi.fn().mockRejectedValue(new Error('load failed')),
      markAllAsRead: vi.fn().mockRejectedValue(new Error('mark all failed')),
      markAsRead: vi.fn().mockRejectedValue(new Error('mark failed')),
      bellNotifications: [makeNotification('1', { sourceUrl: '/target' })],
    })

    const controller = useNotificationBellController({
      rootRef: ref(document.createElement('div')),
      store,
      router: { push: vi.fn() },
    })

    await controller.togglePanel()
    await controller.handleMarkAllRead()
    await controller.handleNotificationClick(store.bellNotifications[0]!)

    expect(mockToastError).toHaveBeenCalledTimes(3)
    expect(mockToastError).toHaveBeenCalledWith('notification-error')
  })

  it('closes only on clicks outside the bell root', () => {
    const root = document.createElement('div')
    const inside = document.createElement('button')
    root.appendChild(inside)
    document.body.appendChild(root)
    const outside = document.createElement('div')
    document.body.appendChild(outside)

    const controller = useNotificationBellController({
      rootRef: ref(root),
      store: makeStore(),
      router: { push: vi.fn() },
    })

    controller.showPanel.value = true
    controller.handleDocumentClick(new MouseEvent('click', { bubbles: true }))
    expect(controller.showPanel.value).toBe(false)

    controller.showPanel.value = true
    Object.defineProperty(window, 'event', { configurable: true, value: undefined })
    controller.handleDocumentClick({ target: inside } as MouseEvent)
    expect(controller.showPanel.value).toBe(true)

    controller.handleDocumentClick({ target: outside } as MouseEvent)
    expect(controller.showPanel.value).toBe(false)
  })

  it('delegates lifecycle hooks to the store', () => {
    const store = makeStore()
    const controller = useNotificationBellController({
      rootRef: ref(document.createElement('div')),
      store,
      router: { push: vi.fn() },
    })

    controller.start()
    controller.stop()

    expect(store.connectSSE).toHaveBeenCalledTimes(1)
    expect(store.stopPolling).toHaveBeenCalledTimes(1)
  })
})
