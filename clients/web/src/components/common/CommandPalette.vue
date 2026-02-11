<template>
  <Teleport to="body">
    <Transition name="overlay">
      <div v-if="isOpen" class="palette-overlay" @click.self="close">
        <div class="palette-modal animate-modal-in">
          <div class="palette-input-wrap">
            <svg class="palette-icon" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <circle cx="11" cy="11" r="8" />
              <path d="m21 21-4.3-4.3" />
            </svg>
            <input
              ref="inputRef"
              v-model="searchQuery"
              class="palette-input"
              :placeholder="t('nav.searchCoursePlaceholder')"
              @keydown.down.prevent="moveDown"
              @keydown.up.prevent="moveUp"
              @keydown.enter.prevent="selectCurrent"
            />
            <kbd class="palette-esc">ESC</kbd>
          </div>

          <div class="palette-body">
            <div v-if="loading" class="palette-loading">
              <div class="spinner" />
            </div>

            <template v-else-if="searchQuery.trim()">
              <div v-if="results.length === 0" class="palette-empty">
                {{ t('common.empty.result') }}
              </div>
              <div v-else class="palette-section">
                <div class="palette-section-title">{{ t('nav.searchResults') }}</div>
                <button
                  v-for="(item, idx) in results"
                  :key="item.id"
                  class="palette-item"
                  :class="{ active: activeIndex === idx }"
                  @click="selectItem(item)"
                  @mouseenter="activeIndex = idx"
                >
                  <div class="palette-item-info">
                    <span class="palette-item-name">{{ item.name }}</span>
                  </div>
                  <span v-if="item.departmentName" class="palette-item-dept">
                    {{ item.departmentName }}
                  </span>
                </button>
              </div>
            </template>

            <template v-else>
              <div v-if="recentSearches.length > 0" class="palette-section">
                <div class="palette-section-title">{{ t('nav.recentSearches') }}</div>
                <button
                  v-for="(term, idx) in recentSearches"
                  :key="term"
                  class="palette-item"
                  :class="{ active: activeIndex === idx }"
                  @click="searchQuery = term"
                  @mouseenter="activeIndex = idx"
                >
                  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <circle cx="12" cy="12" r="10" />
                    <polyline points="12,6 12,12 16,14" />
                  </svg>
                  <span class="palette-item-name">{{ term }}</span>
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
import { useCommandPalette } from '@/composables/useCommandPalette'
import { searchCourses } from '@/api/course'
import type { Course } from '@/types/course'

const { t } = useI18n()
const router = useRouter()
const { isOpen, searchQuery, close } = useCommandPalette()

const inputRef = ref<HTMLInputElement | null>(null)
const results = ref<Course[]>([])
const loading = ref(false)
const activeIndex = ref(0)

const RECENT_KEY = 'recent-searches'
const recentSearches = ref<string[]>(
  JSON.parse(localStorage.getItem(RECENT_KEY) || '[]')
)

function saveRecent(term: string) {
  const list = recentSearches.value.filter((s) => s !== term)
  list.unshift(term)
  recentSearches.value = list.slice(0, 5)
  localStorage.setItem(RECENT_KEY, JSON.stringify(recentSearches.value))
}

let searchTimer: ReturnType<typeof setTimeout> | null = null

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
  searchTimer = setTimeout(async () => {
    try {
      const res = await searchCourses(q, 10)
      results.value = res.data || []
    } catch {
      results.value = []
    } finally {
      loading.value = false
    }
  }, 300)
})

onUnmounted(() => {
  if (searchTimer) clearTimeout(searchTimer)
})

watch(isOpen, (val) => {
  if (val) {
    nextTick(() => inputRef.value?.focus())
  } else {
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
.palette-overlay {
  position: fixed;
  inset: 0;
  background: var(--bg-overlay);
  z-index: var(--z-modal);
  display: flex;
  align-items: flex-start;
  justify-content: center;
  padding-top: 15vh;
}

.palette-modal {
  width: 100%;
  max-width: 640px;
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius-xl);
  box-shadow: var(--shadow-xl);
  overflow: hidden;
}

.palette-input-wrap {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  padding: var(--space-4) var(--space-5);
  border-bottom: 1px solid var(--border);
}

.palette-icon {
  color: var(--text-muted);
  flex-shrink: 0;
}

.palette-input {
  flex: 1;
  border: none;
  outline: none;
  background: none;
  font-size: var(--text-lg);
  font-family: var(--font-sans);
  color: var(--text-primary);
}

.palette-input::placeholder {
  color: var(--text-muted);
}

.palette-esc {
  font-family: var(--font-sans);
  font-size: var(--text-xs);
  padding: 2px 8px;
  background: var(--bg-tertiary);
  border-radius: var(--radius-xs);
  color: var(--text-muted);
  border: none;
  flex-shrink: 0;
}

.palette-body {
  max-height: 400px;
  overflow-y: auto;
  padding: var(--space-2) 0;
}

.palette-loading {
  display: flex;
  justify-content: center;
  padding: var(--space-8);
}

.spinner {
  width: 24px;
  height: 24px;
  border: 2px solid var(--border);
  border-top-color: var(--brand-primary);
  border-radius: var(--radius-full);
  animation: spin 0.6s linear infinite;
}

.palette-empty {
  padding: var(--space-8);
  text-align: center;
  color: var(--text-muted);
  font-size: var(--text-sm);
}

.palette-section {
  padding: var(--space-1) 0;
}

.palette-section-title {
  padding: var(--space-2) var(--space-5);
  font-size: var(--text-xs);
  font-weight: var(--weight-medium);
  color: var(--text-muted);
  text-transform: uppercase;
  letter-spacing: var(--tracking-wide);
}

.palette-item {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  width: 100%;
  padding: var(--space-2-5) var(--space-5);
  text-align: left;
  color: var(--text-primary);
  font-size: var(--text-sm);
  transition: background var(--duration-fast) var(--ease-smooth);
  cursor: pointer;
}

/* space-2-5 不存在，用 padding 值代替 */
.palette-item {
  padding: 10px var(--space-5);
}

.palette-item.active,
.palette-item:hover {
  background: var(--bg-hover);
}

.palette-item-info {
  flex: 1;
  display: flex;
  align-items: center;
  gap: var(--space-2);
  min-width: 0;
}

.palette-item-name {
  font-weight: var(--weight-medium);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.palette-item-meta {
  color: var(--text-muted);
  font-size: var(--text-xs);
  white-space: nowrap;
}

.palette-item-dept {
  font-size: var(--text-xs);
  color: var(--text-muted);
  padding: 2px 8px;
  background: var(--bg-secondary);
  border-radius: var(--radius-full);
  white-space: nowrap;
  flex-shrink: 0;
}

.overlay-enter-active {
  animation: overlayIn var(--duration-base) var(--ease-out);
}

.overlay-leave-active {
  animation: overlayIn var(--duration-fast) var(--ease-out) reverse;
}

@media (max-width: 767px) {
  .palette-overlay {
    padding-top: var(--space-4);
    padding-left: var(--space-4);
    padding-right: var(--space-4);
  }

  .palette-modal {
    border-radius: var(--radius-lg);
  }
}
</style>
