<template>
  <router-link
    :to="`/review/courses/${course.id}`"
    class="flex items-center gap-1.5 px-3 py-1.5 text-xs no-underline transition-colors duration-fast cursor-pointer"
    :class="isActive
      ? 'text-primary font-semibold bg-primary/[0.08]'
      : 'text-text-primary hover:bg-bg-hover'"
  >
    <span class="truncate shrink">{{ course.name }}</span>
    <span v-if="course.credits" class="shrink-0 text-[10px] text-text-muted">{{ course.credits }}学分</span>
    <span class="ml-auto shrink-0 text-[10px] tabular-nums" :class="isActive ? 'text-primary/60' : 'text-text-muted'">{{ course.reviewCount }}评</span>
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
