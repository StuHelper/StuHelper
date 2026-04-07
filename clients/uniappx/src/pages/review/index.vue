<script setup lang="ts">
import { ref } from 'vue'
import { onShow } from '@dcloudio/uni-app'
import { api } from '@/api'
import type { components } from '@/api'
import { assertMutationSuccess, unwrapListData } from '@/api/result'
import { setPageTitle, translate } from '@/i18n'
import { useAuthStore } from '@/stores/auth'
import { averageRating, formatShortDate, truncateText } from '@/utils/format'
import { DEFAULT_PAGE_SIZE } from '@/config/pagination'

type VoteType = 'like' | 'dislike'
const LOCAL_VOTES_STORAGE_KEY = 'stuhelper:uniappx:review-votes'

const authStore = useAuthStore()
const t = translate
const loading = ref(false)
const loadingMore = ref(false)
const sort = ref<'time' | 'likes' | 'rating'>('time')
const reviews = ref<components['schemas']['Review'][]>([])
const voting = ref<Record<string, boolean>>({})
const localVotes = ref<Record<string, VoteType | null>>(loadLocalVotes())
const page = ref(1)
const hasMore = ref(true)

const VALID_VOTE_VALUES = new Set<string | null>(['like', 'dislike', null])

function loadLocalVotes(): Record<string, VoteType | null> {
  try {
    const raw = uni.getStorageSync(LOCAL_VOTES_STORAGE_KEY)
    if (typeof raw !== 'string' || !raw) return {}
    const parsed: unknown = JSON.parse(raw)
    if (typeof parsed !== 'object' || parsed === null || Array.isArray(parsed)) return {}
    const result: Record<string, VoteType | null> = {}
    for (const [key, value] of Object.entries(parsed as Record<string, unknown>)) {
      if (typeof key === 'string' && VALID_VOTE_VALUES.has(value as string | null)) {
        result[key] = value as VoteType | null
      }
    }
    return result
  } catch {
    return {}
  }
}

function persistLocalVotes() {
  try {
    uni.setStorageSync(LOCAL_VOTES_STORAGE_KEY, JSON.stringify(localVotes.value))
  } catch {
    // ignore storage failures
  }
}

async function loadReviews() {
  loading.value = true
  page.value = 1
  hasMore.value = true
  try {
    const result = await api.review.getLatestReviews({ page: 1, pageSize: DEFAULT_PAGE_SIZE, sort: sort.value })
    const data = unwrapListData<components['schemas']['Review']>(result)
    reviews.value = data.list
    hasMore.value = data.list.length >= DEFAULT_PAGE_SIZE
  } catch (error) {
    uni.showToast({
      title: error instanceof Error ? error.message : t('review.index.loadFailed'),
      icon: 'none',
    })
  } finally {
    loading.value = false
  }
}

async function loadMore() {
  if (loadingMore.value || !hasMore.value) return
  loadingMore.value = true
  try {
    page.value++
    const result = await api.review.getLatestReviews({ page: page.value, pageSize: DEFAULT_PAGE_SIZE, sort: sort.value })
    const data = unwrapListData<components['schemas']['Review']>(result)
    reviews.value = [...reviews.value, ...data.list]
    hasMore.value = data.list.length >= 20
  } catch {
    page.value = Math.max(1, page.value - 1)
  } finally {
    loadingMore.value = false
  }
}

async function vote(review: components['schemas']['Review'], type: VoteType) {
  if (!(await authStore.requireAuth(t('review.index.requireAuthVote')))) return
  if (voting.value[review.id]) return
  voting.value = { ...voting.value, [review.id]: true }
  const previousVote = localVotes.value[review.id] ?? null
  localVotes.value = { ...localVotes.value, [review.id]: type }
  persistLocalVotes()

  const nextReviews = reviews.value.map((item) => {
    if (item.id !== review.id) return item
    let likeCount = item.likeCount
    let dislikeCount = item.dislikeCount
    if (previousVote === 'like') likeCount -= 1
    if (previousVote === 'dislike') dislikeCount -= 1
    if (type === 'like') likeCount += 1
    if (type === 'dislike') dislikeCount += 1
    return { ...item, likeCount, dislikeCount }
  })
  reviews.value = nextReviews

  try {
    assertMutationSuccess(await api.review.voteReview(review.id, { voteType: type }))
  } catch (error) {
    localVotes.value = { ...localVotes.value, [review.id]: previousVote }
    persistLocalVotes()
    reviews.value = reviews.value.map((item) => (item.id === review.id ? review : item))
    uni.showToast({
      title: error instanceof Error ? error.message : t('review.index.voteFailed'),
      icon: 'none',
    })
  } finally {
    voting.value = { ...voting.value, [review.id]: false }
  }
}

function openCourse(courseID: number) {
  uni.navigateTo({ url: `/pages/course/detail?id=${courseID}` })
}

