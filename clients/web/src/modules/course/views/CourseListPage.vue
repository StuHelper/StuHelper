<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter, type LocationQueryRaw, type LocationQueryValue } from 'vue-router'
import { BookOpen, ChevronDown, ChevronUp, Search, X } from 'lucide-vue-next'
import CourseThemeProvider from '@/modules/course/theme/CourseThemeProvider.vue'
import SkeletonCard from '@/components/common/SkeletonCard.vue'
import { api } from '@/api'
import { readGroupedCourseListPayload } from '@/modules/course/coursePayload'
import { usePinyinSearch, type PinyinSearchItem } from '@/composables/usePinyinSearch'
import type { Course } from '@stuhelper/shared/types/business/course'

interface DepartmentGroup {
  name: string
  courses: Course[]
  expanded: boolean
}

interface CourseSearchItem extends PinyinSearchItem {
  id: number
  name: string
  course: Course
  departmentName: string
}

const { t } = useI18n()
const route = useRoute()
const router = useRouter()

const loading = ref(true)
const error = ref<string | null>(null)
const departmentGroups = ref<DepartmentGroup[]>([])
let searchTimer: ReturnType<typeof setTimeout> | undefined

const allCourseSearchItems = computed<CourseSearchItem[]>(() =>
  departmentGroups.value.flatMap((group) =>
    group.courses.map((course) => ({
      id: course.id,
      name: [
        course.name,
        course.code,
        group.name,
      ].filter(Boolean).join(' '),
      course,
      departmentName: group.name,
    })),
  ),
)

const { query: searchQuery, results: searchResults } = usePinyinSearch<CourseSearchItem>({
  items: allCourseSearchItems,
  maxResults: 10000,
  sortBy: (a, b) => {
    const reviewDiff = (b.course.reviewCount ?? 0) - (a.course.reviewCount ?? 0)
    return reviewDiff !== 0 ? reviewDiff : a.course.name.localeCompare(b.course.name)
  },
})

searchQuery.value = readRouteSearchQuery()

const searchActive = computed(() => Boolean(searchQuery.value.trim()))
const matchedCourseIDs = computed(() => new Set(searchResults.value.map((item) => item.course.id)))
const visibleDepartmentGroups = computed(() => {
  if (!searchActive.value) {
    return departmentGroups.value.map((group, index) => ({ ...group, originalIndex: index }))
  }

  return departmentGroups.value
    .map((group, index) => ({
      ...group,
      courses: group.courses.filter((course) => matchedCourseIDs.value.has(course.id)),
      expanded: true,
      originalIndex: index,
    }))
    .filter((group) => group.courses.length > 0)
})
const visibleCourseCount = computed(() =>
  visibleDepartmentGroups.value.reduce((sum, group) => sum + group.courses.length, 0),
)
const advancedSearchRoute = computed(() => ({
  path: '/search',
  query: searchQuery.value.trim() ? { courseName: searchQuery.value.trim() } : {},
}))

const allExpanded = computed(() =>
  departmentGroups.value.length > 0 &&
  departmentGroups.value.every((g) => g.expanded),
)

const allCollapsed = computed(() =>
  departmentGroups.value.length > 0 &&
  departmentGroups.value.every((g) => !g.expanded),
)

function firstRouteQueryValue(value: LocationQueryValue | LocationQueryValue[] | undefined): string | null {
  if (Array.isArray(value)) {
    return value.find((item): item is string => typeof item === 'string') ?? null
  }
  return typeof value === 'string' ? value : null
}

function readRouteSearchQuery(): string {
  return firstRouteQueryValue(route.query.q)?.trim() ?? ''
}

function courseListQueryWithSearch(q: string): LocationQueryRaw {
  const nextQuery: LocationQueryRaw = {}
  for (const [key, value] of Object.entries(route.query)) {
    if (key === 'q') continue
    nextQuery[key] = value
  }
  if (q) {
    nextQuery.q = q
  }
  return nextQuery
}

