<script setup lang="ts">
import { computed, ref } from 'vue'
import { onShow } from '@dcloudio/uni-app'
import { api } from '@/api'
import type { components } from '@/api'
import { unwrapData, unwrapListData } from '@/api/result'
import { getHomeFeatures, getUniappxAppNotice } from '@/config/featureSurface'
import { translate } from '@/i18n'
import { useAuthStore } from '@/stores/auth'

const authStore = useAuthStore()
const t = translate
const loading = ref(false)
const courseCount = ref(0)
const reviewCount = ref(0)
const departmentCount = ref(0)
const hotCourses = ref<components['schemas']['HotCourse'][]>([])
const lastLoadedAt = ref(0)
const DASHBOARD_STALE_MS = 60_000

function isSuccessfulResult(result: { error?: unknown; response?: { status?: number } }): boolean {
  const status = result.response?.status ?? 0
  return !result.error && status >= 200 && status < 300
}

const greeting = computed(() =>
  authStore.isAuthenticated
    ? t('home.greetingBack', { name: authStore.displayName })
    : t('home.greetingGuest')
)
const shortcuts = computed(() => getHomeFeatures())
const appNotice = computed(() => getUniappxAppNotice())

async function loadDashboard(force = false) {
  const now = Date.now()
  if (
    !force &&
    lastLoadedAt.value > 0 &&
    now - lastLoadedAt.value < DASHBOARD_STALE_MS &&
    (courseCount.value > 0 || reviewCount.value > 0 || hotCourses.value.length > 0)
  ) {
    return
  }
  loading.value = true
  try {
    const [courseStatsResult, reviewStatsResult, hotCoursesResult, bootstrapResult] = await Promise.allSettled([
      api.course.getStats(),
      api.review.getReviewStats(),
      api.review.getHotCourses({ period: 'month', limit: 5 }),
      authStore.bootstrapSession(),
    ])

    let hasSuccessfulData = false

    if (courseStatsResult.status === 'fulfilled' && isSuccessfulResult(courseStatsResult.value)) {
      const courseStats = unwrapData<components['schemas']['CourseStats']>(courseStatsResult.value)
      courseCount.value = courseStats.courseCount
      departmentCount.value = courseStats.departmentCount
      hasSuccessfulData = true
    }

    if (reviewStatsResult.status === 'fulfilled' && isSuccessfulResult(reviewStatsResult.value)) {
      const reviewStats = unwrapData<components['schemas']['ReviewStats']>(reviewStatsResult.value)
      reviewCount.value = reviewStats.reviewCount
      departmentCount.value = reviewStats.departmentCount || departmentCount.value
      hasSuccessfulData = true
    }

    if (hotCoursesResult.status === 'fulfilled' && isSuccessfulResult(hotCoursesResult.value)) {
      hotCourses.value = unwrapListData<components['schemas']['HotCourse']>(hotCoursesResult.value).list
      hasSuccessfulData = true
    }

    if (bootstrapResult.status === 'rejected' && !hasSuccessfulData) {
      throw bootstrapResult.reason
    }

    if (hasSuccessfulData) {
      lastLoadedAt.value = Date.now()
      return
    }

    throw new Error(t('home.loadFailed'))
  } catch (error) {
    uni.showToast({
      title: error instanceof Error ? error.message : t('home.loadFailed'),
      icon: 'none',
    })
  } finally {
    loading.value = false
  }
}

function go(path: string) {
  if (path.startsWith('/pages/home') || path.startsWith('/pages/course') || path.startsWith('/pages/review') || path.startsWith('/pages/user')) {
    const tabPages = ['/pages/home/index', '/pages/course/index', '/pages/review/index', '/pages/user/index']
    if (tabPages.includes(path)) {
      uni.switchTab({ url: path })
      return
    }
  }
  uni.navigateTo({ url: path })
}

function openCourse(courseID: number) {
  uni.navigateTo({ url: `/pages/course/detail?id=${courseID}` })
}

onShow(() => {
  void loadDashboard()
})
</script>

