<template>
  <div
    class="max-w-[900px] mx-auto p-6 animate-fade-in max-sm:p-4"
    :class="{ '!max-w-none !p-0': isPanelMode }"
  >
    <!-- Loading -->
    <div v-if="loading" class="flex flex-col gap-6" role="status" aria-busy="true" :aria-label="t('common.actions.loading')">
      <div class="shimmer-box h-[180px] rounded-xl bg-bg-secondary" />
      <div class="shimmer-box h-[44px] rounded-xl bg-bg-secondary" />
      <div class="shimmer-box h-[300px] rounded-xl bg-bg-secondary" />
    </div>

    <!-- Error State -->
    <div v-else-if="error" class="text-center py-12">
      <p class="text-text-muted mb-4">{{ t('common.loadFailed') }}</p>
      <button
        class="px-4 py-2 bg-primary text-white rounded-lg border-none cursor-pointer text-sm"
        @click="fetchCourse()"
      >
        {{ t('common.actions.retry') }}
      </button>
    </div>

    <template v-else-if="course">
      <!-- Course Header — single inline row -->
      <header class="flex items-center gap-3 flex-wrap mb-5 max-sm:gap-2">
        <h1 class="font-sans text-2xl font-extrabold -tracking-wide text-text-primary max-sm:text-xl">
          {{ course.name }}
        </h1>
        <span v-if="course.departmentName" class="inline-flex items-center py-0.5 px-2.5 text-xs font-medium text-primary bg-primary/10 rounded-full">
          {{ course.departmentName }}
        </span>
        <span v-if="course.credits" class="inline-flex items-center py-0.5 px-2.5 text-xs text-text-secondary bg-bg-secondary rounded-full">
          {{ t('review.course.credits', { n: course.credits }) }}
        </span>
        <span v-if="course.code" class="inline-flex items-center py-0.5 px-2.5 text-xs text-text-secondary bg-bg-secondary rounded-full font-mono">
          {{ course.code }}
        </span>
        <div v-if="ratingStats" class="inline-flex items-center gap-1 text-sm font-semibold text-warning">
          <span>{{ overallAvgRating.toFixed(1) }}</span>
          <span class="text-text-muted font-normal text-xs">({{ total }}{{ t('review.course.reviewUnit') }})</span>
        </div>
        <div class="flex items-center gap-2 ml-auto max-sm:ml-0">
          <button
            class="py-1.5 px-4 text-sm font-medium text-white bg-success border-none rounded-lg cursor-pointer transition-[opacity,transform] duration-fast hover:opacity-90 hover:-translate-y-px"
            @click="goToPostPage"
          >
            {{ t('review.hub.postReview') }}
          </button>
          <FavoriteButton :course-i-d="courseID" />
        </div>
      </header>

      <!-- Tab Bar -->
      <nav class="sticky top-[var(--navbar-height)] z-10 py-3 mb-5 bg-bg-glass backdrop-blur-lg backdrop-saturate-150 border-b border-white/10 dark:border-white/5">
        <TabBar :tabs="tabItems" :model-value="activeTab" @update:model-value="handleTabChange" />
      </nav>

      <!-- Tab: 概览 -->
      <section v-if="activeTab === 'overview'" class="animate-fade-in">
        <div class="grid grid-cols-2 gap-5 max-sm:grid-cols-1">
          <div v-if="ratingStats" class="bg-bg-card border border-border rounded-xl p-5 shadow-card">
            <h3 class="text-sm font-semibold text-text-secondary mb-4">{{ t('review.rating.overall') }}</h3>
            <DimensionBars :dimensions="dimensionList" />
          </div>
          <div v-if="ratingStats" class="bg-bg-card border border-border rounded-xl p-5 shadow-card">
            <h3 class="text-sm font-semibold text-text-secondary mb-4">{{ t('review.course.reviews') }}</h3>
            <RatingDistribution :distribution="ratingDistribution" />
          </div>
        </div>
      </section>

      <!-- Tab: 测评 -->
      <section v-if="activeTab === 'reviews'" class="animate-fade-in">

        <TeacherFilter
          v-if="uniqueTeachers.length > 0"
          v-model="selectedTeacher"
          :teachers="uniqueTeachers"
          class="my-3"
        />

        <div class="my-4">
          <TabBar :tabs="sortTabs" :model-value="sortBy" @update:model-value="handleSortChange" />
        </div>

        <div class="flex flex-col gap-4 mb-4">
          <ReviewCard v-for="r in filteredReviews" :key="r.id" :review="r" @moderated="handleReviewModerated" />
        </div>

        <EmptyState v-if="!reviewsLoading && reviews.length === 0" :title="t('review.course.noReviews')" />

        <InfiniteScroll :loading="reviewsLoading" :has-more="hasMore" @load-more="loadMoreReviews" />
      </section>

      <!-- Tab: 统计 -->
      <section v-if="activeTab === 'stats'" class="animate-fade-in">
        <SemesterStatsGrid :rating-stats="ratingStats" />
      </section>

      <!-- Tab: 教师 -->
      <section v-if="activeTab === 'teachers'" class="animate-fade-in">
        <div v-if="courseTeachers.length" class="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <TeacherStatsCard v-for="teacher in courseTeachers" :key="teacher.teacherID" :stats="teacher" />
        </div>
        <EmptyState v-else :title="t('review.course.noTeacherData')" />
      </section>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onUnmounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import TabBar from '@/components/common/TabBar.vue'

