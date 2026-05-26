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
        @click="handleSearch"
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
        v-if="searchError"
        role="alert"
        class="mb-4 rounded-xl border border-warning/30 bg-warning/10 px-4 py-3 text-sm text-warning"
      >
        {{ searchError }}
      </div>

      <!-- No Results -->
      <div
        v-else-if="resultCourses.length === 0 && resultReviews.length === 0"
        class="text-center py-16 text-text-muted"
      >
        <SearchX :size="48" class="mx-auto mb-4 opacity-50" />
        <p class="text-base">{{ t('review.search.noResults') }}</p>
      </div>

      <!-- Results Content -->
      <template v-else>
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
        </section>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, computed, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { ArrowLeft, Search, SearchX } from 'lucide-vue-next'
import ReviewCard from '@/components/business/review/ReviewCard.vue'
import { api } from '@/api'
import { getErrorMessage } from '@/api/errors'
import { useToast } from '@/composables/useToast'
import {
  readCourseListPayload,
  readDepartmentArrayPayload,
  readTermArrayPayload,
} from '@/modules/course/coursePayload'
import type { Department, Term, Course } from '@stuhelper/shared/course'
import type { Review } from '@stuhelper/shared/review'

const { t } = useI18n()
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
const searchError = ref('')
const referenceError = ref('')

let abortController: AbortController | null = null

// --- Computed ---

const coursesWithReviews = computed(() =>
  resultCourses.value.filter(c => c.reviewCount > 0)
)

const coursesWithoutReviews = computed(() =>
  resultCourses.value.filter(c => c.reviewCount === 0)
)

// --- Navigation ---

function goHome() {
  router.push({ name: 'home' })
}

function backToForm() {
  showResults.value = false
  resultCourses.value = []
  resultReviews.value = []
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

async function handleSearch() {
  if (!validateForm()) return
  if (searching.value) return

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
  searchError.value = ''

  try {
    // 根据表单内容构造课程与评测查询条件
    const courseQuery = form.courseName.trim() || form.courseCode.trim()

    const coursePromise = courseQuery
      ? api.course.searchCourses(courseQuery, { pageSize: 50 }, { signal })
      : form.departmentID > 0
        ? api.course.getCourses({ departmentID: form.departmentID, pageSize: 50 })
        : Promise.resolve(null)

    const reviewPromise = api.review.searchReviewsPage(
      {
        q: courseQuery || undefined,
        departmentID: form.departmentID > 0 ? form.departmentID : undefined,
        teacherName: form.teacherName.trim() || undefined,
        termID: form.termID || undefined,
        pageSize: 50,
        sort: 'time',
      },
      { signal },
    )

    const [courseRes, reviewRes] = await Promise.allSettled([coursePromise, reviewPromise])

    if (signal.aborted) return

    if (courseRes.status === 'fulfilled' && courseRes.value) {
      try {
        resultCourses.value = readCourseListPayload(
          courseRes.value.data?.data,
          'Invalid course search response',
        )
      } catch (error) {
        resultCourses.value = []
        searchError.value = getErrorMessage(error, t('common.loadFailed'))
      }
    } else if (courseRes.status === 'rejected') {
      searchError.value = getErrorMessage(courseRes.reason, t('common.loadFailed'))
    }

    if (reviewRes.status === 'fulfilled') {
      resultReviews.value = reviewRes.value.list
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

// --- Lifecycle ---

onMounted(() => {
  loadDepartments()
  loadTerms()
})

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
