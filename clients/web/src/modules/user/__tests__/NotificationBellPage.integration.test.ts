// @vitest-environment jsdom

import { beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent, nextTick } from 'vue'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createI18n } from 'vue-i18n'
import { createMemoryHistory, createRouter } from 'vue-router'
import type { Notification } from '@/types/notification'

const mockGetNotifications = vi.fn()
const mockMarkAsRead = vi.fn()
const mockMarkAllAsRead = vi.fn()
const mockDeleteNotification = vi.fn()

vi.mock('@/api', () => ({
  api: {
    notification: {
      getNotifications: mockGetNotifications,
      markAsRead: mockMarkAsRead,
      markAllAsRead: mockMarkAllAsRead,
      deleteNotification: mockDeleteNotification,
    },
  },
  NOTIFICATION_STREAM_PATH: '/api/v1/course/review/user/notifications/stream',
}))

const { useNotificationStore } = await import('@/stores/notification')
const NotificationBell = (await import('@/components/common/NotificationBell.vue')).default
const NotificationsPage = (await import('@/modules/user/views/NotificationsPage.vue')).default

const NotificationItemStub = defineComponent({
  name: 'NotificationItemStub',
  props: {
    notification: { type: Object, required: true },
    deletable: { type: Boolean, default: false },
  },
  emits: ['click', 'delete'],
  template: `
    <div class="page-item" :data-id="notification.id" :data-read="String(notification.isRead)">
      <button class="page-item-click" @click="$emit('click', notification)">
        {{ notification.id }}
      </button>
      <button v-if="deletable" class="page-item-delete" @click="$emit('delete', notification)">
        delete
      </button>
    </div>
  `,
})

const NotificationListStub = defineComponent({
  name: 'NotificationListStub',
  props: {
    notifications: { type: Array, default: () => [] },
    loading: { type: Boolean, default: false },
  },
  emits: ['click'],
  template: `
    <div class="bell-list">
      <div v-if="loading" class="bell-loading">loading</div>
      <button
        v-for="notification in notifications"
        :key="notification.id"
        class="bell-item"
        :data-id="notification.id"
        :data-read="String(notification.isRead)"
        @click="$emit('click', notification)"
      >
        {{ notification.id }}
      </button>
    </div>
  `,
})

const InfiniteScrollStub = defineComponent({
  name: 'InfiniteScrollStub',
  props: {
    loading: { type: Boolean, default: false },
    hasMore: { type: Boolean, default: false },
  },
  emits: ['load-more'],
  template: '<div class="infinite-scroll"><slot /></div>',
})

const EmptyStateStub = defineComponent({
  name: 'EmptyStateStub',
  props: {
    title: { type: String, default: '' },
    description: { type: String, default: '' },
  },
  template: '<div class="empty-state">{{ title }}|{{ description }}</div>',
})

const TestHost = defineComponent({
  name: 'NotificationBellPageTestHost',
  components: { NotificationBell, NotificationsPage },
  template: `
    <div>
      <NotificationBell />
      <NotificationsPage />
    </div>
  `,
})

const messages = {
  en: {
    user: {
      notification: {
        title: 'Notifications',
        unreadCount: 'Unread {count}',
        preferences: 'Preferences',
        markAllRead: 'Mark all read',
        filterAll: 'All',
        filterUnread: 'Unread',
        filterRead: 'Read',
        empty: 'Empty',
        emptyDesc: 'Nothing here',
        bell: 'Notifications Bell',
        bellMarkAllRead: 'Bell mark all read',
        viewAll: 'View all',
      },
    },
  },
}

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

async function mountHost() {
  const pinia = createPinia()
  setActivePinia(pinia)
  const store = useNotificationStore(pinia)

  // Avoid real SSE/polling side effects in jsdom; tests drive store state directly.
  store.connectSSE = vi.fn()
  store.stopPolling = vi.fn()
  store.fetchPageNotifications = vi.fn().mockResolvedValue(undefined)
  store.fetchBellNotifications = vi.fn().mockResolvedValue(undefined)

  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/', component: { template: '<div />' } },
      { path: '/notifications', component: { template: '<div />' } },
      { path: '/notifications/preferences', component: { template: '<div />' } },
      { path: '/target', component: { template: '<div />' } },
    ],
  })
  await router.push('/notifications')
  await router.isReady()

  const i18n = createI18n({
    legacy: false,
    locale: 'en',
    messages,
  })

  const wrapper = mount(TestHost, {
    global: {
      plugins: [pinia, router, i18n],
      stubs: {
        NotificationItem: NotificationItemStub,
        NotificationList: NotificationListStub,
        InfiniteScroll: InfiniteScrollStub,
        EmptyState: EmptyStateStub,
        transition: false,
      },
    },
  })

  await flushPromises()

  return { wrapper, store }
}

