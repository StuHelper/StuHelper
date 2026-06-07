<template>
  <Teleport to="body">
    <Transition name="overlay">
      <div
        v-if="isOpen"
        class="palette-overlay fixed inset-0 bg-bg-overlay z-[var(--z-modal-backdrop)] flex items-start justify-center pt-[15vh] max-md:pt-4 max-md:px-4"
        @click.self="close"
      >
        <div
          ref="modalRef"
          class="w-full max-w-[640px] bg-bg-card rounded-xl shadow-xl overflow-hidden animate-modal-in max-md:rounded-lg glow-border"
          role="dialog"
          aria-modal="true"
          :aria-label="t('nav.searchCoursePlaceholder')"
          @keydown="trapFocus"
        >
          <div class="flex items-center gap-3 px-5 py-4 border-b border-border-light">
            <Search class="text-text-muted shrink-0" :size="20" />
            <input
              ref="inputRef"
              v-model="searchQuery"
              class="flex-1 border-none outline-none bg-transparent text-lg font-sans text-text-primary placeholder:text-text-muted"
              :placeholder="t('nav.searchCoursePlaceholder')"
              role="combobox"
              :aria-expanded="results.length > 0 ? 'true' : 'false'"
              aria-haspopup="listbox"
              :aria-controls="results.length > 0 ? 'palette-listbox' : undefined"
              :aria-activedescendant="results.length > 0 ? `palette-option-${activeIndex}` : undefined"
              @keydown.down.prevent="moveDown"
              @keydown.up.prevent="moveUp"
              @keydown.enter.prevent="selectCurrent"
              @keydown.esc="close"
            />
            <kbd class="font-sans text-xs py-px px-2 bg-bg-tertiary rounded text-text-muted shrink-0">ESC</kbd>
          </div>

          <div class="max-h-[400px] overflow-y-auto py-2">
            <div v-if="loading" class="flex justify-center p-8">
              <div class="w-6 h-6 border-2 border-border border-t-primary rounded-full animate-spin" />
            </div>

            <template v-else-if="searchQuery.trim()">
              <div v-if="searchError" role="alert" class="p-8 text-center text-danger text-sm">
                {{ searchError }}
              </div>
              <div v-else-if="results.length === 0" class="p-8 text-center text-text-muted text-sm">
                {{ t('common.empty.result') }}
              </div>
              <div v-else class="py-1">
                <div class="py-2 px-5 text-xs font-medium text-text-muted uppercase tracking-wide">{{ t('nav.searchResults') }}</div>
                <div role="listbox" id="palette-listbox">
                  <button
                    v-for="(item, idx) in results"
                    :key="`${item.type}-${item.id}`"
                    :id="`palette-option-${idx}`"
                    class="flex items-center gap-3 w-full py-2.5 px-5 text-left text-text-primary text-sm cursor-pointer transition-colors duration-fast ease-smooth hover:bg-bg-hover"
                    :class="{ '!bg-bg-hover': activeIndex === idx }"
                    role="option"
                    :aria-selected="activeIndex === idx"
                    @click="selectItem(item)"
                    @mouseenter="activeIndex = idx"
                  >
                    <div class="flex-1 flex items-center gap-2 min-w-0">
                      <span class="font-medium whitespace-nowrap overflow-hidden text-ellipsis">{{ item.name }}</span>
                      <span v-if="item.detail" class="text-xs text-text-muted whitespace-nowrap overflow-hidden text-ellipsis">{{ item.detail }}</span>
                    </div>
                    <span class="text-xs text-text-muted py-px px-2 bg-bg-secondary rounded-full whitespace-nowrap shrink-0">
                      {{ item.badge }}
                    </span>
                  </button>
                </div>
              </div>
            </template>

            <template v-else>
              <div v-if="recentSearches.length > 0" class="py-1">
                <div class="py-2 px-5 text-xs font-medium text-text-muted uppercase tracking-wide">{{ t('nav.recentSearches') }}</div>
                <button
                  v-for="(term, idx) in recentSearches"
                  :key="term"
                  class="flex items-center gap-3 w-full py-2.5 px-5 text-left text-text-primary text-sm cursor-pointer transition-colors duration-fast ease-smooth hover:bg-bg-hover"
                  :class="{ '!bg-bg-hover': activeIndex === idx }"
                  @click="searchQuery = term"
                  @mouseenter="activeIndex = idx"
                >
                  <Clock :size="14" />
                  <span class="font-medium whitespace-nowrap overflow-hidden text-ellipsis">{{ term }}</span>
                </button>
              </div>
            </template>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { ref, watch, nextTick, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { Search, Clock } from 'lucide-vue-next'
import { useCommandPalette } from '@/composables/useCommandPalette'
import { api } from '@/api'
import {
  readCourseListPayload,
  readTeacherSummaryPagePayload,
  type TeacherSummaryPayload,
} from '@/modules/course/coursePayload'
import { safeGetLocalStorageItem, safeSetLocalStorageItem } from '@/utils/browserStorage'
import type { Course } from '@stuhelper/shared/course'

const { t } = useI18n()
const router = useRouter()
const { isOpen, searchQuery, close } = useCommandPalette()

const inputRef = ref<HTMLInputElement | null>(null)
const modalRef = ref<HTMLElement | null>(null)

type PaletteResult =
  | {
      type: 'course'
      id: number
      name: string
      detail: string
      badge: string
    }
  | {
      type: 'teacher'
      id: number
      name: string
      detail: string
      badge: string
    }

const results = ref<PaletteResult[]>([])
const loading = ref(false)
const searchError = ref('')
const activeIndex = ref(0)

// 保存打开前的 body overflow 原始值，关闭时恢复而非无条件置空
let savedBodyOverflow = ''

const RECENT_KEY = 'recent-searches'
// 最大搜索历史条数常量化
const MAX_RECENT = 5
const recentSearches = ref<string[]>((() => {
  try {
    const parsed = JSON.parse(safeGetLocalStorageItem(RECENT_KEY) || '[]')
    return Array.isArray(parsed) ? parsed.filter((s): s is string => typeof s === 'string').slice(0, MAX_RECENT) : []
  } catch (_error) { void _error;
    return []
  }
})())

function saveRecent(term: string) {
  const trimmed = term.trim()
  if (!trimmed) return
  const list = recentSearches.value.filter((s) => s !== trimmed)
  list.unshift(trimmed)
  recentSearches.value = list.slice(0, MAX_RECENT)
  safeSetLocalStorageItem(RECENT_KEY, JSON.stringify(recentSearches.value))
}

let searchTimer: ReturnType<typeof setTimeout> | null = null
let searchController: AbortController | undefined

function courseResult(course: Course): PaletteResult {
  return {
    type: 'course',
    id: course.id,
    name: course.name,
    detail: [course.code, course.departmentName].filter(Boolean).join(' · '),
    badge: t('nav.courses'),
  }
}

function teacherResult(teacher: TeacherSummaryPayload): PaletteResult {
  return {
    type: 'teacher',
    id: teacher.teacherID,
    name: teacher.teacherName,
    detail: teacher.departmentName ?? '',
    badge: t('nav.teacher'),
  }
}

// 焦点陷阱：Tab/Shift+Tab 在对话框内循环
function trapFocus(e: KeyboardEvent) {
  if (e.key !== 'Tab' || !modalRef.value) return
  const focusable = modalRef.value.querySelectorAll<HTMLElement>(
    'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])'
  )
  if (focusable.length === 0) return
  const first = focusable[0]
  const last = focusable[focusable.length - 1]
  if (e.shiftKey) {
    if (document.activeElement === first) { e.preventDefault(); last.focus() }
  } else {
    if (document.activeElement === last) { e.preventDefault(); first.focus() }
  }
}

watch(searchQuery, (val) => {
  activeIndex.value = 0
  if (searchTimer) clearTimeout(searchTimer)

  const q = val.trim()
  if (!q) {
    if (searchController) { searchController.abort(); searchController = undefined }
    results.value = []
    searchError.value = ''
    loading.value = false
    return
  }

  loading.value = true
  searchError.value = ''
  searchTimer = setTimeout(async () => {
    if (searchController) searchController.abort()
    const controller = new AbortController()
    searchController = controller

    const [courseResponse, teacherResponse] = await Promise.allSettled([
      api.course.searchCourses(q, 10, { signal: controller.signal }),
      api.rating.listTeachers(
        { q, sort: 'reviews', pageSize: 10 },
        { signal: controller.signal },
      ),
    ])

    if (controller.signal.aborted) return

    let failed = false
    const nextResults: PaletteResult[] = []

    if (courseResponse.status === 'fulfilled') {
      try {
        const list = readCourseListPayload(
          courseResponse.value.data?.data,
          'Invalid course search response',
        )
        const seen = new Set<number>()
        nextResults.push(
          ...list
            .filter((c: Course) => {
              if (seen.has(c.id)) return false
              seen.add(c.id)
              return true
            })
            .map(courseResult),
        )
      } catch (_error) { void _error;
        failed = true
      }
    } else {
      failed = !controller.signal.aborted
    }

    if (teacherResponse.status === 'fulfilled') {
      try {
        const page = readTeacherSummaryPagePayload(
          teacherResponse.value.data?.data,
          'Invalid teacher search response',
        )
        const seen = new Set<number>()
        nextResults.push(
          ...page.list
            .filter((teacher) => {
              if (seen.has(teacher.teacherID)) return false
              seen.add(teacher.teacherID)
              return true
            })
            .map(teacherResult),
        )
      } catch (_error) { void _error;
        failed = true
      }
    } else {
      failed = !controller.signal.aborted
    }

    if (searchQuery.value.trim() === q) {
      results.value = nextResults
      searchError.value = failed && nextResults.length === 0
        ? t('common.loadFailed')
        : ''
    }

    if (!controller.signal.aborted && searchQuery.value.trim() === q) {
      loading.value = false
    }
  }, 300)
})

onUnmounted(() => {
  // 清理 timer 后置空
  if (searchTimer) { clearTimeout(searchTimer); searchTimer = null }
  if (searchController) { searchController.abort(); searchController = undefined }
  // 组件卸载时恢复 body overflow，防止路由切换后滚动被锁定
  if (isOpen.value) {
    document.body.style.overflow = savedBodyOverflow
    isOpen.value = false
  }
})

watch(isOpen, (val) => {
  if (val) {
    // 保存当前 overflow 值，避免关闭时覆盖其他模态框的滚动锁定
    savedBodyOverflow = document.body.style.overflow
    document.body.style.overflow = 'hidden'
    nextTick(() => inputRef.value?.focus())
  } else {
    document.body.style.overflow = savedBodyOverflow
    // 关闭时清理搜索 timer 和 inflight 请求
    if (searchTimer) { clearTimeout(searchTimer); searchTimer = null }
    if (searchController) { searchController.abort(); searchController = undefined }
    searchQuery.value = ''
    results.value = []
    searchError.value = ''
    activeIndex.value = 0
  }
})

function moveDown() {
  const max = searchQuery.value.trim()
    ? results.value.length
    : recentSearches.value.length
  if (activeIndex.value < max - 1) activeIndex.value++
}

function moveUp() {
  if (activeIndex.value > 0) activeIndex.value--
}

function selectCurrent() {
  if (searchQuery.value.trim() && results.value[activeIndex.value]) {
    selectItem(results.value[activeIndex.value])
  } else if (!searchQuery.value.trim() && recentSearches.value[activeIndex.value]) {
    // 最近搜索列表中按 Enter，填充搜索词触发搜索
    searchQuery.value = recentSearches.value[activeIndex.value]
  }
}

function selectItem(item: PaletteResult) {
  saveRecent(item.name)
  close()
  router.push(item.type === 'teacher' ? `/teachers/${item.id}` : `/courses/${item.id}`)
}
</script>

<style scoped>
.overlay-enter-active {
  animation: overlayIn var(--duration-base) var(--ease-out);
}

.overlay-leave-active {
  animation: overlayIn var(--duration-fast) var(--ease-out) reverse;
}
</style>