import EmptyState from '@/components/common/EmptyState.vue'
import InfiniteScroll from '@/components/common/InfiniteScroll.vue'
import FavoriteButton from '@/components/business/review/FavoriteButton.vue'
import ReviewCard from '@/components/business/review/ReviewCard.vue'
import DimensionBars from '@/components/business/review/DimensionBars.vue'
import RatingDistribution from '@/components/business/review/RatingDistribution.vue'
import SemesterStatsGrid from '@/components/business/review/SemesterStatsGrid.vue'
import TeacherFilter from '@/components/business/review/TeacherFilter.vue'
import TeacherStatsCard from '@/components/business/review/TeacherStatsCard.vue'
import { api } from '@/api'
import type { Course, CourseRatingStatsResponse, TeacherStats } from '@/types/course'
import type { Review } from '@/types/review'
import { toReviews } from '@/types/review'
import { useReviewPost } from '@/composables/useReviewPost'

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
const courseTeachers = ref<TeacherStats[]>([])

// Tab 状态
type CourseDetailTab = 'overview' | 'reviews' | 'stats' | 'teachers'
const activeTab = ref<CourseDetailTab>('overview')
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
const limit = 20
const total = ref(0)
const hasMore = computed(() => reviews.value.length < total.value)

const { lastPostedAt } = useReviewPost()

// 排序
type SortOption = 'time' | 'likes' | 'rating'
const validSorts: SortOption[] = ['time', 'likes', 'rating']
const sortBy = ref<SortOption>('time')
const sortTabs = computed(() => [
  { value: 'time', label: t('review.filters.latest') },
  { value: 'likes', label: t('review.filters.hottest') }
])

// 教师筛选（M-75: 从 courseTeachers API 获取完整教师列表，而非从分页 reviews 中提取）
const selectedTeacher = ref('')
const uniqueTeachers = computed(() => {
  return courseTeachers.value
    .map(t => t.teacherName)
    .filter(Boolean)
    .sort()
})
const filteredReviews = computed(() => {
  if (!selectedTeacher.value) return reviews.value
  return reviews.value.filter(r => r.teacherName === selectedTeacher.value)
})

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

const error = ref(false)

function syncTabFromRoute() {
  activeTab.value = route.name === 'course-reviews' || route.name === 'review-course-detail'
    ? 'reviews'
    : 'overview'
}

function handleTabChange(tab: string) {
  const nextTab = tab as CourseDetailTab

  if (nextTab === 'overview') {
    activeTab.value = 'overview'
    if (route.name !== 'course-detail') {
      router.push({ name: 'course-detail', params: { id: courseID.value } })
    }
    return
  }

  if (nextTab === 'reviews') {
    activeTab.value = 'reviews'
    if (route.name !== 'course-reviews') {
      router.push({ name: 'course-reviews', params: { id: courseID.value } })
    }
    return
  }

  activeTab.value = nextTab
}

function goToPostPage() {
  router.push({ name: 'course-review-post', params: { id: courseID.value } })
}

const fetchCourse = async () => {
  try {
    const res = await api.course.getCourse(courseID.value)
    course.value = res.data?.data ?? null
    error.value = false
  } catch {
    course.value = null
    error.value = true
  }
}

const fetchRatingStats = async () => {
  try {
    const res = await api.rating.getCourseStats(courseID.value)
    ratingStats.value = res.data?.data ?? null
  } catch {
    // 评分统计可能不存在
  }
}

