<template>
  <div class="min-h-screen bg-bg-base">
    <!-- Search Form View -->
    <div v-if="!showResults" class="max-w-2xl mx-auto px-6 py-8 animate-fade-in">
      <!-- Header -->
      <div class="flex items-center gap-3 mb-2">
        <button
          class="flex items-center gap-1.5 px-3 py-2 text-sm rounded-lg text-text-secondary hover:text-primary hover:bg-bg-elevated transition-colors"
          @click="goHome"
        >
          <ArrowLeft :size="16" />
          {{ t('common.actions.back') }}
        </button>
        <h1 class="text-2xl font-bold m-0 text-text-primary">
          {{ t('review.search.title') }}
        </h1>
      </div>
      <p class="text-sm mb-8 text-text-muted">
        {{ t('review.search.subtitle') }}
      </p>

      <div
        v-if="referenceError"
        role="alert"
        class="mb-6 rounded-xl border border-warning/30 bg-warning/10 px-4 py-3 text-sm text-warning flex items-center justify-between gap-3"
      >
        <span>{{ referenceError }}</span>
        <button
          type="button"
          class="shrink-0 rounded-full bg-warning/15 px-3 py-1 text-xs font-semibold text-warning transition-colors hover:bg-warning/25"
          @click="retryReferenceData"
        >
          {{ t('common.actions.retry') }}
        </button>
      </div>

      <!-- Course Conditions Section -->
      <div class="bg-bg-card rounded-2xl shadow-md p-6 mb-6">
        <h2 class="text-lg font-semibold text-text-primary mb-4 flex items-center gap-2">
          <span class="w-1 h-5 bg-primary rounded-full" />
          {{ t('review.search.courseConditions') }}
        </h2>

        <!-- Course Name -->
        <div class="mb-4">
          <label
            for="advanced-course-name"
            class="block text-sm font-medium mb-1.5 text-text-secondary"
          >
            {{ t('review.search.courseName') }}
          </label>
          <input
            id="advanced-course-name"
            v-model="form.courseName"
            type="text"
            autocomplete="off"
            data-1p-ignore
            data-lpignore="true"
            data-form-type="other"
            class="w-full px-4 py-3 bg-bg-elevated rounded-lg text-text-primary placeholder-text-muted focus:border-primary focus:ring-2 focus:ring-primary/20 outline-none transition-all disabled:opacity-50"
            :class="{ 'border-danger focus:border-danger focus:ring-danger/20': validationError }"
            :placeholder="t('review.search.courseName')"
            :disabled="!!form.courseCode"
          />
          <span class="block text-xs mt-1 text-text-muted">
            {{ t('review.search.courseNameHelper') }}
          </span>
        </div>

        <!-- Course Code -->
        <div class="mb-4">
          <label
            for="advanced-course-code"
            class="block text-sm font-medium mb-1.5 text-text-secondary"
          >
            {{ t('review.search.courseCode') }}
          </label>
          <input
            id="advanced-course-code"
            v-model="form.courseCode"
            type="text"
            autocomplete="off"
            data-1p-ignore
            data-lpignore="true"
            data-form-type="other"
            class="w-full px-4 py-3 bg-bg-elevated rounded-lg text-text-primary placeholder-text-muted focus:border-primary focus:ring-2 focus:ring-primary/20 outline-none transition-all disabled:opacity-50"
            :placeholder="t('review.search.courseCode')"
            :disabled="!!form.courseName"
          />
          <span class="block text-xs mt-1 text-text-muted">
            {{ t('review.search.courseCodeHelper') }}
          </span>
        </div>

        <!-- Department -->
        <div>
          <label
            for="advanced-department"
            class="block text-sm font-medium mb-1.5 text-text-secondary"
          >
            {{ t('review.search.department') }}
          </label>
          <select
            id="advanced-department"
            v-model="form.departmentID"
            class="w-full px-4 py-3 bg-bg-elevated rounded-lg text-text-primary focus:border-primary focus:ring-2 focus:ring-primary/20 outline-none transition-all"
          >
            <option :value="0">{{ t('review.search.allDepartments') }}</option>
            <option v-for="dept in departments" :key="dept.id" :value="dept.id">
              {{ dept.name }}
            </option>
          </select>
        </div>
      </div>

      <!-- Review Conditions Section -->
      <div class="bg-bg-card rounded-2xl shadow-md p-6 mb-6">
        <h2 class="text-lg font-semibold text-text-primary mb-4 flex items-center gap-2">
          <span class="w-1 h-5 bg-secondary rounded-full" />
          {{ t('review.search.reviewConditions') }}
        </h2>

        <!-- Teacher Name -->
        <div class="mb-4">
          <label
            for="advanced-teacher-name"
            class="block text-sm font-medium mb-1.5 text-text-secondary"
          >
            {{ t('review.search.teacherName') }}
          </label>
          <input
            id="advanced-teacher-name"
            v-model="form.teacherName"
            type="text"
            autocomplete="off"
            data-1p-ignore
            data-lpignore="true"
            data-form-type="other"
            class="w-full px-4 py-3 bg-bg-elevated rounded-lg text-text-primary placeholder-text-muted focus:border-primary focus:ring-2 focus:ring-primary/20 outline-none transition-all"
            :placeholder="t('review.search.teacherName')"
          />
          <span class="block text-xs mt-1 text-text-muted">
            {{ t('review.search.teacherNameHelper') }}
          </span>
        </div>

        <!-- Semester -->
        <div>
          <label
            for="advanced-term"
            class="block text-sm font-medium mb-1.5 text-text-secondary"
          >
            {{ t('review.search.semester') }}
          </label>
          <select
            id="advanced-term"
            v-model="form.termID"
            class="w-full px-4 py-3 bg-bg-elevated rounded-lg text-text-primary focus:border-primary focus:ring-2 focus:ring-primary/20 outline-none transition-all"
          >
            <option value="">{{ t('review.search.anySemester') }}</option>
            <option v-for="term in terms" :key="term.id" :value="term.id">
              {{ term.name }}
            </option>
          </select>
        </div>
      </div>

      <!-- Validation Error -->
      <div
        v-if="validationError"
        class="bg-danger/10 text-danger rounded-lg p-3 text-sm font-medium mb-4"
      >
        {{ validationError }}
      </div>

      <!-- Search Button -->
      <button
        class="w-full bg-primary hover:bg-primary-dark text-white rounded-xl py-3 font-semibold text-base flex items-center justify-center gap-2 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
        :disabled="searching"
        @click="handleSearch()"
      >
        <Search :size="18" />
        {{ searching ? t('common.actions.loading') : t('review.search.searchBtn') }}
      </button>
    </div>

    <!-- Results View -->
    <div v-else class="max-w-[900px] mx-auto px-6 py-8 animate-fade-in">
      <!-- Results Header -->
      <div class="flex items-center gap-3 mb-6">
        <button
          class="flex items-center gap-1.5 px-3 py-2 text-sm rounded-lg text-text-secondary hover:text-primary hover:bg-bg-elevated transition-colors"
          @click="backToForm"
        >
          <ArrowLeft :size="16" />
          {{ t('common.actions.back') }}
        </button>
        <h1 class="text-2xl font-bold m-0 text-text-primary">
          {{ t('review.search.results') }}
        </h1>
      </div>

      <!-- Loading State -->
      <div v-if="searching" class="flex flex-col items-center justify-center py-16 gap-4">
        <div
          class="w-8 h-8 rounded-full border-3 animate-spin border-border border-t-primary"
        />
        <span class="text-text-muted">{{ t('common.actions.loading') }}</span>
      </div>

      <div
        v-else
      >
        <div
          v-if="searchError"
          role="alert"
          class="mb-4 rounded-xl border border-warning/30 bg-warning/10 px-4 py-3 text-sm text-warning"
        >
          {{ searchError }}
        </div>

        <!-- No Results -->
        <div
          v-if="resultCourses.length === 0 && resultReviews.length === 0 && !searchError"
          class="text-center py-16 text-text-muted"
        >
          <SearchX :size="48" class="mx-auto mb-4 opacity-50" />
          <p class="text-base">{{ t('review.search.noResults') }}</p>
        </div>

        <!-- Results Content -->
        <template v-else-if="resultCourses.length > 0 || resultReviews.length > 0">
          <!-- Courses With Reviews -->
          <section v-if="coursesWithReviews.length > 0" class="mb-8">
            <h2 class="text-lg font-semibold text-text-primary mb-4 flex items-center gap-2">
              <span class="w-1 h-5 bg-primary rounded-full" />
              {{ t('review.search.coursesWithReviews') }}
              <span class="text-xs font-bold bg-primary/15 text-primary px-2 py-0.5 rounded-full">
                {{ coursesWithReviews.length }}
              </span>
            </h2>
            <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
              <router-link
                v-for="(course, idx) in coursesWithReviews"
                :key="course.id"
                :to="`/courses/${course.id}/reviews`"
                class="bg-bg-card rounded-xl shadow-card p-4 no-underline flex items-center justify-between text-text-primary border border-transparent hover:border-primary/50 hover:shadow-md transition-all stagger-item"
                :style="{ animationDelay: `${Math.min(idx, 8) * 60}ms` }"
              >
                <div class="min-w-0 flex-1">
                  <div class="font-semibold text-sm truncate">{{ course.name }}</div>
                  <div class="text-xs mt-1 flex items-center gap-2 text-text-muted">
                    <span v-if="course.code" class="font-mono">{{ course.code }}</span>
                    <span v-if="course.departmentName">{{ course.departmentName }}</span>
                  </div>
                </div>
                <div class="flex items-center gap-2 shrink-0 ml-3">
                  <span class="text-xs font-bold border border-secondary text-secondary px-2 py-0.5 rounded-full">
                    {{ course.reviewCount }} {{ t('review.course.reviewUnit') }}
                  </span>
                </div>
              </router-link>
            </div>
          </section>

          <!-- Courses Without Reviews -->
          <section v-if="coursesWithoutReviews.length > 0" class="mb-8">
            <h2 class="text-lg font-semibold text-text-primary mb-4 flex items-center gap-2">
              <span class="w-1 h-5 bg-border rounded-full" />
              {{ t('review.search.coursesWithoutReviews') }}
              <span class="text-xs font-bold bg-bg-elevated text-text-secondary px-2 py-0.5 rounded-full">
                {{ coursesWithoutReviews.length }}
              </span>
            </h2>
            <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
              <router-link
                v-for="course in coursesWithoutReviews"
                :key="course.id"
                :to="`/courses/${course.id}`"
                class="bg-bg-card rounded-xl p-4 no-underline flex items-center justify-between text-text-primary border border-transparent hover:border-primary/50 transition-all"
              >
                <div class="min-w-0 flex-1">
                  <div class="font-semibold text-sm truncate">{{ course.name }}</div>
                  <div class="text-xs mt-1 flex items-center gap-2 text-text-muted">
                    <span v-if="course.code" class="font-mono">{{ course.code }}</span>
                    <span v-if="course.departmentName">{{ course.departmentName }}</span>
                  </div>
                </div>
              </router-link>
            </div>
          </section>

          <div
            v-if="resultCourses.length > 0 && (hasMoreCourses || coursesLoadMoreError)"
            class="mb-8 flex flex-col items-center gap-3"
          >
            <p
              v-if="coursesLoadMoreError"
              role="alert"
              class="rounded-lg border border-danger/30 bg-danger/10 px-3 py-2 text-sm text-danger"
            >
              {{ coursesLoadMoreError }}
            </p>
            <button
              v-if="hasMoreCourses"
              type="button"
              class="inline-flex h-11 items-center justify-center gap-2 rounded-lg border border-border-light bg-bg-card px-5 text-sm font-semibold text-text-secondary transition-colors hover:bg-bg-elevated hover:text-text-primary disabled:cursor-wait disabled:opacity-70"
              :disabled="coursesLoadingMore"
              @click="loadMoreCourses"
            >
              <RefreshCw v-if="coursesLoadingMore" :size="16" class="animate-spin" />
              <span>{{ coursesLoadingMore ? t('common.actions.loading') : t('common.actions.loadMore') }}</span>
            </button>
          </div>

          <!-- Review Results -->
          <section v-if="resultReviews.length > 0">
            <h2 class="text-lg font-semibold text-text-primary mb-4 flex items-center gap-2">
              <span class="w-1 h-5 bg-secondary rounded-full" />
              {{ t('review.search.reviewResults') }}
              <span class="text-xs font-bold bg-primary/15 text-primary px-2 py-0.5 rounded-full">
                {{ resultReviews.length }}
              </span>
            </h2>
            <div class="flex flex-col gap-4">
              <ReviewCard
                v-for="(review, idx) in resultReviews"
                :key="review.id"
                :review="review"
                class="stagger-item"
                :style="{ animationDelay: `${Math.min(idx, 8) * 60}ms` }"
              />
            </div>
            <div
              v-if="hasMoreReviews || reviewsLoadMoreError"
              class="mt-5 flex flex-col items-center gap-3"
            >
              <p
                v-if="reviewsLoadMoreError"
                role="alert"
                class="rounded-lg border border-danger/30 bg-danger/10 px-3 py-2 text-sm text-danger"
              >
                {{ reviewsLoadMoreError }}
              </p>
              <button
                v-if="hasMoreReviews"
                type="button"
                class="inline-flex h-11 items-center justify-center gap-2 rounded-lg border border-border-light bg-bg-card px-5 text-sm font-semibold text-text-secondary transition-colors hover:bg-bg-elevated hover:text-text-primary disabled:cursor-wait disabled:opacity-70"
                :disabled="reviewsLoadingMore"
                @click="loadMoreReviews"
              >
                <RefreshCw v-if="reviewsLoadingMore" :size="16" class="animate-spin" />
                <span>{{ reviewsLoadingMore ? t('common.actions.loading') : t('common.actions.loadMore') }}</span>
              </button>
            </div>
          </section>
        </template>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, computed, onUnmounted, nextTick, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { ArrowLeft, RefreshCw, Search, SearchX } from 'lucide-vue-next'
