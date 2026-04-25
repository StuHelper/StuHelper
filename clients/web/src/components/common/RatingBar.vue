<template>
  <div class="flex items-center gap-3">
    <span class="text-sm text-text-secondary min-w-[60px] whitespace-nowrap">{{ label }}</span>
    <div class="flex-1 h-2 bg-bg-secondary rounded-full overflow-hidden">
      <div
        class="h-full rounded-full transition-all duration-700 ease-out"
        :style="{ width: `${percentage}%`, background: fillColor }"
      />
    </div>
    <span class="text-sm font-semibold text-text-primary min-w-[32px] text-right font-mono">{{ displayValue }}</span>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { getScaledRatingColor } from '@/design-system/rating'

const props = defineProps<{
  label: string
  value: number
  max?: number
  showPercentage?: boolean
}>()

const max = computed(() => props.max ?? 5)
const percentage = computed(() =>
  Math.min(100, (props.value / max.value) * 100)
)
const displayValue = computed(() =>
  props.showPercentage
    ? `${Math.round(percentage.value)}%`
    : props.value.toFixed(1)
)

const fillColor = computed(() => getScaledRatingColor(props.value, max.value))
</script>
