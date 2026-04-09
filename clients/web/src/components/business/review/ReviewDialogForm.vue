<template>
  <!-- Teacher selector -->
  <div class="relative">
    <label for="review-teacher-input" class="font-medium text-text-primary text-sm mb-1.5 block">{{ t('review.post.teacherLabel') }} <span class="text-text-muted font-normal text-xs">({{ t('review.post.teacherOptional') }})</span></label>
    <div v-if="loadingTeachers" class="text-xs text-text-muted py-2">{{ t('review.post.teacherLoading') }}</div>
    <div v-else class="relative">
      <input
        id="review-teacher-input"
        :value="teacherQuery"
        autocomplete="off"
        class="w-full p-3 bg-bg-secondary rounded-lg text-sm text-text-primary font-sans transition-[border-color,box-shadow] duration-fast focus:outline-none focus:border-primary focus:shadow-[0_0_0_3px_rgba(59,130,246,0.1)] focus:bg-bg-card"
        :placeholder="t('review.post.teacherPlaceholder')"
        @focus="$emit('openDropdown')"
        @input="$emit('teacherInput', ($event.target as HTMLInputElement).value)"
      />
      <button v-if="teacherQuery" class="absolute right-3 top-1/2 -translate-y-1/2 text-text-muted hover:text-text-primary" @click="$emit('clearTeacher')">&times;</button>
      <div v-if="teacherDropdownOpen && filteredTeachers.length > 0" class="absolute left-0 right-0 mt-1 rounded-lg bg-bg-card shadow-md max-h-[160px] overflow-y-auto z-10">
        <button
          v-for="teacher in filteredTeachers"
          :key="teacher.teacherID"
          class="flex items-center justify-between w-full p-2.5 text-left text-sm text-text-primary hover:bg-bg-hover transition-colors duration-fast"
          @mousedown.prevent="$emit('selectTeacher', teacher)"
        >
          <span>{{ teacher.teacherName }}</span>
          <span class="text-xs text-text-muted">{{ teacher.departmentName }}</span>
        </button>
      </div>
    </div>
  </div>

  <!-- Term selector -->
  <div class="relative">
    <label for="review-term-select" class="font-medium text-text-primary text-sm mb-1.5 block">{{ t('review.post.termLabel') }} <span class="text-danger text-xs">*</span></label>
    <select
      id="review-term-select"
      :value="selectedTermID"
      class="w-full p-3 bg-bg-secondary border rounded-lg text-sm text-text-primary font-sans transition-[border-color,box-shadow] duration-fast focus:outline-none focus:border-primary focus:shadow-[0_0_0_3px_rgba(59,130,246,0.1)] focus:bg-bg-card"
      :class="attempted && !selectedTermID ? 'border-danger' : 'border-border'"
      @change="$emit('update:selectedTermID', ($event.target as HTMLSelectElement).value)"
    >
      <option value="" disabled>{{ t('review.post.termPlaceholder') }}</option>
      <option v-for="term in termOptions" :key="term.id" :value="term.id">
        {{ term.name }}
      </option>
    </select>
    <span v-if="attempted && !selectedTermID" class="block text-xs text-danger mt-1">{{ t('review.post.termMissing') }}</span>
  </div>

  <!-- Title -->
  <div class="relative">
    <label for="review-title-input" class="font-medium text-text-primary text-sm mb-1.5 block">{{ t('review.post.titleRequired') }} <span class="text-danger text-xs">*</span></label>
    <input
      id="review-title-input"
      :value="title"
      autocomplete="off"
      class="w-full p-3 bg-bg-secondary border rounded-lg text-sm text-text-primary font-sans transition-[border-color,box-shadow] duration-fast focus:outline-none focus:border-primary focus:shadow-[0_0_0_3px_rgba(59,130,246,0.1)] focus:bg-bg-card"
      :class="attempted && titleInvalid ? 'border-danger' : 'border-border'"
      :placeholder="t('review.post.titlePlaceholder')"
      :maxlength="titleMax"
      @input="$emit('update:title', ($event.target as HTMLInputElement).value)"
    />
    <span v-if="attempted && titleInvalid" class="block text-xs text-danger mt-1">{{ t('review.post.titleMissing') }}</span>
    <span v-else class="block text-right text-xs text-text-muted mt-1">
      {{ t('review.validation.charCount', { current: title.length, max: titleMax }) }}
    </span>
  </div>

  <!-- Ratings -->
  <div :class="{ 'ring-1 ring-danger rounded-lg': attempted && ratingsInvalid }">
    <RatingGroup ref="ratingGroupRef" :model-value="ratings" @update:model-value="$emit('update:ratings', $event)" />
  </div>
  <span v-if="attempted && ratingsInvalid" class="block text-xs text-danger -mt-2">{{ t('review.post.ratingMissing') }}</span>

  <!-- Content -->
  <div class="relative">
    <label for="review-content-input" class="font-medium text-text-primary text-sm mb-1.5 block">{{ t('review.post.detailedReview') }} <span class="text-danger text-xs">*</span></label>
    <textarea
      id="review-content-input"
      :value="content"
      class="w-full p-3 bg-bg-secondary border rounded-lg text-sm text-text-primary font-sans resize-vertical min-h-[120px] transition-[border-color,box-shadow] duration-fast focus:outline-none focus:border-primary focus:shadow-[0_0_0_3px_rgba(59,130,246,0.1)] focus:bg-bg-card"
      :class="attempted && contentInvalid ? 'border-danger' : 'border-border'"
      :placeholder="t('review.post.contentPlaceholder')"
      :aria-describedby="contentError ? 'review-dialog-content-error' : undefined"
      :maxlength="contentMax"
      rows="5"
      @input="$emit('update:content', ($event.target as HTMLTextAreaElement).value)"
    />
    <span v-if="contentError || (attempted && contentInvalid)" id="review-dialog-content-error" class="block text-xs text-danger mt-1">
      {{ contentError || t('review.post.contentMinError', { min: contentMin }) }}
    </span>
  </div>

  <!-- Grade -->
  <div class="relative">
    <label for="review-grade-input" class="font-medium text-text-primary text-sm mb-1.5 block">{{ t('review.post.gradeLabel') }} <span class="text-text-muted font-normal text-xs">({{ t('review.post.gradeOptional') }})</span></label>
    <input
      id="review-grade-input"
      :value="grade"
      autocomplete="off"
      class="w-full p-3 bg-bg-secondary rounded-lg text-sm text-text-primary font-sans transition-[border-color,box-shadow] duration-fast focus:outline-none focus:border-primary focus:shadow-[0_0_0_3px_rgba(59,130,246,0.1)] focus:bg-bg-card"
      :placeholder="t('review.post.gradePlaceholder')"
      maxlength="20"
      @input="$emit('update:grade', ($event.target as HTMLInputElement).value)"
    />
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import RatingGroup from './RatingGroup.vue'
import type { TeacherStats } from '@/types/course'
import type { ReviewRatings } from '@/types/review'

defineProps<{
  // Teacher
  teacherQuery: string
  teacherDropdownOpen: boolean
  loadingTeachers: boolean
  filteredTeachers: TeacherStats[]
  // Term
  selectedTermID: string
  termOptions: Array<{ id: string; name: string }>
  // Title
  title: string
  titleMax: number
  titleInvalid: boolean
  // Ratings
  ratings: ReviewRatings
  ratingsInvalid: boolean
  // Content
  content: string
  contentMin: number
  contentMax: number
  contentInvalid: boolean
  contentError: string
  // Grade
  grade: string
  // Validation state
  attempted: boolean
}>()

defineEmits<{
  'update:title': [value: string]
  'update:content': [value: string]
  'update:grade': [value: string]
  'update:ratings': [value: ReviewRatings]
  'update:selectedTermID': [value: string]
  selectTeacher: [teacher: TeacherStats]
  clearTeacher: []
  openDropdown: []
  teacherInput: [value: string]
}>()

const { t } = useI18n()

const ratingGroupRef = ref<InstanceType<typeof RatingGroup> | null>(null)

defineExpose({ ratingGroupRef })
</script>
