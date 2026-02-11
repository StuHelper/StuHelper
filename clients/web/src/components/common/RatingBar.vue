<template>
  <div class="rating-bar">
    <span class="rating-bar__label">{{ label }}</span>
    <div class="rating-bar__track">
      <div
        class="rating-bar__fill"
        :style="{ width: `${percentage}%`, background: fillColor }"
      />
    </div>
    <span class="rating-bar__value font-mono">{{ displayValue }}</span>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

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

const fillColor = computed(() => {
  const ratio = props.value / max.value
  if (ratio >= 0.8) return 'var(--rating-5)'
  if (ratio >= 0.6) return 'var(--rating-4)'
  if (ratio >= 0.4) return 'var(--rating-3)'
  if (ratio >= 0.2) return 'var(--rating-2)'
  return 'var(--rating-1)'
})
</script>

<style scoped>
.rating-bar {
  display: flex;
  align-items: center;
  gap: var(--space-3);
}

.rating-bar__label {
  font-size: var(--text-sm);
  color: var(--text-secondary);
  min-width: 60px;
  white-space: nowrap;
}

.rating-bar__track {
  flex: 1;
  height: 8px;
  background: var(--bg-secondary);
  border-radius: var(--radius-full);
  overflow: hidden;
}

.rating-bar__fill {
  height: 100%;
  border-radius: var(--radius-full);
  transition: width var(--duration-slower) var(--ease-out);
}

.rating-bar__value {
  font-size: var(--text-sm);
  font-weight: var(--weight-semibold);
  color: var(--text-primary);
  min-width: 32px;
  text-align: right;
}
</style>
