<template>
  <div class="course-detail-page" :class="{ 'panel-mode': isPanelMode }">
    <!-- Loading -->
    <div v-if="loading" class="loading-state">
      <div class="skeleton-hero" />
      <div class="skeleton-tabs" />
      <div class="skeleton-content" />
    </div>

    <template v-else-if="course">
      <!-- Hero -->
      <header class="course-hero">
        <div class="hero-inner">
          <h1 class="hero-title">{{ course.name }}</h1>
          <div class="hero-meta">
            <span v-if="course.departmentName" class="meta-pill">
              {{ course.departmentName }}
            </span>
            <span v-if="course.credits" class="meta-pill">
              {{ t('review.course.credits', { n: course.credits }) }}
            </span>
            <span v-if="course.code" class="meta-pill font-mono">
              {{ course.code }}
            </span>
          </div>
          <div class="hero-actions">
            <button class="hero-post-btn" @click="scrollToReviews">
              {{ t('review.hub.postReview') }}
            </button>
            <FavoriteButton :course-i-d="courseID" />
          </div>
        </div>
        <div class="hero-rating" v-if="ratingStats">
          <RatingCircle
            :value="overallAvgRating"
            :size="88"
            :stroke-width="6"
            :subtitle="t('review.course.basedOnReviews', { count: total })"
          />
        </div>
      </header>

      <!-- Tab Bar -->
      <nav class="tab-bar">
        <TabBar :tabs="tabItems" :model-value="activeTab" @update:model-value="activeTab = $event" />
      </nav>

      <!-- Tab: 概览 -->
      <section v-if="activeTab === 'overview'" class="tab-content">
        <div class="overview-grid">
          <div class="overview-card" v-if="ratingStats">
            <h3 class="card-title">{{ t('review.rating.overall') }}</h3>
            <DimensionBars :dimensions="dimensionList" />
          </div>
          <div class="overview-card" v-if="ratingStats">
            <h3 class="card-title">{{ t('review.course.reviews') }}</h3>
            <RatingDistribution :distribution="ratingDistribution" />
          </div>
        </div>
      </section>

      <!-- Tab: 测评 -->
      <section v-if="activeTab === 'reviews'" ref="reviewsSection" class="tab-content">
        <ReviewForm :course-i-d="courseID" @posted="handlePosted" />

        <div class="reviews-header">
          <TabBar :tabs="sortTabs" :model-value="sortBy" @update:model-value="handleSortChange" />
        </div>

        <div class="review-list">
          <ReviewCard v-for="r in reviews" :key="r.id" :review="r" />
        </div>

        <EmptyState v-if="!reviewsLoading && reviews.length === 0" :title="t('review.course.noReviews')" />

        <InfiniteScroll :loading="reviewsLoading" :has-more="hasMore" @load-more="loadMoreReviews" />
      </section>

      <!-- Tab: 统计 -->
      <section v-if="activeTab === 'stats'" class="tab-content">
        <CourseRatingChart :course-i-d="courseID" />
      </section>

      <!-- Tab: 教师 -->
      <section v-if="activeTab === 'teachers'" class="tab-content">
        <EmptyState :title="t('review.course.noTeacherData')" />
      </section>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch, nextTick } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import TabBar from '@/components/common/TabBar.vue'
import RatingCircle from '@/components/common/RatingCircle.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import InfiniteScroll from '@/components/common/InfiniteScroll.vue'
import FavoriteButton from '@/components/review/FavoriteButton.vue'
import ReviewCard from '@/components/review/ReviewCard.vue'
import ReviewForm from '@/components/review/ReviewForm.vue'
import DimensionBars from '@/components/review/DimensionBars.vue'
import RatingDistribution from '@/components/review/RatingDistribution.vue'
import CourseRatingChart from '@/components/review/CourseRatingChart.vue'
import { getCourse, getCourseRatingStats } from '@/api/course'
import { getCourseReviews } from '@/api/review'
import type { Course, CourseRatingStatsResponse } from '@/types/course'
import type { Review } from '@/types/review'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const courseID = computed(() => Number(route.params.id))

