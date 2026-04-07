<template>
  <div class="relative w-[600px] shrink-0 max-tablet:w-full">
    <div
      class="flex items-center gap-2 px-3 py-2 rounded-full bg-bg-secondary transition-all duration-fast ease-smooth"
      :class="isFocused && 'border-primary shadow-glow-sm'"
    >
      <Search :size="16" class="text-text-muted shrink-0" />
      <input
        ref="inputRef"
        v-model="query"
        type="text"
        class="flex-1 bg-transparent border-none outline-none text-sm text-text-primary placeholder:text-text-muted min-w-0"
        :placeholder="t('review.topBar.searchPlaceholder')"
        :aria-label="t('review.topBar.searchPlaceholder')"
        aria-autocomplete="list"
        role="combobox"
        :aria-expanded="showDropdown"
        aria-haspopup="listbox"
        aria-controls="inline-search-listbox"
        :aria-activedescendant="activeIndex >= 0 ? `search-item-${activeIndex}` : undefined"
        @focus="handleFocus"
        @blur="handleBlur"
        @keydown="handleKeydown"
      />
      <kbd class="hidden lg:inline font-sans text-xs py-px px-1.5 bg-bg-tertiary rounded text-text-muted">{{ isMac ? '⌘' : 'Ctrl' }}K</kbd>
    </div>

    <!-- 下拉面板 -->
    <div
      v-if="showDropdown"
      id="inline-search-listbox"
      role="listbox"
      class="absolute top-full left-0 right-0 mt-1 bg-bg-glass-heavy backdrop-blur-xl backdrop-saturate-150 border border-white/15 dark:border-white/8 rounded-xl shadow-lg z-50 max-h-[400px] overflow-y-auto"
    >
      <!-- 搜索结果 -->
      <template v-if="query.trim()">
        <div v-if="isLoading" class="flex items-center justify-center py-6" aria-busy="true" :aria-label="t('common.actions.loading')">
          <div class="w-5 h-5 border-2 border-border border-t-primary rounded-full animate-spin" aria-hidden="true" />
        </div>
        <div v-else-if="results.length === 0" class="py-6 text-center text-sm text-text-muted">
          {{ t('review.chart.emptyTitle') }}
        </div>
        <template v-else>
          <button
            v-for="(course, idx) in results"
            :id="`search-item-${idx}`"
            :key="course.id"
            role="option"
            :aria-selected="idx === activeIndex"
            class="w-full flex items-center gap-2 px-4 py-2.5 text-left cursor-pointer transition-colors duration-fast hover:bg-bg-hover"
            :class="idx === activeIndex && 'bg-bg-hover'"
            @mouseenter="activeIndex = idx"
            @click="selectCourse(course)"
          >
            <span class="text-sm font-medium text-text-primary truncate">{{ course.name }}</span>
            <span class="shrink-0 text-xs text-text-muted"><template v-if="course.credits">{{ t('review.course.creditsBadge', { n: course.credits }) }} · </template>{{ course.departmentName }}</span>
            <span class="shrink-0 text-xs tabular-nums text-text-muted ml-auto">{{ t('review.course.reviewCountBadge', { count: course.reviewCount }) }}</span>
          </button>
        </template>
      </template>

      <!-- 最近搜索 -->
      <template v-else-if="recentSearches.length > 0">
        <div class="px-4 py-2 text-xs text-text-muted font-medium">{{ t('review.topBar.recentSearches') }}</div>
        <button
          v-for="(item, idx) in recentSearches"
          :id="`search-item-${idx}`"
          :key="item.id"
          role="option"
          :aria-selected="idx === activeIndex"
          class="w-full flex items-center gap-2 px-4 py-2.5 text-left cursor-pointer transition-colors duration-fast hover:bg-bg-hover"
          :class="idx === activeIndex && 'bg-bg-hover'"
          @mouseenter="activeIndex = idx"
          @click="selectRecent(item)"
        >
          <span class="text-sm font-medium text-text-primary truncate">{{ item.name }}</span>
          <span class="shrink-0 text-xs text-text-muted"><template v-if="item.credits">{{ t('review.course.creditsBadge', { n: item.credits }) }} · </template>{{ item.departmentName }}</span>
          <span class="shrink-0 text-xs tabular-nums text-text-muted ml-auto">{{ t('review.course.reviewCountBadge', { count: item.reviewCount }) }}</span>
        </button>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, onMounted, onUnmounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { Search } from 'lucide-vue-next'
import { api } from '@/api'
import type { Course } from '@/types/course'

interface RecentItem {
  id: number
  name: string
  credits: number
  departmentName: string
  reviewCount: number
}

const RECENT_KEY = 'search-recent'
const MAX_RECENT = 5

const { t } = useI18n()
const router = useRouter()

const inputRef = ref<HTMLInputElement>()
const query = ref('')
const results = ref<Course[]>([])
const isLoading = ref(false)
const isFocused = ref(false)
const activeIndex = ref(-1)
const recentSearches = ref<RecentItem[]>(loadRecent())

const isMac = /mac/i.test(navigator.userAgent)

// 下拉面板显示条件：聚焦 && (有输入 || 有最近搜索)
const showDropdown = computed(() =>
  isFocused.value && (query.value.trim().length > 0 || recentSearches.value.length > 0)
)

