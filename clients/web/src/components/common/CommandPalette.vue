<template>
  <Teleport to="body">
    <Transition name="overlay">
      <div
        v-if="isOpen"
        class="palette-overlay fixed inset-0 bg-bg-overlay z-50 flex items-start justify-center pt-[15vh] max-md:pt-4 max-md:px-4"
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
import { searchCourses } from '@/api/course'
import type { Course } from '@/types/course'

const { t } = useI18n()
const router = useRouter()
const { isOpen, searchQuery, close } = useCommandPalette()

const inputRef = ref<HTMLInputElement | null>(null)
const modalRef = ref<HTMLElement | null>(null)
const results = ref<Course[]>([])
const loading = ref(false)
const activeIndex = ref(0)

const RECENT_KEY = 'recent-searches'
const recentSearches = ref<string[]>((() => {
  try {
    return JSON.parse(localStorage.getItem(RECENT_KEY) || '[]')
  } catch {
    return []
  }
})())

function saveRecent(term: string) {
  const list = recentSearches.value.filter((s) => s !== term)
  list.unshift(term)
  recentSearches.value = list.slice(0, 5)
  localStorage.setItem(RECENT_KEY, JSON.stringify(recentSearches.value))
}

let searchTimer: ReturnType<typeof setTimeout> | null = null

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
    results.value = []
    loading.value = false
    return
  }

  loading.value = true
  const currentQuery = q
  searchTimer = setTimeout(async () => {
    try {
      const res = await searchCourses(currentQuery, 10)
      if (searchQuery.value.trim() === currentQuery) {
        results.value = res.data?.list || []
      }
    } catch {
      if (searchQuery.value.trim() === currentQuery) {
        results.value = []
      }
    } finally {
      if (searchQuery.value.trim() === currentQuery) {
        loading.value = false
      }
    }
  }, 300)
})

onUnmounted(() => {
  if (searchTimer) clearTimeout(searchTimer)
})

watch(isOpen, (val) => {
  if (val) {
    document.body.style.overflow = 'hidden'
    nextTick(() => inputRef.value?.focus())
  } else {
    document.body.style.overflow = ''
    if (searchTimer) clearTimeout(searchTimer)
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
  }
}

function selectItem(course: Course) {
  saveRecent(course.name)
  close()
  router.push(`/review/courses/${course.id}`)
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