// 面板模式：作为 ReviewPage 子路由时
const isPanelMode = computed(() => {
  return route.matched.some(r => r.name === 'review')
})

// 页面状态
const loading = ref(false)
const course = ref<Course | null>(null)
const ratingStats = ref<CourseRatingStatsResponse | null>(null)

// Tab 状态
const activeTab = ref('overview')
const tabItems = computed(() => [
  { value: 'overview', label: t('review.course.overview') },
  { value: 'reviews', label: t('review.course.reviews') },
  { value: 'stats', label: t('review.course.stats') },
  { value: 'teachers', label: t('review.course.teachers') }
])

// 测评列表状态
const reviews = ref<Review[]>([])
const reviewsLoading = ref(false)
const page = ref(1)
const pageSize = 20
const total = ref(0)
const hasMore = computed(() => reviews.value.length < total.value)
const reviewsSection = ref<HTMLElement | null>(null)

// 排序
const sortBy = ref('time')
const sortTabs = computed(() => [
  { value: 'time', label: t('review.filters.latest') },
  { value: 'likes', label: t('review.filters.hottest') }
])

// 总体平均评分（从维度计算）
const overallAvgRating = computed(() => {
  const dims = ratingStats.value?.overall?.dimensions
  if (!dims?.length) return 0
  const sum = dims.reduce((acc, d) => acc + d.avgRating, 0)
  return sum / dims.length
})

// 评分维度列表
const dimensionList = computed(() => {
  if (!ratingStats.value?.overall?.dimensions) return []
  return ratingStats.value.overall.dimensions.map(d => ({
    key: d.key,
    name: d.name,
    avgRating: d.avgRating
  }))
})

// 评分分布
const ratingDistribution = computed(() => {
  if (!ratingStats.value?.overall?.dimensions?.length) return {}
  const dist: Record<number, number> = { 1: 0, 2: 0, 3: 0, 4: 0, 5: 0 }
  const firstDim = ratingStats.value.overall.dimensions[0]
  if (firstDim?.distribution) {
    for (const [key, val] of Object.entries(firstDim.distribution)) {
      dist[Number(key)] = val
    }
  }
  return dist
})

const fetchCourse = async () => {
  const res = await getCourse(courseID.value)
  course.value = res.data
}

const fetchRatingStats = async () => {
  try {
    const res = await getCourseRatingStats(courseID.value)
    ratingStats.value = res.data
  } catch {
    // 评分统计可能不存在
  }
}

const fetchReviews = async (append = false) => {
  reviewsLoading.value = true
  try {
    const res = await getCourseReviews(courseID.value, {
      page: page.value,
      pageSize,
      sort: sortBy.value as 'time' | 'likes' | 'rating'
    })
    const list = res.data?.list || []
    reviews.value = append ? [...reviews.value, ...list] : list
    total.value = res.data?.total || 0
  } catch {
    // 评论加载失败静默处理，UI 已有空状态展示
  } finally {
    reviewsLoading.value = false
  }
}

const loadMoreReviews = () => {
  if (reviewsLoading.value || !hasMore.value) return
  page.value++
  fetchReviews(true)
}

const handleSortChange = (val: string) => {
  sortBy.value = val
  page.value = 1
  fetchReviews()
}

const handlePosted = () => {
  page.value = 1
  fetchReviews()
  fetchRatingStats()
}

const scrollToReviews = () => {
  activeTab.value = 'reviews'
  nextTick(() => {
    reviewsSection.value?.scrollIntoView({ behavior: 'smooth' })
  })
}

onMounted(async () => {
  if (isNaN(courseID.value) || courseID.value <= 0) {
    router.replace({ name: 'teaching-hub' })
    return
  }

  loading.value = true
  try {
    await Promise.all([fetchCourse(), fetchRatingStats(), fetchReviews()])
  } finally {
    loading.value = false
  }
})

