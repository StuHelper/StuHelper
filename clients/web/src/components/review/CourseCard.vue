<template>
  <router-link :to="`/review/courses/${course.id}`" class="quick-card">
    <div class="quick-card__info">
      <span class="quick-card__name">{{ course.name }}</span>
    </div>
    <div class="quick-card__meta">
      <span v-if="course.departmentName" class="quick-card__dept">
        {{ course.departmentName }}
      </span>
      <span class="quick-card__reviews font-mono">
        {{ course.reviewCount }}
      </span>
    </div>
  </router-link>
</template>

<script setup lang="ts">
import type { Course } from '@/types/course'

defineProps<{ course: Course }>()
</script>

<style scoped>
.quick-card {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--space-3) var(--space-4);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  text-decoration: none;
  color: var(--text-primary);
  transition: border-color var(--duration-fast),
    box-shadow var(--duration-fast);
}

.quick-card:hover {
  border-color: var(--brand-primary);
  box-shadow: var(--shadow-glow-sm);
}

.quick-card__info {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
  min-width: 0;
}

.quick-card__name {
  font-size: var(--text-sm);
  font-weight: var(--weight-semibold);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.quick-card__meta {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  flex-shrink: 0;
}

.quick-card__dept {
  font-size: var(--text-xs);
  color: var(--text-muted);
  padding: 2px 8px;
  background: var(--bg-secondary);
  border-radius: var(--radius-full);
}

.quick-card__reviews {
  font-size: var(--text-xs);
  color: var(--text-muted);
}
</style>
