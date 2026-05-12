<script setup lang="ts">
import { ref, computed, onMounted, onBeforeUnmount } from 'vue'
import { useI18n } from 'vue-i18n'
import { Search, X, User, BookOpen } from 'lucide-vue-next'
import { api } from '@/api'
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

const searchQuery = ref('')
const searchResults = ref<TeacherEntry[]>([])
const searchTotal = ref(0)
const searching = ref(false)
const loading = ref(true)
const popularTeachers = ref<TeacherEntry[]>([])

let searchTimer: ReturnType<typeof setTimeout> | undefined

async function loadPopularTeachers() {
  loading.value = true
  try {
    const res = await api.rating.listHotTeachers(20)
    const list = res.data?.data?.list
    popularTeachers.value = (list ?? []).map(mapTeacher)
  } finally {
    loading.value = false
  }
}

function mapTeacher(raw: { teacherID: number; teacherName: string; departmentName?: string; avgRating?: number | null; reviewCount: number; courseCount: number }): TeacherEntry {
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
  if (!searchQuery.value.trim()) {
    searchResults.value = []
    searchTotal.value = 0
    return
  }
  searchTimer = setTimeout(() => doSearch(), 350)
}

async function doSearch() {
  const q = searchQuery.value.trim()
  if (!q) return

  searching.value = true
  try {
    const res = await api.rating.listTeachers({ q, sort: 'reviews', pageSize: 30 })
    const data = res.data?.data
    searchResults.value = (data?.list ?? []).map(mapTeacher)
    searchTotal.value = data?.total ?? 0
  } catch (_error) { void _error;
    searchResults.value = []
    searchTotal.value = 0
  } finally {
    searching.value = false
  }
}

function clearSearch() {
  searchQuery.value = ''
  searchResults.value = []
  searchTotal.value = 0
}

onMounted(loadPopularTeachers)

onBeforeUnmount(() => {
  if (searchTimer) clearTimeout(searchTimer)
})

const displayTeachers = computed(() =>
  searchQuery.value.trim() ? searchResults.value : popularTeachers.value,
)

const showEmpty = computed(
  () => !loading.value && !searching.value && displayTeachers.value.length === 0,
)
</script>

<template>
  <div class="min-h-screen bg-bg-base">
    <div class="animate-fade-in max-w-4xl mx-auto py-10 px-4 sm:px-6">
      <!-- Header -->
      <header class="text-center mb-10">
        <h1 class="text-3xl sm:text-4xl font-extrabold tracking-tight gradient-text">
          {{ t('teaching.title') }}
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

      <!-- Teacher cards -->
      <div v-else-if="displayTeachers.length > 0" class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
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

      <!-- Empty -->
      <div v-else-if="showEmpty" class="text-center py-12 text-text-muted">
        <User :size="48" class="mx-auto mb-4 opacity-30" />
        <p>{{ searchQuery.trim() ? t('teaching.hub.noResults') : t('teaching.hub.noTeachers') }}</p>
      </div>
    </div>
  </div>
</template>
