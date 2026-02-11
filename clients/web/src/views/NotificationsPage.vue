<template>
  <div class="notifications-page">
    <div class="page-header">
      <h1>{{ t('user.notification.title') }}</h1>
      <button
        v-if="hasUnread"
        class="mark-all-btn"
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
      <div v-if="notifications.length > 0" class="notification-list">
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
import { computed, onMounted } from 'vue'
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

let page = 1

onMounted(() => {
  store.fetchNotifications(1)
})

const loadMore = () => {
  page++
  store.fetchNotifications(page)
}

const handleMarkAllRead = () => {
  store.markAllAsRead()
}

const handleClick = (n: Notification) => {
  store.markAsRead(n.id)
  if (n.relatedType && n.relatedID) {
    if (n.relatedType === 'review') {
      router.push(`/review/courses/${n.relatedID}`)
    }
  }
}
</script>

<style scoped>
.notifications-page {
  max-width: 600px;
  margin: 0 auto;
  padding: var(--space-6);
  animation: fadeIn var(--duration-base) var(--ease-out);
}

.page-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: var(--space-6);
  padding-bottom: var(--space-4);
  border-bottom: 1px solid var(--border);
}

.page-header h1 {
  font-family: var(--font-sans);
  font-size: var(--text-xl);
  font-weight: var(--weight-extrabold);
  letter-spacing: var(--tracking-tight);
  color: var(--text-primary);
  margin: 0;
}

.mark-all-btn {
  padding: var(--space-1) var(--space-3);
  background: transparent;
  border: 1px solid var(--border);
  border-radius: var(--radius-full);
  color: var(--text-muted);
  font-size: var(--text-sm);
  cursor: pointer;
  transition: all var(--duration-fast);
}

.mark-all-btn:hover {
  border-color: var(--text-primary);
  color: var(--text-primary);
}

.notification-list {
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  overflow: hidden;
}
</style>
