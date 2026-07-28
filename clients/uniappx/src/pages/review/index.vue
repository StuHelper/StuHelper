<script setup lang="ts">
import { ref } from 'vue'
import { onShow } from '@dcloudio/uni-app'
import { api } from '@/api'
import type { components } from '@/api'
import { assertMutationSuccess, unwrapListData } from '@/api/result'
import A11yButton from '@/components/A11yButton.vue'
import { setPageTitle, translate } from '@/i18n'
import { useAuthStore } from '@/stores/auth'
import { averageRating, formatShortDate, truncateText } from '@/utils/format'
import { DEFAULT_PAGE_SIZE } from '@/config/pagination'
import {
  applyOptimisticVote,
  createReviewVoteState,
  getDisplayVoteCount,
  type VoteType,
} from '@stuhelper/shared/review'

const authStore = useAuthStore()
const t = translate
const loading = ref(false)
const loadingMore = ref(false)
const sort = ref<'time' | 'likes' | 'rating'>('time')
const reviews = ref<components['schemas']['Review'][]>([])
const voting = ref<Record<string, boolean>>({})
const page = ref(1)
const total = ref(0)
const hasMore = ref(false)
let loadGeneration = 0

async function loadReviews() {
  const generation = ++loadGeneration
  const requestedSort = sort.value
  loading.value = true
  loadingMore.value = false
  try {
    const result = await api.review.getLatestReviews({
      page: 1,
      pageSize: DEFAULT_PAGE_SIZE,
      sort: requestedSort,
    })
    const data = unwrapListData<components['schemas']['Review']>(result)
    if (generation !== loadGeneration) return
    reviews.value = data.list
    page.value = 1
    total.value = data.total
    hasMore.value = reviews.value.length < total.value
  } catch (error) {
    if (generation !== loadGeneration) return
    uni.showToast({
      title: error instanceof Error ? error.message : t('review.index.loadFailed'),
      icon: 'none',
    })
  } finally {
    if (generation === loadGeneration) {
      loading.value = false
    }
  }
}

async function loadMore() {
  if (loading.value || loadingMore.value || !hasMore.value) return
  const generation = loadGeneration
  const nextPage = page.value + 1
  const requestedSort = sort.value
  loadingMore.value = true
  try {
    const result = await api.review.getLatestReviews({
      page: nextPage,
      pageSize: DEFAULT_PAGE_SIZE,
      sort: requestedSort,
    })
    const data = unwrapListData<components['schemas']['Review']>(result)
    if (generation !== loadGeneration) return
    reviews.value = [...reviews.value, ...data.list]
    page.value = nextPage
    total.value = data.total
    hasMore.value = reviews.value.length < total.value
  } catch (error) {
    if (generation !== loadGeneration) return
    uni.showToast({
      title: error instanceof Error ? error.message : t('review.index.loadFailed'),
      icon: 'none',
    })
  } finally {
    if (generation === loadGeneration) {
      loadingMore.value = false
    }
  }
}

async function vote(review: components['schemas']['Review'], type: VoteType) {
  if (voting.value[review.id]) return
  voting.value = { ...voting.value, [review.id]: true }
  let previousReview: components['schemas']['Review'] | null = null

  try {
    if (!(await authStore.requireAuth(t('review.index.requireAuthVote')))) return
    previousReview = { ...review }
    const nextState = applyOptimisticVote(
      createReviewVoteState({ userVote: review.userVote ?? null }),
      type,
    )

    reviews.value = reviews.value.map((item) => {
      if (item.id !== review.id) return item
      return {
        ...item,
        userVote: nextState.userVote ?? undefined,
        likeCount: getDisplayVoteCount(item.likeCount, nextState.likeOffset),
        dislikeCount: getDisplayVoteCount(item.dislikeCount, nextState.dislikeOffset),
      }
    })

    assertMutationSuccess(await api.review.voteReview(review.id, { voteType: type }))
  } catch (error) {
    if (previousReview) {
      reviews.value = reviews.value.map((item) => (
        item.id === review.id ? previousReview as components['schemas']['Review'] : item
      ))
    }
    uni.showToast({
      title: error instanceof Error ? error.message : t('review.index.voteFailed'),
      icon: 'none',
    })
  } finally {
    voting.value = { ...voting.value, [review.id]: false }
  }
}

