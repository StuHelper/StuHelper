import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import type { Notification } from '@stuhelper/shared/notification'

const mockGetNotifications = vi.fn()
const mockGetUnreadCount = vi.fn()
const mockMarkAsRead = vi.fn()
const mockMarkAllAsRead = vi.fn()

vi.mock('@/api', () => ({
  api: {
    notification: {
      getNotifications: mockGetNotifications,
      getUnreadCount: mockGetUnreadCount,
      markAsRead: mockMarkAsRead,
      markAllAsRead: mockMarkAllAsRead,
    },
  },
}))

// Import after mocks
const { useNotificationStore } = await import('@/stores/notification')

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

describe('useNotificationStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  describe('fetchNotifications', () => {
    it('loads first page and replaces notifications', async () => {
      const list = [
        makeNotification('1', { title: 'test1', isRead: false }),
        makeNotification('2', { title: 'test2', isRead: true }),
      ]
      mockGetNotifications.mockResolvedValue({
        data: { data: { list, total: 2 } },
      })

      const store = useNotificationStore()
      await store.fetchPageNotifications(1, 20)

      expect(store.pageNotifications).toEqual(list)
      expect(store.pageTotal).toBe(2)
      expect(store.pageLoading).toBe(false)
      expect(store.pageHasMore).toBe(false)
    })

    it('appends deduplicated items on subsequent pages', async () => {
      const store = useNotificationStore()

      mockGetNotifications.mockResolvedValue({
        data: { data: { list: [makeNotification('1', { title: 'a' })], total: 3 } },
      })
      await store.fetchPageNotifications(1, 1)

      // Page 2 with overlap
      mockGetNotifications.mockResolvedValue({
        data: {
          data: {
            list: [
              makeNotification('1', { title: 'a' }), // duplicate
              makeNotification('2', { title: 'b' }),
            ],
            total: 3,
          },
        },
      })
      await store.fetchPageNotifications(2, 2)

      expect(store.pageNotifications).toHaveLength(2) // no duplicate
      expect(store.pageHasMore).toBe(true)
    })

    it('stops pagination when an out-of-range page returns no items', async () => {
      const store = useNotificationStore()

      mockGetNotifications.mockResolvedValue({
        data: {
          data: {
            list: [makeNotification('1', { title: 'a' })],
            total: 2,
          },
        },
      })
      await store.fetchPageNotifications(1, 1)

      mockGetNotifications.mockResolvedValue({
        data: { data: { list: [], total: 2 } },
      })
      await store.fetchPageNotifications(3, 1)

      expect(store.pageNotifications).toHaveLength(1)
      expect(store.pageHasMore).toBe(false)
    })

    it('sets fetchError on failure', async () => {
      mockGetNotifications.mockRejectedValue(new Error('network error'))

      const store = useNotificationStore()
      await expect(store.fetchPageNotifications()).rejects.toThrow('network error')

      expect(store.pageFetchError).toBeTruthy()
      expect(store.pageFetchError?.message).toBe('network error')
      expect(store.pageLoading).toBe(false)
    })

    it('fails closed when page response is missing data', async () => {
      const list = [makeNotification('1', { title: 'test' })]
      mockGetNotifications
        .mockResolvedValueOnce({
          data: { data: { list, total: 1 } },
        })
        .mockResolvedValueOnce({
          data: { data: null },
        })

      const store = useNotificationStore()
      await store.fetchPageNotifications(1, 20)

      await expect(store.fetchPageNotifications(2, 20)).rejects.toThrow(
        'Invalid notification page response',
      )
      expect(store.pageNotifications).toEqual(list)
      expect(store.pageTotal).toBe(1)
      expect(store.pageFetchError?.message).toBe(
        'Invalid notification page response',
      )
      expect(store.pageLoading).toBe(false)
    })

    it('fails closed when a notification item is malformed', async () => {
      const list = [makeNotification('1', { title: 'test' })]
      mockGetNotifications
        .mockResolvedValueOnce({
          data: { data: { list, total: 1 } },
        })
        .mockResolvedValueOnce({
          data: {
            data: {
              list: [
                {
                  id: '2',
                  type: 'unsupported',
                  title: 'bad',
                  isRead: false,
                  createdAt: '2026-04-08T00:00:00Z',
                },
              ],
              total: 2,
            },
          },
        })

      const store = useNotificationStore()
      await store.fetchPageNotifications(1, 20)

      await expect(store.fetchPageNotifications(2, 20)).rejects.toThrow(
        'Invalid notification page response',
      )
      expect(store.pageNotifications).toEqual(list)
      expect(store.pageTotal).toBe(1)
    })

    it('accepts freshman notification types', async () => {
      const list = [
        makeNotification('1', { type: 'freshman_approved' }),
        makeNotification('2', { type: 'freshman_rejected' }),
        makeNotification('3', { type: 'freshman_near_expiry' }),
      ]
      mockGetNotifications.mockResolvedValue({
        data: { data: { list, total: 3 } },
      })

      const store = useNotificationStore()
      await store.fetchPageNotifications(1, 20)

      expect(store.pageNotifications.map(item => item.type)).toEqual([
        'freshman_approved',
        'freshman_rejected',
        'freshman_near_expiry',
      ])
      expect(store.pageFetchError).toBeNull()
    })

    it('normalizes invalid page params', async () => {
      mockGetNotifications.mockResolvedValue({
        data: { data: { list: [], total: 0 } },
      })

      const store = useNotificationStore()
      await store.fetchPageNotifications(-1, 0)

      expect(mockGetNotifications).toHaveBeenCalledWith(1, 20)
    })
  })

  describe('fetchBellNotifications', () => {
    it('loads a bell preview without polluting page state', async () => {
      mockGetNotifications.mockResolvedValue({
        data: {
          data: {
            list: [
              makeNotification('1', { title: 'bell-1' }),
              makeNotification('2', { title: 'bell-2' }),
            ],
            total: 2,
          },
        },
      })

      const store = useNotificationStore()
      await store.fetchBellNotifications(1, 5)

      expect(store.bellNotifications).toHaveLength(2)
      expect(store.pageNotifications).toHaveLength(0)
      expect(store.bellLoading).toBe(false)
    })

    it('fails closed when bell response is missing list data', async () => {
      mockGetNotifications.mockResolvedValue({
        data: { data: null },
      })

      const store = useNotificationStore()
      await expect(store.fetchBellNotifications(1, 5)).rejects.toThrow(
        'Invalid notification page response',
      )

      expect(store.bellNotifications).toEqual([])
      expect(store.bellLoading).toBe(false)
    })
  })

  describe('fetchUnreadCount', () => {
    it('updates unread count', async () => {
      mockGetUnreadCount.mockResolvedValue({
        data: { data: { count: 5 } },
      })

      const store = useNotificationStore()
      await store.fetchUnreadCount()

      expect(store.unreadCount).toBe(5)
      expect(store.hasUnread).toBe(true)
    })

    it('handles zero unread', async () => {
      mockGetUnreadCount.mockResolvedValue({
        data: { data: { count: 0 } },
      })

      const store = useNotificationStore()
      await store.fetchUnreadCount()

      expect(store.unreadCount).toBe(0)
      expect(store.hasUnread).toBe(false)
    })

    it('does not overwrite unread count when count response is malformed', async () => {
      mockGetUnreadCount.mockResolvedValue({
        data: { data: null },
      })

      const store = useNotificationStore()
      store.unreadCount = 7
      await store.fetchUnreadCount()

      expect(store.unreadCount).toBe(7)
    })
  })

  describe('markAsRead', () => {
    it('marks a notification as read and decrements unread count', async () => {
      mockMarkAsRead.mockResolvedValue({})
      mockGetNotifications.mockResolvedValue({
        data: {
          data: {
            list: [makeNotification('1', { title: 'test' })],
            total: 1,
          },
        },
      })

      const store = useNotificationStore()
      await store.fetchPageNotifications()
      store.unreadCount = 3

      await store.markAsRead('1')

      expect(store.pageNotifications[0].isRead).toBe(true)
      expect(store.unreadCount).toBe(2)
    })

    it('does not decrement below zero', async () => {
      mockMarkAsRead.mockResolvedValue({})
      mockGetNotifications.mockResolvedValue({
        data: {
          data: {
            list: [makeNotification('1', { title: 'test' })],
            total: 1,
          },
        },
      })

      const store = useNotificationStore()
      await store.fetchPageNotifications()
      store.unreadCount = 0

      await store.markAsRead('1')

      expect(store.unreadCount).toBe(0)
    })
  })

  describe('markAllAsRead', () => {
    it('marks all notifications as read', async () => {
      mockMarkAllAsRead.mockResolvedValue({})
      mockGetNotifications.mockResolvedValue({
        data: {
          data: {
            list: [
              makeNotification('1'),
              makeNotification('2'),
            ],
            total: 2,
          },
        },
      })

      const store = useNotificationStore()
      await store.fetchPageNotifications()
      store.unreadCount = 2

      await store.markAllAsRead()

      expect(store.pageNotifications.every(n => n.isRead)).toBe(true)
      expect(store.unreadCount).toBe(0)
    })
  })

  describe('reset', () => {
    it('resets all state', async () => {
      mockGetNotifications.mockResolvedValue({
        data: {
          data: {
            list: [makeNotification('1')],
            total: 1,
          },
        },
      })

      const store = useNotificationStore()
      await store.fetchPageNotifications()
      store.unreadCount = 5

      store.reset()

      expect(store.pageNotifications).toEqual([])
      expect(store.pageTotal).toBe(0)
      expect(store.unreadCount).toBe(0)
      expect(store.pageLoading).toBe(false)
      expect(store.pageHasMore).toBe(false)
      expect(store.pageFetchError).toBeNull()
    })
  })
})
