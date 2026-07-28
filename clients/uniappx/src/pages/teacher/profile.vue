<script setup lang="ts">
import { ref } from 'vue'
import { onLoad, onShow } from '@dcloudio/uni-app'
import A11yButton from '@/components/A11yButton.vue'
import { api } from '@/api'
import type { components } from '@/api'
import { unwrapOptionalData } from '@/api/result'
import { setPageTitle, translate } from '@/i18n'

const t = translate
const teacherID = ref(0)
const loading = ref(false)
const loadError = ref(false)
const teacher = ref<components['schemas']['TeacherRatingStatsResponse'] | null>(null)
const lastLoadedAt = ref(0)
const STALE_MS = 30_000
let teacherLoadPromise: Promise<void> | null = null

async function performTeacherLoad() {
  if (!teacherID.value) return
  loading.value = true
  loadError.value = false
  try {
    const result = await api.rating.getTeacherStats(teacherID.value)
    teacher.value = unwrapOptionalData<components['schemas']['TeacherRatingStatsResponse']>(result)
    lastLoadedAt.value = Date.now()
  } catch (error) {
    loadError.value = true
    uni.showToast({
      title: error instanceof Error ? error.message : t('teacher.profile.loadFailed'),
      icon: 'none',
    })
  } finally {
    loading.value = false
  }
}

function loadTeacher(): Promise<void> {
  if (teacherLoadPromise) return teacherLoadPromise
  teacherLoadPromise = performTeacherLoad().finally(() => {
    teacherLoadPromise = null
  })
  return teacherLoadPromise
}

function openCourse(id: number) {
  uni.navigateTo({ url: `/pages/course/detail?id=${id}` })
}

onLoad((options) => {
  setPageTitle('common.pageTitles.teacherProfile')
  teacherID.value = Number(options?.id || 0)
  void loadTeacher()
})

onShow(() => {
  setPageTitle('common.pageTitles.teacherProfile')
  if (!teacherID.value) return
  if (Date.now() - lastLoadedAt.value < STALE_MS) return
  void loadTeacher()
})
</script>

<template>
  <scroll-view class="teacher-page" scroll-y>
    <view v-if="loading" class="state-card"><text>{{ t('common.loading') }}</text></view>
    <view v-else-if="loadError && !teacher" class="state-card" role="alert">
      <text>{{ t('teacher.profile.loadFailed') }}</text>
      <A11yButton class="retry-btn" data-testid="uni-teacher-retry" @tap="loadTeacher">
        {{ t('common.retry') }}
      </A11yButton>
    </view>
    <view v-else-if="!teacher" class="state-card"><text>{{ t('teacher.profile.missing') }}</text></view>
    <view v-else>
      <view class="hero-card">
        <text class="teacher-name">{{ teacher.teacherName }}</text>
        <text class="teacher-dept">{{ teacher.departmentName }}</text>
        <view class="stats-row">
          <view class="stat-item">
            <text class="stat-value">{{ teacher.avgRating ? teacher.avgRating.toFixed(1) : '--' }}</text>
            <text class="stat-label">{{ t('teacher.profile.averageRating') }}</text>
          </view>
          <view class="stat-item">
            <text class="stat-value">{{ teacher.courseCount }}</text>
            <text class="stat-label">{{ t('teacher.profile.courseCount') }}</text>
          </view>
          <view class="stat-item">
            <text class="stat-value">{{ teacher.reviewCount }}</text>
            <text class="stat-label">{{ t('teacher.profile.reviewCount') }}</text>
          </view>
        </view>
      </view>

      <view class="section-card">
        <text class="section-title">{{ t('teacher.profile.coursesTitle') }}</text>
        <view v-if="teacher.courses.length === 0" class="empty-text">
          <text>{{ t('teacher.profile.noCourses') }}</text>
        </view>
        <view v-else>
          <A11yButton
            v-for="course in teacher.courses"
            :key="course.id"
            class="course-row"
            :data-testid="`uni-teacher-course-${course.id}`"
            @tap="openCourse(course.id)"
          >
            <view>
              <text class="course-name">{{ course.name }}</text>
              <text class="course-meta">{{ t('common.reviewCount', { count: course.reviewCount }) }}</text>
            </view>
            <text class="course-score">{{ course.avgRating ? course.avgRating.toFixed(1) : '--' }}</text>
          </A11yButton>
        </view>
      </view>
    </view>
  </scroll-view>
</template>

<style scoped>
.teacher-page {
  min-height: 100vh;
  background: #f8fafc;
}

.state-card,
.hero-card,
.section-card {
  margin: 24rpx;
  background: #ffffff;
  border-radius: 24rpx;
  box-shadow: 0 10rpx 30rpx rgba(15, 23, 42, 0.05);
}

.state-card {
  padding: 40rpx;
  text-align: center;
  color: #64748b;
}

.retry-btn {
  margin-top: 20rpx;
  height: 72rpx;
  border-radius: 18rpx;
  background: #4f46e5;
  color: #ffffff;
  font-size: 26rpx;
}

.hero-card {
  padding: 36rpx;
}

.teacher-name {
  display: block;
  font-size: 40rpx;
  font-weight: 700;
  color: #0f172a;
}

.teacher-dept {
  display: block;
  margin-top: 10rpx;
  font-size: 24rpx;
  color: #64748b;
}

.stats-row {
  display: flex;
  gap: 16rpx;
  margin-top: 24rpx;
}

.stat-item {
  flex: 1;
  padding: 20rpx;
  border-radius: 20rpx;
  background: #eef2ff;
}

.stat-value {
  display: block;
  font-size: 34rpx;
  font-weight: 700;
  color: #4338ca;
}

.stat-label {
  display: block;
  margin-top: 8rpx;
  font-size: 22rpx;
  color: #6366f1;
}

.section-card {
  padding: 30rpx;
}

.section-title {
  display: block;
  font-size: 30rpx;
  font-weight: 700;
  color: #0f172a;
  margin-bottom: 20rpx;
}

.course-row {
  display: flex;
  width: 100%;
  justify-content: space-between;
  align-items: center;
  padding: 22rpx 0;
  border: 0;
  border-bottom: 1rpx solid #e2e8f0;
  background: transparent;
  text-align: left;
  line-height: inherit;
}

.course-row:last-child {
  border-bottom: none;
}

.course-name {
  display: block;
  font-size: 28rpx;
  font-weight: 600;
  color: #0f172a;
}

.course-meta,
.empty-text {
  font-size: 24rpx;
  color: #64748b;
}

.course-meta {
  display: block;
  margin-top: 8rpx;
}

.course-score {
  font-size: 32rpx;
  font-weight: 700;
  color: #4f46e5;
}
</style>
