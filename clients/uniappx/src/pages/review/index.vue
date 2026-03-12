<script setup lang="ts">
import { ref, onMounted } from 'vue'
import type { Review } from '@stuhelper/shared'

const reviews = ref<Review[]>([])
const loading = ref(false)

const loadReviews = async () => {
  loading.value = true
  try {
    // TODO: 调用 API
    await new Promise(resolve => setTimeout(resolve, 1000))
    reviews.value = []
  } finally {
    loading.value = false
  }
}

const navigateToCourse = (courseID: number) => {
  uni.navigateTo({ url: `/pages/course/detail?id=${courseID}` })
}

onMounted(() => {
  loadReviews()
})
</script>

<template>
  <view class="review-page">
    <!-- Loading -->
    <view v-if="loading" class="loading">
      <text>加载中...</text>
    </view>

    <!-- Review List -->
    <view v-else class="review-list">
      <view
        v-for="review in reviews"
        :key="review.id"
        class="review-card"
        @tap="navigateToCourse(review.courseID)"
      >
        <view class="review-header">
          <text class="course-name">{{ review.courseName }}</text>
          <text class="rating">{{ review.title }}</text>
        </view>
        <text class="review-content">{{ review.content }}</text>
        <view class="review-footer">
          <text class="likes">👍 {{ review.likeCount }}</text>
          <text class="date">{{ review.createdAt }}</text>
        </view>
      </view>

      <!-- Empty State -->
      <view v-if="reviews.length === 0" class="empty">
        <text class="empty-text">暂无评课数据</text>
      </view>
    </view>
  </view>
</template>

<style scoped>
.review-page {
  min-height: 100vh;
  background: #F8F9FA;
}

.loading {
  padding: 80rpx;
  text-align: center;
  color: #6B7280;
}

.review-list {
  padding: 24rpx;
}

.review-card {
  background: #FFFFFF;
  border-radius: 16rpx;
  padding: 32rpx;
  margin-bottom: 24rpx;
  box-shadow: 0 2rpx 8rpx rgba(0, 0, 0, 0.05);
}

.review-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16rpx;
}

.course-name {
  font-size: 32rpx;
  font-weight: 600;
  color: #1F2937;
}

.rating {
  font-size: 28rpx;
  color: #F59E0B;
}

.review-content {
  font-size: 28rpx;
  color: #4B5563;
  line-height: 1.6;
  margin-bottom: 16rpx;
}

.review-footer {
  display: flex;
  justify-content: space-between;
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
