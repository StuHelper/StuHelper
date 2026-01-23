<template>
  <div class="department-list">
    <div
      v-for="category in categories"
      :key="category.key"
      class="category-section"
    >
      <button
        class="category-header"
        :class="{ expanded: expandedCategories.includes(category.key) }"
        @click="toggleCategory(category.key)"
      >
        <span class="category-name">{{ category.label }}</span>
        <span class="category-count">{{ getDeptCount(category.key) }}</span>
        <svg class="chevron" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M6 9l6 6 6-6"/>
        </svg>
      </button>

      <Transition name="collapse">
        <div v-if="expandedCategories.includes(category.key)" class="category-content">
          <button
            v-for="dept in getDeptsByCategory(category.key)"
            :key="dept.id"
            class="dept-item"
            :class="{ active: selectedId === dept.id }"
            @click="handleSelect(dept)"
          >
            {{ dept.name }}
          </button>
        </div>
      </Transition>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import type { Department } from '@/types/course'

const props = defineProps<{
  departments: Department[]
  selectedId?: number
}>()

const emit = defineEmits<{
  select: [dept: Department]
}>()

const expandedCategories = ref(['school'])

const categories = [
  { key: 'school', label: '院系课程' },
  { key: 'elective', label: '通选课' },
  { key: 'pe', label: '体育课' },
  { key: 'english', label: '英语课' },
  { key: 'pols', label: '政治课' }
]

const getDeptsByCategory = (category: string) => {
  return props.departments.filter(d => d.category === category)
}

const getDeptCount = (category: string) => {
  return getDeptsByCategory(category).length
}

const toggleCategory = (key: string) => {
  const index = expandedCategories.value.indexOf(key)
  if (index > -1) {
    expandedCategories.value.splice(index, 1)
  } else {
    expandedCategories.value.push(key)
  }
}

const handleSelect = (dept: Department) => {
  emit('select', dept)
}
</script>

<style scoped>
.department-list {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.category-section {
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  overflow: hidden;
}

.category-header {
  width: 100%;
  display: flex;
  align-items: center;
  gap: var(--space-3);
  padding: var(--space-3) var(--space-4);
  background: var(--bg-card);
  color: var(--text-primary);
  font-weight: 500;
  text-align: left;
  transition: background var(--duration-fast);
}

.category-header:hover {
  background: var(--bg-elevated);
}

.category-name {
  flex: 1;
}

.category-count {
  font-size: var(--text-sm);
  color: var(--text-muted);
}

.chevron {
  width: 18px;
  height: 18px;
  color: var(--text-muted);
  transition: transform var(--duration-fast);
}

.category-header.expanded .chevron {
  transform: rotate(180deg);
}

.category-content {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-2);
  padding: var(--space-3);
  background: var(--bg-secondary);
}

.dept-item {
  padding: var(--space-2) var(--space-3);
  font-size: var(--text-sm);
  color: var(--text-secondary);
  background: var(--bg-card);
  border-radius: var(--radius-sm);
  transition: all var(--duration-fast);
}

.dept-item:hover {
  color: var(--accent);
  background: var(--bg-elevated);
}

.dept-item.active {
  color: var(--bg-primary);
  background: var(--accent);
}

/* Collapse Transition */
.collapse-enter-active,
.collapse-leave-active {
  transition: all var(--duration-base) var(--ease-out);
  overflow: hidden;
}

.collapse-enter-from,
.collapse-leave-to {
  opacity: 0;
  max-height: 0;
  padding-top: 0;
  padding-bottom: 0;
}

.collapse-enter-to,
.collapse-leave-from {
  max-height: 500px;
}
</style>
