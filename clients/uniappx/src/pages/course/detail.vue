<script setup lang="ts">
import { computed, ref } from 'vue'
import { onLoad, onShow } from '@dcloudio/uni-app'
import { api } from '@/api'
import type { components } from '@/api'
import { assertMutationSuccess, unwrapData, unwrapListData, unwrapOptionalData } from '@/api/result'
import { setPageTitle, translate } from '@/i18n'
import { useAuthStore } from '@/stores/auth'
import { averageRating, truncateText } from '@/utils/format'

const authStore = useAuthStore()
const t = translate
const courseID = ref(0)
const loading = ref(false)
const course = ref<components['schemas']['Course'] | null>(null)
const stats = ref<components['schemas']['CourseRatingStatsResponse'] | null>(null)
const teachers = ref<components['schemas']['CourseTeacherStats'][]>([])
const reviews = ref<components['schemas']['Review'][]>([])
const favoriteLoading = ref(false)
const isFavorite = ref(false)
const lastLoadedAt = ref(0)
const lastLoadedAuthenticated = ref<boolean | null>(null)
const DETAIL_STALE_MS = 30_000

// Reply state per review
const expandedReview = ref<string | null>(null)
const replies = ref<components['schemas']['Reply'][]>([])
const repliesLoading = ref(false)
const replyText = ref('')
const replySubmitting = ref(false)

async function loadReplies(reviewId: string) {
  repliesLoading.value = true
  replies.value = []
  try {
    const result = await api.reply.getReplies(reviewId, { page: 1, pageSize: 50 })
    replies.value = unwrapListData<components['schemas']['Reply']>(result).list
  } catch (_error) { void _error;
    uni.showToast({ title: t('course.detail.repliesLoadFailed'), icon: 'none' })
  } finally {
    repliesLoading.value = false
  }
}

async function toggleReplies(reviewId: string) {
  if (expandedReview.value === reviewId) {
    expandedReview.value = null
    return
  }
  expandedReview.value = reviewId
  replyText.value = ''
  await loadReplies(reviewId)
}

async function submitReply(reviewId: string) {
  const text = replyText.value.trim()
  if (!text) return
  if (!(await authStore.requireAuth(t('course.detail.requireAuthReply')))) return
  replySubmitting.value = true
  try {
    assertMutationSuccess(await api.reply.createReply(reviewId, { content: text }))
    replyText.value = ''
    uni.showToast({ title: t('course.detail.replySent'), icon: 'none' })
    expandedReview.value = reviewId
    await loadReplies(reviewId)
  } catch (error) {
    uni.showToast({ title: error instanceof Error ? error.message : t('course.detail.replyFailed'), icon: 'none' })
  } finally {
    replySubmitting.value = false
  }
}

const summaryDimensions = computed(() => stats.value?.overall?.dimensions || [])

async function loadDetail(force = false) {
  if (!courseID.value) return
  const now = Date.now()
  if (
    !force &&
    lastLoadedAt.value > 0 &&
    now - lastLoadedAt.value < DETAIL_STALE_MS &&
    lastLoadedAuthenticated.value === authStore.isAuthenticated
  ) {
    return
  }
  loading.value = true
  try {
    const [courseResult, statsResult, teachersResult, reviewsResult] = await Promise.all([
      api.course.getCourse(courseID.value),
      api.rating.getCourseStats(courseID.value),
      api.rating.getCourseTeachers(courseID.value),
      api.review.getReviews(courseID.value, { page: 1, pageSize: 10, sort: 'likes' }),
    ])

    course.value = unwrapOptionalData<components['schemas']['Course']>(courseResult)
    stats.value = unwrapOptionalData<components['schemas']['CourseRatingStatsResponse']>(statsResult)
    teachers.value = unwrapData<components['schemas']['CourseTeacherStats'][]>(teachersResult)
    reviews.value = unwrapListData<components['schemas']['Review']>(reviewsResult).list

    if (authStore.isAuthenticated) {
      const favoriteStatusResult = await api.user.getFavoriteStatus(courseID.value)
      const favoriteStatus = unwrapData<{ favorited: boolean }>(favoriteStatusResult)
      isFavorite.value = favoriteStatus.favorited
    } else {
      isFavorite.value = false
    }
    lastLoadedAuthenticated.value = authStore.isAuthenticated
    lastLoadedAt.value = Date.now()
  } catch (error) {
    uni.showToast({
      title: error instanceof Error ? error.message : t('course.detail.loadFailed'),
      icon: 'none',
    })
  } finally {
    loading.value = false
  }
}

