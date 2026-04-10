<template>
  <div class="max-w-[600px] mx-auto p-6 animate-fade-in">
    <div class="flex items-start justify-between gap-4 mb-4 pb-4 border-b border-border-light">
      <div>
        <h1 class="font-sans text-xl font-extrabold tracking-tight text-text-primary m-0">
          {{ t('user.notification.title') }}
          <!-- L-42: 未读数量变化时屏幕阅读器播报 -->
          <span v-if="hasUnread" class="sr-only" aria-live="polite" role="status">
            {{ t('user.notification.unreadCount', { count: unreadCount }) }}
          </span>
        </h1>
        <div class="mt-3 flex items-center gap-2">
          <button
            v-for="option in filterOptions"
            :key="option.value"
            type="button"
            class="rounded-full px-3 py-1 text-sm border transition-colors duration-fast"
            :class="activeFilter === option.value
              ? 'border-text-primary text-text-primary'
              : 'border-border-light text-text-muted hover:text-text-primary hover:border-text-primary'"
            @click="activeFilter = option.value"
          >
            {{ option.label }}
          </button>
        </div>
      </div>

      <button
        v-if="hasUnread"
        type="button"
        class="py-1 px-3 bg-transparent rounded-full text-text-muted text-sm cursor-pointer transition-all duration-fast hover:border-text-primary hover:text-text-primary"
        @click="handleMarkAllRead"
      >
        {{ t('user.notification.markAllRead') }}
      </button>
    </div>

    <InfiniteScroll
      :loading="loading"
      :has-more="hasMore"
      @load-more="loadMore"
    >
      <div v-if="visibleNotifications.length > 0" class="border-0 rounded-md overflow-hidden">
        <NotificationItem
          v-for="(n, index) in visibleNotifications"
          :key="n.id"
          :notification="n"
          class="stagger-item"
          :style="{ animationDelay: `${Math.min(index, 10) * 50}ms` }"
          @click="handleClick(n)"
        />
      </div>
      <EmptyState
        v-else-if="!loading"
        :title="t('user.notification.empty')"
        :description="t('user.notification.emptyDesc')"
      />
    </InfiniteScroll>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { storeToRefs } from 'pinia'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { useNotificationStore } from '@/stores/notification'
import type { Notification } from '@/types/notification'
import NotificationItem from '@/components/common/NotificationItem.vue'
import InfiniteScroll from '@/components/common/InfiniteScroll.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import { useNotificationSSESync, type NotificationFilter } from '@/composables/useNotificationSSESync'

const router = useRouter()
const { t } = useI18n()
const store = useNotificationStore()
const {
  pageNotifications,
  pageTotal,
  pageLoading,
  pageHasMore,
  unreadCount,
  hasUnread,
  lastSSEEvent,
} = storeToRefs(store)

const activeFilter = ref<NotificationFilter>('all')

useNotificationSSESync(pageNotifications, pageTotal, activeFilter, lastSSEEvent)

const filterOptions = [
  { value: 'all' as const, label: t('user.notification.filterAll') },
  { value: 'unread' as const, label: t('user.notification.filterUnread') },
  { value: 'read' as const, label: t('user.notification.filterRead') },
]

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
const hasMore = computed(() => activeFilter.value === 'all' && pageHasMore.value)
const page = ref(1)

onMounted(() => {
  void store.fetchPageNotifications(1).catch(() => {})
})

const loadMore = async () => {
  if (loading.value || !hasMore.value) return
  page.value++
  try {
    await store.fetchPageNotifications(page.value)
  } catch {
    page.value = Math.max(1, page.value - 1)
  }
}

const handleMarkAllRead = () => {
  store.markAllAsRead().catch(() => {})
}

const handleClick = async (n: Notification) => {
  store.markAsRead(n.id).catch(() => {})

  if (n.courseID) {
    router.push(`/courses/${n.courseID}/reviews`)
    return
  }

  if (!n.relatedType || !n.relatedID) return

  if (n.relatedType === 'course') {
    router.push(`/courses/${n.relatedID}/reviews`)
    return
  }

  if (n.relatedType === 'review') {
    router.push({ name: 'user-reviews' })
  }
}
</script>
