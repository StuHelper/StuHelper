<template>
  <section class="mb-6">
    <h3 class="text-sm font-semibold uppercase tracking-wider text-text-muted mb-4">
      {{ t('review.detail.ratingTitle') }} &gt;
    </h3>

    <!-- No data at all -->
    <div
      v-if="!ratingStats || !ratingStats.byTerm || ratingStats.byTerm.length === 0"
      class="bg-bg-card rounded-xl shadow-card p-8 text-center"
    >
      <p class="text-text-muted mb-3">{{ t('review.detail.noData') }}</p>
      <button
        class="py-2 px-6 text-sm font-medium text-white bg-accent rounded-full transition-opacity duration-fast hover:opacity-90"
        @click="$emit('post')"
      >
        {{ t('review.detail.writeFirst') }}
      </button>
    </div>

    <!-- Semester rating cards grid -->
    <div
      v-else
      class="grid gap-4"
      style="grid-template-columns: repeat(auto-fill, minmax(260px, 1fr))"
    >
      <div
        v-for="(term, idx) in ratingStats.byTerm"
        :key="term.termID ?? term.termName"
        class="glass-card rounded-xl shadow-card p-4 stagger-item"
        :style="{ animationDelay: `${idx * 80}ms` }"
      >
        <!-- Semester header -->
        <div class="flex items-center gap-2 mb-3">
          <span class="text-sm font-bold text-text-primary">
            {{ term.termName }}
          </span>
          <span
            v-if="termReviewCount(term) > 0"
            class="inline-flex items-center text-[0.65rem] font-medium px-2 py-0.5 h-[18px] rounded-full bg-primary/10 text-primary"
          >
            {{ termReviewCount(term) }} {{ t('review.course.reviewUnit') }}
          </span>
          <span
            v-if="termReviewCount(term) > 0 && termReviewCount(term) < 3"
            class="inline-flex items-center text-[0.65rem] font-medium px-2 py-0.5 h-[18px] rounded-full bg-warning/15 text-warning"
          >
            {{ t('review.detail.insufficientData') }}
          </span>
        </div>

        <!-- Insufficient data message -->
        <div v-if="termReviewCount(term) > 0 && termReviewCount(term) < 3" class="mb-2">
          <p class="text-xs m-0 text-text-muted">
            {{ t('review.detail.insufficientHint') }}
          </p>
        </div>

        <!-- Rating bars (4 dimensions) -->
        <div v-if="term.dimensions && term.dimensions.length > 0" class="flex flex-col gap-2.5">
          <div v-for="dim in term.dimensions" :key="dim.key" class="flex items-center gap-2">
            <span class="text-xs w-16 shrink-0 text-right text-text-secondary">
              {{ dimensionLabel(dim.key, t) }}
            </span>
            <div class="flex-1 h-2 bg-bg-secondary rounded-full overflow-hidden">
              <div
                class="h-full rounded-full transition-all duration-slow"
                :style="{
                  width: `${(dim.avgRating / 5) * 100}%`,
                  backgroundColor: ratingBarColor(dim.avgRating)
                }"
              />
            </div>
            <span class="text-xs font-mono w-8 text-right" :style="{ color: ratingBarColor(dim.avgRating) }">
              {{ dim.avgRating.toFixed(1) }}
            </span>
          </div>
        </div>

        <!-- No dimensions -->
        <div v-else class="text-center py-2">
          <p class="text-xs text-text-muted">{{ t('review.stats.noData') }}</p>
        </div>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import type { CourseRatingStatsResponse } from '@/types/course'
import { dimensionLabel, ratingBarColor, termReviewCount } from '@/modules/review/ratingHelpers'

defineProps<{
  ratingStats: CourseRatingStatsResponse | null
}>()

defineEmits<{
  post: []
}>()

const { t } = useI18n()
</script>
