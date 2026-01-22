<template>
  <div class="rating-group">
    <div v-if="loading" class="loading">
      <el-skeleton :rows="4" animated />
    </div>
    <template v-else>
      <div class="rating-item" v-for="dim in dimensions" :key="dim.key">
        <div class="label-wrapper">
          <span class="label">{{ dim.name }}</span>
          <el-tooltip v-if="dim.description" :content="dim.description" placement="top">
            <el-icon class="info-icon"><InfoFilled /></el-icon>
          </el-tooltip>
        </div>
        <div class="stars">
          <span
            v-for="star in 5"
            :key="star"
            :class="['star', { active: (modelValue[dim.key] || 0) >= star }]"
            @click="handleSelect(dim.key, star as RatingValue)"
            @mouseenter="hoverKey = dim.key; hoverValue = star"
            @mouseleave="hoverKey = ''; hoverValue = 0"
          >
            <el-icon>
              <StarFilled v-if="getStarState(dim.key, star)" />
              <Star v-else />
            </el-icon>
          </span>
          <span class="rating-text">{{ getRatingText(modelValue[dim.key]) }}</span>
        </div>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { StarFilled, Star, InfoFilled } from '@element-plus/icons-vue'
import { getRatingDimensions } from '@/api/course'
import type { RatingDimension, RatingValue } from '@/types/course'
import type { ReviewRatings } from '@/types/review'

const props = defineProps<{
  modelValue: ReviewRatings
}>()

const emit = defineEmits<{
  'update:modelValue': [value: ReviewRatings]
}>()

const loading = ref(true)
const dimensions = ref<RatingDimension[]>([])
const hoverKey = ref('')
const hoverValue = ref(0)

const ratingTexts: Record<number, string> = {
  1: '很差',
  2: '较差',
  3: '一般',
  4: '较好',
  5: '很好'
}

const getRatingText = (value: number | undefined) => {
  if (!value) return ''
  return ratingTexts[value] || ''
}

const getStarState = (key: string, star: number) => {
  if (hoverKey.value === key) {
    return star <= hoverValue.value
  }
  return star <= (props.modelValue[key] || 0)
}

const handleSelect = (key: string, value: RatingValue) => {
  emit('update:modelValue', { ...props.modelValue, [key]: value })
}

onMounted(async () => {
  try {
    const res = await getRatingDimensions()
    dimensions.value = res.data.filter(d => d.isActive)
  } catch (e) {
    console.error('Failed to load rating dimensions:', e)
  } finally {
    loading.value = false
  }
})

defineExpose({ dimensions })
</script>

<style scoped>
.rating-group {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.loading {
  padding: 8px 0;
}

.rating-item {
  display: flex;
  align-items: center;
  gap: 12px;
}

.label-wrapper {
  display: flex;
  align-items: center;
  gap: 4px;
  width: 100px;
}

.label {
  font-size: 14px;
  color: #606266;
}

.info-icon {
  font-size: 14px;
  color: #909399;
  cursor: help;
}

.stars {
  display: flex;
  align-items: center;
  gap: 4px;
}

.star {
  font-size: 24px;
  cursor: pointer;
  transition: all 0.2s;
  color: #c0c4cc;
}

.star:hover {
  transform: scale(1.1);
}

.star.active {
  color: #f7ba2a;
}

.rating-text {
  margin-left: 8px;
  font-size: 12px;
  color: #909399;
  min-width: 32px;
}
</style>
