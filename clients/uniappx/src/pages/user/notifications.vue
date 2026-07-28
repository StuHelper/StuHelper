<script setup lang="ts">
import { computed, ref } from 'vue'
import { onShow } from '@dcloudio/uni-app'
import A11yButton from '@/components/A11yButton.vue'
import { api } from '@/api'
import type { components } from '@/api'
import { assertMutationSuccess, unwrapListData } from '@/api/result'
import { usePagedList } from '@/composables/usePagedList'
import { setPageTitle, translate } from '@/i18n'
import { useAuthStore } from '@/stores/auth'
import { formatDateTime } from '@/utils/format'
import { DEFAULT_PAGE_SIZE } from '@/config/pagination'

const authStore = useAuthStore()
const t = translate
const lastLoadedAt = ref(0)
const markingRead = ref<Record<string, boolean>>({})
const markingAll = ref(false)
const STALE_MS = 30_000
const {
  items: notifications,
  loading,
  loadingMore,
  hasMore,
  refresh: refreshNotifications,
  loadMore,
} = usePagedList<components['schemas']['Notification']>({
  pageSize: DEFAULT_PAGE_SIZE,
  async fetchPage(page, pageSize) {
    const result = await api.notification.getNotifications(page, pageSize)
    return unwrapListData<components['schemas']['Notification']>(result)
  },
  onError(error) {
    uni.showToast({
      title: error instanceof Error ? error.message : t('user.notifications.loadFailed'),
      icon: 'none',
    })
  },
})
const hasUnread = computed(() => notifications.value.some(item => !item.isRead))

async function loadNotifications() {
  if (!(await authStore.requireAuth(t('user.notifications.requireAuth')))) return
  if (await refreshNotifications()) {
    lastLoadedAt.value = Date.now()
  }
}

async function markRead(id: string) {
  const notification = notifications.value.find(item => item.id === id)
  if (!notification || notification.isRead || markingAll.value || markingRead.value[id]) return
  markingRead.value = { ...markingRead.value, [id]: true }
  try {
    assertMutationSuccess(await api.notification.markAsRead(id))
    notifications.value = notifications.value.map((item) => item.id === id ? { ...item, isRead: true } : item)
  } catch (error) {
    uni.showToast({
      title: error instanceof Error ? error.message : t('user.notifications.actionFailed'),
      icon: 'none',
    })
  } finally {
    markingRead.value = { ...markingRead.value, [id]: false }
  }
}

async function markAllRead() {
  if (markingAll.value || !hasUnread.value) return
  markingAll.value = true
  try {
    assertMutationSuccess(await api.notification.markAllAsRead())
    notifications.value = notifications.value.map((item) => ({ ...item, isRead: true }))
    uni.showToast({ title: t('user.notifications.allReadDone'), icon: 'none' })
  } catch (error) {
    uni.showToast({
      title: error instanceof Error ? error.message : t('user.notifications.actionFailed'),
      icon: 'none',
    })
  } finally {
    markingAll.value = false
  }
}

onShow(() => {
  setPageTitle('common.pageTitles.notifications')
  if (Date.now() - lastLoadedAt.value < STALE_MS) return
  void loadNotifications()
})
</script>

<template>
  <scroll-view class="notifications-page" scroll-y>
    <view class="toolbar">
      <A11yButton class="mark-all-btn" data-testid="uni-notification-mark-all" :disabled="markingAll || !hasUnread" @tap="markAllRead">
        {{ markingAll ? t('common.processing') : t('user.notifications.markAllRead') }}
      </A11yButton>
    </view>
    <view v-if="loading" class="state-card"><text>{{ t('common.loading') }}</text></view>
    <view v-else-if="notifications.length === 0" class="state-card"><text>{{ t('user.notifications.empty') }}</text></view>
    <view v-else class="list-wrap">
      <A11yButton
        v-for="item in notifications"
        :key="item.id"
        class="notification-card"
        :class="{ unread: !item.isRead }"
        :data-testid="`uni-notification-card-${item.id}`"
        :disabled="item.isRead || markingAll || markingRead[item.id]"
        @tap="markRead(item.id)"
      >
        <view class="notification-head">
          <text class="notification-title">{{ item.title }}</text>
          <text
            v-if="!item.isRead"
            class="badge"
            :data-testid="`uni-notification-unread-${item.id}`"
          >
            {{ t('user.notifications.unread') }}
          </text>
        </view>
        <text v-if="item.content" class="notification-content">{{ item.content }}</text>
        <view class="notification-meta">
          <text>{{ item.type }}</text>
          <text>{{ formatDateTime(item.createdAt) }}</text>
        </view>
      </A11yButton>
      <A11yButton v-if="hasMore" class="load-more" data-testid="uni-notification-load-more" :disabled="loadingMore" @tap="loadMore">
        {{ loadingMore ? t('common.loading') : t('common.loadMore') }}
      </A11yButton>
    </view>
  </scroll-view>
</template>

<style scoped>
.notifications-page {
  min-height: 100vh;
  background: #f8fafc;
}

.toolbar {
  padding: 24rpx 24rpx 0;
}

.mark-all-btn {
  height: 76rpx;
  border-radius: 20rpx;
  background: #eef2ff;
  color: #4338ca;
  font-size: 26rpx;
  font-weight: 600;
}

.state-card {
  margin: 24rpx;
  padding: 40rpx;
  background: #ffffff;
  border-radius: 24rpx;
  text-align: center;
  color: #64748b;
}

.list-wrap {
  padding: 24rpx;
}

.notification-card {
  display: block;
  width: 100%;
  margin-bottom: 18rpx;
  padding: 28rpx;
  border: 0;
  background: #ffffff;
  border-radius: 24rpx;
  box-shadow: 0 10rpx 30rpx rgba(15, 23, 42, 0.05);
  text-align: left;
  line-height: inherit;
}

.notification-card.unread {
  border: 2rpx solid #c7d2fe;
}

.notification-head,
.notification-meta {
  display: flex;
  justify-content: space-between;
  gap: 16rpx;
}

.notification-title {
  flex: 1;
  font-size: 28rpx;
  font-weight: 700;
  color: #0f172a;
}

.badge {
  font-size: 22rpx;
  color: #4338ca;
}

.notification-content {
  display: block;
  margin-top: 14rpx;
  font-size: 24rpx;
  line-height: 1.7;
  color: #334155;
}

.notification-meta {
  margin-top: 16rpx;
  font-size: 22rpx;
  color: #64748b;
}

.load-more {
  display: block;
  width: 100%;
  padding: 28rpx;
  border: 0;
  background: transparent;
  text-align: center;
  color: #4f46e5;
  font-size: 26rpx;
}
</style>