import ReviewCard from '@/components/business/review/ReviewCard.vue'
import { api } from '@/api'
import { getErrorMessage } from '@/api/errors'
import { useToast } from '@/composables/useToast'
import { updatePageMeta } from '@/composables/usePageMeta'
import {
  readCoursePagePayload,
  readDepartmentArrayPayload,
  readTermArrayPayload,
} from '@/modules/course/coursePayload'
import type { Department, Term, Course } from '@stuhelper/shared/course'
import type { Review } from '@stuhelper/shared/review'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const toast = useToast()

// --- State ---

interface SearchForm {
  courseName: string
  courseCode: string
  departmentID: number
  teacherName: string
  termID: string
}

const form = reactive<SearchForm>({
  courseName: '',
  courseCode: '',
  departmentID: 0,
  teacherName: '',
  termID: '',
})

const departments = ref<Department[]>([])
const terms = ref<Term[]>([])
const searching = ref(false)
const showResults = ref(false)
const validationError = ref('')
const resultCourses = ref<Course[]>([])
const resultReviews = ref<Review[]>([])
const courseTotal = ref(0)
const reviewTotal = ref(0)
const coursePage = ref(1)
const reviewPage = ref(1)
const coursesLoadingMore = ref(false)
const reviewsLoadingMore = ref(false)
const searchError = ref('')
const coursesLoadMoreError = ref('')
const reviewsLoadMoreError = ref('')
const referenceError = ref('')
const SEARCH_PAGE_SIZE = 50