onShow(() => {
  setPageTitle('common.pageTitles.reviewSquare')
  void loadReviews()
})
</script>

<template>
  <scroll-view class="review-page" scroll-y>
    <view class="filter-row">
      <button class="filter-btn" :class="{ active: sort === 'time' }" @tap="sort = 'time'; loadReviews()">
        {{ t('review.index.sort.latest') }}
      </button>
      <button class="filter-btn" :class="{ active: sort === 'likes' }" @tap="sort = 'likes'; loadReviews()">
        {{ t('review.index.sort.hot') }}
      </button>
      <button class="filter-btn" :class="{ active: sort === 'rating' }" @tap="sort = 'rating'; loadReviews()">
        {{ t('review.index.sort.top') }}
      </button>
    </view>

    <view v-if="loading" class="state-card"><text>{{ t('common.loading') }}</text></view>
    <view v-else-if="reviews.length === 0" class="state-card"><text>{{ t('review.index.empty') }}</text></view>
    <view v-else class="list-wrap">
      <view v-for="review in reviews" :key="review.id" class="review-card">
        <view class="review-head" @tap="openCourse(review.courseID)">
          <view class="review-main">
            <text class="course-name">{{ review.courseName || t('common.courseFallback', { id: review.courseID }) }}</text>
            <text class="review-title">{{ review.title }}</text>
          </view>
          <text class="review-score">{{ averageRating(review.ratings) }}</text>
        </view>
        <text v-if="review.teacherName" class="review-teacher">{{ review.teacherName }}</text>
        <text class="review-content">{{ truncateText(review.content, 160) }}</text>
        <view class="review-meta">
          <text>{{ review.termName || review.termID }}</text>
          <text>{{ formatShortDate(review.createdAt) }}</text>
        </view>
        <view class="action-row">
          <button class="vote-btn" :class="{ active: localVotes[review.id] === 'like' }" :disabled="voting[review.id]" @tap="vote(review, 'like')">
            👍 {{ review.likeCount }}
          </button>
          <button class="vote-btn down" :class="{ active: localVotes[review.id] === 'dislike' }" :disabled="voting[review.id]" @tap="vote(review, 'dislike')">
            👎 {{ review.dislikeCount }}
          </button>
          <button class="detail-btn" @tap="openCourse(review.courseID)">{{ t('review.index.viewCourse') }}</button>
        </view>
      </view>
      <view v-if="hasMore" class="load-more" @tap="loadMore">
        <text>{{ loadingMore ? t('common.loading') : t('common.loadMore') }}</text>
      </view>
    </view>
  </scroll-view>
</template>

<style scoped>
.review-page {
  min-height: 100vh;
  background: #f8fafc;
}

.filter-row {
  display: flex;
  gap: 16rpx;
  padding: 24rpx;
}

.filter-btn,
.vote-btn,
.detail-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 20rpx;
  font-size: 24rpx;
}

.filter-btn {
  flex: 1;
  height: 76rpx;
  background: #ffffff;
  color: #475569;
}

.filter-btn.active {
  background: #4f46e5;
  color: #ffffff;
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

.review-card {
  margin-bottom: 18rpx;
  padding: 28rpx;
  background: #ffffff;
  border-radius: 24rpx;
  box-shadow: 0 10rpx 30rpx rgba(15, 23, 42, 0.05);
}

.review-head {
  display: flex;
  justify-content: space-between;
  gap: 16rpx;
}

.review-main {
  flex: 1;
}

.course-name {
  display: block;
  font-size: 24rpx;
  color: #4f46e5;
}

.review-title {
  display: block;
  margin-top: 8rpx;
  font-size: 30rpx;
  font-weight: 700;
  color: #0f172a;
}

.review-score {
  font-size: 32rpx;
  font-weight: 700;
  color: #4f46e5;
}

.review-teacher,
.review-meta {
  font-size: 22rpx;
  color: #64748b;
}

.review-teacher {
  display: block;
  margin-top: 10rpx;
}

.review-content {
  display: block;
  margin-top: 14rpx;
  font-size: 26rpx;
  line-height: 1.7;
  color: #334155;
}

.review-meta {
  display: flex;
  justify-content: space-between;
  margin-top: 18rpx;
}

.action-row {
  display: flex;
  gap: 16rpx;
  margin-top: 18rpx;
}

.vote-btn,
.detail-btn {
  flex: 1;
  height: 72rpx;
  background: #eef2ff;
  color: #4338ca;
}

.vote-btn.down {
  background: #fef2f2;
  color: #dc2626;
}

.vote-btn.active {
  border: 2rpx solid currentColor;
}

.detail-btn {
  background: #f8fafc;
  color: #0f172a;
}

.load-more {
  padding: 28rpx;
  text-align: center;
  color: #4f46e5;
  font-size: 26rpx;
}
</style>
