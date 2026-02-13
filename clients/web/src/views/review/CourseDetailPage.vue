<template>
  <div
    class="max-w-[900px] mx-auto p-6 animate-fade-in max-sm:p-4"
    :class="{ '!max-w-none !p-0': isPanelMode }"
  >
    <!-- Loading -->
    <div v-if="loading" class="flex flex-col gap-6">
      <div class="h-[180px] rounded-xl bg-[length:200%_100%] bg-bg-secondary animate-shimmer" style="background-image: linear-gradient(90deg, var(--color-bg-secondary) 25%, var(--color-bg-hover) 50%, var(--color-bg-secondary) 75%)" />
      <div class="h-[44px] rounded-xl bg-[length:200%_100%] bg-bg-secondary animate-shimmer" style="background-image: linear-gradient(90deg, var(--color-bg-secondary) 25%, var(--color-bg-hover) 50%, var(--color-bg-secondary) 75%)" />
      <div class="h-[300px] rounded-xl bg-[length:200%_100%] bg-bg-secondary animate-shimmer" style="background-image: linear-gradient(90deg, var(--color-bg-secondary) 25%, var(--color-bg-hover) 50%, var(--color-bg-secondary) 75%)" />
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
            @click="course && openPostModal(course)"
          >
            {{ t('review.hub.postReview') }}
          </button>
          <FavoriteButton :course-i-d="courseID" />
        </div>
      </header>

      <!-- Tab Bar -->
      <nav class="sticky top-[var(--navbar-height)] z-10 py-3 mb-5 bg-bg-glass backdrop-blur-lg backdrop-saturate-150 border-b border-white/10 dark:border-white/5">
        <TabBar :tabs="tabItems" :model-value="activeTab" @update:model-value="activeTab = $event" />
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
          <ReviewCard v-for="r in filteredReviews" :key="r.id" :review="r" />
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
import { ref, computed, onMounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import TabBar from '@/components/common/TabBar.vue'

import EmptyState from '@/components/common/EmptyState.vue'
import InfiniteScroll from '@/components/common/InfiniteScroll.vue'
import FavoriteButton from '@/components/review/FavoriteButton.vue'
import ReviewCard from '@/components/review/ReviewCard.vue'
import DimensionBars from '@/components/review/DimensionBars.vue'
import RatingDistribution from '@/components/review/RatingDistribution.vue'
import SemesterStatsGrid from '@/components/review/SemesterStatsGrid.vue'
import TeacherFilter from '@/components/review/TeacherFilter.vue'
import TeacherStatsCard from '@/components/review/TeacherStatsCard.vue'
import { getCourse, getCourseRatingStats, getCourseTeachers } from '@/api/course'
import { getCourseReviews } from '@/api/review'
import type { Course, CourseRatingStatsResponse, TeacherStats } from '@/types/course'
import type { Review } from '@/types/review'
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

const { openPostModal, lastPostedAt } = useReviewPost()

// 排序
const sortBy = ref('time')
const sortTabs = computed(() => [
  { value: 'time', label: t('review.filters.latest') },
  { value: 'likes', label: t('review.filters.hottest') }
])

// 教师筛选
const selectedTeacher = ref('')
const uniqueTeachers = computed(() => {
  const names = new Set<string>()
  for (const r of reviews.value) {
    if (r.teacherName) names.add(r.teacherName)
  }
  return [...names].sort()
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

const fetchCourseTeachers = async () => {
  try {
    const res = await getCourseTeachers(courseID.value)
    courseTeachers.value = res.data || []
  } catch {
    // 教师数据加载失败静默处理
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
  selectedTeacher.value = ''
  fetchReviews()
}

const handlePosted = () => {
  page.value = 1
  fetchReviews()
  fetchRatingStats()
}

// 通过 ReviewDialog 发布测评后刷新数据
watch(lastPostedAt, () => {
  if (lastPostedAt.value > 0) handlePosted()
})

onMounted(async () => {
  if (isNaN(courseID.value) || courseID.value <= 0) {
    router.replace({ name: 'teaching-hub' })
    return
  }

  loading.value = true
  try {
    await Promise.all([fetchCourse(), fetchRatingStats(), fetchReviews(), fetchCourseTeachers()])
  } finally {
    loading.value = false
  }
})

// 路由参数变化时重新加载数据
watch(courseID, async (newID, oldID) => {
  if (newID === oldID || isNaN(newID) || newID <= 0) return
  page.value = 1
  sortBy.value = 'time'
  selectedTeacher.value = ''
  activeTab.value = 'overview'
  loading.value = true
  try {
    await Promise.all([fetchCourse(), fetchRatingStats(), fetchReviews(), fetchCourseTeachers()])
  } finally {
    loading.value = false
  }
})
</script>
