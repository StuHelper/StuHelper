<template>
  <router-link
    :to="`/courses/${course.id}`"
    class="flex items-center justify-between px-4 py-3 rounded-lg no-underline text-text-primary glass-card hover-lift"
  >
    <div class="flex flex-col gap-1 min-w-0">
      <span class="text-sm font-semibold whitespace-nowrap overflow-hidden text-ellipsis">{{ course.name }}</span>
    </div>
    <div class="flex items-center gap-2 shrink-0">
      <span v-if="course.departmentName" class="text-xs text-text-muted px-2 py-0.5 bg-bg-secondary rounded-full">
        {{ course.departmentName }}
      </span>
      <span class="text-xs text-text-muted font-mono">
        {{ course.reviewCount }}
      </span>
    </div>
  </router-link>
</template>

<script setup lang="ts">
import type { Course, FavoriteCourse } from '@/types/course'

defineProps<{ course: Course | FavoriteCourse }>()
</script>
