<template>
  <router-link
    :to="`/review/courses/${course.id}`"
    class="course-list-item"
    :class="{ active: isActive }"
  >
    <span class="course-list-item__name">{{ course.name }}</span>
  </router-link>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import type { Course } from '@/types/course'

const props = defineProps<{ course: Course }>()

const route = useRoute()
const isActive = computed(() => {
  return route.params.id && Number(route.params.id) === props.course.id
})
</script>

<style scoped>
.course-list-item {
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: var(--space-2) var(--space-3);
  border-radius: var(--radius-sm);
  text-decoration: none;
  color: var(--text-primary);
  transition: background var(--duration-fast) var(--ease-smooth);
  cursor: pointer;
}

.course-list-item:hover {
  background: var(--bg-hover);
}

.course-list-item.active {
  background: color-mix(in srgb, var(--brand-primary) 10%, transparent);
}

.course-list-item__name {
  font-size: var(--text-sm);
  font-weight: var(--weight-medium);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

</style>
