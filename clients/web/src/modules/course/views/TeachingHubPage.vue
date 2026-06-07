<template>
  <div class="min-h-screen bg-bg-base">
    <div class="animate-fade-in max-w-4xl mx-auto py-10 px-4 sm:px-6">
      <!-- Hero Header -->
      <header class="text-center mb-10">
        <h1 class="text-3xl sm:text-4xl font-extrabold tracking-tight gradient-text">
          {{ t('review.home.title') }}
        </h1>
        <p v-if="reviewStats" class="mt-2 text-base text-text-secondary">
          {{ t('review.home.subtitle', { count: reviewStats.reviewCount }) }}
        </p>
        <p v-if="currentTerm" class="mt-1 text-sm font-medium text-secondary">
          {{ t('review.home.dataUpdated', { term: currentTerm }) }}
        </p>
      </header>

      <!-- Search bar with glass effect -->
      <div class="relative mb-10 max-w-2xl mx-auto">
        <div class="relative">
          <Search
            class="absolute left-4 top-1/2 -translate-y-1/2 pointer-events-none text-text-muted"
            :size="20"
          />
          <input
            ref="searchInputRef"
            v-model="query"
            type="text"
            autocomplete="off"
            data-1p-ignore
            data-lpignore="true"
            data-form-type="other"
            class="w-full pl-12 pr-12 py-4 text-base rounded-xl outline-none
                   bg-bg-glass-heavy backdrop-blur-sm
                   shadow-card
                   text-text-primary placeholder:text-text-muted
                   transition-all duration-base ease-smooth
                   focus:shadow-glow-primary focus:ring-1 focus:ring-primary/30"
            :placeholder="t('review.home.searchPlaceholder')"
            :aria-label="t('review.home.searchPlaceholder')"
            @keydown="onSearchKeyDown"
            @focus="showDropdown = true"
          />
          <button
            v-if="query"
            class="absolute right-4 top-1/2 -translate-y-1/2 p-0 text-text-muted hover:text-text-primary transition-colors duration-fast"
            :aria-label="t('review.home.clear')"
            @click="clearSearch"
          >
            <X :size="18" />
          </button>
        </div>

        <!-- Autocomplete dropdown -->
        <div
          v-if="showDropdown && query.trim()"
          id="course-hub-search-listbox"
          role="listbox"
          class="absolute left-0 right-0 mt-1 rounded-xl overflow-hidden z-[var(--z-dropdown)]
                 bg-bg-card shadow-lg
                 max-h-[360px] overflow-y-auto"
        >
          <template v-if="displayedResults.length > 0">
            <div
              v-for="(course, idx) in displayedResults"
              :key="course.id"
              role="option"
              :aria-selected="idx === selectedIndex"
              class="flex items-center justify-between px-4 py-3 cursor-pointer transition-colors duration-fast"
              :class="idx === selectedIndex ? 'bg-bg-hover' : 'bg-transparent'"
              @mouseenter="selectedIndex = idx"
              @click="navigateToCourse(course.id)"
            >
              <div class="flex-1 min-w-0">
                <span class="text-sm font-medium truncate block text-text-primary">
                  {{ course.name }}
                </span>
                <span v-if="course.departmentName" class="text-xs mt-0.5 block text-text-muted">
                  {{ course.departmentName }}
                </span>
              </div>
              <span class="ml-3 shrink-0 text-xs font-medium px-2.5 py-1 rounded-full
                           bg-primary-alpha text-primary">
                {{ course.reviewCount }}{{ t('review.home.reviewUnit') }}
              </span>
            </div>
          </template>
          <div v-else-if="remoteSearchLoading" class="px-4 py-6 text-center text-sm text-text-muted">
            {{ t('common.actions.loading') }}
          </div>
          <div v-else-if="visibleSearchError" role="alert" class="px-4 py-6 text-center text-sm text-danger">
            {{ visibleSearchError }}
          </div>
          <div v-else class="px-4 py-6 text-center text-sm text-text-muted">
            <p>{{ t('review.home.courseNotFound') }}</p>
            <p class="mt-1">
              <router-link to="/about" class="text-primary hover:text-primary-light" @click="showDropdown = false">
                {{ t('review.home.contactUs') }}
              </router-link>
              {{ t('review.home.contactFeedback') }}
              {{ t('review.home.tryAdvancedSearch') }}
              <router-link :to="advancedSearchRoute" class="text-primary hover:text-primary-light" @click="showDropdown = false">
                {{ t('review.home.advancedSearch') }}
              </router-link>
            </p>
          </div>
        </div>
      </div>

      <!-- CTA cards -->
      <div class="grid grid-cols-1 sm:grid-cols-2 gap-6 mb-12">
        <!-- Browse courses card -->
        <ScrollReveal :delay="0">
        <div class="glass-card shadow-card rounded-xl p-6 hover-lift">
          <div class="w-12 h-12 flex items-center justify-center rounded-xl mb-4 bg-primary-alpha">
            <BookOpen :size="26" class="text-primary" />
          </div>
          <h2 class="text-lg font-bold mb-2 text-text-primary">
            {{ t('review.home.browseCourses') }}
          </h2>
          <p class="text-sm mb-5 leading-relaxed text-text-secondary">
            {{ t('review.home.browseCoursesDesc') }}
          </p>
          <router-link
            to="/courses/list"
            class="inline-block no-underline text-sm font-medium
                   px-5 py-2.5 rounded-lg
                   bg-primary text-white
                   hover:bg-primary-dark
                   transition-colors duration-fast"
          >
            {{ t('review.home.viewAll') }}
          </router-link>
        </div>
        </ScrollReveal>

        <!-- Post review card -->
        <ScrollReveal :delay="120">
        <div class="glass-card shadow-card rounded-xl p-6 hover-lift">
          <div class="w-12 h-12 flex items-center justify-center rounded-xl mb-4 bg-accent-alpha">
            <PenLine :size="26" class="text-accent" />
          </div>
          <h2 class="text-lg font-bold mb-2 text-text-primary">
            {{ t('review.home.postReview') }}
          </h2>
          <p class="text-sm mb-5 leading-relaxed text-text-secondary">
            {{ t('review.home.postReviewDesc') }}
          </p>
          <router-link
            :to="{ name: 'course-review-post' }"
            class="inline-block no-underline text-sm font-medium
                   px-5 py-2.5 rounded-lg
                   bg-accent text-white
                   hover:bg-accent-dark
                   transition-colors duration-fast"
          >
            {{ t('review.home.writeReview') }}
          </router-link>
        </div>
        </ScrollReveal>
      </div>

      <!-- Hot courses grid -->
      <section v-if="hotCourses.length > 0">
        <h2 class="text-xl font-bold mb-6 gradient-text inline-block">
          {{ t('review.home.hotCourses') }}
        </h2>
        <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
          <div
            v-for="(course, idx) in hotCourses"
            :key="course.courseID"
            class="glass-card shadow-card rounded-xl p-5 cursor-pointer hover-lift stagger-item"
            :style="{ animationDelay: `${idx * 80}ms` }"
            @click="navigateToCourse(course.courseID)"
          >
            <h3 class="text-base font-semibold mb-2 truncate text-text-primary">
              {{ course.courseName }}
            </h3>
            <div class="flex items-center justify-between">
              <span class="text-xs font-medium px-2.5 py-1 rounded-full
                           bg-primary-alpha text-primary">
                {{ course.reviewCount }}{{ t('review.home.reviewUnit') }}
              </span>
            </div>
          </div>
        </div>
      </section>

      <!-- Snackbar for errors -->
      <Transition name="snackbar">
        <div
          v-if="errorMessage"
          role="alert"
          class="fixed bottom-6 left-1/2 -translate-x-1/2 px-6 py-3 rounded-lg text-sm font-medium z-[var(--z-toast)]
                 bg-danger text-white shadow-lg"
        >
          {{ errorMessage }}
        </div>
      </Transition>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, onMounted, onBeforeUnmount, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { Search, X, BookOpen, PenLine } from 'lucide-vue-next'
