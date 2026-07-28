<script setup lang="ts">
import { ref } from 'vue'
import { onShow } from '@dcloudio/uni-app'
import A11yButton from '@/components/A11yButton.vue'
import { api } from '@/api'
import type { components } from '@/api'
import { unwrapListData } from '@/api/result'
import { usePagedList } from '@/composables/usePagedList'
import { setPageTitle, translate } from '@/i18n'
import { DEFAULT_PAGE_SIZE } from '@/config/pagination'

const t = translate
const query = ref('')
const {
  items: courses,
  loading,
  loadingMore,
  hasMore,
  refresh: refreshCourses,
  loadMore: loadMoreCourses,
} = usePagedList<components['schemas']['Course']>({
  pageSize: DEFAULT_PAGE_SIZE,
  async fetchPage(page, pageSize) {
    const result = await api.course.getCourses({
      page,
      pageSize,
      q: query.value.trim() || undefined,
      sort: 'reviewCount',
    })
    return unwrapListData<components['schemas']['Course']>(result)
  },
  onError(error) {
    uni.showToast({
      title: error instanceof Error ? error.message : t('course.index.loadFailed'),
      icon: 'none',
    })
  },
})

function handleSearch() {
  void refreshCourses()
}

function loadMore() {
  void loadMoreCourses()
}

function openCourse(id: number) {
  uni.navigateTo({ url: `/pages/course/detail?id=${id}` })
}

onShow(() => {
  setPageTitle('common.pageTitles.courseList')
  if (courses.value.length > 0) return
  void refreshCourses()
})
</script>

<template>
  <scroll-view class="course-page" scroll-y>
    <view class="search-bar">
      <input
        v-model="query"
        class="search-input"
        data-testid="uni-course-search-input"
        :placeholder="t('course.index.searchPlaceholder')"
        @confirm="handleSearch"
      />
      <A11yButton class="search-btn" data-testid="uni-course-search-submit" @tap="handleSearch">{{ t('common.search') }}</A11yButton>
    </view>

    <view v-if="loading" class="state-card"><text>{{ t('common.loading') }}</text></view>
    <view v-else-if="courses.length === 0" class="state-card">
      <text>{{ t('course.index.noResults') }}</text>
    </view>
    <view v-else class="list-wrap">
      <A11yButton v-for="course in courses" :key="course.id" class="course-card" :data-testid="`uni-course-card-${course.id}`" @tap="openCourse(course.id)">
        <view class="course-main">
          <text class="course-name">{{ course.name }}</text>
          <text class="course-code">{{ course.code || t('common.unavailableCourseCode') }}</text>
        </view>
        <view class="course-side">
          <text class="course-count">{{ course.reviewCount }}</text>
          <text class="course-count-label">{{ t('home.stats.reviews') }}</text>
        </view>
        <view class="course-meta">
          <text>{{ course.departmentName || t('common.unclassifiedDepartment') }}</text>
          <text>{{ t('common.creditValue', { value: course.credits }) }}</text>
        </view>
      </A11yButton>

      <A11yButton v-if="hasMore" class="more-btn" data-testid="uni-course-load-more" :disabled="loadingMore" @tap="loadMore">
        {{ loadingMore ? t('common.loading') : t('common.loadMore') }}
      </A11yButton>
    </view>
  </scroll-view>
</template>

<style scoped>
.course-page {
  min-height: 100vh;
  background: #f8fafc;
}

.search-bar {
  position: sticky;
  top: 0;
  z-index: 10;
  display: flex;
  gap: 16rpx;
  padding: 24rpx;
  background: #f8fafc;
}

.search-input {
  flex: 1;
  height: 84rpx;
  border-radius: 22rpx;
  background: #ffffff;
  border: 2rpx solid #e2e8f0;
  padding: 0 24rpx;
  font-size: 28rpx;
}

.search-btn,
.more-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 84rpx;
  border-radius: 22rpx;
  font-size: 28rpx;
  font-weight: 600;
  background: #4f46e5;
  color: #ffffff;
}

.search-btn {
  width: 160rpx;
}

.more-btn {
  margin: 12rpx 24rpx 36rpx;
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
  padding: 0 24rpx 24rpx;
}

.course-card {
  display: block;
  width: 100%;
  margin-bottom: 18rpx;
  padding: 28rpx;
  border: 0;
  background: #ffffff;
  border-radius: 24rpx;
  box-shadow: 0 10rpx 30rpx rgba(15, 23, 42, 0.04);
  text-align: left;
  line-height: inherit;
}

.course-main {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
}

.course-name {
  flex: 1;
  font-size: 32rpx;
  font-weight: 700;
  color: #0f172a;
}

.course-code {
  margin-left: 16rpx;
  font-size: 22rpx;
  color: #64748b;
}

.course-side {
  margin-top: 16rpx;
  display: inline-flex;
  flex-direction: column;
  align-items: center;
  padding: 14rpx 18rpx;
  border-radius: 18rpx;
  background: #eef2ff;
}

.course-count {
  font-size: 30rpx;
  font-weight: 700;
  color: #4338ca;
}

.course-count-label {
  font-size: 20rpx;
  color: #6366f1;
}

.course-meta {
  display: flex;
  justify-content: space-between;
  margin-top: 18rpx;
  font-size: 24rpx;
  color: #475569;
}
</style>
