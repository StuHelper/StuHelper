<script setup lang="ts">
import { ref } from 'vue'
import { onLoad } from '@dcloudio/uni-app'
import type { Course } from '@stuhelper/shared'

const courseId = ref('')
const course = ref<Course | null>(null)
const loading = ref(false)

onLoad((options) => {
  courseId.value = options?.id || ''
  loadCourseDetail()
})

const loadCourseDetail = async () => {
  loading.value = true
  try {
    // TODO: 调用 API
    await new Promise(resolve => setTimeout(resolve, 1000))
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <view class="course-detail">
    <view v-if="loading" class="loading">
      <text>加载中...</text>
    </view>

    <view v-else-if="course" class="content">
      <view class="header">
        <text class="course-name">{{ course.name }}</text>
        <text class="course-code">{{ course.code }}</text>
      </view>

      <view class="info-section">
        <view class="info-item">
          <text class="label">院系：</text>
          <text class="value">{{ course.departmentName }}</text>
        </view>
        <view class="info-item">
          <text class="label">学分：</text>
          <text class="value">{{ course.credits }}</text>
        </view>
        <view class="info-item">
          <text class="label">评论数：</text>
          <text class="value">{{ course.reviewCount }}</text>
        </view>
      </view>
    </view>

    <view v-else class="empty">
      <text>课程不存在</text>
    </view>
  </view>
</template>

<style scoped>
.course-detail {
  min-height: 100vh;
  background: #F8F9FA;
}

.loading,
.empty {
  padding: 80rpx;
  text-align: center;
  color: #6B7280;
}

.content {
  padding: 24rpx;
}

.header {
  background: #FFFFFF;
  border-radius: 16rpx;
  padding: 32rpx;
  margin-bottom: 24rpx;
}

.course-name {
  font-size: 36rpx;
  font-weight: 600;
  color: #1F2937;
  display: block;
  margin-bottom: 12rpx;
}

.course-code {
  font-size: 28rpx;
  color: #6B7280;
}

.info-section {
  background: #FFFFFF;
  border-radius: 16rpx;
  padding: 32rpx;
}

.info-item {
  display: flex;
  margin-bottom: 16rpx;
  font-size: 28rpx;
}

.label {
  color: #6B7280;
  width: 120rpx;
}

.value {
  color: #1F2937;
  flex: 1;
}
</style>
