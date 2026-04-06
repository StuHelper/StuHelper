<template>
  <div class="relative inline-flex items-center justify-center" :style="{ width: `${resolvedSize}px`, height: `${resolvedSize}px` }">
    <svg class="absolute inset-0 -rotate-90" :width="resolvedSize" :height="resolvedSize" :viewBox="`0 0 ${resolvedSize} ${resolvedSize}`">
      <circle
        :cx="center"
        :cy="center"
        :r="radius"
        fill="none"
        :stroke-width="resolvedStrokeWidth"
        :style="{ stroke: 'var(--color-bg-secondary)' }"
      />
      <circle
        class="transition-all duration-700 ease-out"
        :cx="center"
        :cy="center"
        :r="radius"
        fill="none"
        :stroke-width="resolvedStrokeWidth"
        :stroke-dasharray="circumference"
        :stroke-dashoffset="dashOffset"
        :stroke="strokeColor"
        stroke-linecap="round"
      />
    </svg>
    <div class="flex flex-col items-center z-[1]">
      <span class="text-xl font-bold text-text-primary leading-none font-mono">{{ displayValue }}</span>
      <span v-if="subtitle" class="text-xs text-text-muted mt-0.5">{{ subtitle }}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{
  value: number
  max?: number
  size?: number
  strokeWidth?: number
  subtitle?: string
}>()

const resolvedSize = computed(() => props.size ?? 80)
const resolvedStrokeWidth = computed(() => props.strokeWidth ?? 6)
// 防止 max 为 0 时除零
const resolvedMax = computed(() => Math.max(props.max ?? 5, 0.01))
const center = computed(() => resolvedSize.value / 2)
const radius = computed(() => center.value - resolvedStrokeWidth.value)
const circumference = computed(() => 2 * Math.PI * radius.value)

const percentage = computed(() =>
  Math.min(1, props.value / resolvedMax.value)
)

const dashOffset = computed(() =>
  circumference.value * (1 - percentage.value)
)

const displayValue = computed(() => props.value.toFixed(1))

const strokeColor = computed(() => {
  const ratio = percentage.value
  if (ratio >= 0.8) return 'var(--color-rating-5)'
  if (ratio >= 0.6) return 'var(--color-rating-4)'
  if (ratio >= 0.4) return 'var(--color-rating-3)'
  if (ratio >= 0.2) return 'var(--color-rating-2)'
  return 'var(--color-rating-1)'
})
</script>
