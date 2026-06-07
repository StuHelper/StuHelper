<template>
  <div class="max-w-[900px] mx-auto p-6 animate-fade-in max-sm:p-4">
    <!-- Loading State -->
    <div v-if="loading" class="flex flex-col gap-6">
      <div
        class="h-[120px] rounded-lg bg-[length:200%_100%] animate-shimmer"
        :style="{ backgroundImage: 'linear-gradient(90deg, var(--color-bg-secondary) 25%, var(--color-bg-tertiary) 50%, var(--color-bg-secondary) 75%)' }"
      />
      <div
        class="h-[300px] rounded-lg bg-[length:200%_100%] animate-shimmer"
        :style="{ backgroundImage: 'linear-gradient(90deg, var(--color-bg-secondary) 25%, var(--color-bg-tertiary) 50%, var(--color-bg-secondary) 75%)' }"
      />
    </div>

    <!-- Content -->
    <template v-else-if="teacher">
      <!-- Teacher Header -->
      <header class="flex items-center gap-5 p-6 bg-bg-card rounded-xl mb-6 max-sm:flex-col max-sm:text-center">
        <div class="p-[3px] bg-gradient-to-br from-primary to-accent rounded-full shrink-0">
          <div class="w-16 h-16 bg-bg-card rounded-full flex items-center justify-center text-text-muted">
            <User :size="32" :stroke-width="1.5" />
          </div>
        </div>
        <div class="flex-1">
          <h1 class="text-xl font-extrabold tracking-tight text-text-primary m-0 mb-2">{{ teacher.teacherName }}</h1>
          <span class="inline-block text-xs text-text-secondary bg-bg-secondary py-0.5 px-2.5 rounded-full">{{ teacher.departmentName }}</span>
        </div>
        <div class="flex shrink-0 flex-col items-center gap-1 max-sm:w-full">
          <RatingCircle
            :value="teacher.avgRating || 0"
            :size="72"
            :stroke-width="5"
          >
            <template v-if="teacher.avgRating !== null">
              <EmojiRating :value="teacher.avgRating" size="md" />
            </template>
            <span v-else class="text-base font-semibold text-text-muted">
              -
            </span>
          </RatingCircle>
          <span class="text-sm font-semibold tabular-nums text-text-primary">
            {{ teacher.avgRating !== null ? `${formatRating(teacher.avgRating)} / 5` : t('teaching.profile.noRating') }}
          </span>
          <span class="text-xs text-text-muted">{{ t('teaching.profile.overallRating') }}</span>
        </div>
      </header>

      <!-- Stats Cards -->
      <section class="flex items-center justify-center gap-8 mb-6 py-5 border-b border-border-light max-sm:gap-4">
        <div class="flex flex-col items-center text-center">
          <span class="w-9 h-9 flex items-center justify-center mb-2 text-primary">
            <BookOpen :size="20" />
          </span>
          <span class="font-display text-xl font-bold text-text-primary tabular-nums">{{ teacher.courseCount }}</span>
          <span class="text-xs text-text-muted mt-1">{{ t('teaching.profile.courseCount') }}</span>
        </div>
        <div class="flex flex-col items-center text-center">
          <span class="w-9 h-9 flex items-center justify-center mb-2 text-accent">
            <MessageSquare :size="20" />
          </span>
          <span class="font-display text-xl font-bold text-text-primary tabular-nums">{{ teacher.reviewCount }}</span>
          <span class="text-xs text-text-muted mt-1">{{ t('teaching.profile.reviewCount') }}</span>
        </div>
        <div class="flex flex-col items-center text-center">
          <span class="w-9 h-9 flex items-center justify-center mb-2 text-primary">
            <TrendingUp :size="20" />
          </span>
          <span class="font-display text-xl font-bold text-text-primary tabular-nums">{{ trendText }}</span>
          <span class="text-xs text-text-muted mt-1">{{ t('teaching.profile.ratingTrend') }}</span>
        </div>
      </section>

      <!-- Rating Trend Chart -->
      <section class="bg-bg-card rounded-lg p-5 mb-6">
        <h2 class="text-base font-semibold text-text-primary m-0 mb-4">{{ t('teaching.profile.ratingTrend') }}</h2>
        <div
          v-if="teacher.ratingTrend.length > 0"
          class="flex h-[250px] items-end gap-2 rounded-lg border border-border-light bg-bg-elevated/40 p-4"
          role="img"
          :aria-label="t('teaching.profile.ratingTrendChartAria')"
        >
          <div
            v-for="point in teacher.ratingTrend"
            :key="point.termID || point.termName"
            class="flex min-w-0 flex-1 flex-col items-center gap-2"
          >
            <div class="flex h-[170px] w-full max-w-20 items-end rounded-lg bg-bg-card px-2 py-1">
              <div
                class="w-full rounded-md bg-gradient-to-t from-primary/70 to-accent transition-all duration-base"
                :style="{ height: `${ratingTrendHeight(point.avgRating)}%` }"
                aria-hidden="true"
              />
            </div>
            <span class="text-xs font-semibold tabular-nums text-text-primary">
              {{ formatRating(point.avgRating) }}
            </span>
            <span class="max-w-full truncate text-center text-[11px] text-text-muted">
              {{ point.termName }}
            </span>
          </div>
        </div>
        <div v-else class="rounded-lg border border-border-light bg-bg-elevated/40 p-8 text-center text-sm text-text-muted">
          {{ t('common.empty.data') }}
        </div>
      </section>

      <!-- Courses List -->
      <section class="bg-bg-card rounded-lg p-5">
        <h2 class="text-base font-semibold text-text-primary m-0 mb-4">{{ t('teaching.profile.courseList') }}</h2>
        <div class="flex flex-col gap-2">
          <router-link
            v-for="course in teacher.courses"
            :key="course.id"
            :to="`/courses/${course.id}`"
            class="flex items-center justify-between py-3 px-4 border-b border-border-light no-underline transition-all duration-fast last:border-b-0 hover:bg-bg-hover hover:pl-6"
          >
            <div class="flex flex-col gap-1">
              <span class="font-medium text-text-primary">{{ course.name }}</span>
              <span class="text-xs text-text-muted">{{ t('teaching.profile.reviewsCount', { count: course.reviewCount }) }}</span>
            </div>
            <div class="flex items-center gap-1">
              <EmojiRating v-if="course.avgRating" :value="course.avgRating" size="sm" />
              <span v-else class="text-text-muted">-</span>
            </div>
          </router-link>
        </div>
      </section>

      <!-- Teacher Reviews -->
      <section class="mt-6">
        <div class="mb-4 flex items-center justify-between gap-3">
          <h2 class="text-base font-semibold text-text-primary m-0">{{ t('teaching.profile.latestReviews') }}</h2>
          <span
            v-if="teacherReviewsTotal > 0"
            class="shrink-0 rounded-full bg-primary/10 px-2.5 py-1 text-xs font-semibold text-primary"
          >
            {{ t('teaching.profile.reviewsCount', { count: teacherReviewsTotal }) }}
          </span>
        </div>

        <div
          v-if="teacherReviewsLoading && teacherReviews.length === 0"
          class="flex flex-col gap-4"
          role="status"
          :aria-label="t('teaching.profile.reviewsLoading')"
        >
          <div
            v-for="i in 2"
            :key="i"
            class="h-40 rounded-xl bg-[length:200%_100%] animate-shimmer"
            :style="{ backgroundImage: 'linear-gradient(90deg, var(--color-bg-secondary) 25%, var(--color-bg-tertiary) 50%, var(--color-bg-secondary) 75%)' }"
          />
        </div>

        <div
          v-else-if="teacherReviewsError"
          role="alert"
          class="rounded-lg border border-warning/20 bg-warning/5 p-6 text-center text-sm text-text-muted"
        >
          <p class="m-0 mb-3">{{ teacherReviewsError }}</p>
          <button
            type="button"
            class="rounded-lg bg-primary px-4 py-2 text-white hover:bg-primary/90 cursor-pointer"
            @click="fetchTeacherReviews(true)"
          >
            {{ t('common.actions.retry') }}
          </button>
        </div>

        <div
          v-else-if="teacherReviews.length === 0"
          class="rounded-lg border border-border-light bg-bg-elevated/40 p-8 text-center text-sm text-text-muted"
        >
          {{ t('teaching.profile.reviewsEmpty') }}
        </div>

        <div v-else class="flex flex-col gap-4">
          <ReviewCard
            v-for="(review, index) in teacherReviews"
            :key="review.id"
            :review="review"
            class="stagger-item"
            :style="{ animationDelay: `${Math.min(index, 8) * 60}ms` }"
            @moderated="() => fetchTeacherReviews(true)"
          />
          <div v-if="teacherReviewsHasMore" class="flex justify-center pt-1">
            <button
              type="button"
              class="rounded-lg border border-border-light px-4 py-2 text-sm font-semibold text-text-secondary transition-colors hover:border-primary hover:text-primary disabled:cursor-not-allowed disabled:opacity-60"
              :disabled="teacherReviewsLoading"
              @click="fetchTeacherReviews(false)"
            >
              {{ teacherReviewsLoading ? t('common.actions.loading') : t('common.actions.loadMore') }}
            </button>
          </div>
        </div>
      </section>
    </template>

    <!-- Error State -->
    <div v-else-if="loadError" role="alert" class="flex flex-col items-center justify-center p-12 text-center text-text-muted">
      <p>{{ loadError }}</p>
      <button
        v-if="canRetryTeacherLoad"
        type="button"
        class="mt-4 rounded-lg bg-primary px-4 py-2 text-white hover:bg-primary/90 cursor-pointer"
        data-teacher-profile-retry-button
        @click="() => fetchTeacher()"
      >
        {{ t('common.actions.retry') }}
      </button>
    </div>
    <div v-else class="flex flex-col items-center justify-center p-12 text-center text-text-muted">
      <p>{{ t('teaching.profile.notFound') }}</p>
      <router-link to="/" class="mt-4 text-primary no-underline hover:underline">{{ t('teaching.profile.backToHome') }}</router-link>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { api } from '@/api'