function changeSort(nextSort: 'time' | 'likes' | 'rating') {
  if (sort.value === nextSort && reviews.value.length > 0) return
  sort.value = nextSort
  void loadReviews()
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
      <A11yButton data-testid="uni-review-sort-time" class="filter-btn" :class="{ active: sort === 'time' }" :aria-pressed="sort === 'time'" @tap="changeSort('time')">
        {{ t('review.index.sort.latest') }}
      </A11yButton>
      <A11yButton data-testid="uni-review-sort-likes" class="filter-btn" :class="{ active: sort === 'likes' }" :aria-pressed="sort === 'likes'" @tap="changeSort('likes')">
        {{ t('review.index.sort.hot') }}
      </A11yButton>
      <A11yButton data-testid="uni-review-sort-rating" class="filter-btn" :class="{ active: sort === 'rating' }" :aria-pressed="sort === 'rating'" @tap="changeSort('rating')">
        {{ t('review.index.sort.top') }}
      </A11yButton>
    </view>

    <view v-if="loading" class="state-card"><text>{{ t('common.loading') }}</text></view>
    <view v-else-if="reviews.length === 0" class="state-card"><text>{{ t('review.index.empty') }}</text></view>
    <view v-else class="list-wrap">
      <view v-for="review in reviews" :key="review.id" class="review-card" :data-testid="`uni-review-card-${review.id}`">
        <A11yButton class="review-head" :data-testid="`uni-review-open-${review.id}`" @tap="openCourse(review.courseID)">
          <view class="review-main">
            <text class="course-name">{{ review.courseName || t('common.courseFallback', { id: review.courseID }) }}</text>
            <text class="review-title">{{ review.title }}</text>
          </view>
          <text class="review-score">{{ averageRating(review.ratings) }}</text>
        </A11yButton>
        <text v-if="review.teacherName" class="review-teacher">{{ review.teacherName }}</text>
        <text class="review-content">{{ truncateText(review.content, 160) }}</text>
        <view class="review-meta">
          <text>{{ review.termName || review.termID }}</text>
          <text>{{ formatShortDate(review.createdAt) }}</text>
        </view>
        <view class="action-row">
          <A11yButton class="vote-btn" :data-testid="`uni-review-like-${review.id}`" :class="{ active: review.userVote === 'like' }" :aria-pressed="review.userVote === 'like'" :disabled="voting[review.id]" @tap="vote(review, 'like')">
            👍 {{ review.likeCount }}
          </A11yButton>
          <A11yButton class="vote-btn down" :data-testid="`uni-review-dislike-${review.id}`" :class="{ active: review.userVote === 'dislike' }" :aria-pressed="review.userVote === 'dislike'" :disabled="voting[review.id]" @tap="vote(review, 'dislike')">
            👎 {{ review.dislikeCount }}
          </A11yButton>
          <A11yButton class="detail-btn" :data-testid="`uni-review-view-course-${review.id}`" @tap="openCourse(review.courseID)">{{ t('review.index.viewCourse') }}</A11yButton>
        </view>
      </view>
      <A11yButton v-if="hasMore" class="load-more" data-testid="uni-review-load-more" :disabled="loadingMore" @tap="loadMore">
        {{ loadingMore ? t('common.loading') : t('common.loadMore') }}
      </A11yButton>
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
  width: 100%;
  padding: 0;
  border: 0;
  background: transparent;
  text-align: left;
  line-height: inherit;
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
  display: block;
  width: 100%;
  padding: 28rpx;
  border: 0;
  background: transparent;
  text-align: center;
  color: #4f46e5;
  font-size: 26rpx;
}

.load-more[disabled] {
  opacity: 0.6;
}
</style>
