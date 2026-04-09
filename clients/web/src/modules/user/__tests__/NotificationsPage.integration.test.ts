// @vitest-environment jsdom

import { beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent, nextTick } from 'vue'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createI18n } from 'vue-i18n'
import { createMemoryHistory, createRouter } from 'vue-router'

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
const NotificationsPage = (await import('@/modules/user/views/NotificationsPage.vue')).default

const NotificationItemStub = defineComponent({
  name: 'NotificationItemStub',
  props: {
    notification: { type: Object, required: true },
    deletable: { type: Boolean, default: false },
  },
  emits: ['click', 'delete'],
  template: `
    <div class="notification-item" :data-id="notification.id" :data-read="String(notification.isRead)">
      <button class="item-click" @click="$emit('click', notification)">
        {{ notification.id }}
      </button>
      <button v-if="deletable" class="item-delete" @click="$emit('delete', notification)">
        delete
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
      },
    },
  },
}

function makeNotification(id: string, isRead: boolean) {
  return {
    id,
    type: 'reply',
    title: `notification-${id}`,
    isRead,
  }
}

async function mountPage() {
  const pinia = createPinia()
  setActivePinia(pinia)

  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/notifications', component: { template: '<div />' } },
      { path: '/notifications/preferences', component: { template: '<div />' } },
    ],
  })
  await router.push('/notifications')
  await router.isReady()

  const i18n = createI18n({
    legacy: false,
    locale: 'en',
    messages,
  })

  const wrapper = mount(NotificationsPage, {
    global: {
      plugins: [pinia, router, i18n],
      stubs: {
        NotificationItem: NotificationItemStub,
        InfiniteScroll: InfiniteScrollStub,
        EmptyState: EmptyStateStub,
      },
    },
  })

  await flushPromises()

  return {
    wrapper,
    store: useNotificationStore(pinia),
  }
}

describe('NotificationsPage.vue integration', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders initial API data and updates rendered list on notification SSE', async () => {
    const n1 = makeNotification('1', false)
    const n2 = makeNotification('2', false)

    mockGetNotifications.mockResolvedValueOnce({
      data: { data: { list: [n2], total: 1, unread: 1 } },
    })

    const { wrapper, store } = await mountPage()

    let items = wrapper.findAll('.notification-item')
    expect(items).toHaveLength(1)
    expect(items[0].attributes('data-id')).toBe('2')

    store.lastSSEEvent = { type: 'notification', data: n1, seq: 1 }
    await nextTick()

    items = wrapper.findAll('.notification-item')
    expect(items).toHaveLength(2)
    expect(items[0].attributes('data-id')).toBe('1')
    expect(items[1].attributes('data-id')).toBe('2')
  })

  it('does not render new unread notification while "read" filter is active', async () => {
    const n1 = makeNotification('1', false)
    const n3 = makeNotification('3', true)

    mockGetNotifications
      .mockResolvedValueOnce({
        data: { data: { list: [n3], total: 1, unread: 0 } },
      })
      .mockResolvedValueOnce({
        data: { data: { list: [n3], total: 1, unread: 0 } },
      })

    const { wrapper, store } = await mountPage()

    const buttons = wrapper.findAll('button')
    const readButton = buttons.find(button => button.text() === 'Read')
    expect(readButton).toBeTruthy()
    await readButton!.trigger('click')
    await flushPromises()

    store.lastSSEEvent = { type: 'notification', data: n1, seq: 1 }
    await nextTick()

    const items = wrapper.findAll('.notification-item')
    expect(items).toHaveLength(1)
    expect(items[0].attributes('data-id')).toBe('3')
  })

  it('prevents double-decrement when local unread mark-read is followed by the same SSE event', async () => {
    const n1 = makeNotification('1', false)
    const n2 = makeNotification('2', false)

    mockGetNotifications
      .mockResolvedValueOnce({
        data: { data: { list: [n1, n2], total: 2, unread: 2 } },
      })
      .mockResolvedValueOnce({
        data: { data: { list: [n1, n2], total: 2, unread: 2 } },
      })
    mockMarkAsRead.mockResolvedValue({})

    const { wrapper, store } = await mountPage()

    const buttons = wrapper.findAll('button')
    const unreadButton = buttons.find(button => button.text() === 'Unread')
    expect(unreadButton).toBeTruthy()
    await unreadButton!.trigger('click')
    await flushPromises()

    let items = wrapper.findAll('.notification-item')
    expect(items).toHaveLength(2)

    await items[0].find('.item-click').trigger('click')
    await flushPromises()

    items = wrapper.findAll('.notification-item')
    expect(items).toHaveLength(1)
    expect(items[0].attributes('data-id')).toBe('2')

    store.lastSSEEvent = { type: 'notification_read', data: { id: '1' }, seq: 1 }
    await nextTick()

    items = wrapper.findAll('.notification-item')
    expect(items).toHaveLength(1)
    expect(items[0].attributes('data-id')).toBe('2')
  })
})
