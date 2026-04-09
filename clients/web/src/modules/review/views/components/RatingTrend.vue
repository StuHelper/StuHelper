<template>
  <section v-if="ratingTrend.length >= 2" class="mb-6">
    <h3 class="text-sm font-semibold uppercase tracking-wider text-text-muted mb-3">
      {{ t('review.detail.trendTitle') }}
    </h3>
    <div class="glass-card rounded-xl shadow-card p-4">
      <div class="flex items-end gap-1 h-[120px]">
        <div
          v-for="(point, idx) in ratingTrend"
          :key="idx"
          class="flex-1 flex flex-col items-center gap-1"
        >
          <span class="text-xs font-mono font-medium text-primary">
            {{ point.avgRating.toFixed(1) }}
          </span>
          <div
            class="w-full rounded-t-md bg-primary/20 transition-all duration-base"
            :style="{ height: `${Math.max(8, (point.avgRating / 5) * 100)}%` }"
          >
            <div
              class="w-full h-full rounded-t-md bg-gradient-to-t from-primary/60 to-primary"
            />
          </div>
          <span class="text-[10px] text-text-muted truncate max-w-full text-center">
            {{ point.termName }}
          </span>
        </div>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

defineProps<{
  ratingTrend: { termName: string; avgRating: number }[]
}>()

const { t } = useI18n()
</script>