<template>
  <scroll-view class="home-page" scroll-y>
    <view class="hero-card">
      <text class="hero-title">{{ t('home.heroTitle') }}</text>
      <text class="hero-subtitle">{{ greeting }}</text>
      <text class="hero-caption">{{ appNotice }}</text>
      <view class="stats-row">
        <view class="stat-card">
          <text class="stat-value">{{ courseCount }}</text>
          <text class="stat-label">{{ t('home.stats.courses') }}</text>
        </view>
        <view class="stat-card">
          <text class="stat-value">{{ reviewCount }}</text>
          <text class="stat-label">{{ t('home.stats.reviews') }}</text>
        </view>
        <view class="stat-card">
          <text class="stat-value">{{ departmentCount }}</text>
          <text class="stat-label">{{ t('home.stats.departments') }}</text>
        </view>
      </view>
    </view>

    <view class="section">
      <text class="section-title">{{ t('home.shortcutsTitle') }}</text>
      <view class="shortcut-grid">
        <view
          v-for="item in shortcuts"
          :key="item.path"
          class="shortcut-card"
          @tap="go(item.path)"
        >
          <text class="shortcut-icon">{{ item.icon }}</text>
          <text class="shortcut-title">{{ item.title }}</text>
          <text class="shortcut-desc">{{ item.desc }}</text>
        </view>
      </view>
    </view>

    <view class="section">
      <view class="section-header">
        <text class="section-title">{{ t('home.hotCoursesTitle') }}</text>
        <text class="section-link" @tap="go('/pages/review/index')">{{ t('home.viewAll') }}</text>
      </view>
      <view v-if="loading" class="loading-card"><text>{{ t('common.loading') }}</text></view>
      <view v-else-if="hotCourses.length === 0" class="empty-card">
        <text>{{ t('home.noHotCourses') }}</text>
      </view>
      <view v-else class="list-card">
        <view v-for="course in hotCourses" :key="course.courseID" class="list-row" @tap="openCourse(course.courseID)">
          <view class="list-main">
            <text class="list-title">{{ course.courseName }}</text>
            <text class="list-meta">{{ t('common.reviewCount', { count: course.reviewCount }) }}</text>
          </view>
          <text class="list-score">{{ course.avgRating.toFixed(1) }}</text>
        </view>
      </view>
    </view>
  </scroll-view>
</template>

<style scoped>
.home-page {
  min-height: 100vh;
  background: #f8fafc;
}

.hero-card {
  margin: 24rpx;
  padding: 36rpx;
  border-radius: 28rpx;
  background: linear-gradient(135deg, #4f46e5 0%, #7c3aed 100%);
  color: #ffffff;
}

.hero-title {
  display: block;
  font-size: 42rpx;
  font-weight: 700;
}

.hero-subtitle {
  display: block;
  margin-top: 12rpx;
  font-size: 26rpx;
  color: rgba(255, 255, 255, 0.88);
}

.hero-caption {
  display: block;
  margin-top: 16rpx;
  font-size: 22rpx;
  line-height: 1.6;
  color: rgba(255, 255, 255, 0.78);
}

.stats-row {
  display: flex;
  gap: 16rpx;
  margin-top: 28rpx;
}

.stat-card {
  flex: 1;
  background: rgba(255, 255, 255, 0.14);
  border: 2rpx solid rgba(255, 255, 255, 0.14);
  border-radius: 20rpx;
  padding: 20rpx;
}

.stat-value {
  display: block;
  font-size: 36rpx;
  font-weight: 700;
}

.stat-label {
  display: block;
  margin-top: 8rpx;
  font-size: 22rpx;
  color: rgba(255, 255, 255, 0.78);
}

.section {
  margin: 24rpx;
}

.section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.section-title {
  font-size: 30rpx;
  font-weight: 700;
  color: #0f172a;
}

.section-link {
  color: #4f46e5;
  font-size: 24rpx;
}

.shortcut-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 18rpx;
  margin-top: 20rpx;
}

.shortcut-card,
.loading-card,
.empty-card,
.list-card {
  background: #ffffff;
  border-radius: 24rpx;
  box-shadow: 0 10rpx 30rpx rgba(15, 23, 42, 0.05);
}

.shortcut-card {
  padding: 28rpx 20rpx;
}

.shortcut-icon {
  font-size: 44rpx;
}

.shortcut-title {
  display: block;
  margin-top: 16rpx;
  font-size: 28rpx;
  font-weight: 700;
  color: #0f172a;
}

.shortcut-desc {
  display: block;
  margin-top: 10rpx;
  font-size: 22rpx;
  color: #64748b;
  line-height: 1.5;
}

.loading-card,
.empty-card {
  margin-top: 20rpx;
  padding: 36rpx;
  text-align: center;
  color: #64748b;
}

.list-card {
  margin-top: 20rpx;
  overflow: hidden;
}

.list-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 28rpx;
  border-bottom: 1rpx solid #e2e8f0;
}

.list-row:last-child {
  border-bottom: none;
}

.list-main {
  flex: 1;
}

.list-title {
  display: block;
  font-size: 28rpx;
  font-weight: 600;
  color: #0f172a;
}

.list-meta {
  display: block;
  margin-top: 8rpx;
  font-size: 22rpx;
  color: #64748b;
}

.list-score {
  font-size: 32rpx;
  font-weight: 700;
  color: #4f46e5;
}
</style>
