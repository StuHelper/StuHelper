<script setup lang="ts">
import { ref } from 'vue'
import { onShow } from '@dcloudio/uni-app'
import { api } from '@/api'
import type { components } from '@/api'
import { assertMutationSuccess, unwrapListData } from '@/api/result'
import { setPageTitle, translate } from '@/i18n'
import { useAuthStore } from '@/stores/auth'
import { formatDateTime } from '@/utils/format'
import { DEFAULT_PAGE_SIZE } from '@/config/pagination'

const authStore = useAuthStore()
const t = translate
const loading = ref(false)
const loadingMore = ref(false)
const notifications = ref<components['schemas']['Notification'][]>([])
const page = ref(1)
const hasMore = ref(true)
const lastLoadedAt = ref(0)
const STALE_MS = 30_000

async function loadNotifications() {
  if (!(await authStore.requireAuth(t('user.notifications.requireAuth')))) return
  loading.value = true
  page.value = 1
  hasMore.value = true
  try {
    const result = await api.notification.getNotifications(1, DEFAULT_PAGE_SIZE)
    const data = unwrapListData<components['schemas']['Notification']>(result)
    notifications.value = data.list
    hasMore.value = data.list.length >= DEFAULT_PAGE_SIZE
    lastLoadedAt.value = Date.now()
  } catch (error) {
    uni.showToast({
      title: error instanceof Error ? error.message : t('user.notifications.loadFailed'),
      icon: 'none',
    })
  } finally {
    loading.value = false
  }
}

async function loadMore() {
  if (loadingMore.value || !hasMore.value) return
  loadingMore.value = true
  try {
    page.value++
    const result = await api.notification.getNotifications(page.value, DEFAULT_PAGE_SIZE)
    const data = unwrapListData<components['schemas']['Notification']>(result)
    notifications.value = [...notifications.value, ...data.list]
    hasMore.value = data.list.length >= DEFAULT_PAGE_SIZE
  } catch (_error) { void _error;
    page.value = Math.max(1, page.value - 1)
  } finally {
    loadingMore.value = false
  }
}

async function markRead(id: string) {
  try {
    assertMutationSuccess(await api.notification.markAsRead(id))
    notifications.value = notifications.value.map((item) => item.id === id ? { ...item, isRead: true } : item)
  } catch (error) {
    uni.showToast({
      title: error instanceof Error ? error.message : t('user.notifications.actionFailed'),
      icon: 'none',
    })
  }
}

async function markAllRead() {
  try {
    assertMutationSuccess(await api.notification.markAllAsRead())
    notifications.value = notifications.value.map((item) => ({ ...item, isRead: true }))
    uni.showToast({ title: t('user.notifications.allReadDone'), icon: 'none' })
  } catch (error) {
    uni.showToast({
      title: error instanceof Error ? error.message : t('user.notifications.actionFailed'),
      icon: 'none',
    })
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
      <button class="mark-all-btn" @tap="markAllRead">{{ t('user.notifications.markAllRead') }}</button>
    </view>
    <view v-if="loading" class="state-card"><text>{{ t('common.loading') }}</text></view>
    <view v-else-if="notifications.length === 0" class="state-card"><text>{{ t('user.notifications.empty') }}</text></view>
    <view v-else class="list-wrap">
      <view v-for="item in notifications" :key="item.id" class="notification-card" :class="{ unread: !item.isRead }" @tap="markRead(item.id)">
        <view class="notification-head">
          <text class="notification-title">{{ item.title }}</text>
          <text v-if="!item.isRead" class="badge">{{ t('user.notifications.unread') }}</text>
        </view>
        <text v-if="item.content" class="notification-content">{{ item.content }}</text>
        <view class="notification-meta">
          <text>{{ item.type }}</text>
          <text>{{ formatDateTime(item.createdAt) }}</text>
        </view>
      </view>
      <view v-if="hasMore" class="load-more" @tap="loadMore">
        <text>{{ loadingMore ? t('common.loading') : t('common.loadMore') }}</text>
      </view>
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
  margin-bottom: 18rpx;
  padding: 28rpx;
  background: #ffffff;
  border-radius: 24rpx;
  box-shadow: 0 10rpx 30rpx rgba(15, 23, 42, 0.05);
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
  padding: 28rpx;
  text-align: center;
  color: #4f46e5;
  font-size: 26rpx;
}
</style>
