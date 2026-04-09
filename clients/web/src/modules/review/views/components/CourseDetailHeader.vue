<template>
  <header class="flex items-center gap-3 flex-wrap mb-6">
    <h4 class="text-xl font-bold m-0 text-text-primary">
      {{ course.name }}
    </h4>
    <span
      v-if="course.departmentName"
      class="inline-flex items-center text-xs font-medium px-2.5 py-0.5 rounded-full bg-primary/10 text-primary"
    >
      {{ course.departmentName }}
    </span>
    <span
      v-if="total > 0"
      class="inline-flex items-center text-xs font-medium px-2.5 py-0.5 rounded-full bg-primary/10 text-primary"
    >
      {{ total }} {{ t('review.course.reviewUnit') }}
    </span>
    <div class="flex items-center gap-2 ml-auto">
      <FavoriteButton :course-i-d="courseID" />
      <button
        class="py-1.5 px-5 text-sm font-medium text-white bg-accent rounded-full transition-opacity duration-fast hover:opacity-90"
        @click="$emit('post')"
      >
        {{ t('review.hub.postReview') }}
      </button>
    </div>
  </header>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import FavoriteButton from '@/components/business/review/FavoriteButton.vue'
import type { Course } from '@/types/course'

defineProps<{
  course: Course
  courseID: number
  total: number
}>()

defineEmits<{
  post: []
}>()

const { t } = useI18n()
</script>
