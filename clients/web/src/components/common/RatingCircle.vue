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
      <slot>
        <span class="h-3 w-3 rounded-full" :style="{ backgroundColor: strokeColor }" aria-hidden="true" />
      </slot>
      <span v-if="subtitle" class="text-xs text-text-muted mt-0.5">{{ subtitle }}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { getScaledRatingColor } from '@/design-system/rating'

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

const strokeColor = computed(() => getScaledRatingColor(props.value, resolvedMax.value))
</script>
