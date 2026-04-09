<template>
  <section v-if="uniqueTeachers.length > 0" class="mb-6">
    <h3 class="text-sm font-semibold uppercase tracking-wider text-text-muted mb-3">
      {{ t('review.detail.teacherTitle') }} &gt;
    </h3>
    <div class="flex flex-wrap gap-2">
      <button
        v-for="teacher in teacherChips"
        :key="teacher"
        class="rounded-full px-3 py-1 text-[0.8125rem] font-semibold transition-all duration-base"
        :class="selectedTeacher === teacher
          ? 'bg-primary text-white shadow-glow-primary scale-105'
          : 'bg-bg-elevated text-text-secondary hover:scale-[1.02] hover:shadow-sm'"
        @click="$emit('select', teacher)"
      >
        {{ teacher === '' ? t('review.filter.all') : teacher }}
      </button>
    </div>
  </section>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

defineProps<{
  uniqueTeachers: string[]
  teacherChips: string[]
  selectedTeacher: string
}>()

defineEmits<{
  select: [name: string]
}>()

const { t } = useI18n()
</script>
