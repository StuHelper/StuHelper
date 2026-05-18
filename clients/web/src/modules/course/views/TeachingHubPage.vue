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
        <p class="mt-1 text-sm font-medium text-secondary">
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
          class="absolute left-0 right-0 mt-1 rounded-xl overflow-hidden z-[var(--z-dropdown)]
                 bg-bg-card shadow-lg
                 max-h-[360px] overflow-y-auto"
        >
          <template v-if="results.length > 0">
            <div
              v-for="(course, idx) in results"
              :key="course.id"
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
          <div v-else class="px-4 py-6 text-center text-sm text-text-muted">
            <p>{{ t('review.home.courseNotFound') }}</p>
            <p class="mt-1">
              <router-link to="/about" class="text-primary hover:text-primary-light" @click="showDropdown = false">
                {{ t('review.home.contactUs') }}
              </router-link>
              {{ t('review.home.contactFeedback') }}
              {{ t('review.home.tryAdvancedSearch') }}
              <router-link to="/search" class="text-primary hover:text-primary-light" @click="showDropdown = false">
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
            to="/courses/reviews"
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
import { ref, onMounted, onBeforeUnmount } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { Search, X, BookOpen, PenLine } from 'lucide-vue-next'
import { api } from '@/api'
import { usePinyinSearch, type PinyinSearchItem } from '@/composables/usePinyinSearch'
import ScrollReveal from '@/components/animated/ScrollReveal.vue'

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
}

const { t } = useI18n()
const router = useRouter()

const allCourses = ref<CourseItem[]>([])
const hotCourses = ref<HotCourse[]>([])
const reviewStats = ref<ReviewStats | null>(null)
const errorMessage = ref('')
const showDropdown = ref(false)
const searchInputRef = ref<HTMLInputElement | null>(null)
const currentTerm = ref('')

let errorTimer: ReturnType<typeof setTimeout> | undefined

const { query, results, selectedIndex, handleKeyDown, clear } = usePinyinSearch<CourseItem>({
  items: allCourses,
  maxResults: 10,
  sortBy: (a, b) => b.reviewCount - a.reviewCount,
})

function onSearchKeyDown(e: KeyboardEvent) {
  if (e.key === 'Escape') {
    showDropdown.value = false
    return
  }
  const selected = handleKeyDown(e)
  if (selected) {
    navigateToCourse(selected.id)
  }
}

function navigateToCourse(courseId: number) {
  showDropdown.value = false
  clear()
  router.push(`/courses/${courseId}`)
}

function clearSearch() {
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

onMounted(async () => {
  document.addEventListener('click', handleClickOutside)

  const [coursesRes, statsRes, hotRes, termsRes] = await Promise.allSettled([
    api.course.getCourses({ page: 1, pageSize: 100 }),
    api.review.getReviewStats(),
    api.review.getHotCourses({ limit: 6 }),
    api.course.getTerms(),
  ])

  if (coursesRes.status === 'fulfilled') {
    const list = coursesRes.value.data?.data?.list
    if (list) {
      allCourses.value = list.map((c) => ({
        id: c.id,
        name: c.name,
        departmentName: c.departmentName ?? undefined,
        reviewCount: c.reviewCount,
      }))
    }
  } else {
    showError(String(coursesRes.reason))
  }

  if (statsRes.status === 'fulfilled') {
    const data = statsRes.value.data?.data
    if (data) {
      reviewStats.value = {
        courseCount: data.courseCount,
        reviewCount: data.reviewCount,
        departmentCount: data.departmentCount,
      }
    }
  }

  if (hotRes.status === 'fulfilled') {
    const list = hotRes.value.data?.data?.list
    if (list) {
      hotCourses.value = list.slice(0, 6)
    }
  }

  if (termsRes.status === 'fulfilled') {
    const terms = termsRes.value.data?.data
    if (Array.isArray(terms) && terms.length > 0) {
      currentTerm.value = terms[0].name ?? terms[0].id ?? ''
    }
  }
})

onBeforeUnmount(() => {
  document.removeEventListener('click', handleClickOutside)
  if (errorTimer) clearTimeout(errorTimer)
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
