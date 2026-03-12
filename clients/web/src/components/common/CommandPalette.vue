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
          class="w-full max-w-[640px] bg-bg-card border border-border rounded-xl shadow-xl overflow-hidden animate-modal-in max-md:rounded-lg"
          role="dialog"
          aria-modal="true"
          @keydown="trapFocus"
        >
          <div class="flex items-center gap-3 px-5 py-4 border-b border-border">
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
              <div v-if="results.length === 0" class="p-8 text-center text-text-muted text-sm">
                {{ t('common.empty.result') }}
              </div>
              <div v-else class="py-1">
                <div class="py-2 px-5 text-xs font-medium text-text-muted uppercase tracking-wide">{{ t('nav.searchResults') }}</div>
                <div role="listbox" id="palette-listbox">
                  <button
                    v-for="(item, idx) in results"
                    :key="item.id"
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
                  </div>
                  <span v-if="item.departmentName" class="text-xs text-text-muted py-px px-2 bg-bg-secondary rounded-full whitespace-nowrap shrink-0">
                    {{ item.departmentName }}
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
import type { Course } from '@/types/course'

const { t } = useI18n()
const router = useRouter()
const { isOpen, searchQuery, close } = useCommandPalette()

const inputRef = ref<HTMLInputElement | null>(null)
const modalRef = ref<HTMLElement | null>(null)
const results = ref<Course[]>([])
const loading = ref(false)
const activeIndex = ref(0)

// M-19: 保存打开前的 body overflow 原始值，关闭时恢复而非无条件置空
let savedBodyOverflow = ''

const RECENT_KEY = 'recent-searches'
// M-104: 最大搜索历史条数常量化
const MAX_RECENT = 5
const recentSearches = ref<string[]>((() => {
  try {
    const parsed = JSON.parse(localStorage.getItem(RECENT_KEY) || '[]')
    return Array.isArray(parsed) ? parsed.filter((s): s is string => typeof s === 'string').slice(0, MAX_RECENT) : []
  } catch {
    return []
  }
})())

function saveRecent(term: string) {
  const trimmed = term.trim()
  if (!trimmed) return
  const list = recentSearches.value.filter((s) => s !== trimmed)
  list.unshift(trimmed)
  recentSearches.value = list.slice(0, MAX_RECENT)
  try {
    localStorage.setItem(RECENT_KEY, JSON.stringify(recentSearches.value))
  } catch {
    // M-31: localStorage 不可用时静默忽略
  }
}

let searchTimer: ReturnType<typeof setTimeout> | null = null
let searchController: AbortController | undefined

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
    loading.value = false
    return
  }

  loading.value = true
  searchTimer = setTimeout(async () => {
    if (searchController) searchController.abort()
    const controller = new AbortController()
    searchController = controller

    try {
      const res = await api.course.searchCourses(q, 10, { signal: controller.signal })
      if (controller.signal.aborted) return
      // M-93: 按 ID 去重，防止后端返回重复课程
      const list = res.data?.data?.list || []
      const seen = new Set<number>()
      results.value = list.filter((c: Course) => {
        if (seen.has(c.id)) return false
        seen.add(c.id)
        return true
      })
    } catch {
      if (controller.signal.aborted) return
      if (searchQuery.value.trim() === q) {
        results.value = []
      }
    } finally {
      if (!controller.signal.aborted && searchQuery.value.trim() === q) {
        loading.value = false
      }
    }
  }, 300)
})

onUnmounted(() => {
  // L-50: 清理 timer 后置空
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
    // M-19: 保存当前 overflow 值，避免关闭时覆盖其他模态框的滚动锁定
    savedBodyOverflow = document.body.style.overflow
    document.body.style.overflow = 'hidden'
    nextTick(() => inputRef.value?.focus())
  } else {
    document.body.style.overflow = savedBodyOverflow
    // M-119: 关闭时清理搜索 timer 和 inflight 请求
    if (searchTimer) { clearTimeout(searchTimer); searchTimer = null }
    if (searchController) { searchController.abort(); searchController = undefined }
    searchQuery.value = ''
    results.value = []
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

function selectItem(course: Course) {
  saveRecent(course.name)
  close()
  router.push(`/courses/${course.id}`)
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
