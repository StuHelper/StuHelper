<template>
  <div class="flex flex-col gap-3">
    <RatingBar
      v-for="dim in dimensions"
      :key="dim.key"
      :label="dim.name"
      :value="dim.avgRating"
      :max="5"
    />
  </div>
</template>

<script setup lang="ts">
import RatingBar from '@/components/common/RatingBar.vue'

defineProps<{
  dimensions: Array<{
    key: string
    name: string
    avgRating: number
  }>
}>()
</script>
