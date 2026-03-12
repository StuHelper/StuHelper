<script setup lang="ts">
import { ref } from 'vue'
import { onLoad } from '@dcloudio/uni-app'
import type { TeacherStats } from '@stuhelper/shared'

const teacherId = ref('')
const teacher = ref<TeacherStats | null>(null)
const loading = ref(false)

onLoad((options) => {
  teacherId.value = options?.id || ''
  loadTeacherProfile()
})

const loadTeacherProfile = async () => {
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
  <view class="teacher-profile">
    <view v-if="loading" class="loading">
      <text>加载中...</text>
    </view>

    <view v-else-if="teacher" class="content">
      <view class="header">
        <text class="teacher-name">{{ teacher.teacherName }}</text>
        <text class="department">{{ teacher.departmentName }}</text>
      </view>
    </view>

    <view v-else class="empty">
      <text>教师不存在</text>
    </view>
  </view>
</template>

<style scoped>
.teacher-profile {
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
  text-align: center;
}

.teacher-name {
  font-size: 36rpx;
  font-weight: 600;
  color: #1F2937;
  display: block;
  margin-bottom: 12rpx;
}

.department {
  font-size: 28rpx;
  color: #6B7280;
}
</style>