// 路由参数变化时重新加载数据
watch(courseID, async (newID, oldID) => {
  if (newID === oldID || isNaN(newID) || newID <= 0) return
  page.value = 1
  sortBy.value = 'time'
  activeTab.value = 'overview'
  loading.value = true
  try {
    await Promise.all([fetchCourse(), fetchRatingStats(), fetchReviews()])
  } finally {
    loading.value = false
  }
})
</script>

<style scoped>
.course-detail-page {
  max-width: 900px;
  margin: 0 auto;
  padding: var(--space-6);
  animation: fadeIn var(--duration-base) var(--ease-out);
}

.course-detail-page.panel-mode {
  max-width: none;
  padding: 0;
}

/* Loading State */
.loading-state {
  display: flex;
  flex-direction: column;
  gap: var(--space-6);
}

.skeleton-hero,
.skeleton-tabs,
.skeleton-content {
  background: linear-gradient(90deg, var(--bg-secondary) 25%, var(--bg-tertiary) 50%, var(--bg-secondary) 75%);
  background-size: 200% 100%;
  border-radius: var(--radius-lg);
  animation: shimmer 1.5s ease-in-out infinite;
}

.skeleton-hero {
  height: 180px;
}

.skeleton-tabs {
  height: 44px;
}

.skeleton-content {
  height: 300px;
}

/* Course Hero */
.course-hero {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: var(--space-6);
  padding: var(--space-8) var(--space-6);
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius-xl);
  margin-bottom: var(--space-5);
}

.hero-inner {
  flex: 1;
  min-width: 0;
}

.hero-title {
  font-family: var(--font-sans);
  font-size: var(--text-2xl);
  font-weight: var(--weight-extrabold);
  letter-spacing: var(--tracking-tight);
  color: var(--text-primary);
  margin: 0 0 var(--space-3) 0;
}

.hero-meta {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-2);
  margin-bottom: var(--space-4);
}

.meta-pill {
  display: inline-flex;
  align-items: center;
  padding: 2px 10px;
  font-size: var(--text-xs);
  color: var(--text-secondary);
  background: var(--bg-secondary);
  border-radius: var(--radius-full);
  text-decoration: none;
}

.meta-teacher {
  color: var(--brand-primary);
  background: color-mix(in srgb, var(--brand-primary) 10%, transparent);
}

.meta-teacher:hover {
  background: color-mix(in srgb, var(--brand-primary) 18%, transparent);
}

.hero-actions {
  display: flex;
  align-items: center;
  gap: var(--space-3);
}

.hero-post-btn {
  padding: var(--space-2) var(--space-5);
  font-size: var(--text-sm);
  font-weight: var(--weight-medium);
  color: white;
  background: var(--gradient-brand);
  border: none;
  border-radius: var(--radius-full);
  cursor: pointer;
  transition: opacity var(--duration-fast), transform var(--duration-fast);
}

.hero-post-btn:hover {
  opacity: 0.9;
  transform: translateY(-1px);
}

.hero-rating {
  flex-shrink: 0;
}

/* Tab Bar */
.tab-bar {
  position: sticky;
  top: 56px;
  z-index: 10;
  padding: var(--space-3) 0;
  margin-bottom: var(--space-5);
  background: var(--bg-base);
}

/* Tab Content */
.tab-content {
  animation: fadeIn var(--duration-base) var(--ease-out);
}

/* Overview Grid */
.overview-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: var(--space-5);
}

.overview-card {
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  padding: var(--space-5);
}

.card-title {
  font-size: var(--text-sm);
  font-weight: var(--weight-semibold);
  color: var(--text-secondary);
  margin: 0 0 var(--space-4) 0;
}

/* Reviews Header */
.reviews-header {
  margin: var(--space-4) 0;
}

.review-list {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
  margin-bottom: var(--space-4);
}

/* Responsive */
@media (max-width: 640px) {
  .course-detail-page {
    padding: var(--space-4);
  }

  .course-hero {
    flex-direction: column;
    padding: var(--space-5) var(--space-4);
  }

  .hero-title {
    font-size: var(--text-xl);
  }

  .hero-rating {
    align-self: center;
  }

  .overview-grid {
    grid-template-columns: 1fr;
  }
}
</style>
