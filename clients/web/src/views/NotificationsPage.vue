<template>
  <div class="max-w-[600px] mx-auto p-6 animate-fade-in">
    <div class="flex items-center justify-between mb-6 pb-4 border-b border-border">
      <h1 class="font-sans text-xl font-extrabold tracking-tight text-text-primary m-0">{{ t('user.notification.title') }}</h1>
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
