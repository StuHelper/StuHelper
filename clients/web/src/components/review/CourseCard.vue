<template>
  <article class="course-card" @click="handleClick">
    <div class="card-content">
      <h3 class="course-name">{{ course.name }}</h3>
      <div class="course-meta">
        <span v-if="course.departmentName" class="department">
          {{ course.departmentName }}
        </span>
        <span v-if="course.credits" class="credits">
          {{ course.credits }}学分
        </span>
      </div>
    </div>
    <div class="card-stats">
      <div class="stat-item reviews">
        <span class="stat-value">{{ course.reviewCount || 0 }}</span>
        <span class="stat-label">测评</span>
      </div>
    </div>
    <div class="card-arrow">
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <path d="M9 18l6-6-6-6"/>
      </svg>
    </div>
  </article>
</template>

<script setup lang="ts">
import type { Course } from '@/types/course'

const props = defineProps<{
  course: Course
}>()

const emit = defineEmits<{
  click: [course: Course]
}>()

const handleClick = () => {
  emit('click', props.course)
}
</script>

<style scoped>
.course-card {
  display: flex;
  align-items: center;
  gap: var(--space-4);
  padding: var(--space-4);
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  cursor: pointer;
  transition: all var(--duration-base) var(--ease-out);
}

.course-card:hover {
  border-color: var(--border-accent);
  background: var(--bg-elevated);
  transform: translateX(4px);
}

.card-content {
  flex: 1;
  min-width: 0;
}

.course-name {
  font-size: var(--text-base);
  font-weight: 500;
  color: var(--text-primary);
  margin: 0 0 var(--space-1) 0;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.course-meta {
  display: flex;
  gap: var(--space-3);
  font-size: var(--text-sm);
  color: var(--text-muted);
}

.card-stats {
  display: flex;
  gap: var(--space-4);
}

.stat-item {
  text-align: center;
}

.stat-value {
  display: block;
  font-size: var(--text-lg);
  font-weight: 600;
  color: var(--accent);
}

.stat-label {
  font-size: var(--text-xs);
  color: var(--text-muted);
}

.card-arrow {
  color: var(--text-muted);
  transition: all var(--duration-fast);
}

.card-arrow svg {
  width: 20px;
  height: 20px;
}

.course-card:hover .card-arrow {
  color: var(--accent);
  transform: translateX(4px);
}
</style>
