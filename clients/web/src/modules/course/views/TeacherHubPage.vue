<script setup lang="ts">
import { ref, computed, onMounted, onBeforeUnmount, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter, type LocationQueryRaw, type LocationQueryValue } from 'vue-router'
import { Search, X, User, BookOpen, RefreshCw } from 'lucide-vue-next'
import { api } from '@/api'
import {
  readTeacherSummaryListPayload,
  readTeacherSummaryPagePayload,
  type TeacherSummaryPayload,
} from '@/modules/course/coursePayload'
import EmojiRating from '@/components/business/review/EmojiRating.vue'

interface TeacherEntry {
  teacherID: number
  teacherName: string
  departmentName?: string
  avgRating: number | null
  reviewCount: number
  courseCount: number
}

const { t } = useI18n()
const route = useRoute()
const router = useRouter()

const searchQuery = ref(readRouteSearchQuery())
const searchResults = ref<TeacherEntry[]>([])
const searchTotal = ref(0)
const searchPage = ref(1)
const searching = ref(false)
const searchLoadingMore = ref(false)
const loading = ref(true)
const errorMessage = ref('')
const loadMoreError = ref('')
const popularTeachers = ref<TeacherEntry[]>([])
const TEACHER_SEARCH_PAGE_SIZE = 30

let searchTimer: ReturnType<typeof setTimeout> | undefined
let searchRequestSeq = 0

function firstRouteQueryValue(value: LocationQueryValue | LocationQueryValue[] | undefined): string | null {
  if (Array.isArray(value)) {
    return value.find((item): item is string => typeof item === 'string') ?? null
  }
  return typeof value === 'string' ? value : null
}

function readRouteSearchQuery(): string {
  return firstRouteQueryValue(route.query.q)?.trim() ?? ''
}

