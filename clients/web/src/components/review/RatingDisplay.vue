<template>
  <span class="rating-display">
    <span class="stars">
      <el-icon v-for="star in 5" :key="star" :class="star <= value ? 'active' : ''">
        <StarFilled v-if="star <= value" />
        <Star v-else />
      </el-icon>
    </span>
    <span v-if="showValue" class="value">{{ value.toFixed(1) }}</span>
    <span v-if="showLabel && label" class="label">{{ label }}</span>
  </span>
</template>

<script setup lang="ts">
import { StarFilled, Star } from '@element-plus/icons-vue'

defineProps<{
  value: number
  label?: string
  showLabel?: boolean
  showValue?: boolean
}>()
</script>

<style scoped>
.rating-display {
  display: inline-flex;
  align-items: center;
  gap: 4px;
}

.stars {
  display: inline-flex;
  gap: 2px;
}

.stars .el-icon {
  font-size: 14px;
  color: #c0c4cc;
}

.stars .el-icon.active {
  color: #f7ba2a;
}

.value {
  font-size: 14px;
  font-weight: 500;
  color: #303133;
}

.label {
  font-size: 12px;
  color: #606266;
}
</style>