let abortController: AbortController | null = null

// --- Computed ---

const coursesWithReviews = computed(() =>
  resultCourses.value.filter(c => c.reviewCount > 0)
)

const coursesWithoutReviews = computed(() =>
  resultCourses.value.filter(c => c.reviewCount === 0)
)
const hasMoreCourses = computed(() => resultCourses.value.length < courseTotal.value)
const hasMoreReviews = computed(() => resultReviews.value.length < reviewTotal.value)

// --- Navigation ---

function goHome() {
  router.push({ name: 'home' })
}

function scrollSearchPageToTop() {
  window.scrollTo({ top: 0, left: 0, behavior: 'auto' })
}

function updateSearchFormPageMeta() {
  updatePageMeta({
    title: t('review.search.title'),
    description: t('review.search.subtitle'),
  })
}

async function backToForm() {
  showResults.value = false
  resultCourses.value = []
  resultReviews.value = []
  searchError.value = ''
  validationError.value = ''
  resetSearchPagination()
  updateSearchFormPageMeta()
  await nextTick()
  scrollSearchPageToTop()
}

// --- Data loading ---

async function loadDepartments() {
  try {
    const res = await api.course.getDepartments()
    departments.value = readDepartmentArrayPayload(
      res.data?.data,
      'Invalid departments response',
    )
  } catch (_error) { void _error;
    departments.value = []
    referenceError.value = t('common.loadFailed')
  }
}