import { getErrorStatus, isApiError } from '@/api/errors'
import { updatePageMeta } from '@/composables/usePageMeta'
import RatingCircle from '@/components/common/RatingCircle.vue'
import EmojiRating from '@/components/business/review/EmojiRating.vue'
import ReviewCard from '@/components/business/review/ReviewCard.vue'
import { User, BookOpen, MessageSquare, TrendingUp } from 'lucide-vue-next'
import type { Review } from '@stuhelper/shared/review'

interface TeacherCourse {
  id: number
  name: string
  avgRating: number | null
  reviewCount: number
}

interface RatingTrendItem {
  termID: string
  termName: string
  avgRating: number
}

interface TeacherDetail {
  teacherID: number
  teacherName: string
  departmentName: string
  avgRating: number | null
  courseCount: number
  reviewCount: number
  courses: TeacherCourse[]
  ratingTrend: RatingTrendItem[]
}

const route = useRoute()
const router = useRouter()
const { t } = useI18n()
const teacherID = computed(() => Number(route.params.id))

const loading = ref(true)
const teacher = ref<TeacherDetail | null>(null)
const loadError = ref('')
const canRetryTeacherLoad = ref(false)
const teacherReviews = ref<Review[]>([])
const teacherReviewsLoading = ref(false)
const teacherReviewsError = ref('')
const teacherReviewsTotal = ref(0)
const teacherReviewsPage = ref(1)
const teacherReviewsPageSize = 6
let teacherRequestSeq = 0
let teacherReviewsRequestSeq = 0