const fetchReviews = async (append = false, expectedVersion?: number) => {
  reviewsLoading.value = true
  try {
    const res = await api.review.getReviews(courseID.value, {
      page: page.value,
      pageSize: limit,
      sort: validSorts.includes(sortBy.value) ? sortBy.value : 'time'
    })
    // H-45: 仅在显式传入版本号时检查是否过期，undefined 表示不做版本检查
    if (expectedVersion !== undefined && expectedVersion !== loadVersion) return
    const list = toReviews(res.data?.data?.list || [])
    reviews.value = append ? [...reviews.value, ...list] : list
    total.value = res.data?.data?.total || 0
  } catch {
    // 评论加载失败静默处理，UI 已有空状态展示
  } finally {
    // H-45: undefined 时始终更新 loading 状态
    if (expectedVersion === undefined || expectedVersion === loadVersion) {
      reviewsLoading.value = false
    }
  }
}

const loadMoreReviews = () => {
  if (reviewsLoading.value || !hasMore.value) return
  page.value++
  fetchReviews(true)
}

const handleSortChange = (val: string) => {
  sortBy.value = val as SortOption
  page.value = 1
  selectedTeacher.value = ''
  // M-171: 递增 loadVersion 使进行中的旧请求被丢弃
  const version = ++loadVersion
  fetchReviews(false, version)
}

const handlePosted = () => {
  page.value = 1
  const version = ++loadVersion
  fetchReviews(false, version)
  fetchRatingStats()
}

const handleReviewModerated = () => {
  page.value = 1
  const version = ++loadVersion
  fetchReviews(false, version)
}

// 通过 ReviewDialog 发布测评后刷新数据
// M-60: 在 watch 之前初始化快照，避免 immediate watch 先执行导致遗漏
let lastPostedAtSnapshot = lastPostedAt.value
watch(lastPostedAt, (val) => {
  if (val > lastPostedAtSnapshot) {
    lastPostedAtSnapshot = val
    handlePosted()
  }
})

// H-09/M-46: 使用 loadVersion 防止并发请求交错写入
let loadVersion = 0

watch(() => route.name, () => {
  syncTabFromRoute()
}, { immediate: true })

// 组件卸载时递增 loadVersion，使所有进行中的请求响应被丢弃
onUnmounted(() => {
  ++loadVersion
})

// 统一数据加载入口：初始挂载 + 路由参数变化
// M-229: 快速切换 courseID 时，旧 Promise.all 响应到达后丢弃，防止数据不一致
watch(courseID, async (newID, oldID) => {
  if (oldID !== undefined && (newID === oldID || isNaN(newID) || newID <= 0)) return
  if (isNaN(newID) || newID <= 0) {
    router.replace({ name: 'course-hub' })
    return
  }

  const version = ++loadVersion
  // 重置数据状态，避免新数据加载前显示上一课程的内容
  course.value = null
  ratingStats.value = null
  courseTeachers.value = []
  reviews.value = []
  total.value = 0
  page.value = 1
  sortBy.value = 'time'
  selectedTeacher.value = ''
  syncTabFromRoute()
  loading.value = true
  try {
    const [courseRes, statsRes, reviewsRes, teachersRes] = await Promise.all([
      api.course.getCourse(newID).catch(() => null),
      api.rating.getCourseStats(newID).catch(() => null),
      api.review.getReviews(newID, { page: 1, pageSize: limit, sort: 'time' }).catch(() => null),
      api.rating.getCourseTeachers(newID).catch(() => null)
    ])
    // 版本已过期，丢弃全部结果
    if (version !== loadVersion) return
    course.value = courseRes?.data?.data ?? null
    error.value = !courseRes
    ratingStats.value = statsRes?.data?.data ?? null
    const list = (reviewsRes?.data?.data?.list || []) as Review[]
    reviews.value = list
    total.value = reviewsRes?.data?.data?.total || 0
    courseTeachers.value = teachersRes?.data?.data || []
  } finally {
    if (version === loadVersion) {
      loading.value = false
    }
  }
}, { immediate: true })
</script>

<style scoped>
/* L-38: 使用 transform: translateX() 替代 background-position 动画，降低 GPU 开销 */
.shimmer-box {
  position: relative;
  overflow: hidden;
}
.shimmer-box::after {
  content: '';
  position: absolute;
  inset: 0;
  background: linear-gradient(90deg, transparent 25%, var(--color-bg-hover) 50%, transparent 75%);
  transform: translateX(-100%);
  animation: shimmer-slide 1.5s infinite;
}
@keyframes shimmer-slide {
  to { transform: translateX(100%); }
}
</style>