import { api } from '@/api'
import { readCourseListPayload, readCoursePagePayload, readTermArrayPayload } from '@/modules/course/coursePayload'
import { usePinyinSearch, type PinyinSearchItem } from '@/composables/usePinyinSearch'
import ScrollReveal from '@/components/animated/ScrollReveal.vue'
import type { Course } from '@stuhelper/shared/course'

interface CourseItem extends PinyinSearchItem {
  id: number
  name: string
  departmentName?: string
  reviewCount: number
}

interface HotCourse {
  courseID: number
  courseName: string
  reviewCount: number
  avgRating: number
}

interface ReviewStats {
  courseCount: number
  reviewCount: number
  departmentCount: number
  userCount: number
}

const { t } = useI18n()
const router = useRouter()

const allCourses = ref<CourseItem[]>([])
const remoteResults = ref<CourseItem[]>([])
const hotCourses = ref<HotCourse[]>([])
const reviewStats = ref<ReviewStats | null>(null)
const errorMessage = ref('')
const searchCatalogError = ref('')
const remoteSearchError = ref('')
const remoteSearchLoading = ref(false)
const showDropdown = ref(false)
const searchInputRef = ref<HTMLInputElement | null>(null)
const currentTerm = ref('')

let errorTimer: ReturnType<typeof setTimeout> | undefined
let remoteSearchTimer: ReturnType<typeof setTimeout> | undefined
let remoteSearchController: AbortController | undefined

