<template>
  <div class="flex flex-col gap-2">
    <div
      v-for="star in [5, 4, 3, 2, 1]"
      :key="star"
      class="flex items-center gap-2"
    >
      <span class="font-mono text-xs font-medium text-text-muted w-4 text-center">{{ star }}</span>
      <div class="flex-1 h-2 bg-bg-secondary rounded-full overflow-hidden">
        <div
          class="h-full rounded-full transition-all duration-slower ease-out"
          :style="{ width: `${getPercentage(star)}%`, background: `var(--rating-${star})` }"
        />
      </div>
      <span class="font-mono text-xs text-text-muted w-7 text-right">{{ getCount(star) }}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{
  distribution: Record<number, number>
}>()

const total = computed(() =>
  Object.values(props.distribution).reduce((a, b) => a + b, 0)
)

function getCount(star: number): number {
  return props.distribution[star] || 0
}

function getPercentage(star: number): number {
  if (total.value === 0) return 0
  return (getCount(star) / total.value) * 100
}
</script>
