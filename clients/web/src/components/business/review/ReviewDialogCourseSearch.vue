<template>
  <div v-if="!selectedCourse">
    <input
      :value="courseQuery"
      autocomplete="off"
      role="combobox"
      :aria-expanded="courseResults.length > 0"
      aria-haspopup="listbox"
      aria-autocomplete="list"
      :aria-activedescendant="highlightedIndex >= 0 ? `course-option-${courseResults[highlightedIndex]?.id}` : undefined"
      aria-controls="course-search-listbox"
      class="w-full p-3 bg-bg-secondary rounded-lg text-sm text-text-primary font-sans transition-[border-color,box-shadow] duration-fast focus:outline-none focus:border-primary focus:shadow-[0_0_0_3px_rgba(59,130,246,0.1)] focus:bg-bg-card"
      :placeholder="t('review.post.searchCourse')"
      :aria-label="t('review.post.searchCourseLabel')"
      @input="$emit('update:courseQuery', ($event.target as HTMLInputElement).value)"
      @keydown="$emit('keydown', $event)"
    />
    <div v-if="courseResults.length > 0" id="course-search-listbox" role="listbox" class="border-0 rounded-lg max-h-[200px] overflow-y-auto mt-2">
      <button
        v-for="(c, idx) in courseResults"
        :id="`course-option-${c.id}`"
        :key="c.id"
        role="option"
        :aria-selected="idx === highlightedIndex"
        class="flex items-center gap-2 w-full p-3 text-left text-sm text-text-primary cursor-pointer transition-[background] duration-fast hover:bg-bg-hover"
        :class="{ 'bg-bg-hover': idx === highlightedIndex }"
        @click="$emit('select', c)"
      >
        <span class="font-medium truncate">{{ c.name }}</span>
        <span class="shrink-0 text-xs text-text-muted"><template v-if="c.credits">{{ t('review.course.creditsBadge', { n: c.credits }) }} · </template>{{ c.departmentName }}</span>
        <span class="shrink-0 text-xs tabular-nums text-text-muted ml-auto">{{ t('review.course.reviewCountBadge', { count: c.reviewCount ?? 0 }) }}</span>
      </button>
    </div>
  </div>

  <!-- Selected course chip -->
  <div v-else class="flex items-center justify-between p-3 bg-primary/[0.06] rounded-lg border border-primary/15">
    <span class="font-semibold text-sm">{{ selectedCourse.name }}</span>
    <button class="text-xs text-primary cursor-pointer" @click="$emit('deselect')">
      {{ t('common.actions.edit') }}
    </button>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import type { Course } from '@/types/course'

defineProps<{
  selectedCourse: Course | null
  courseQuery: string
  courseResults: Course[]
  highlightedIndex: number
}>()

defineEmits<{
  select: [course: Course]
  deselect: []
  'update:courseQuery': [value: string]
  keydown: [event: KeyboardEvent]
}>()

const { t } = useI18n()
</script>
