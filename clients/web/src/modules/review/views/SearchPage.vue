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

      <!-- Course Conditions Section -->
      <div class="bg-bg-card rounded-2xl shadow-md p-6 mb-6">
        <h2 class="text-lg font-semibold text-text-primary mb-4 flex items-center gap-2">
          <span class="w-1 h-5 bg-primary rounded-full" />
          {{ t('review.search.courseConditions') }}
        </h2>

        <!-- Course Name -->
        <div class="mb-4">
          <label class="block text-sm font-medium mb-1.5 text-text-secondary">
            {{ t('review.search.courseName') }}
          </label>
          <input
            v-model="form.courseName"
            type="text"
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
          <label class="block text-sm font-medium mb-1.5 text-text-secondary">
            {{ t('review.search.courseCode') }}
          </label>
          <input
            v-model="form.courseCode"
            type="text"
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
          <label class="block text-sm font-medium mb-1.5 text-text-secondary">
            {{ t('review.search.department') }}
          </label>
          <select
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
          <label class="block text-sm font-medium mb-1.5 text-text-secondary">
            {{ t('review.search.teacherName') }}
          </label>
          <input
            v-model="form.teacherName"
            type="text"
            class="w-full px-4 py-3 bg-bg-elevated rounded-lg text-text-primary placeholder-text-muted focus:border-primary focus:ring-2 focus:ring-primary/20 outline-none transition-all"
            :placeholder="t('review.search.teacherName')"
          />
          <span class="block text-xs mt-1 text-text-muted">
            {{ t('review.search.teacherNameHelper') }}
          </span>
        </div>

        <!-- Semester -->
        <div>
          <label class="block text-sm font-medium mb-1.5 text-text-secondary">
            {{ t('review.search.semester') }}
          </label>
          <select
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
            <div
              v-for="(review, idx) in resultReviews"
              :key="review.id"
              class="bg-bg-card rounded-xl shadow-card p-5 stagger-item"
              :style="{ animationDelay: `${Math.min(idx, 8) * 60}ms` }"
            >
              <!-- Review Header -->
              <div class="flex items-center gap-2 mb-2">
                <router-link
                  :to="`/courses/${review.courseID}/reviews`"
                  class="text-base font-bold no-underline truncate text-text-primary hover:text-primary transition-colors"
                >
                  {{ review.title || review.courseName || `Course #${review.courseID}` }}
                </router-link>
                <span
                  v-if="reviewAvgRating(review) > 0"
                  class="text-xs font-bold font-mono py-px px-2 rounded-full shrink-0"
                  :style="{ backgroundColor: ratingBgColor(review), color: ratingTextColor(review) }"
                >
                  {{ reviewAvgRating(review).toFixed(1) }}
                </span>
              </div>

              <!-- Review Meta -->
              <div class="flex items-center gap-2 mb-3 text-xs text-text-muted">
                <span v-if="review.teacherName" class="font-medium text-primary">
                  {{ review.teacherName }}
                </span>
                <span v-if="review.teacherName">&middot;</span>
                <span v-if="review.termName">{{ review.termName }}</span>
              </div>

              <!-- Review Content -->
              <div
                class="text-sm leading-relaxed break-words line-clamp-3 text-text-secondary"
                v-text="review.content"
              />

              <!-- Emoji Ratings -->
              <div
                v-if="hasRatings(review)"
                class="flex flex-wrap gap-3 mt-4 pt-3 border-t border-border-light"
              >
                <span
                  v-for="dim in reviewDimensions(review)"
                  :key="dim.key"
                  class="text-xs flex items-center gap-1 text-text-secondary"
                >
                  <EmojiRating :value="Math.round(dim.value)" size="sm" />
                  <span>{{ dim.label }}</span>
                  <span class="font-mono font-medium text-text-primary">
                    {{ dim.value.toFixed(1) }}
                  </span>
                </span>
              </div>

              <!-- Controversial Badge -->
              <ControversialBadge
                v-if="review.dislikeCount >= 5"
                :dislike-count="review.dislikeCount"
                class="mt-3"
              />

              <!-- Vote Counts -->
              <div
                class="flex items-center gap-4 mt-3 pt-3 text-xs border-t border-border-light text-text-muted"
              >
                <span class="flex items-center gap-1">
                  <Heart :size="14" />
                  {{ review.likeCount }}
                </span>
                <span class="flex items-center gap-1">
                  <ThumbsDown :size="14" />
                  {{ review.dislikeCount }}
                </span>
                <span class="flex items-center gap-1">
                  <MessageCircle :size="14" />
                  {{ review.replyCount }}
                </span>
              </div>
            </div>
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
import { ArrowLeft, Search, SearchX, Heart, ThumbsDown, MessageCircle } from 'lucide-vue-next'
import EmojiRating from '@/components/business/review/EmojiRating.vue'
import ControversialBadge from '@/components/business/review/ControversialBadge.vue'
import { api } from '@/api'
import type { Department, Term, Course } from '@/types/course'
import type { Review } from '@/types/review'
import { getRatingColor } from '@/modules/course/theme'

const { t } = useI18n()
const router = useRouter()

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

let abortController: AbortController | null = null

// --- Computed ---

const coursesWithReviews = computed(() =>
  resultCourses.value.filter(c => c.reviewCount > 0)
)

const coursesWithoutReviews = computed(() =>
  resultCourses.value.filter(c => c.reviewCount === 0)
)

// --- Rating helpers ---

function reviewAvgRating(review: Review): number {
  const values = Object.values(review.ratings || {})
  if (values.length === 0) return 0
  return values.reduce((a, b) => a + b, 0) / values.length
}

function ratingTextColor(review: Review): string {
  return getRatingColor(reviewAvgRating(review))
}

function ratingBgColor(review: Review): string {
  const color = getRatingColor(reviewAvgRating(review))
  return `color-mix(in srgb, ${color} 15%, transparent)`
}

function hasRatings(review: Review): boolean {
  return Object.keys(review.ratings || {}).length > 0
}

function reviewDimensions(review: Review): Array<{ key: string; label: string; value: number }> {
  if (!review.ratings) return []
  return Object.entries(review.ratings).map(([key, value]) => ({
    key,
    label: t(`review.ratingEmoji.${key}`, t('review.ratingEmoji.fallback')),
    value: Math.max(0, Math.min(5, value)),
  }))
}

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
    departments.value = res.data?.data || []
  } catch {
    departments.value = []
  }
}

async function loadTerms() {
  try {
    const res = await api.course.getTerms()
    terms.value = res.data?.data || []
  } catch {
    terms.value = []
  }
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

  // Cancel any in-flight request
  if (abortController) {
    abortController.abort()
  }
  abortController = new AbortController()
  const signal = abortController.signal

  searching.value = true
  showResults.value = true
  resultCourses.value = []
  resultReviews.value = []

  try {
    // Build search queries from form
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

    const [courseRes, reviewRes] = await Promise.all([
      coursePromise.catch(() => null),
      reviewPromise.catch(() => null),
    ])

    if (signal.aborted) return

    // Process course results
    if (courseRes) {
      const raw = courseRes.data?.data
      const courseList: Course[] = (raw && 'list' in raw ? raw.list : raw) as Course[] ?? []
      resultCourses.value = courseList
    }

    // Process review results
    if (reviewRes) {
      resultReviews.value = reviewRes.list
    }
  } catch {
    if (!signal.aborted) {
      resultCourses.value = []
      resultReviews.value = []
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