async function loadTerms() {
  try {
    const res = await api.course.getTerms()
    terms.value = readTermArrayPayload(
      res.data?.data,
      'Invalid terms response',
    )
  } catch (_error) { void _error;
    terms.value = []
    referenceError.value = t('common.loadFailed')
  }
}

function retryReferenceData() {
  referenceError.value = ''
  void loadDepartments()
  void loadTerms()
}

// --- Search ---

function validateForm(): boolean {
  validationError.value = ''
  const hasCondition =
    form.courseName.trim() !== '' ||
    form.courseCode.trim() !== '' ||
    form.departmentID > 0 ||
    form.teacherName.trim() !== '' ||
    form.termID !== ''

  if (!hasCondition) {
    validationError.value = t('review.search.atLeastOne')
    return false
  }
  return true
}

function readQueryString(...keys: string[]): string {
  for (const key of keys) {
    const value = route.query[key]
    const text = Array.isArray(value) ? value[0] : value
    if (typeof text === 'string' && text.trim()) {
      return text.trim()
    }
  }
  return ''
}

function readQueryPositiveInteger(key: string): number {
  const value = readQueryString(key)
  if (!value) return 0
  const parsed = Number(value)
  return Number.isInteger(parsed) && parsed > 0 ? parsed : 0
}

