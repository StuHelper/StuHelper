<template>
  <div class="rating-circle" :style="{ width: `${size}px`, height: `${size}px` }">
    <svg :width="size" :height="size" :viewBox="`0 0 ${size} ${size}`">
      <circle
        class="rating-circle__track"
        :cx="center"
        :cy="center"
        :r="radius"
        fill="none"
        :stroke-width="strokeWidth"
      />
      <circle
        class="rating-circle__fill"
        :cx="center"
        :cy="center"
        :r="radius"
        fill="none"
        :stroke-width="strokeWidth"
        :stroke-dasharray="circumference"
        :stroke-dashoffset="dashOffset"
        :stroke="strokeColor"
        stroke-linecap="round"
      />
    </svg>
    <div class="rating-circle__label">
      <span class="rating-circle__value font-mono">{{ displayValue }}</span>
      <span v-if="subtitle" class="rating-circle__subtitle">{{ subtitle }}</span>
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

const size = computed(() => props.size ?? 80)
const strokeWidth = computed(() => props.strokeWidth ?? 6)
const max = computed(() => props.max ?? 5)
const center = computed(() => size.value / 2)
const radius = computed(() => center.value - strokeWidth.value)
const circumference = computed(() => 2 * Math.PI * radius.value)

const percentage = computed(() =>
  Math.min(1, props.value / max.value)
)

const dashOffset = computed(() =>
  circumference.value * (1 - percentage.value)
)

const displayValue = computed(() => props.value.toFixed(1))

const strokeColor = computed(() => {
  const ratio = percentage.value
  if (ratio >= 0.8) return 'var(--rating-5)'
  if (ratio >= 0.6) return 'var(--rating-4)'
  if (ratio >= 0.4) return 'var(--rating-3)'
  if (ratio >= 0.2) return 'var(--rating-2)'
  return 'var(--rating-1)'
})
</script>

<style scoped>
.rating-circle {
  position: relative;
  display: inline-flex;
  align-items: center;
  justify-content: center;
}

.rating-circle svg {
  transform: rotate(-90deg);
  position: absolute;
  inset: 0;
}

.rating-circle__track {
  stroke: var(--bg-secondary);
}

.rating-circle__fill {
  transition: stroke-dashoffset var(--duration-slower) var(--ease-out);
}

.rating-circle__label {
  display: flex;
  flex-direction: column;
  align-items: center;
  z-index: 1;
}

.rating-circle__value {
  font-size: var(--text-xl);
  font-weight: var(--weight-bold);
  color: var(--text-primary);
  line-height: 1;
}

.rating-circle__subtitle {
  font-size: var(--text-xs);
  color: var(--text-muted);
  margin-top: 2px;
}
</style>
