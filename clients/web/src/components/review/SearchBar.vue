<template>
  <div class="search-bar" :class="{ focused: isFocused, expanded: showResults }">
    <div class="search-input-wrapper">
      <svg class="search-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <circle cx="11" cy="11" r="8"/>
        <path d="m21 21-4.35-4.35"/>
      </svg>
      <input
        ref="inputRef"
        v-model="query"
        type="text"
        class="search-input"
        :placeholder="placeholder"
        @focus="handleFocus"
        @blur="handleBlur"
        @input="handleInput"
        @keydown.down.prevent="navigateDown"
        @keydown.up.prevent="navigateUp"
        @keydown.enter.prevent="selectCurrent"
        @keydown.escape="handleEscape"
      />
      <button v-if="query" class="clear-btn" @click="clearQuery">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M18 6L6 18M6 6l12 12"/>
        </svg>
      </button>
    </div>

    <Transition name="dropdown">
      <div v-if="showResults" class="search-results">
        <div v-if="loading" class="loading-state">
          <span class="spinner"></span>
          <span>搜索中...</span>
        </div>
        <template v-else-if="results.length">
          <button
            v-for="(course, index) in results"
            :key="course.id"
            class="result-item"
            :class="{ active: index === activeIndex }"
            @click="selectCourse(course)"
            @mouseenter="activeIndex = index"
          >
            <span class="course-name">{{ course.name }}</span>
            <span class="course-meta">
              <span v-if="course.departmentName">{{ course.departmentName }}</span>
              <span v-if="course.credits">{{ course.credits }}学分</span>
            </span>
          </button>
        </template>
        <div v-else-if="query.length >= 2" class="empty-state">
          未找到相关课程
        </div>
      </div>
    </Transition>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { searchCourses } from '@/api/course'
import type { Course } from '@/types/course'

withDefaults(defineProps<{
  placeholder?: string
}>(), {
  placeholder: '搜索课程名称...'
})

const emit = defineEmits<{
  select: [course: Course]
}>()

const inputRef = ref<HTMLInputElement>()
const query = ref('')
const results = ref<Course[]>([])
const loading = ref(false)
const isFocused = ref(false)
const showResults = ref(false)
const activeIndex = ref(-1)

let searchTimeout: ReturnType<typeof setTimeout>

const handleFocus = () => {
  isFocused.value = true
  if (query.value.length >= 2) {
    showResults.value = true
  }
}

const handleBlur = () => {
  setTimeout(() => {
    isFocused.value = false
    showResults.value = false
  }, 200)
}

const handleInput = () => {
  activeIndex.value = -1
  if (query.value.length < 2) {
    results.value = []
    showResults.value = false
    return
  }
  showResults.value = true
  clearTimeout(searchTimeout)
  searchTimeout = setTimeout(doSearch, 300)
}

const doSearch = async () => {
  if (query.value.length < 2) return
  loading.value = true
  try {
    const res = await searchCourses(query.value, 8)
    results.value = res.data || []
  } catch (e) {
    console.error('Search failed:', e)
    results.value = []
  } finally {
    loading.value = false
  }
}

const selectCourse = (course: Course) => {
  emit('select', course)
  query.value = ''
  results.value = []
  showResults.value = false
}

const clearQuery = () => {
  query.value = ''
  results.value = []
  inputRef.value?.focus()
}

const navigateDown = () => {
  if (activeIndex.value < results.value.length - 1) {
    activeIndex.value++
  }
}

const navigateUp = () => {
  if (activeIndex.value > 0) {
    activeIndex.value--
  }
}

const selectCurrent = () => {
  if (activeIndex.value >= 0 && results.value[activeIndex.value]) {
    selectCourse(results.value[activeIndex.value])
  }
}

const handleEscape = () => {
  showResults.value = false
  inputRef.value?.blur()
}
</script>

<style scoped>
.search-bar {
  position: relative;
  width: 100%;
  max-width: 500px;
}

.search-input-wrapper {
  position: relative;
  display: flex;
  align-items: center;
}

.search-icon {
  position: absolute;
  left: var(--space-4);
  width: 20px;
  height: 20px;
  color: var(--text-muted);
  pointer-events: none;
  transition: color var(--duration-fast);
}

.search-bar.focused .search-icon {
  color: var(--accent);
}

.search-input {
  width: 100%;
  padding: var(--space-3) var(--space-4);
  padding-left: calc(var(--space-4) + 28px);
  padding-right: var(--space-10);
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius-full);
  color: var(--text-primary);
  font-size: var(--text-base);
  transition: all var(--duration-base) var(--ease-out);
}

.search-input::placeholder {
  color: var(--text-muted);
}

.search-input:focus {
  outline: none;
  border-color: var(--accent);
  box-shadow: 0 0 0 3px rgba(201, 162, 39, 0.15);
}

.clear-btn {
  position: absolute;
  right: var(--space-3);
  width: 24px;
  height: 24px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--text-muted);
  border-radius: 50%;
  transition: all var(--duration-fast);
}

.clear-btn:hover {
  color: var(--text-primary);
  background: var(--bg-hover);
}

.clear-btn svg {
  width: 16px;
  height: 16px;
}

/* Results Dropdown */
.search-results {
  position: absolute;
  top: calc(100% + 8px);
  left: 0;
  right: 0;
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-xl);
  overflow: hidden;
  z-index: var(--z-dropdown);
}

.loading-state,
.empty-state {
  padding: var(--space-6);
  text-align: center;
  color: var(--text-muted);
  display: flex;
  align-items: center;
  justify-content: center;
  gap: var(--space-2);
}

.spinner {
  width: 16px;
  height: 16px;
  border: 2px solid var(--border);
  border-top-color: var(--accent);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

.result-item {
  width: 100%;
  padding: var(--space-3) var(--space-4);
  display: flex;
  justify-content: space-between;
  align-items: center;
  text-align: left;
  color: var(--text-primary);
  transition: background var(--duration-fast);
}

.result-item:hover,
.result-item.active {
  background: var(--bg-hover);
}

.course-name {
  font-weight: 500;
}

.course-meta {
  display: flex;
  gap: var(--space-2);
  font-size: var(--text-sm);
  color: var(--text-muted);
}

/* Transitions */
.dropdown-enter-active,
.dropdown-leave-active {
  transition: all var(--duration-fast) var(--ease-out);
}

.dropdown-enter-from,
.dropdown-leave-to {
  opacity: 0;
  transform: translateY(-8px);
}
</style>
