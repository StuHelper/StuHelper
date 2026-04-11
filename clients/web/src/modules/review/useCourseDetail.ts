import { ref, computed, watch, onUnmounted, nextTick } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'

import { api } from '@/api'
import { getErrorMessage } from '@/api/errors'
import { useUserStore } from '@/stores/user'
import { useToast } from '@/composables/useToast'
import { useReviewPost } from '@/composables/useReviewPost'

import type { Course, CourseRatingStatsResponse, TeacherStats } from '@/types/course'
import type { Review } from '@/types/review'

const PAGE_SIZE = 20

export function useCourseDetail() {
  const { t } = useI18n()
  const route = useRoute()
  const router = useRouter()
  const toast = useToast()
  const userStore = useUserStore()
  const { lastPostedAt } = useReviewPost()

  const courseID = computed(() => Number(route.params.id))

  const isPanelMode = computed(() => {
    return route.matched.some(r => r.name === 'review')
  })

  // ── Page state ──
  const loading = ref(false)
  const error = ref(false)
  const contentReady = ref(false)
  const course = ref<Course | null>(null)
  const ratingStats = ref<CourseRatingStatsResponse | null>(null)
  const courseTeachers = ref<TeacherStats[]>([])
  const ratingTrend = ref<{ termName: string; avgRating: number }[]>([])

  // ── Reviews ──
  const reviews = ref<Review[]>([])
  const reviewsLoading = ref(false)
  const page = ref(1)
  const total = ref(0)
  const hasMore = computed(() => reviews.value.length < total.value)

  // ── Teacher filter ──
  const selectedTeacher = ref('')
  const uniqueTeachers = computed(() => {
    return courseTeachers.value
      .map(t => t.teacherName)
      .filter(Boolean)
      .sort()
  })
  const teacherChips = computed(() => ['', ...uniqueTeachers.value])

  const filteredReviews = computed(() => {
    if (!selectedTeacher.value) return reviews.value
    return reviews.value.filter(r => r.teacherName === selectedTeacher.value)
  })

  function selectTeacher(name: string) {
    selectedTeacher.value = name
  }

  // ── Data fetching ──
  let loadVersion = 0

  const fetchReviews = async (append = false, expectedVersion?: number) => {
    reviewsLoading.value = true
    try {
      const pageData = await api.review.getReviewsPage(courseID.value, {
        page: page.value,
        pageSize: PAGE_SIZE,
        sort: 'time',
      })
      if (expectedVersion !== undefined && expectedVersion !== loadVersion) return
      reviews.value = append ? [...reviews.value, ...pageData.list] : pageData.list
      total.value = pageData.total
    } catch (fetchErr) {
      if (expectedVersion === undefined || expectedVersion === loadVersion) {
        toast.error(getErrorMessage(fetchErr, t('review.course.loadFailed')))
      }
    } finally {
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

  const fetchRatingStats = async () => {
    try {
      const res = await api.rating.getCourseStats(courseID.value)
      ratingStats.value = res.data?.data ?? null
    } catch (fetchErr) {
      toast.error(getErrorMessage(fetchErr, t('common.loadFailed')))
    }
  }

  function refreshReviews() {
    page.value = 1
    const version = ++loadVersion
    fetchReviews(false, version)
    fetchRatingStats()
  }

  const fetchAll = async () => {
    const id = courseID.value
    if (isNaN(id) || id <= 0) {
      router.replace({ name: 'course-hub' })
      return
    }

    const version = ++loadVersion
    course.value = null
    ratingStats.value = null
    courseTeachers.value = []
    reviews.value = []
    total.value = 0
    page.value = 1
    selectedTeacher.value = ''
    contentReady.value = false
    loading.value = true

    try {
      const [courseRes, statsRes, reviewsRes, teachersRes, trendRes] = await Promise.all([
        api.course.getCourse(id).catch(() => null),
        api.rating.getCourseStats(id).catch(() => null),
        api.review.getReviewsPage(id, { page: 1, pageSize: PAGE_SIZE, sort: 'time' }).catch(() => null),
        api.rating.getCourseTeachers(id).catch(() => null),
        api.rating.getRatingTrend(id).catch(() => null),
      ])

      if (version !== loadVersion) return

      course.value = courseRes?.data?.data ?? null
      if (course.value) {
        userStore.setFavoriteStatus(id, (course.value as { isFavorited?: boolean }).isFavorited === true)
      }
      error.value = !courseRes
      ratingStats.value = statsRes?.data?.data ?? null
      reviews.value = reviewsRes?.list ?? []
      total.value = reviewsRes?.total ?? 0
      courseTeachers.value = teachersRes?.data?.data || []
      ratingTrend.value = trendRes?.data?.data?.trend || []
    } finally {
      if (version === loadVersion) {
        loading.value = false
        await nextTick()
        contentReady.value = true
      }
    }
  }

  // ── Navigation ──
  function goToPostPage() {
    router.push({ name: 'course-review-post', params: { id: courseID.value } })
  }

  // Watch for post events
  let lastPostedAtSnapshot = lastPostedAt.value
  watch(lastPostedAt, (val) => {
    if (val > lastPostedAtSnapshot) {
      lastPostedAtSnapshot = val
      refreshReviews()
    }
  })

  onUnmounted(() => {
    ++loadVersion
  })

  // Main data load
  watch(courseID, async (newID, oldID) => {
    if (oldID !== undefined && (newID === oldID || isNaN(newID) || newID <= 0)) return
    await fetchAll()
  }, { immediate: true })

  return {
    courseID,
    isPanelMode,
    loading,
    error,
    contentReady,
    course,
    ratingStats,
    ratingTrend,
    reviews,
    reviewsLoading,
    total,
    hasMore,
    selectedTeacher,
    uniqueTeachers,
    teacherChips,
    filteredReviews,
    selectTeacher,
    loadMoreReviews,
    refreshReviews,
    fetchAll,
    goToPostPage,
  }
}