async function toggleFavorite() {
  if (!(await authStore.requireAuth(t('course.detail.requireAuthFavorite')))) return
  if (!course.value || favoriteLoading.value) return
  favoriteLoading.value = true
  try {
    if (isFavorite.value) {
      assertMutationSuccess(await api.user.removeFavorite(course.value.id))
      isFavorite.value = false
      uni.showToast({ title: t('course.detail.favoriteRemoved'), icon: 'none' })
    } else {
      assertMutationSuccess(await api.user.addFavorite(course.value.id))
      isFavorite.value = true
      uni.showToast({ title: t('course.detail.favoriteSaved'), icon: 'success' })
    }
  } catch (error) {
    uni.showToast({
      title: error instanceof Error ? error.message : t('course.detail.favoriteFailed'),
      icon: 'none',
    })
  } finally {
    favoriteLoading.value = false
  }
}

async function goPostReview() {
  if (!(await authStore.requireAuth(t('course.detail.requireAuthReview')))) return
  uni.navigateTo({ url: `/pages/review/post?courseID=${courseID.value}` })
}

function goTeacher(teacherID: number) {
  uni.navigateTo({ url: `/pages/teacher/profile?id=${teacherID}` })
}

onLoad((options) => {
  setPageTitle('common.pageTitles.courseDetail')
  courseID.value = Number(options?.id || options?.courseID || 0)
  void loadDetail(true)
})

onShow(() => {
  setPageTitle('common.pageTitles.courseDetail')
  if (!courseID.value) return
  void loadDetail()
})
</script>

<template>
  <scroll-view class="detail-page" scroll-y>
    <view v-if="loading" class="state-card"><text>{{ t('common.loading') }}</text></view>
    <view v-else-if="!course" class="state-card"><text>{{ t('course.detail.missing') }}</text></view>
    <view v-else>
      <view class="hero-card">
        <view class="hero-top">
          <view>
            <text class="course-name">{{ course.name }}</text>
            <text class="course-code">{{ course.code || t('common.unavailableCourseCode') }}</text>
          </view>
          <button class="favorite-btn" data-testid="uni-course-favorite" :disabled="favoriteLoading" @tap="toggleFavorite">
            {{ favoriteLoading ? t('common.processing') : isFavorite ? t('course.detail.favorited') : t('course.detail.favorite') }}
          </button>
        </view>
        <view class="hero-meta">
          <text>{{ course.departmentName || t('common.unclassifiedDepartment') }}</text>
          <text>{{ t('common.creditValue', { value: course.credits }) }}</text>
          <text>{{ t('common.reviewCount', { count: course.reviewCount }) }}</text>
        </view>
        <button class="primary-btn" @tap="goPostReview">{{ t('course.detail.writeReview') }}</button>
      </view>

      <view class="section-card">
        <text class="section-title">{{ t('course.detail.summaryTitle') }}</text>
        <view v-if="summaryDimensions.length === 0" class="empty-text">
          <text>{{ t('course.detail.noStats') }}</text>
        </view>
        <view v-else>
          <view v-for="dimension in summaryDimensions" :key="dimension.key" class="dimension-row">
            <view>
              <text class="dimension-name">{{ dimension.name }}</text>
              <text class="dimension-meta">{{ t('common.ratingCount', { count: dimension.ratingCount }) }}</text>
            </view>
            <text class="dimension-score">{{ dimension.avgRating.toFixed(1) }}</text>
          </view>
        </view>
      </view>

      <view class="section-card">
        <text class="section-title">{{ t('course.detail.teachersTitle') }}</text>
        <view v-if="teachers.length === 0" class="empty-text"><text>{{ t('course.detail.noTeachers') }}</text></view>
        <view v-else>
          <view v-for="teacher in teachers" :key="teacher.teacherID" class="teacher-row" @tap="goTeacher(teacher.teacherID)">
            <view>
              <text class="teacher-name">{{ teacher.teacherName }}</text>
              <text class="teacher-meta">
                {{ teacher.departmentName }} · {{ t('common.reviewCount', { count: teacher.reviewCount }) }}
              </text>
            </view>
            <text class="teacher-score">{{ teacher.avgRating ? teacher.avgRating.toFixed(1) : '--' }}</text>
          </view>
        </view>
      </view>

      <view class="section-card">
        <text class="section-title">{{ t('course.detail.featuredReviewsTitle') }}</text>
        <view v-if="reviews.length === 0" class="empty-text"><text>{{ t('course.detail.noReviews') }}</text></view>
        <view v-else>
          <view v-for="review in reviews" :key="review.id" class="review-card">
            <view class="review-head">
              <text class="review-title">{{ review.title }}</text>
              <text class="review-score">{{ averageRating(review.ratings) }}</text>
            </view>
            <text v-if="review.teacherName" class="review-teacher">{{ review.teacherName }}</text>
            <text class="review-content">{{ truncateText(review.content, 180) }}</text>
            <view class="review-meta">
              <text>👍 {{ review.likeCount }}</text>
              <text class="reply-toggle" :data-testid="`uni-review-replies-${review.id}`" @tap="toggleReplies(review.id)">💬 {{ review.replyCount }}</text>
              <text>{{ review.termName || review.termID }}</text>
            </view>
            <!-- Replies section -->
            <view v-if="expandedReview === review.id" class="replies-section">
              <view v-if="repliesLoading" class="reply-loading"><text>{{ t('common.loading') }}</text></view>
              <view v-else-if="replies.length > 0" class="reply-list">
                <view v-for="reply in replies" :key="reply.id" class="reply-item">
                  <text class="reply-content">{{ reply.content }}</text>
                  <text class="reply-time">{{ reply.createdAt }}</text>
                </view>
              </view>
              <view v-else class="reply-empty"><text>{{ t('course.detail.noReplies') }}</text></view>
              <view class="reply-input-row">
                <input
                  v-model="replyText"
                  class="reply-input"
                  :data-testid="`uni-review-reply-input-${review.id}`"
                  :placeholder="t('course.detail.replyPlaceholder')"
                  :disabled="replySubmitting"
                />
                <button class="reply-submit-btn" :data-testid="`uni-review-reply-submit-${review.id}`" :disabled="replySubmitting || !replyText.trim()" @tap="submitReply(review.id)">
                  {{ t('course.detail.replySubmit') }}
                </button>
              </view>
            </view>
          </view>
        </view>
      </view>
    </view>
  </scroll-view>
