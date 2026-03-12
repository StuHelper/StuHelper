<script setup lang="ts">
import { ref, onMounted } from 'vue'
import type { Course } from '@stuhelper/shared'

const courses = ref<Course[]>([])
const loading = ref(false)
const searchQuery = ref('')

const loadCourses = async () => {
  loading.value = true
  try {
    // TODO: 调用 API
    await new Promise(resolve => setTimeout(resolve, 1000))
    courses.value = []
  } finally {
    loading.value = false
  }
}

const navigateToDetail = (id: number) => {
  uni.navigateTo({ url: `/pages/course/detail?id=${id}` })
}

onMounted(() => {
  loadCourses()
})
</script>

<template>
  <view class="course-page">
    <!-- Search Bar -->
    <view class="search-bar">
      <input
        v-model="searchQuery"
        class="search-input"
        placeholder="搜索课程名称或代码"
        placeholder-class="search-placeholder"
      />
    </view>

    <!-- Loading -->
    <view v-if="loading" class="loading">
      <text>加载中...</text>
    </view>

    <!-- Course List -->
    <view v-else class="course-list">
      <view
        v-for="course in courses"
        :key="course.id"
        class="course-card"
        @tap="navigateToDetail(course.id)"
      >
        <view class="course-header">
          <text class="course-name">{{ course.name }}</text>
          <text class="course-code">{{ course.code }}</text>
        </view>
        <view class="course-info">
          <text class="course-dept">{{ course.departmentName }}</text>
          <text class="course-credits">{{ course.credits }}学分</text>
        </view>
      </view>

      <!-- Empty State -->
      <view v-if="courses.length === 0" class="empty">
        <text class="empty-text">暂无课程数据</text>
      </view>
    </view>
  </view>
</template>

<style scoped>
.course-page {
  min-height: 100vh;
  background: #F8F9FA;
}

.search-bar {
  padding: 24rpx;
  background: #FFFFFF;
}

.search-input {
  width: 100%;
  height: 80rpx;
  padding: 0 32rpx;
  background: #F3F4F6;
  border-radius: 40rpx;
  font-size: 28rpx;
}

.search-placeholder {
  color: #9CA3AF;
}

.loading {
  padding: 80rpx;
  text-align: center;
  color: #6B7280;
}

.course-list {
  padding: 24rpx;
}

.course-card {
  background: #FFFFFF;
  border-radius: 16rpx;
  padding: 32rpx;
  margin-bottom: 24rpx;
  box-shadow: 0 2rpx 8rpx rgba(0, 0, 0, 0.05);
}

.course-header {
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

.course-code {
  font-size: 24rpx;
  color: #6B7280;
}

.course-info {
  display: flex;
  justify-content: space-between;
  font-size: 28rpx;
  color: #6B7280;
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
