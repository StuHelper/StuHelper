<script setup lang="ts">
import { ref } from 'vue'
import { onShow } from '@dcloudio/uni-app'
import { api } from '@/api'
import type { components } from '@/api'
import { unwrapListData } from '@/api/result'
import { setPageTitle, translate } from '@/i18n'
import { useAuthStore } from '@/stores/auth'
import { averageRating, formatDateTime } from '@/utils/format'

const authStore = useAuthStore()
const t = translate
const loading = ref(false)
const loadingMore = ref(false)
const votes = ref<components['schemas']['Review'][]>([])
const page = ref(1)
const hasMore = ref(true)

async function loadVotes() {
  if (!(await authStore.requireAuth(t('user.votes.requireAuth')))) return
  loading.value = true
  page.value = 1
  hasMore.value = true
  try {
    const result = await api.user.getMyVotes(1, 20, 'like')
    const data = unwrapListData<components['schemas']['Review']>(result)
    votes.value = data.list
    hasMore.value = data.list.length >= 20
  } catch (error) {
    uni.showToast({ title: error instanceof Error ? error.message : t('user.votes.loadFailed'), icon: 'none' })
  } finally {
    loading.value = false
  }
}

async function loadMore() {
  if (loadingMore.value || !hasMore.value) return
  loadingMore.value = true
  try {
    page.value++
    const result = await api.user.getMyVotes(page.value, 20, 'like')
    const data = unwrapListData<components['schemas']['Review']>(result)
    votes.value = [...votes.value, ...data.list]
    hasMore.value = data.list.length >= 20
  } catch {
    page.value = Math.max(1, page.value - 1)
  } finally {
    loadingMore.value = false
  }
}

function openCourse(id: number) {
  uni.navigateTo({ url: `/pages/course/detail?id=${id}` })
}

onShow(() => {
  setPageTitle('common.pageTitles.myVotes')
  void loadVotes()
})
</script>

<template>
  <scroll-view class="page" scroll-y>
    <view v-if="loading" class="state-card"><text>{{ t('common.loading') }}</text></view>
    <view v-else-if="votes.length === 0" class="state-card"><text>{{ t('user.votes.empty') }}</text></view>
    <view v-else class="list-wrap">
      <view v-for="review in votes" :key="review.id" class="card" @tap="openCourse(review.courseID)">
        <text class="title">{{ review.title }}</text>
        <text class="meta">
          {{ review.courseName || t('common.courseFallback', { id: review.courseID }) }} ·
          {{ formatDateTime(review.createdAt) }}
        </text>
        <text class="content">{{ review.content }}</text>
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
