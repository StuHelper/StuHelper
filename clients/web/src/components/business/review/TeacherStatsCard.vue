<template>
  <div class="border-0 rounded-md p-5 transition-all duration-base ease-out hover:shadow-md hover:-translate-y-0.5">
    <div class="flex items-center gap-3 mb-4">
      <div class="w-11 h-11 bg-bg-secondary rounded-sm flex items-center justify-center text-text-muted">
        <User :size="28" :stroke-width="1.5" aria-hidden="true" />
      </div>
      <div class="flex-1">
        <h3 class="font-serif text-base font-semibold text-text-primary m-0 mb-1">{{ stats.teacherName }}</h3>
        <p class="text-sm text-text-muted m-0">{{ stats.departmentName }}</p>
      </div>
    </div>

    <div class="grid grid-cols-3 gap-3 py-4 border-t border-b border-border-light">
      <div class="flex flex-col items-center text-center">
        <span class="h-7 flex items-center justify-center text-accent">
          <EmojiRating v-if="stats.avgRating" :value="stats.avgRating" size="lg" />
          <span v-else class="text-text-muted">-</span>
        </span>
        <span class="text-xs text-text-muted mt-1">{{ t('teaching.profile.avgRating') }}</span>
      </div>
      <div class="flex flex-col items-center text-center">
        <span class="font-display text-xl font-bold text-accent tabular-nums">{{ stats.courseCount }}</span>
        <span class="text-xs text-text-muted mt-1">{{ t('teaching.profile.courseCount') }}</span>
      </div>
      <div class="flex flex-col items-center text-center">
        <span class="font-display text-xl font-bold text-accent tabular-nums">{{ stats.reviewCount }}</span>
        <span class="text-xs text-text-muted mt-1">{{ t('teaching.profile.reviewCount') }}</span>
      </div>
    </div>

    <div v-if="stats.tags?.length" class="flex flex-wrap gap-2 mt-4">
      <span
        v-for="tag in stats.tags"
        :key="tag"
        class="py-1 px-2 bg-bg-elevated rounded-sm text-xs text-text-secondary"
      >
        {{ tag }}
      </span>
    </div>

    <router-link
      :to="`/teachers/${stats.teacherID}`"
      class="flex items-center justify-center gap-1 mt-4 p-2 text-accent text-sm font-medium no-underline transition-all duration-fast hover:gap-2"
    >
      {{ t('teaching.profile.viewDetail') }}
      <ChevronRight :size="16" class="transition-transform duration-fast hover:translate-x-0.5" />
    </router-link>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { User, ChevronRight } from 'lucide-vue-next'
import type { TeacherStats } from '@stuhelper/shared/course'
import EmojiRating from './EmojiRating.vue'

const { t } = useI18n()

defineProps<{
  stats: TeacherStats
}>()
</script>