function readDepartmentGroups(payload: unknown): DepartmentGroup[] {
  return readGroupedCourseListPayload(payload).map(group => ({
    name: group.departmentName || t('review.filters.all'),
    courses: group.courses,
    expanded: true,
  }))
}

function expandAll(): void {
  departmentGroups.value = departmentGroups.value.map((g) => ({
    ...g,
    expanded: true,
  }))
}

function collapseAll(): void {
  departmentGroups.value = departmentGroups.value.map((g) => ({
    ...g,
    expanded: false,
  }))
}

function toggleDepartment(index: number): void {
  departmentGroups.value = departmentGroups.value.map((g, i) =>
    i === index ? { ...g, expanded: !g.expanded } : g,
  )
}

function handleSearchInput(): void {
  if (searchTimer) clearTimeout(searchTimer)
  const q = searchQuery.value.trim()
  searchTimer = setTimeout(() => {
    void router.replace({ path: route.path, query: courseListQueryWithSearch(q) })
  }, 250)
}

function clearSearch(): void {
  if (searchTimer) clearTimeout(searchTimer)
  searchQuery.value = ''
  void router.replace({ path: route.path, query: courseListQueryWithSearch('') })
}

async function fetchCourses(): Promise<void> {
  loading.value = true
  error.value = null
  try {
    const res = await api.course.getCoursesGrouped()
    departmentGroups.value = readDepartmentGroups(res.data?.data)
  } catch (_error) { void _error;
    error.value = t('review.courseList.loadFailed')
    departmentGroups.value = []
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  void fetchCourses()
})

watch(
  () => route.query.q,
  () => {
    const q = readRouteSearchQuery()
    if (searchQuery.value !== q) {
      searchQuery.value = q
    }
  },
)

onBeforeUnmount(() => {
  if (searchTimer) clearTimeout(searchTimer)
})
</script>

