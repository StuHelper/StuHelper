<template>
  <div class="max-w-[600px] mx-auto p-6 animate-fade-in">
    <div class="flex items-center justify-between mb-6 pb-4 border-b border-border-light">
      <h1 class="font-sans text-xl font-extrabold tracking-tight text-text-primary m-0">
        {{ t('user.notification.title') }}
        <!-- L-42: 未读数量变化时屏幕阅读器播报 -->
        <span v-if="hasUnread" class="sr-only" aria-live="polite" role="status">
          {{ t('user.notification.unreadCount', { count: store.unreadCount }) }}
        </span>
      </h1>
      <button
        v-if="hasUnread"
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
      <div v-if="notifications.length > 0" class="border-0 rounded-md overflow-hidden">
        <NotificationItem
          v-for="(n, index) in notifications"
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
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { useNotificationStore } from '@/stores/notification'
import type { Notification } from '@/types/notification'
import NotificationItem from '@/components/common/NotificationItem.vue'
import InfiniteScroll from '@/components/common/InfiniteScroll.vue'
import EmptyState from '@/components/common/EmptyState.vue'

const router = useRouter()
const { t } = useI18n()
const store = useNotificationStore()

const notifications = computed(() => store.notifications)
const loading = computed(() => store.loading)
const hasMore = computed(() => store.hasMore)
const hasUnread = computed(() => store.hasUnread)

const page = ref(1)

onMounted(() => {
  void store.fetchNotifications(1).catch(() => {})
})

const loadMore = async () => {
  // 防止上一请求未完成时重复翻页
  if (loading.value || !hasMore.value) return
  page.value++
  try {
    await store.fetchNotifications(page.value)
  } catch {
    page.value = Math.max(1, page.value - 1)
  }
}

const handleMarkAllRead = () => {
  store.markAllAsRead().catch(() => {})
}

const handleClick = async (n: Notification) => {
  store.markAsRead(n.id).catch(() => {})

  // 优先使用后端提供的 courseID 精准跳转
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
    // courseID 不可用时的 fallback：跳转到用户评价列表
    router.push({ name: 'user-reviews' })
  }
}
</script>
