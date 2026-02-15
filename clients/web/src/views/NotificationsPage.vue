<template>
  <div class="max-w-[600px] mx-auto p-6 animate-fade-in">
    <div class="flex items-center justify-between mb-6 pb-4 border-b border-border">
      <h1 class="font-sans text-xl font-extrabold tracking-tight text-text-primary m-0">
        {{ t('user.notification.title') }}
        <!-- L-42: 未读数量变化时屏幕阅读器播报 -->
        <span v-if="hasUnread" class="sr-only" aria-live="polite" role="status">
          {{ t('user.notification.unreadCount', { count: store.unreadCount }) }}
        </span>
      </h1>
      <button
        v-if="hasUnread"
        class="py-1 px-3 bg-transparent border border-border rounded-full text-text-muted text-sm cursor-pointer transition-all duration-fast hover:border-text-primary hover:text-text-primary"
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
      <div v-if="notifications.length > 0" class="border border-border rounded-md overflow-hidden">
        <NotificationItem
          v-for="n in notifications"
          :key="n.id"
          :notification="n"
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
  store.fetchNotifications(1)
})

const loadMore = () => {
  // M-17: 防止上一请求未完成时重复翻页
  if (loading.value || !hasMore.value) return
  page.value++
  store.fetchNotifications(page.value)
}

const handleMarkAllRead = () => {
  store.markAllAsRead()
}

const handleClick = (n: Notification) => {
  store.markAsRead(n.id)
  if (n.relatedType && n.relatedID) {
    // relatedType='review' 时 relatedID 为 review UUID，非 course ID
    // 暂不导航（需后端扩展 courseID 字段后才能正确跳转）
    if (n.relatedType === 'course') {
      router.push(`/review/courses/${n.relatedID}`)
    }
  }
}
</script>