const { query, results: localResults, selectedIndex, clear } = usePinyinSearch<CourseItem>({
  items: allCourses,
  maxResults: 10,
  sortBy: (a, b) => b.reviewCount - a.reviewCount,
})

const displayedResults = computed(() => {
  const seen = new Set<number>()
  const merged: CourseItem[] = []

  for (const course of [...localResults.value, ...remoteResults.value]) {
    if (seen.has(course.id)) continue
    seen.add(course.id)
    merged.push(course)
    if (merged.length >= 10) break
  }

  return merged
})

const visibleSearchError = computed(() => {
  if (searchCatalogError.value) return searchCatalogError.value
  if (displayedResults.value.length > 0) return ''
  return remoteSearchError.value
})

const advancedSearchRoute = computed(() => {
  const courseName = query.value.trim()
  return courseName
    ? { name: 'search', query: { courseName } }
    : { name: 'search' }
})

function onSearchKeyDown(e: KeyboardEvent) {
  if (e.key === 'Escape') {
    showDropdown.value = false
    return
  }
  if (!displayedResults.value.length) return

  if (e.key === 'ArrowDown') {
    e.preventDefault()
    selectedIndex.value = Math.min(selectedIndex.value + 1, displayedResults.value.length - 1)
    return
  }

  if (e.key === 'ArrowUp') {
    e.preventDefault()
    selectedIndex.value = Math.max(selectedIndex.value - 1, 0)
    return
  }

  if (e.key === 'Enter') {
    e.preventDefault()
    const selected = displayedResults.value[selectedIndex.value]
    if (selected) {
      navigateToCourse(selected.id)
    }
  }
}

function navigateToCourse(courseId: number) {
  showDropdown.value = false
  clear()
  router.push(`/courses/${courseId}`)
}

function clearSearch() {
  resetRemoteSearch()
  clear()
  showDropdown.value = false
  searchInputRef.value?.focus()
}

function showError(message: string) {
  errorMessage.value = message
  if (errorTimer) clearTimeout(errorTimer)
  errorTimer = setTimeout(() => {
    errorMessage.value = ''
  }, 4000)
}

function handleClickOutside(e: MouseEvent) {
  const target = e.target as Node
  if (searchInputRef.value && !searchInputRef.value.closest('.relative')?.contains(target)) {
    showDropdown.value = false
  }
}

function isNonNegativeNumber(value: unknown): value is number {
  return typeof value === 'number' && Number.isFinite(value) && value >= 0
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return Boolean(value) && typeof value === 'object' && !Array.isArray(value)
}

function mapCourseItem(raw: Course): CourseItem {
  return {
    id: raw.id,
    name: raw.name,
    departmentName: raw.departmentName,
    reviewCount: raw.reviewCount,
  }
}

function resetRemoteSearch() {
  if (remoteSearchTimer) {
    clearTimeout(remoteSearchTimer)
    remoteSearchTimer = undefined
  }
  if (remoteSearchController) {
    remoteSearchController.abort()
    remoteSearchController = undefined
  }
  remoteResults.value = []
  remoteSearchError.value = ''
  remoteSearchLoading.value = false
}

async function searchRemoteCourses(term: string) {
  const controller = new AbortController()
  remoteSearchController = controller
  remoteSearchLoading.value = true
  remoteSearchError.value = ''

  try {
    const res = await api.course.searchCourses(term, { pageSize: 10 }, { signal: controller.signal })
    if (controller.signal.aborted) return
    remoteResults.value = readCourseListPayload(
      res.data?.data,
      'Invalid course search response',
    ).map(mapCourseItem)
  } catch (_error: unknown) {
    void _error
    if (controller.signal.aborted) return
    remoteResults.value = []
    remoteSearchError.value = t('common.loadFailed')
  } finally {
    if (remoteSearchController === controller) {
      remoteSearchController = undefined
      remoteSearchLoading.value = false
    }
  }
}

watch(query, (value) => {
  selectedIndex.value = 0
  remoteSearchError.value = ''

  if (remoteSearchTimer) {
    clearTimeout(remoteSearchTimer)
    remoteSearchTimer = undefined
  }
  if (remoteSearchController) {
    remoteSearchController.abort()
    remoteSearchController = undefined
  }

  const term = value.trim()
  if (!term || searchCatalogError.value) {
    remoteResults.value = []
    remoteSearchLoading.value = false
    return
  }

  remoteSearchLoading.value = true
  remoteSearchTimer = setTimeout(() => {
    remoteSearchTimer = undefined
    void searchRemoteCourses(term)
  }, 300)
})

