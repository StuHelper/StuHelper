<template>
  <span class="rating-display">
    <span :class="iconClass">
      <i :class="iconName"></i>
    </span>
    <span v-if="showLabel" class="label">{{ label }}</span>
  </span>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { RatingLevel } from '@/types/course'

const props = defineProps<{
  value: RatingLevel
  label?: string
  showLabel?: boolean
}>()

const iconName = computed(() => {
  const icons: Record<RatingLevel, string> = {
    2: 'el-icon-star-filled',
    1: 'el-icon-star-filled',
    0: 'el-icon-minus',
    [-1]: 'el-icon-close',
    [-2]: 'el-icon-close'
  }
  return icons[props.value]
})

const iconClass = computed(() => {
  if (props.value >= 1) return 'text-success'
  if (props.value === 0) return 'text-muted'
  return 'text-danger'
})
</script>

<style scoped>
.rating-display {
  display: inline-flex;
  align-items: center;
  gap: 4px;
}
.text-success { color: #67c23a; }
.text-muted { color: #909399; }
.text-danger { color: #f56c6c; }
.label { font-size: 12px; color: #606266; }
</style>