const trendText = computed(() => {
  if (!teacher.value?.ratingTrend?.length) return '-'
  const trend = teacher.value.ratingTrend
  if (trend.length < 2) return t('teaching.profile.trend.stable')
  const diff = trend[trend.length - 1].avgRating - trend[0].avgRating
  if (diff > 0.2) return t('teaching.profile.trend.up')
  if (diff < -0.2) return t('teaching.profile.trend.down')
  return t('teaching.profile.trend.stable')
})
const teacherReviewsHasMore = computed(() => teacherReviews.value.length < teacherReviewsTotal.value)

function isNonNegativeNumber(value: unknown): value is number {
  return typeof value === 'number' && Number.isFinite(value) && value >= 0
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return Boolean(value) && typeof value === 'object' && !Array.isArray(value)
}

function readPositiveInteger(
  record: Record<string, unknown>,
  key: string,
): number {
  const value = record[key]
  if (typeof value !== 'number' || !Number.isInteger(value) || value <= 0) {
    throw new Error('Invalid teacher stats response')
  }
  return value
}

function readNonNegativeInteger(
  record: Record<string, unknown>,
  key: string,
): number {
  const value = record[key]
  if (typeof value !== 'number' || !Number.isInteger(value) || value < 0) {
    throw new Error('Invalid teacher stats response')
  }
  return value
}

function readString(record: Record<string, unknown>, key: string): string {
  const value = record[key]
  if (typeof value !== 'string') {
    throw new Error('Invalid teacher stats response')
  }
  return value
}

function readRating(record: Record<string, unknown>, key: string): number {
  const value = record[key]
  if (!isNonNegativeNumber(value) || value > 5) {
    throw new Error('Invalid teacher stats response')
  }
  return value
}

function readOptionalRating(record: Record<string, unknown>, key: string): number | null {
  const value = record[key]
  if (value === undefined || value === null) {
    return null
  }
  if (!isNonNegativeNumber(value) || value > 5) {
    throw new Error('Invalid teacher stats response')
  }
  return value
}

function readTeacherCourse(payload: unknown): TeacherCourse {
  if (!isRecord(payload)) {
    throw new Error('Invalid teacher stats response')
  }

  return {
    id: readPositiveInteger(payload, 'id'),
    name: readString(payload, 'name'),
    avgRating: readOptionalRating(payload, 'avgRating'),
    reviewCount: readNonNegativeInteger(payload, 'reviewCount'),
  }
}

function readRatingTrendItem(payload: unknown): RatingTrendItem {
  if (!isRecord(payload)) {
    throw new Error('Invalid teacher stats response')
  }

  return {
    termID: readString(payload, 'termID'),
    termName: readString(payload, 'termName'),
    avgRating: readRating(payload, 'avgRating'),
  }
}