</template>

<style scoped>
.detail-page {
  min-height: 100vh;
  background: #f8fafc;
}

.state-card,
.section-card,
.hero-card {
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

.hero-card {
  padding: 32rpx;
}

.hero-top {
  display: flex;
  justify-content: space-between;
  gap: 16rpx;
}

.course-name {
  display: block;
  font-size: 36rpx;
  font-weight: 700;
  color: #0f172a;
}

.course-code,
.hero-meta,
.dimension-meta,
.teacher-meta,
.review-teacher,
.review-meta,
.empty-text {
  color: #64748b;
  font-size: 24rpx;
}

.course-code {
  display: block;
  margin-top: 8rpx;
}

.hero-meta {
  display: flex;
  justify-content: space-between;
  margin-top: 18rpx;
}

.favorite-btn,
.primary-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 20rpx;
  font-size: 26rpx;
  font-weight: 600;
}

.favorite-btn {
  min-width: 148rpx;
  padding: 0 18rpx;
  height: 76rpx;
  background: #eef2ff;
  color: #4338ca;
}

.primary-btn {
  margin-top: 24rpx;
  height: 84rpx;
  background: #4f46e5;
  color: #ffffff;
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

.dimension-row,
.teacher-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 20rpx 0;
  border-bottom: 1rpx solid #e2e8f0;
}

.dimension-row:last-child,
.teacher-row:last-child {
  border-bottom: none;
}

.dimension-name,
.teacher-name,
.review-title {
  display: block;
  font-size: 28rpx;
  font-weight: 600;
  color: #0f172a;
}

.dimension-score,
.teacher-score,
.review-score {
  font-size: 32rpx;
  font-weight: 700;
  color: #4f46e5;
}

.review-card {
  padding: 24rpx 0;
  border-bottom: 1rpx solid #e2e8f0;
}

.review-card:last-child {
  border-bottom: none;
}

.review-head {
  display: flex;
  justify-content: space-between;
  gap: 16rpx;
}

.review-teacher {
  display: block;
  margin-top: 8rpx;
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
  margin-top: 16rpx;
}

.reply-toggle {
  color: #4f46e5;
}

.replies-section {
  margin-top: 16rpx;
  padding-top: 16rpx;
  border-top: 1rpx solid #e2e8f0;
}

.reply-loading,
.reply-empty {
  padding: 16rpx 0;
  text-align: center;
  color: #64748b;
  font-size: 24rpx;
}

.reply-item {
  padding: 12rpx 0;
  border-bottom: 1rpx solid #f1f5f9;
}

.reply-item:last-child {
  border-bottom: none;
}

.reply-content {
  display: block;
  font-size: 24rpx;
  color: #334155;
  line-height: 1.6;
}

.reply-time {
  display: block;
  margin-top: 4rpx;
  font-size: 20rpx;
  color: #94a3b8;
}

.reply-input-row {
  display: flex;
  gap: 12rpx;
  margin-top: 16rpx;
}

.reply-input {
  flex: 1;
  height: 68rpx;
  padding: 0 16rpx;
  border-radius: 16rpx;
  background: #f8fafc;
  border: 1rpx solid #e2e8f0;
  font-size: 24rpx;
}

.reply-submit-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  min-width: 120rpx;
  height: 68rpx;
  border-radius: 16rpx;
  background: #4f46e5;
  color: #fff;
  font-size: 24rpx;
}
</style>