// 防抖搜索
let debounceTimer: ReturnType<typeof setTimeout> | undefined
// AbortController 取消前一次搜索请求
let searchController: AbortController | undefined

watch(query, (val) => {
  activeIndex.value = -1
  if (debounceTimer) clearTimeout(debounceTimer)

  const trimmed = val.trim()
  if (!trimmed) {
    if (searchController) { searchController.abort(); searchController = undefined }
    results.value = []
    isLoading.value = false
    return
  }

  isLoading.value = true
  debounceTimer = setTimeout(async () => {
    // 取消前一次仍在进行的请求
    if (searchController) searchController.abort()
    const controller = new AbortController()
    searchController = controller

    try {
      const res = await api.course.searchCourses(trimmed, 10, { signal: controller.signal })
      // 请求已被取消则忽略结果
      if (controller.signal.aborted) return
      const list = res.data?.data?.list || []
      // 按 ID 去重，防止后端返回重复课程
      const seen = new Set<number>()
      results.value = list.filter((c: Course) => {
        if (seen.has(c.id)) return false
        seen.add(c.id)
        return true
      })
    } catch (err) {
      // 被取消的请求不更新状态（API 客户端会将 AbortError 包装为 ApiError）
      if (controller.signal.aborted) return
      if (import.meta.env.DEV) { console.warn('[InlineSearch] Search failed:', err) }
      if (query.value.trim() === trimmed) {
        results.value = []
      }
    } finally {
      if (!controller.signal.aborted && query.value.trim() === trimmed) {
        isLoading.value = false
      }
    }
  }, 300)
})

// blur 延迟关闭使用 undefined 初始值，统一清理
let blurTimer: ReturnType<typeof setTimeout> | undefined

function handleFocus() {
  if (blurTimer !== undefined) { clearTimeout(blurTimer); blurTimer = undefined }
  isFocused.value = true
}

function handleBlur() {
  // 延迟关闭，让 click 事件先触发；handleFocus 会取消此定时器
  blurTimer = setTimeout(() => {
    blurTimer = undefined
    isFocused.value = false
  }, 150)
}

function handleKeydown(e: KeyboardEvent) {
  const itemCount = query.value.trim() ? results.value.length : recentSearches.value.length

  if (e.key === 'ArrowDown') {
    e.preventDefault()
    activeIndex.value = activeIndex.value < itemCount - 1 ? activeIndex.value + 1 : 0
  } else if (e.key === 'ArrowUp') {
    e.preventDefault()
    activeIndex.value = activeIndex.value > 0 ? activeIndex.value - 1 : itemCount - 1
  } else if (e.key === 'Enter') {
    e.preventDefault()
    if (query.value.trim() && activeIndex.value >= 0 && activeIndex.value < results.value.length) {
      selectCourse(results.value[activeIndex.value])
    } else if (!query.value.trim() && activeIndex.value >= 0 && activeIndex.value < recentSearches.value.length) {
      selectRecent(recentSearches.value[activeIndex.value])
    }
  } else if (e.key === 'Escape') {
    e.preventDefault()
    inputRef.value?.blur()
  }
}

function selectCourse(course: Course) {
  saveRecent({
    id: course.id,
    name: course.name,
    credits: course.credits,
    departmentName: course.departmentName || '',
    reviewCount: course.reviewCount,
  })
  query.value = ''
  isFocused.value = false
  router.push(`/courses/${course.id}`)
}

function selectRecent(item: RecentItem) {
  query.value = ''
  isFocused.value = false
  router.push(`/courses/${item.id}`)
}

// 最近搜索持久化
function loadRecent(): RecentItem[] {
  try {
    const raw = localStorage.getItem(RECENT_KEY)
    if (!raw) return []
    const parsed: unknown = JSON.parse(raw)
    if (!Array.isArray(parsed)) return []
    return parsed.filter((item): item is RecentItem =>
      typeof item === 'object' && item !== null && typeof item.name === 'string' && typeof item.id === 'number'
    ).slice(0, MAX_RECENT)
  } catch {
    return []
  }
}

function saveRecent(item: RecentItem) {
  const updated = [item, ...recentSearches.value.filter(s => s.id !== item.id)].slice(0, MAX_RECENT)
  recentSearches.value = updated
  try {
    localStorage.setItem(RECENT_KEY, JSON.stringify(updated))
  } catch {
    // localStorage 不可用时静默忽略
  }
}

// Cmd+K / Ctrl+K 全局快捷键聚焦搜索框
function handleGlobalKeydown(e: KeyboardEvent) {
  // Cmd+K / Ctrl+K works from any context (including editable fields)
  if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
    e.preventDefault()
    inputRef.value?.focus()
    return
  }

  const target = e.target as HTMLElement | null
  const isEditable = target instanceof HTMLInputElement
    || target instanceof HTMLTextAreaElement
    || target?.isContentEditable === true

  if (isEditable) return

  if (!e.metaKey && !e.ctrlKey && e.key === '/') {
    e.preventDefault()
    inputRef.value?.focus()
  }
}

defineExpose({ inputRef })

onMounted(() => {
  document.addEventListener('keydown', handleGlobalKeydown)
})

onUnmounted(() => {
  document.removeEventListener('keydown', handleGlobalKeydown)
  if (debounceTimer) clearTimeout(debounceTimer)
  if (blurTimer) clearTimeout(blurTimer)
  if (searchController) searchController.abort()
})
</script>