function findButtonByText(wrapper: ReturnType<typeof mount>, text: string) {
  return wrapper.findAll('button').find(button => button.text() === text)
}

describe('NotificationBell + NotificationsPage shared-store integration', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('switching page filter does not pollute bell global list', async () => {
    const unread = makeNotification('1')
    const read = makeNotification('2', { isRead: true })

    const { wrapper, store } = await mountHost()
    store.pageNotifications = [unread, read]
    store.bellNotifications = [unread, read]
    store.unreadCount = 1
    await nextTick()

    const readFilter = findButtonByText(wrapper, 'Read')
    expect(readFilter).toBeTruthy()
    await readFilter!.trigger('click')
    await flushPromises()

    const pageItems = wrapper.findAll('.page-item')
    expect(pageItems).toHaveLength(1)
    expect(pageItems[0].attributes('data-id')).toBe('2')

    const bellButton = wrapper.get('button[aria-label="Notifications Bell"]')
    await bellButton.trigger('click')
    await flushPromises()

    const bellItems = wrapper.findAll('.bell-item')
    expect(store.fetchBellNotifications).toHaveBeenCalledWith(1, 5)
    expect(bellItems).toHaveLength(2)
    expect(bellItems.map(item => item.attributes('data-id'))).toEqual(['1', '2'])
  })

  it('shared SSE notification updates both bell and page in the same app state', async () => {
    const existing = makeNotification('2')
    const incoming = makeNotification('1')

    const { wrapper, store } = await mountHost()
    store.pageNotifications = [existing]
    store.bellNotifications = [existing]
    store.unreadCount = 1
    await nextTick()

    const bellButton = wrapper.get('button[aria-label="Notifications Bell"]')
    await bellButton.trigger('click')
    await flushPromises()

    expect(wrapper.findAll('.page-item').map(item => item.attributes('data-id'))).toEqual(['2'])
    expect(wrapper.findAll('.bell-item').map(item => item.attributes('data-id'))).toEqual(['2'])

    store.lastSSEEvent = { type: 'notification', data: incoming, seq: 1 }
    await nextTick()

    expect(wrapper.findAll('.page-item').map(item => item.attributes('data-id'))).toEqual(['1', '2'])
    expect(wrapper.findAll('.bell-item').map(item => item.attributes('data-id'))).toEqual(['1', '2'])
  })

  it('page unread action removes local item but keeps bell global item, and duplicate SSE stays idempotent', async () => {
    const n1 = makeNotification('1')
    const n2 = makeNotification('2')

    const { wrapper, store } = await mountHost()
    store.pageNotifications = [n1, n2]
    store.bellNotifications = [n1, n2]
    store.unreadCount = 2
    await nextTick()

    const bellButton = wrapper.get('button[aria-label="Notifications Bell"]')
    await bellButton.trigger('click')
    await flushPromises()

    const unreadFilter = findButtonByText(wrapper, 'Unread')
    expect(unreadFilter).toBeTruthy()
    await unreadFilter!.trigger('click')
    await flushPromises()

    let pageItems = wrapper.findAll('.page-item')
    expect(pageItems.map(item => item.attributes('data-id'))).toEqual(['1', '2'])
    expect(wrapper.findAll('.bell-item').map(item => item.attributes('data-id'))).toEqual(['1', '2'])

    await pageItems[0].find('.page-item-click').trigger('click')
    await flushPromises()

    pageItems = wrapper.findAll('.page-item')
    expect(pageItems.map(item => item.attributes('data-id'))).toEqual(['2'])

    const bellItemsAfterLocalRead = wrapper.findAll('.bell-item')
    expect(bellItemsAfterLocalRead).toHaveLength(2)
    expect(bellItemsAfterLocalRead[0].attributes('data-id')).toBe('1')
    expect(bellItemsAfterLocalRead[0].attributes('data-read')).toBe('true')

    store.lastSSEEvent = { type: 'notification_read', data: { id: '1' }, seq: 1 }
    await nextTick()

    expect(wrapper.findAll('.page-item').map(item => item.attributes('data-id'))).toEqual(['2'])
    expect(wrapper.findAll('.bell-item')).toHaveLength(2)
  })
})
