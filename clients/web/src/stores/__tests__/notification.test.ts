import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'

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
        { id: '1', message: 'test1', isRead: false },
        { id: '2', message: 'test2', isRead: true },
      ]
      mockGetNotifications.mockResolvedValue({
        data: { data: { list, total: 2 } },
      })

      const store = useNotificationStore()
      await store.fetchNotifications(1, 20)

      expect(store.notifications).toEqual(list)
      expect(store.total).toBe(2)
      expect(store.loading).toBe(false)
      expect(store.hasMore).toBe(false)
    })

    it('appends deduplicated items on subsequent pages', async () => {
      const store = useNotificationStore()

      mockGetNotifications.mockResolvedValue({
        data: { data: { list: [{ id: '1', message: 'a', isRead: false }], total: 3 } },
      })
      await store.fetchNotifications(1, 1)

      // Page 2 with overlap
      mockGetNotifications.mockResolvedValue({
        data: {
          data: {
            list: [
              { id: '1', message: 'a', isRead: false }, // duplicate
              { id: '2', message: 'b', isRead: false },
            ],
            total: 3,
          },
        },
      })
      await store.fetchNotifications(2, 2)

      expect(store.notifications).toHaveLength(2) // no duplicate
      expect(store.hasMore).toBe(true)
    })

    it('sets fetchError on failure', async () => {
      mockGetNotifications.mockRejectedValue(new Error('network error'))

      const store = useNotificationStore()
      await expect(store.fetchNotifications()).rejects.toThrow('network error')

      expect(store.fetchError).toBeTruthy()
      expect(store.fetchError?.message).toBe('network error')
      expect(store.loading).toBe(false)
    })

    it('normalizes invalid page params', async () => {
      mockGetNotifications.mockResolvedValue({
        data: { data: { list: [], total: 0 } },
      })

      const store = useNotificationStore()
      await store.fetchNotifications(-1, 0)

      expect(mockGetNotifications).toHaveBeenCalledWith(1, 20)
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
  })

  describe('markAsRead', () => {
    it('marks a notification as read and decrements unread count', async () => {
      mockMarkAsRead.mockResolvedValue({})
      mockGetNotifications.mockResolvedValue({
        data: {
          data: {
            list: [{ id: '1', message: 'test', isRead: false }],
            total: 1,
          },
        },
      })

      const store = useNotificationStore()
      await store.fetchNotifications()
      store.unreadCount = 3

      await store.markAsRead('1')

      expect(store.notifications[0].isRead).toBe(true)
      expect(store.unreadCount).toBe(2)
    })

    it('does not decrement below zero', async () => {
      mockMarkAsRead.mockResolvedValue({})
      mockGetNotifications.mockResolvedValue({
        data: {
          data: {
            list: [{ id: '1', message: 'test', isRead: false }],
            total: 1,
          },
        },
      })

      const store = useNotificationStore()
      await store.fetchNotifications()
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
              { id: '1', isRead: false },
              { id: '2', isRead: false },
            ],
            total: 2,
          },
        },
      })

      const store = useNotificationStore()
      await store.fetchNotifications()
      store.unreadCount = 2

      await store.markAllAsRead()

      expect(store.notifications.every(n => n.isRead)).toBe(true)
      expect(store.unreadCount).toBe(0)
    })
  })

  describe('reset', () => {
    it('resets all state', async () => {
      mockGetNotifications.mockResolvedValue({
        data: {
          data: {
            list: [{ id: '1', isRead: false }],
            total: 1,
          },
        },
      })

      const store = useNotificationStore()
      await store.fetchNotifications()
      store.unreadCount = 5

      store.reset()

      expect(store.notifications).toEqual([])
      expect(store.total).toBe(0)
      expect(store.unreadCount).toBe(0)
      expect(store.loading).toBe(false)
      expect(store.hasMore).toBe(true)
      expect(store.fetchError).toBeNull()
    })
  })
})
