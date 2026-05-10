<template>
  <div class="flex flex-col gap-3">
    <RatingBar
      v-for="dim in dimensions"
      :key="dim.key"
      :label="dimensionLabel(dim.key, dim.name)"
      :value="dim.avgRating"
      :max="5"
    />
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import RatingBar from '@/components/common/RatingBar.vue'
import { ratingDimensionLabel } from '@/modules/review/ratingHelpers'

defineProps<{
  dimensions: Array<{
    key: string
    name: string
    avgRating: number
  }>
}>()

const { t } = useI18n()

function dimensionLabel(key: string, fallback?: string): string {
  return ratingDimensionLabel({ key, fallback, t })
}
</script>
