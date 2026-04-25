<script setup lang="ts">
import { ref } from 'vue'
import { onShow } from '@dcloudio/uni-app'
import { api } from '@/api'
import type { components } from '@/api'
import { unwrapListData } from '@/api/result'
import { setPageTitle, translate } from '@/i18n'
import { useAuthStore } from '@/stores/auth'
import { averageRating, formatDateTime, truncateText } from '@/utils/format'
import { DEFAULT_PAGE_SIZE } from '@/config/pagination'

const authStore = useAuthStore()
const t = translate
const loading = ref(false)
const loadingMore = ref(false)
const reviews = ref<components['schemas']['Review'][]>([])
const page = ref(1)
const hasMore = ref(true)
const lastLoadedAt = ref(0)
const STALE_MS = 30_000

async function loadMyReviews() {
  if (!(await authStore.requireAuth(t('user.reviews.requireAuth')))) return
  loading.value = true
  page.value = 1
  hasMore.value = true
  try {
    const result = await api.user.getMyReviews(1, DEFAULT_PAGE_SIZE)
    const data = unwrapListData<components['schemas']['Review']>(result)
    reviews.value = data.list
    hasMore.value = data.list.length >= DEFAULT_PAGE_SIZE
    lastLoadedAt.value = Date.now()
  } catch (error) {
    uni.showToast({ title: error instanceof Error ? error.message : t('user.reviews.loadFailed'), icon: 'none' })
  } finally {
    loading.value = false
  }
}

async function loadMore() {
  if (loadingMore.value || !hasMore.value) return
  loadingMore.value = true
  try {
    page.value++
    const result = await api.user.getMyReviews(page.value, DEFAULT_PAGE_SIZE)
    const data = unwrapListData<components['schemas']['Review']>(result)
    reviews.value = [...reviews.value, ...data.list]
    hasMore.value = data.list.length >= DEFAULT_PAGE_SIZE
  } catch (_error) { void _error;
    page.value = Math.max(1, page.value - 1)
  } finally {
    loadingMore.value = false
  }
}

function openCourse(id: number) {
  uni.navigateTo({ url: `/pages/course/detail?id=${id}` })
}

onShow(() => {
  setPageTitle('common.pageTitles.myReviews')
  if (Date.now() - lastLoadedAt.value < STALE_MS) return
  void loadMyReviews()
})
</script>

<template>
  <scroll-view class="page" scroll-y>
    <view v-if="loading" class="state-card"><text>{{ t('common.loading') }}</text></view>
    <view v-else-if="reviews.length === 0" class="state-card"><text>{{ t('user.reviews.empty') }}</text></view>
    <view v-else class="list-wrap">
      <view v-for="review in reviews" :key="review.id" class="card" @tap="openCourse(review.courseID)">
        <text class="title">{{ review.title }}</text>
        <text class="meta">
          {{ review.courseName || t('common.courseFallback', { id: review.courseID }) }} ·
          {{ formatDateTime(review.createdAt) }}
        </text>
        <text class="content">{{ truncateText(review.content, 180) }}</text>
        <text class="score">{{ t('common.scorePrefix', { value: averageRating(review.ratings) }) }}</text>
      </view>
      <view v-if="hasMore" class="load-more" @tap="loadMore">
        <text>{{ loadingMore ? t('common.loading') : t('common.loadMore') }}</text>
      </view>
    </view>
  </scroll-view>
</template>

<style scoped>
.page { min-height: 100vh; background: #f8fafc; }
.state-card { margin: 24rpx; padding: 40rpx; background: #fff; border-radius: 24rpx; text-align: center; color: #64748b; }
.list-wrap { padding: 24rpx; }
.card { margin-bottom: 18rpx; padding: 28rpx; background: #fff; border-radius: 24rpx; box-shadow: 0 10rpx 30rpx rgba(15,23,42,.05); }
.title { display: block; font-size: 30rpx; font-weight: 700; color: #0f172a; }
.meta { display: block; margin-top: 10rpx; font-size: 22rpx; color: #64748b; }
.content { display: block; margin-top: 14rpx; font-size: 26rpx; line-height: 1.7; color: #334155; }
.score { display: block; margin-top: 14rpx; font-size: 24rpx; color: #4f46e5; }
.load-more { padding: 28rpx; text-align: center; color: #4f46e5; font-size: 26rpx; }
</style>