function hydrateSearchFormFromRoute(): boolean {
  const courseName = readQueryString('courseName', 'q')
  form.courseName = courseName
  form.courseCode = courseName ? '' : readQueryString('courseCode', 'code')
  form.departmentID = readQueryPositiveInteger('departmentID')
  form.teacherName = readQueryString('teacherName')
  form.termID = readQueryString('termID')

  return (
    form.courseName !== '' ||
    form.courseCode !== '' ||
    form.departmentID > 0 ||
    form.teacherName !== '' ||
    form.termID !== ''
  )
}

function buildSearchCriteriaSummary() {
  const criteria = [
    form.courseName.trim(),
    form.courseCode.trim(),
    departments.value.find(dept => dept.id === form.departmentID)?.name,
    form.teacherName.trim(),
    terms.value.find(term => term.id === form.termID)?.name,
  ].filter((part): part is string => Boolean(part))

  return criteria.slice(0, 3).join(' · ')
}

function updateSearchResultsPageMeta() {
  const summary = buildSearchCriteriaSummary()
  updatePageMeta({
    title: summary
      ? `${summary} · ${t('review.search.results')}`
      : t('review.search.results'),
    description: summary
      ? `${t('review.search.results')} · ${summary}`
      : t('review.search.results'),
  })
}

function resetSearchPagination() {
  courseTotal.value = 0
  reviewTotal.value = 0
  coursePage.value = 1
  reviewPage.value = 1
  coursesLoadMoreError.value = ''
  reviewsLoadMoreError.value = ''
}

async function fetchCourseResults(nextPage: number, signal?: AbortSignal) {
  const courseQuery = form.courseName.trim() || form.courseCode.trim()
  if (!courseQuery && form.departmentID <= 0) {
    return { list: [] as Course[], total: 0 }
  }

  const result = courseQuery
    ? form.departmentID > 0
      ? await api.course.getCourses({
          q: courseQuery,
          departmentID: form.departmentID,
          page: nextPage,
          pageSize: SEARCH_PAGE_SIZE,
        })
      : await api.course.searchCourses(
          courseQuery,
          { page: nextPage, pageSize: SEARCH_PAGE_SIZE },
          { signal },
        )
    : await api.course.getCourses({
        departmentID: form.departmentID,
        page: nextPage,
        pageSize: SEARCH_PAGE_SIZE,
      })

  return readCoursePagePayload(
    result.data?.data,
    'Invalid course search response',
  )
}

async function fetchReviewResults(nextPage: number, signal?: AbortSignal) {
  return api.review.searchReviewsPage(
    {
      q: form.courseName.trim() || form.courseCode.trim() || undefined,
      departmentID: form.departmentID > 0 ? form.departmentID : undefined,
      teacherName: form.teacherName.trim() || undefined,
      termID: form.termID || undefined,
      page: nextPage,
      pageSize: SEARCH_PAGE_SIZE,
      sort: 'time',
    },
    { signal },
  )
}