function readTeacherPayload(payload: unknown): TeacherDetail {
  if (!isRecord(payload)) {
    throw new Error('Invalid teacher stats response')
  }

  if (!Array.isArray(payload.courses) || !Array.isArray(payload.ratingTrend)) {
    throw new Error('Invalid teacher stats response')
  }

  return {
    teacherID: readPositiveInteger(payload, 'teacherID'),
    teacherName: readString(payload, 'teacherName'),
    departmentName: readString(payload, 'departmentName'),
    avgRating: readOptionalRating(payload, 'avgRating'),
    courseCount: readNonNegativeInteger(payload, 'courseCount'),
    reviewCount: readNonNegativeInteger(payload, 'reviewCount'),
    courses: payload.courses.map(readTeacherCourse),
    ratingTrend: payload.ratingTrend.map(readRatingTrendItem),
  }
}

function updateTeacherPageMeta(detail: TeacherDetail) {
  updatePageMeta({
    title: detail.teacherName,
    description: [
      detail.teacherName,
      detail.departmentName,
      t('teaching.profile.reviewsCount', { count: detail.reviewCount }),
    ].join(' · '),
  })
}

function isValidTeacherID(value: number) {
  return Number.isInteger(value) && value > 0
}

function isTeacherNotFoundError(error: unknown) {
  return getErrorStatus(error) === 404 || (isApiError(error) && error.code === 'A0100003')
}

const fetchTeacher = async (id = teacherID.value) => {
  const requestSeq = ++teacherRequestSeq
  loading.value = true
  loadError.value = ''
  canRetryTeacherLoad.value = false
  teacherReviews.value = []
  teacherReviewsError.value = ''
  teacherReviewsTotal.value = 0
  teacherReviewsPage.value = 1
  try {
    const res = await api.rating.getTeacherStats(id)
    const nextTeacher = readTeacherPayload(res.data?.data)
    if (requestSeq !== teacherRequestSeq || teacherID.value !== id) return
    teacher.value = nextTeacher
    updateTeacherPageMeta(nextTeacher)
    await fetchTeacherReviews(true, id)
  } catch (error) {
    if (requestSeq !== teacherRequestSeq || teacherID.value !== id) return
    teacher.value = null
    if (isTeacherNotFoundError(error)) {
      loadError.value = t('teaching.profile.notFound')
      canRetryTeacherLoad.value = false
      updatePageMeta({
        title: t('teaching.profile.notFound'),
        description: t('teaching.profile.notFound'),
      })
    } else {
      loadError.value = t('common.loadFailed')
      canRetryTeacherLoad.value = true
    }
  } finally {
    if (requestSeq === teacherRequestSeq && teacherID.value === id) {
      loading.value = false
    }
  }
}

async function fetchTeacherReviews(reset: boolean, id = teacherID.value) {
  if (!isValidTeacherID(id)) return
  if (!reset && teacherReviewsLoading.value) return
  const requestSeq = ++teacherReviewsRequestSeq
  if (reset) {
    teacherReviews.value = []
    teacherReviewsTotal.value = 0
    teacherReviewsPage.value = 1
  }

  teacherReviewsLoading.value = true
  teacherReviewsError.value = ''
  try {
    const page = teacherReviewsPage.value
    const pageData = await api.review.getLatestReviewsPage({
      page,
      pageSize: teacherReviewsPageSize,
      sort: 'time',
      teacherID: id,
    })
    if (requestSeq !== teacherReviewsRequestSeq || teacherID.value !== id) return
    if (reset) {
      teacherReviews.value = pageData.list
    } else {
      const existingIDs = new Set(teacherReviews.value.map(review => review.id))
      teacherReviews.value.push(...pageData.list.filter(review => !existingIDs.has(review.id)))
    }
    teacherReviewsTotal.value = pageData.total
    teacherReviewsPage.value = page + 1
  } catch (_error) { void _error;
    if (requestSeq !== teacherReviewsRequestSeq || teacherID.value !== id) return
    teacherReviewsError.value = t('teaching.profile.reviewsLoadFailed')
  } finally {
    if (requestSeq === teacherReviewsRequestSeq && teacherID.value === id) {
      teacherReviewsLoading.value = false
    }
  }
}

function formatRating(value: number): string {
  return value.toFixed(1)
}

function ratingTrendHeight(value: number): number {
  return Math.max(8, Math.min(100, (value / 5) * 100))
}

watch(teacherID, async (newID, oldID) => {
  if (newID === oldID) return
  if (!isValidTeacherID(newID)) {
    teacherRequestSeq += 1
    teacherReviewsRequestSeq += 1
    await router.replace({ name: 'course-hub' })
    return
  }
  await fetchTeacher(newID)
}, { immediate: true })
</script>
