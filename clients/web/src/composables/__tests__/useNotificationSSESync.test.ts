import { describe, expect, it } from 'vitest'
import { ref, nextTick } from 'vue'
import { useNotificationSSESync, type PageNotification, type NotificationFilter } from '@/composables/useNotificationSSESync'
import type { SSENotificationEvent } from '@/stores/notification'

/** Helper: create refs and wire up the composable, return mutation helpers */
function setup(
  initialNotifications: PageNotification[] = [],
  initialTotal = 0,
  filter: NotificationFilter = 'all',
) {
  const pageNotifications = ref<PageNotification[]>(initialNotifications)
  const pageTotal = ref(initialTotal)
  const activeFilter = ref<NotificationFilter>(filter)
  const lastSSEEvent = ref<SSENotificationEvent>({ type: 'unread_count', data: null, seq: 0 })

  useNotificationSSESync(pageNotifications, pageTotal, activeFilter, lastSSEEvent)

  let seq = 0
  const emit = async (type: SSENotificationEvent['type'], data: unknown) => {
    lastSSEEvent.value = { type, data, seq: ++seq }
    await nextTick()
  }

  return { pageNotifications, pageTotal, activeFilter, emit }
}

describe('useNotificationSSESync (real composable)', () => {
  const n1: PageNotification = { id: '1', isRead: false }
  const n2: PageNotification = { id: '2', isRead: false }
  const n3: PageNotification = { id: '3', isRead: true }

  describe('notification event', () => {
    it('inserts new notification at head in "all" filter', async () => {
      const { pageNotifications, pageTotal, emit } = setup([n2], 1, 'all')
      await emit('notification', n1)
      expect(pageNotifications.value).toHaveLength(2)
      expect(pageNotifications.value[0].id).toBe('1')
      expect(pageTotal.value).toBe(2)
    })

    it('inserts new notification in "unread" filter', async () => {
      const { pageNotifications, pageTotal, emit } = setup([n2], 1, 'unread')
      await emit('notification', n1)
      expect(pageNotifications.value).toHaveLength(2)
      expect(pageTotal.value).toBe(2)
    })

    it('does NOT insert in "read" filter', async () => {
      const { pageNotifications, pageTotal, emit } = setup([n3], 1, 'read')
      await emit('notification', n1)
      expect(pageNotifications.value).toHaveLength(1)
      expect(pageTotal.value).toBe(1)
    })

    it('deduplicates if already present', async () => {
      const { pageNotifications, pageTotal, emit } = setup([n1, n2], 2, 'all')
      await emit('notification', n1)
      expect(pageNotifications.value).toHaveLength(2)
      expect(pageTotal.value).toBe(2)
    })

    it('ignores malformed notification events', async () => {
      const { pageNotifications, pageTotal, emit } = setup([n2], 1, 'all')
      await emit('notification', { id: 'bad' })
      expect(pageNotifications.value).toEqual([n2])
      expect(pageTotal.value).toBe(1)
    })
  })

  describe('notification_read event', () => {
    it('removes item in "unread" filter', async () => {
      const { pageNotifications, pageTotal, emit } = setup([n1, n2], 2, 'unread')
      await emit('notification_read', { id: '1' })
      expect(pageNotifications.value).toHaveLength(1)
      expect(pageTotal.value).toBe(1)
    })

    it('marks item as read in "all" filter', async () => {
      const { pageNotifications, pageTotal, emit } = setup([n1, n2], 2, 'all')
      await emit('notification_read', { id: '1' })
      expect(pageNotifications.value).toHaveLength(2)
      expect(pageNotifications.value[0].isRead).toBe(true)
      expect(pageTotal.value).toBe(2)
    })

    it('is idempotent: no double-decrement if item already removed', async () => {
      const { pageNotifications, pageTotal, emit } = setup([n2], 1, 'unread')
      await emit('notification_read', { id: '1' }) // n1 not in list
      expect(pageNotifications.value).toHaveLength(1)
      expect(pageTotal.value).toBe(1)
    })

    it('ignores malformed read events', async () => {
      const { pageNotifications, pageTotal, emit } = setup([n1], 1, 'unread')
      await emit('notification_read', { id: 1 })
      expect(pageNotifications.value).toEqual([n1])
      expect(pageTotal.value).toBe(1)
    })
  })

  describe('notification_read_all event', () => {
    it('clears list in "unread" filter', async () => {
      const { pageNotifications, pageTotal, emit } = setup([n1, n2], 2, 'unread')
      await emit('notification_read_all', null)
      expect(pageNotifications.value).toHaveLength(0)
      expect(pageTotal.value).toBe(0)
    })

    it('marks all as read in "all" filter', async () => {
      const { pageNotifications, pageTotal, emit } = setup([n1, n2], 2, 'all')
      await emit('notification_read_all', null)
      expect(pageNotifications.value.every(n => n.isRead)).toBe(true)
      expect(pageTotal.value).toBe(2)
    })

    it('is idempotent on empty list in "unread" filter', async () => {
      const { pageNotifications, pageTotal, emit } = setup([], 0, 'unread')
      await emit('notification_read_all', null)
      expect(pageNotifications.value).toHaveLength(0)
      expect(pageTotal.value).toBe(0)
    })
  })

  describe('notification_deleted event', () => {
    it('removes item from list', async () => {
      const { pageNotifications, pageTotal, emit } = setup([n1, n2], 2, 'all')
      await emit('notification_deleted', { id: '1' })
      expect(pageNotifications.value).toHaveLength(1)
      expect(pageTotal.value).toBe(1)
    })

    it('is idempotent: no double-decrement if item already removed', async () => {
      const { pageNotifications, pageTotal, emit } = setup([n2], 1, 'all')
      await emit('notification_deleted', { id: '1' })
      expect(pageNotifications.value).toHaveLength(1)
      expect(pageTotal.value).toBe(1)
    })
  })

  describe('filter switching', () => {
    it('respects dynamic filter changes', async () => {
      const { pageNotifications, pageTotal, activeFilter, emit } = setup([n1], 1, 'all')

      // Switch to "read" filter, new notification should NOT be inserted
      activeFilter.value = 'read'
      await nextTick()
      await emit('notification', n2)
      expect(pageNotifications.value).toHaveLength(1)
      expect(pageTotal.value).toBe(1)

      // Switch back to "all", new notification should be inserted
      activeFilter.value = 'all'
      await nextTick()
      await emit('notification', n2)
      expect(pageNotifications.value).toHaveLength(2)
      expect(pageTotal.value).toBe(2)
    })
  })
})