<template>
  <CourseThemeProvider>
    <div class="min-h-screen w-full bg-bg-base">
      <!-- Top toolbar -->
      <div class="flex flex-col gap-3 bg-bg-card border-b border-border-light px-6 py-4 md:flex-row md:items-center md:justify-between">
        <div class="min-w-0">
          <h1 class="text-xl font-bold text-text-primary">
            {{ t('review.courseList.title') }}
          </h1>
          <p v-if="searchActive" class="mt-1 text-sm text-text-secondary">
            {{ t('review.courseList.searchCount', { count: visibleCourseCount }) }}
          </p>
        </div>
        <div class="flex flex-col gap-2 sm:flex-row sm:items-center">
          <label class="relative block min-w-0 sm:w-80">
            <span class="sr-only">{{ t('review.courseList.searchLabel') }}</span>
            <Search
              class="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-text-muted"
              :size="16"
            />
            <input
              v-model="searchQuery"
              type="search"
              autocomplete="off"
              data-1p-ignore
              data-lpignore="true"
              data-form-type="other"
              class="h-10 w-full rounded-lg border border-border-light bg-bg-base pl-9 pr-9 text-sm text-text-primary outline-none transition-colors placeholder:text-text-muted focus:border-primary"
              :aria-label="t('review.courseList.searchLabel')"
              :placeholder="t('review.courseList.searchPlaceholder')"
              @input="handleSearchInput"
            />
            <button
              v-if="searchQuery"
              type="button"
              class="absolute right-2 top-1/2 flex h-6 w-6 -translate-y-1/2 items-center justify-center rounded text-text-muted transition-colors hover:bg-bg-elevated hover:text-text-primary"
              :aria-label="t('review.courseList.clearSearch')"
              :title="t('review.courseList.clearSearch')"
              @click="clearSearch"
            >
              <X :size="14" />
            </button>
          </label>
          <div class="flex gap-2">
            <button
              type="button"
              :disabled="allExpanded"
              :aria-label="t('review.courseList.expandAll')"
              :title="t('review.courseList.expandAll')"
              class="flex h-10 w-10 items-center justify-center rounded-lg transition-colors"
              :class="
                allExpanded
                  ? 'bg-bg-elevated text-text-tertiary cursor-not-allowed'
                  : 'bg-primary text-white hover:bg-primary/90 cursor-pointer'
              "
              @click="expandAll"
            >
              <ChevronDown :size="18" />
            </button>
            <button
              type="button"
              :disabled="allCollapsed"
              :aria-label="t('review.courseList.collapseAll')"
              :title="t('review.courseList.collapseAll')"
              class="flex h-10 w-10 items-center justify-center rounded-lg transition-colors"
              :class="
                allCollapsed
                  ? 'bg-bg-elevated text-text-tertiary cursor-not-allowed'
                  : 'bg-primary text-white hover:bg-primary/90 cursor-pointer'
              "
              @click="collapseAll"
            >
              <ChevronUp :size="18" />
            </button>
          </div>
        </div>
      </div>

      <!-- Loading state -->
      <div
        v-if="loading"
        class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4 p-6"
      >
        <SkeletonCard v-for="i in 9" :key="i" />
      </div>

      <!-- Error state -->
  <div v-else-if="error" class="flex flex-col items-center justify-center py-20 text-text-secondary">
        <p class="mb-4">{{ error }}</p>
        <button
          type="button"
          class="rounded-lg bg-primary px-4 py-2 text-white hover:bg-primary/90 cursor-pointer"
          @click="fetchCourses"
        >
          {{ t('common.actions.retry') }}
        </button>
      </div>

      <!-- Empty state -->
      <div v-else-if="departmentGroups.length === 0" class="flex items-center justify-center py-20 text-text-secondary">
        {{ t('review.courseList.noCourses') }}
      </div>

      <!-- Search empty state -->
      <div v-else-if="visibleDepartmentGroups.length === 0" class="flex flex-col items-center justify-center px-6 py-20 text-center text-text-secondary">
        <BookOpen :size="42" class="mb-4 opacity-40" />
        <p class="mb-3 text-sm">{{ t('review.courseList.noSearchResults') }}</p>
        <router-link
          :to="advancedSearchRoute"
          class="text-sm font-semibold text-primary underline underline-offset-4 hover:text-primary/80"
        >
          {{ t('review.courseList.advancedSearch') }}
        </router-link>
      </div>

      <!-- Department groups -->
      <div v-else class="p-6 space-y-4">
        <div
          v-for="group in visibleDepartmentGroups"
          :key="group.name"
          class="rounded-xl bg-bg-card border border-border-light overflow-hidden"
        >
          <button
            type="button"
            class="flex w-full items-center justify-between px-5 py-3 cursor-pointer hover:bg-bg-elevated/50 transition-colors"
            :aria-expanded="group.expanded"
            @click="toggleDepartment(group.originalIndex)"
          >
            <span class="text-base font-semibold text-text-primary">
              {{ group.name }}
              <span class="ml-2 text-sm font-normal text-text-tertiary">({{ group.courses.length }})</span>
            </span>
            <ChevronDown
              :size="16"
              class="text-text-tertiary transition-transform"
              :class="{ 'rotate-180': !group.expanded }"
            />
          </button>

          <div v-show="group.expanded" class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3 px-5 pb-4">
            <router-link
              v-for="course in group.courses"
              :key="course.id"
              :to="`/courses/${course.id}/reviews`"
              class="flex items-center justify-between rounded-lg border border-border-light px-4 py-3 cursor-pointer hover:bg-bg-elevated transition-colors"
            >
              <div class="min-w-0">
                <p class="text-sm font-medium text-text-primary truncate">{{ course.name }}</p>
                <p v-if="course.code" class="text-xs text-text-tertiary mt-0.5">{{ course.code }}</p>
              </div>
              <span class="ml-3 shrink-0 text-xs text-text-secondary">
                {{ t('review.courseList.reviewCount', { count: course.reviewCount }) }}
              </span>
            </router-link>
          </div>
        </div>
      </div>
    </div>
  </CourseThemeProvider>
</template>