async function handleSearch(options: { restart?: boolean } = {}) {
  if (!validateForm()) return
  if (searching.value && !options.restart) return

  // 新搜索开始前取消上一次未完成请求
  if (abortController) {
    abortController.abort()
  }
  abortController = new AbortController()
  const signal = abortController.signal

  searching.value = true
  showResults.value = true
  resultCourses.value = []
  resultReviews.value = []
  resetSearchPagination()
  searchError.value = ''
  updateSearchResultsPageMeta()
  await nextTick()
  scrollSearchPageToTop()

  try {
    const [courseRes, reviewRes] = await Promise.allSettled([
      fetchCourseResults(1, signal),
      fetchReviewResults(1, signal),
    ])

    if (signal.aborted) return

    if (courseRes.status === 'fulfilled') {
      resultCourses.value = courseRes.value.list
      courseTotal.value = courseRes.value.total
      coursePage.value = 1
    } else if (courseRes.status === 'rejected') {
      searchError.value = getErrorMessage(courseRes.reason, t('common.loadFailed'))
    }

    if (reviewRes.status === 'fulfilled') {
      resultReviews.value = reviewRes.value.list
      reviewTotal.value = reviewRes.value.total
      reviewPage.value = 1
    } else {
      const message = getErrorMessage(reviewRes.reason, t('common.loadFailed'))
      searchError.value = searchError.value || message
    }

    if (searchError.value) {
      toast.error(searchError.value || t('common.loadFailed'))
    }
  } catch (error) {
    if (!signal.aborted) {
      resultCourses.value = []
      resultReviews.value = []
      searchError.value = getErrorMessage(error, t('common.loadFailed'))
      toast.error(searchError.value)
    }
  } finally {
    if (!signal.aborted) {
      searching.value = false
    }
  }
}

async function syncSearchFromRoute() {
  const shouldSearch = hydrateSearchFormFromRoute()
  if (shouldSearch) {
    await handleSearch({ restart: true })
    return
  }

  if (abortController) {
    abortController.abort()
    abortController = null
  }
  searching.value = false
  await backToForm()
}

async function loadMoreCourses() {
  if (!hasMoreCourses.value || coursesLoadingMore.value || searching.value) return
  const nextPage = coursePage.value + 1
  coursesLoadingMore.value = true
  coursesLoadMoreError.value = ''
  try {
    const data = await fetchCourseResults(nextPage)
    resultCourses.value = [...resultCourses.value, ...data.list]
    courseTotal.value = data.total
    coursePage.value = nextPage
  } catch (error) {
    coursesLoadMoreError.value = getErrorMessage(
      error,
      t('review.search.loadMoreCoursesFailed'),
    )
  } finally {
    coursesLoadingMore.value = false
  }
}

async function loadMoreReviews() {
  if (!hasMoreReviews.value || reviewsLoadingMore.value || searching.value) return
  const nextPage = reviewPage.value + 1
  reviewsLoadingMore.value = true
  reviewsLoadMoreError.value = ''
  try {
    const data = await fetchReviewResults(nextPage)
    resultReviews.value = [...resultReviews.value, ...data.list]
    reviewTotal.value = data.total
    reviewPage.value = nextPage
  } catch (error) {
    reviewsLoadMoreError.value = getErrorMessage(
      error,
      t('review.search.loadMoreReviewsFailed'),
    )
  } finally {
    reviewsLoadingMore.value = false
  }
}

// --- Lifecycle ---

onMounted(async () => {
  updateSearchFormPageMeta()
  await Promise.all([loadDepartments(), loadTerms()])
  await syncSearchFromRoute()
})

watch(
  () => route.query,
  () => {
    void syncSearchFromRoute()
  },
)

onUnmounted(() => {
  if (abortController) {
    abortController.abort()
    abortController = null
  }
})
</script>

<style scoped>
.animate-fade-in {
  animation: fadeIn 0.3s ease-out;
}

@keyframes fadeIn {
  from {
    opacity: 0;
    transform: translateY(8px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}
</style>
