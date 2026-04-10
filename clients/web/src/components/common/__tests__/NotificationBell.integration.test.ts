// @vitest-environment jsdom

import { beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent } from 'vue'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createI18n } from 'vue-i18n'
import { createMemoryHistory, createRouter } from 'vue-router'
import type { Notification } from '@/types/notification'

const { useNotificationStore } = await import('@/stores/notification')
const NotificationBell = (await import('@/components/common/NotificationBell.vue')).default

const NotificationListStub = defineComponent({
  name: 'NotificationListStub',
  props: {
    notifications: { type: Array, default: () => [] },
    loading: { type: Boolean, default: false },
  },
  emits: ['click'],
  template: `
    <div class="notification-list-stub">
      <div v-if="loading" class="loading">loading</div>
      <button
        v-for="notification in notifications"
        :key="notification.id"
        class="notification-list-item"
        :data-id="notification.id"
        @click="$emit('click', notification)"
      >
        {{ notification.id }}
      </button>
    </div>
  `,
})

const messages = {
  en: {
    user: {
      notification: {
        bell: 'Notifications',
        bellMarkAllRead: 'Mark all read',
        viewAll: 'View all',
        empty: 'Empty',
      },
    },
  },
}

function makeNotification(id: string, overrides: Partial<Notification> = {}): Notification {
  return {
    id,
    type: 'reply' as const,
    title: `notification-${id}`,
    isRead: false,
    createdAt: '2026-04-08T00:00:00Z',
    ...overrides,
  }
}

async function mountBell() {
  const pinia = createPinia()
  setActivePinia(pinia)
  const store = useNotificationStore(pinia)
  store.connectSSE = vi.fn()
  store.stopPolling = vi.fn()
  store.fetchBellNotifications = vi.fn().mockResolvedValue(undefined)

  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/', component: { template: '<div />' } },
      { path: '/notifications', component: { template: '<div />' } },
      { path: '/target', component: { template: '<div />' } },
    ],
  })
  await router.push('/')
  await router.isReady()

  const i18n = createI18n({
    legacy: false,
    locale: 'en',
    messages,
  })

  const wrapper = mount(NotificationBell, {
    global: {
      plugins: [pinia, router, i18n],
      stubs: {
        NotificationList: NotificationListStub,
        transition: false,
      },
    },
  })

  await flushPromises()

  return { wrapper, store, router }
}

describe('NotificationBell.vue integration', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('connects SSE on mount and stops polling on unmount', async () => {
    const pinia = createPinia()
    setActivePinia(pinia)
    const store = useNotificationStore(pinia)
    const connectSSE = vi.fn()
    const stopPolling = vi.fn()
    store.connectSSE = connectSSE
    store.stopPolling = stopPolling

    const router = createRouter({
      history: createMemoryHistory(),
      routes: [{ path: '/', component: { template: '<div />' } }],
    })
    await router.push('/')
    await router.isReady()

    const i18n = createI18n({
      legacy: false,
      locale: 'en',
      messages,
    })

    const wrapper = mount(NotificationBell, {
      global: {
        plugins: [pinia, router, i18n],
        stubs: {
          NotificationList: NotificationListStub,
          transition: false,
        },
      },
    })

    expect(connectSSE).toHaveBeenCalledTimes(1)

    wrapper.unmount()
    expect(stopPolling).toHaveBeenCalledTimes(1)
  })

  it('fetches history on first open even if store already has SSE-inserted items, and only once', async () => {
    const { wrapper, store } = await mountBell()
    const fetchBellNotifications = vi.fn().mockResolvedValue(undefined)
    store.fetchBellNotifications = fetchBellNotifications
    store.bellNotifications = [makeNotification('sse-1')]
    await flushPromises()

    const bellButton = wrapper.get('button[aria-label="Notifications"]')

    await bellButton.trigger('click')
    expect(fetchBellNotifications).toHaveBeenCalledTimes(1)
    expect(fetchBellNotifications).toHaveBeenCalledWith(1, 5)

    await bellButton.trigger('click') // close
    await bellButton.trigger('click') // reopen
    expect(fetchBellNotifications).toHaveBeenCalledTimes(1)
  })

  it('clicking a notification marks it read, closes panel, and routes to its href', async () => {
    const { wrapper, store, router } = await mountBell()
    const markAsRead = vi.fn().mockResolvedValue(undefined)
    store.markAsRead = markAsRead
    store.bellNotifications = [makeNotification('1', { sourceUrl: '/target' })]
    await flushPromises()

    const bellButton = wrapper.get('button[aria-label="Notifications"]')
    await bellButton.trigger('click')
    await flushPromises()

    const itemButton = wrapper.get('.notification-list-item')
    await itemButton.trigger('click')
    await flushPromises()
    await flushPromises()

    expect(markAsRead).toHaveBeenCalledWith('1')
    expect(router.currentRoute.value.fullPath).toBe('/target')
    expect(wrapper.get('button[aria-label="Notifications"]').attributes('aria-expanded')).toBe('false')
  })

  it('clicking mark-all-read button delegates to store', async () => {
    const { wrapper, store } = await mountBell()
    const markAllAsRead = vi.fn().mockResolvedValue(undefined)
    store.markAllAsRead = markAllAsRead
    store.unreadCount = 2
    store.bellNotifications = [makeNotification('1'), makeNotification('2')]
    await flushPromises()

    const bellButton = wrapper.get('button[aria-label="Notifications"]')
    await bellButton.trigger('click')
    await flushPromises()

    const markAllButton = wrapper.get('button.text-xs')
    await markAllButton.trigger('click')

    expect(markAllAsRead).toHaveBeenCalledTimes(1)
  })
})