function teacherHubQueryWithSearch(q: string): LocationQueryRaw {
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

function resetSearchState(): void {
  searchResults.value = []
  searchTotal.value = 0
  searchPage.value = 1
  errorMessage.value = ''
  loadMoreError.value = ''
  searching.value = false
  searchLoadingMore.value = false
}

async function loadPopularTeachers() {
  loading.value = true
  errorMessage.value = ''
  try {
    const res = await api.rating.listHotTeachers(20)
    const list = readTeacherSummaryListPayload(
      res.data?.data,
      'Invalid popular teachers response',
    )
    popularTeachers.value = list.map(mapTeacher)
  } catch (_error) { void _error;
    popularTeachers.value = []
    errorMessage.value = t('common.loadFailed')
  } finally {
    loading.value = false
  }
}

function mapTeacher(raw: TeacherSummaryPayload): TeacherEntry {
  return {
    teacherID: raw.teacherID,
    teacherName: raw.teacherName,
    departmentName: raw.departmentName || undefined,
    avgRating: raw.avgRating ?? null,
    reviewCount: raw.reviewCount,
    courseCount: raw.courseCount,
  }
}

function handleSearchInput() {
  if (searchTimer) clearTimeout(searchTimer)
  searchRequestSeq += 1
  const q = searchQuery.value.trim()
  if (!q) {
    resetSearchState()
    void router.replace({ path: route.path, query: teacherHubQueryWithSearch('') })
    return
  }
  searchTimer = setTimeout(() => {
    void router.replace({ path: route.path, query: teacherHubQueryWithSearch(q) })
  }, 350)
}

async function doSearch(nextPage = 1, append = false) {
  const q = searchQuery.value.trim()
  if (!q) return
  const requestSeq = ++searchRequestSeq

  if (append) {
    searchLoadingMore.value = true
    loadMoreError.value = ''
  } else {
    searching.value = true
    errorMessage.value = ''
    loadMoreError.value = ''
  }
  try {
    const res = await api.rating.listTeachers({
      q,
      sort: 'reviews',
      page: nextPage,
      pageSize: TEACHER_SEARCH_PAGE_SIZE,
    })
    const data = readTeacherSummaryPagePayload(
      res.data?.data,
      'Invalid teacher search response',
    )
    if (requestSeq !== searchRequestSeq || searchQuery.value.trim() !== q) return
    const nextTeachers = data.list.map(mapTeacher)
    searchResults.value = append
      ? [...searchResults.value, ...nextTeachers]
      : nextTeachers
    searchTotal.value = data.total
    searchPage.value = nextPage
  } catch (_error) { void _error;
    if (requestSeq !== searchRequestSeq || searchQuery.value.trim() !== q) return
    if (append) {
      loadMoreError.value = t('teaching.hub.loadMoreFailed')
    } else {
      searchResults.value = []
      searchTotal.value = 0
      searchPage.value = 1
      errorMessage.value = t('common.loadFailed')
    }
  } finally {
    if (requestSeq === searchRequestSeq && searchQuery.value.trim() === q) {
      if (append) {
        searchLoadingMore.value = false
      } else {
        searching.value = false
      }
    }
  }
}

function loadMoreSearchResults() {
  if (!hasMoreSearchResults.value || searchLoadingMore.value || searching.value) return
  return doSearch(searchPage.value + 1, true)
}

function clearSearch() {
  searchQuery.value = ''
  if (searchTimer) clearTimeout(searchTimer)
  searchRequestSeq += 1
  resetSearchState()
  void router.replace({ path: route.path, query: teacherHubQueryWithSearch('') })
}

onMounted(() => {
  void loadPopularTeachers()
  if (searchQuery.value.trim()) {
    void doSearch()
  }
})

watch(
  () => route.query.q,
  () => {
    const q = readRouteSearchQuery()
    if (searchTimer) clearTimeout(searchTimer)
    searchRequestSeq += 1
    if (searchQuery.value !== q) {
      searchQuery.value = q
    }
    if (!q) {
      resetSearchState()
      return
    }
    void doSearch()
  },
)

onBeforeUnmount(() => {
  if (searchTimer) clearTimeout(searchTimer)
})

const displayTeachers = computed(() =>
  searchQuery.value.trim() ? searchResults.value : popularTeachers.value,
)
const hasMoreSearchResults = computed(() =>
  Boolean(searchQuery.value.trim()) && searchResults.value.length < searchTotal.value,
)

const showEmpty = computed(
  () => !loading.value && !searching.value && !errorMessage.value && displayTeachers.value.length === 0,
)
</script>

<template>
  <div class="min-h-screen bg-bg-base">
    <div class="animate-fade-in max-w-4xl mx-auto py-10 px-4 sm:px-6">
      <!-- Header -->
      <header class="text-center mb-10">
        <h1 class="text-3xl sm:text-4xl font-extrabold tracking-tight gradient-text">
          {{ t('teaching.hub.title') }}
        </h1>
        <p class="mt-2 text-base text-text-secondary">
          {{ t('teaching.hub.description') }}
        </p>
      </header>

      <!-- Search -->
      <div class="relative mb-10 max-w-2xl mx-auto">
        <Search
          class="absolute left-4 top-1/2 -translate-y-1/2 pointer-events-none text-text-muted"
          :size="20"
        />
        <input
          id="teacher-search-input"
          v-model="searchQuery"
          type="text"
          autocomplete="off"
          data-1p-ignore
          data-lpignore="true"
          data-form-type="other"
          :aria-label="t('teaching.hub.searchPlaceholder')"
          class="w-full pl-12 pr-12 py-4 text-base rounded-xl outline-none
                 bg-bg-glass-heavy backdrop-blur-sm shadow-card
                 text-text-primary placeholder:text-text-muted
                 transition-all duration-base ease-smooth
                 focus:shadow-glow-primary focus:ring-1 focus:ring-primary/30"
          :placeholder="t('teaching.hub.searchPlaceholder')"
          @input="handleSearchInput"
        />
        <button
          v-if="searchQuery"
          type="button"
          :aria-label="t('common.actions.clear')"
          :title="t('common.actions.clear')"
          class="absolute right-4 top-1/2 -translate-y-1/2 p-0 text-text-muted hover:text-text-primary transition-colors duration-fast"
          @click="clearSearch"
        >
          <X :size="18" />
        </button>
      </div>

      <!-- Section title -->
      <h2 v-if="!searchQuery.trim()" class="text-xl font-bold mb-6 gradient-text inline-block">
        {{ t('teaching.hub.popular') }}
      </h2>
      <h2 v-else class="text-xl font-bold mb-6 text-text-primary">
        {{ t('teaching.hub.searchResults') }}
      </h2>

      <!-- Loading -->
      <div v-if="loading || searching" class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
        <div
          v-for="i in 6"
          :key="i"
          class="h-[140px] rounded-xl bg-[length:200%_100%] animate-shimmer"
          :style="{ backgroundImage: 'linear-gradient(90deg, var(--color-bg-secondary) 25%, var(--color-bg-tertiary) 50%, var(--color-bg-secondary) 75%)' }"
        />
      </div>

      <!-- Error -->
      <div v-else-if="errorMessage" class="text-center py-12 text-text-muted">
        <User :size="48" class="mx-auto mb-4 opacity-30" />
        <p class="mb-4">{{ errorMessage }}</p>
        <button
          type="button"
          class="rounded-lg bg-primary px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-primary/90"
          @click="searchQuery.trim() ? doSearch() : loadPopularTeachers()"
        >
          {{ t('common.actions.retry') }}
        </button>
      </div>

      <!-- Teacher cards -->
      <template v-else-if="displayTeachers.length > 0">
        <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
          <router-link
            v-for="(teacher, idx) in displayTeachers"
            :key="teacher.teacherID"
            :to="`/teachers/${teacher.teacherID}`"
            class="glass-card shadow-card rounded-xl p-5 cursor-pointer hover-lift stagger-item no-underline block"
            :style="{ animationDelay: `${idx * 60}ms` }"
          >
            <div class="flex items-center gap-3 mb-3">
              <div class="p-[3px] bg-gradient-to-br from-primary to-accent rounded-full shrink-0">
                <div class="w-10 h-10 bg-bg-card rounded-full flex items-center justify-center">
                  <User :size="18" class="text-text-muted" />
                </div>
              </div>
              <div class="min-w-0">
                <h3 class="text-sm font-bold text-text-primary m-0 truncate">{{ teacher.teacherName }}</h3>
                <span v-if="teacher.departmentName" class="text-xs text-text-muted">{{ teacher.departmentName }}</span>
              </div>
            </div>
            <div class="flex items-center gap-4 text-xs text-text-secondary">
              <span v-if="teacher.avgRating" class="flex items-center gap-1 font-semibold text-primary">
                <EmojiRating :value="teacher.avgRating" size="sm" />
              </span>
              <span class="flex items-center gap-1">
                <BookOpen :size="12" />
                {{ teacher.reviewCount }} {{ t('teaching.hub.reviewUnit') }}
              </span>
            </div>
          </router-link>
        </div>

        <div
          v-if="hasMoreSearchResults || loadMoreError"
          class="mt-6 flex flex-col items-center gap-3"
        >
          <p
            v-if="loadMoreError"
            role="alert"
            class="rounded-lg border border-danger/30 bg-danger/10 px-3 py-2 text-sm text-danger"
          >
            {{ loadMoreError }}
          </p>
          <button
            v-if="hasMoreSearchResults"
            type="button"
            class="inline-flex h-11 items-center justify-center gap-2 rounded-lg border border-border-light bg-bg-card px-5 text-sm font-semibold text-text-secondary transition-colors hover:bg-bg-elevated hover:text-text-primary disabled:cursor-wait disabled:opacity-70"
            :disabled="searchLoadingMore"
            @click="loadMoreSearchResults"
          >
            <RefreshCw v-if="searchLoadingMore" :size="16" class="animate-spin" />
            <span>{{ searchLoadingMore ? t('common.actions.loading') : t('common.actions.loadMore') }}</span>
          </button>
        </div>
      </template>

      <!-- Empty -->
      <div v-else-if="showEmpty" class="text-center py-12 text-text-muted">
        <User :size="48" class="mx-auto mb-4 opacity-30" />
        <p>{{ searchQuery.trim() ? t('teaching.hub.noResults') : t('teaching.hub.noTeachers') }}</p>
      </div>
    </div>
  </div>
</template>
