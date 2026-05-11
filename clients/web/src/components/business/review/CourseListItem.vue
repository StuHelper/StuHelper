<template>
  <router-link
    :to="`/courses/${course.id}`"
    class="grid grid-cols-[minmax(0,1fr)_auto] gap-x-2 gap-y-0.5 px-3 py-2 text-xs no-underline transition-colors duration-fast cursor-pointer"
    :class="isActive
      ? 'text-primary font-semibold bg-primary/[0.08]'
      : 'text-text-primary hover:bg-bg-hover'"
  >
    <span class="min-w-0 break-words leading-snug">{{ course.name }}</span>
    <span class="justify-self-end text-[10px] tabular-nums whitespace-nowrap" :class="isActive ? 'text-primary/60' : 'text-text-muted'">{{ t('review.course.reviewCountBadge', { count: course.reviewCount }) }}</span>
    <span v-if="course.credits" class="text-[10px] text-text-muted">{{ t('review.course.creditsBadge', { n: course.credits }) }}</span>
  </router-link>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import type { Course } from '@stuhelper/shared/course'

const props = defineProps<{ course: Course }>()

const { t } = useI18n()
const route = useRoute()
const isActive = computed(() => {
  return route.params.id && Number(route.params.id) === props.course.id
})
</script>
