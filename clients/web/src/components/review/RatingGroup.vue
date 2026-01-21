<template>
  <div class="rating-group">
    <div class="rating-item" v-for="item in ratingItems" :key="item.key">
      <span class="label">{{ item.label }}</span>
      <div class="options">
        <span
          v-for="opt in options"
          :key="opt.value"
          :class="['option', { active: modelValue[item.key] === opt.value }]"
          @click="handleSelect(item.key, opt.value)"
        >
          {{ opt.icon }}
        </span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { RatingLevel } from '@/types/course'

interface RatingValues {
  recommend: RatingLevel
  content: RatingLevel
  workload: RatingLevel
  exam: RatingLevel
}

const props = defineProps<{
  modelValue: RatingValues
}>()

const emit = defineEmits<{
  'update:modelValue': [value: RatingValues]
}>()

const ratingItems = [
  { key: 'recommend' as const, label: '推荐指数' },
  { key: 'content' as const, label: '内容质量' },
  { key: 'workload' as const, label: '工作量' },
  { key: 'exam' as const, label: '考核难度' }
]

const options = [
  { value: 2 as RatingLevel, icon: '😍' },
  { value: 1 as RatingLevel, icon: '😊' },
  { value: 0 as RatingLevel, icon: '😐' },
  { value: -1 as RatingLevel, icon: '😕' },
  { value: -2 as RatingLevel, icon: '😫' }
]

const handleSelect = (key: keyof RatingValues, value: RatingLevel) => {
  emit('update:modelValue', { ...props.modelValue, [key]: value })
}
</script>

<style scoped>
.rating-group {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.rating-item {
  display: flex;
  align-items: center;
  gap: 12px;
}

.label {
  width: 80px;
  font-size: 14px;
  color: #606266;
}

.options {
  display: flex;
  gap: 8px;
}

.option {
  width: 36px;
  height: 36px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 20px;
  border-radius: 50%;
  cursor: pointer;
  transition: all 0.2s;
  background: #f5f7fa;
}

.option:hover {
  background: #e6e8eb;
  transform: scale(1.1);
}

.option.active {
  background: #409eff;
  transform: scale(1.2);
}
</style>