watch(displayedResults, () => {
  selectedIndex.value = 0
})

function mapHotCourse(raw: unknown): HotCourse {
  if (!isRecord(raw)) {
    throw new Error('Invalid hot courses response')
  }

  const courseID = raw.courseID
  const courseName = raw.courseName
  const reviewCount = raw.reviewCount
  const avgRating = raw.avgRating
  if (
    !isNonNegativeNumber(courseID) ||
    typeof courseName !== 'string' ||
    !isNonNegativeNumber(reviewCount) ||
    !isNonNegativeNumber(avgRating)
  ) {
    throw new Error('Invalid hot courses response')
  }

  return { courseID, courseName, reviewCount, avgRating }
}

function readHotCourseListPayload(payload: unknown): HotCourse[] {
  if (!isRecord(payload) || !Array.isArray(payload.list)) {
    throw new Error('Invalid hot courses response')
  }

  return payload.list.map(mapHotCourse)
}

function readReviewStatsPayload(payload: unknown): ReviewStats {
  if (!isRecord(payload)) {
    throw new Error('Invalid review stats response')
  }

  const courseCount = payload.courseCount
  const reviewCount = payload.reviewCount
  const departmentCount = payload.departmentCount
  const userCount = payload.userCount
  if (
    !isNonNegativeNumber(courseCount) ||
    !isNonNegativeNumber(reviewCount) ||
    !isNonNegativeNumber(departmentCount) ||
    !isNonNegativeNumber(userCount)
  ) {
    throw new Error('Invalid review stats response')
  }

  return { courseCount, reviewCount, departmentCount, userCount }
}

function readCurrentTermPayload(payload: unknown): string {
  const terms = readTermArrayPayload(payload, 'Invalid terms response')
  if (terms.length === 0) return ''

  return (terms.find(term => term.isCurrent) ?? terms[0]).name
}

onMounted(async () => {
  document.addEventListener('click', handleClickOutside)

  const [coursesRes, statsRes, hotRes, termsRes] = await Promise.allSettled([
    api.course.getCourses({ page: 1, pageSize: 100 }),
    api.review.getReviewStats(),
    api.review.getHotCourses({ limit: 6 }),
    api.course.getTerms(),
  ])

  if (coursesRes.status === 'fulfilled') {
    try {
      const data = readCoursePagePayload(
        coursesRes.value.data?.data,
        'Invalid course catalog response',
      )
      allCourses.value = data.list.map(mapCourseItem)
      searchCatalogError.value = ''
    } catch (_error) { void _error;
      allCourses.value = []
      searchCatalogError.value = t('common.loadFailed')
      showError(t('common.loadFailed'))
    }
  } else {
    allCourses.value = []
    searchCatalogError.value = t('common.loadFailed')
    showError(t('common.loadFailed'))
  }

  if (statsRes.status === 'fulfilled') {
    try {
      reviewStats.value = readReviewStatsPayload(statsRes.value.data?.data)
    } catch (_error) { void _error;
      reviewStats.value = null
      showError(t('common.loadFailed'))
    }
  } else {
    reviewStats.value = null
    showError(t('common.loadFailed'))
  }

  if (hotRes.status === 'fulfilled') {
    try {
      hotCourses.value = readHotCourseListPayload(hotRes.value.data?.data).slice(0, 6)
    } catch (_error) { void _error;
      hotCourses.value = []
      showError(t('common.loadFailed'))
    }
  } else {
    hotCourses.value = []
    showError(t('common.loadFailed'))
  }

  if (termsRes.status === 'fulfilled') {
    try {
      currentTerm.value = readCurrentTermPayload(termsRes.value.data?.data)
    } catch (_error) { void _error;
      currentTerm.value = ''
      showError(t('common.loadFailed'))
    }
  } else {
    currentTerm.value = ''
    showError(t('common.loadFailed'))
  }
})

onBeforeUnmount(() => {
  document.removeEventListener('click', handleClickOutside)
  if (errorTimer) clearTimeout(errorTimer)
  resetRemoteSearch()
})
</script>

<style scoped>
/* Snackbar transition */
.snackbar-enter-active,
.snackbar-leave-active {
  transition: opacity var(--duration-slow) var(--ease-out),
              transform var(--duration-slow) var(--ease-out);
}

.snackbar-enter-from,
.snackbar-leave-to {
  opacity: 0;
  transform: translate(-50%, 20px);
}
</style>
