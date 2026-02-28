<template>
  <div class="grid grid-cols-2 gap-4 max-sm:grid-cols-1">
    <div
      v-for="term in terms"
      :key="term.termName"
      class="bg-bg-card border border-border rounded-xl p-5 shadow-card"
    >
      <div class="flex items-center justify-between mb-4">
        <h4 class="text-sm font-semibold text-text-primary m-0">{{ term.termName }}</h4>
        <span
          v-if="getTermRatingCount(term) < 3"
          class="text-[11px] py-0.5 px-2 rounded-full bg-warning/15 text-warning font-medium"
        >
          {{ t('review.stats.insufficient') }}
        </span>
        <span v-else class="text-xs text-text-muted">
          {{ t('review.stats.reviewCount', { count: getTermRatingCount(term) }) }}
        </span>
      </div>

      <template v-if="getTermRatingCount(term) >= 3">
        <div class="flex flex-col gap-3">
          <div v-for="dim in term.dimensions" :key="dim.key" class="flex flex-col gap-1">
            <div class="flex justify-between text-xs">
              <span class="text-text-secondary">{{ dim.name }}</span>
              <span class="font-mono font-medium text-text-primary">{{ dim.avgRating.toFixed(1) }}</span>
            </div>
            <div class="h-2 bg-bg-secondary rounded-full overflow-hidden">
              <div
                class="h-full rounded-full transition-all duration-slow ease-smooth"
                :style="{
                  width: `${(dim.avgRating / 5) * 100}%`,
                  backgroundColor: getRatingColor(dim.avgRating)
                }"
              />
            </div>
          </div>
        </div>
      </template>

      <div v-else class="text-center text-text-muted text-sm py-6">
        {{ t('review.stats.insufficientHint') }}
      </div>
    </div>

    <div v-if="!terms.length" class="col-span-full text-center text-text-muted text-sm py-8">
      {{ t('review.stats.noData') }}
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { CourseRatingStatsResponse, TermRatingStats } from '@/types/course'

const props = defineProps<{
  ratingStats: CourseRatingStatsResponse | null
}>()

const { t } = useI18n()

const terms = computed(() => props.ratingStats?.byTerm || [])

// 从第一个维度取 ratingCount（同一学期各维度 ratingCount 相同）
function getTermRatingCount(term: TermRatingStats): number {
  return term.dimensions?.[0]?.ratingCount ?? 0
}

function getRatingColor(rating: number): string {
  if (rating >= 4) return 'var(--color-rating-5)'
  if (rating >= 3) return 'var(--color-rating-4)'
  if (rating >= 2) return 'var(--color-rating-3)'
  return 'var(--color-rating-1)'
}
</script>
