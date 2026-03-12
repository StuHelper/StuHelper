<script setup lang="ts">
import { ref, onMounted } from 'vue'

const notifications = ref<any[]>([])
const loading = ref(false)

const loadNotifications = async () => {
  loading.value = true
  try {
    // TODO: 调用 API
    await new Promise(resolve => setTimeout(resolve, 1000))
    notifications.value = []
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  loadNotifications()
})
</script>

<template>
  <view class="notifications-page">
    <view v-if="loading" class="loading">
      <text>加载中...</text>
    </view>

    <view v-else class="notification-list">
      <view
        v-for="item in notifications"
        :key="item.id"
        class="notification-item"
      >
        <text class="title">{{ item.title }}</text>
        <text class="content">{{ item.content }}</text>
        <text class="time">{{ item.createdAt }}</text>
      </view>

      <view v-if="notifications.length === 0" class="empty">
        <text class="empty-text">暂无消息</text>
      </view>
    </view>
  </view>
</template>

<style scoped>
.notifications-page {
  min-height: 100vh;
  background: #F8F9FA;
}

.loading {
  padding: 80rpx;
  text-align: center;
  color: #6B7280;
}

.notification-list {
  padding: 24rpx;
}

.notification-item {
  background: #FFFFFF;
  border-radius: 16rpx;
  padding: 32rpx;
  margin-bottom: 24rpx;
}

.title {
  font-size: 32rpx;
  font-weight: 600;
  color: #1F2937;
  display: block;
  margin-bottom: 12rpx;
}

.content {
  font-size: 28rpx;
  color: #4B5563;
  line-height: 1.6;
  display: block;
  margin-bottom: 12rpx;
}

.time {
  font-size: 24rpx;
  color: #9CA3AF;
}

.empty {
  padding: 120rpx 0;
  text-align: center;
}

.empty-text {
  font-size: 28rpx;
  color: #9CA3AF;
}
</style>
