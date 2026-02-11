<template>
  <div class="rating-group">
    <div v-if="loading" class="loading-state">
      <div v-for="i in 5" :key="i" class="skeleton-item">
        <div class="skeleton-label"></div>
        <div class="skeleton-stars"></div>
      </div>
    </div>

    <template v-else>
      <div
        v-for="dim in dimensions"
        :key="dim.key"
        class="rating-item"
        :class="{ 'has-value': modelValue[dim.key] }"
      >
        <div class="item-header">
          <span class="dim-name">{{ dim.name }}</span>
          <span v-if="dim.description" class="dim-desc">{{ dim.description }}</span>
        </div>

        <div class="stars-row">
          <div class="stars">
            <button
              v-for="star in 5"
              :key="star"
              type="button"
              class="star-btn"
              :class="{
                filled: getStarState(dim.key, star),
                hovered: hoverKey === dim.key && star <= hoverValue
              }"
              @click="handleSelect(dim.key, star as RatingValue)"
              @mouseenter="handleHover(dim.key, star)"
              @mouseleave="handleHoverEnd"
            >
              <svg viewBox="0 0 24 24" fill="currentColor">
                <path d="M12 2l3.09 6.26L22 9.27l-5 4.87 1.18 6.88L12 17.77l-6.18 3.25L7 14.14 2 9.27l6.91-1.01L12 2z"/>
              </svg>
            </button>
          </div>
          <Transition name="fade">
            <span v-if="modelValue[dim.key]" class="rating-label">
              {{ getRatingText(modelValue[dim.key]) }}
            </span>
          </Transition>
        </div>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { getRatingDimensions } from '@/api/course'
import type { RatingDimension, RatingValue } from '@/types/course'
import type { ReviewRatings } from '@/types/review'

const { t } = useI18n()

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

const getRatingText = (value: number | undefined) => {
  if (!value) return ''
  const keys: Record<number, string> = {
    1: 'review.rating.veryBad',
    2: 'review.rating.bad',
    3: 'review.rating.average',
    4: 'review.rating.good',
    5: 'review.rating.excellent'
  }
  return keys[value] ? t(keys[value]) : ''
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

const handleHover = (key: string, value: number) => {
  hoverKey.value = key
  hoverValue.value = value
}

const handleHoverEnd = () => {
  hoverKey.value = ''
  hoverValue.value = 0
}

onMounted(async () => {
  try {
    const res = await getRatingDimensions()
    dimensions.value = res.data.filter(d => d.isActive)
  } catch {
    // 评分维度加载失败，UI 显示空状态
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
  gap: var(--space-5);
}

/* Loading State */
.loading-state {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.skeleton-item {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.skeleton-label {
  width: 80px;
  height: 16px;
  background: var(--bg-elevated);
  border-radius: var(--radius-sm);
  animation: pulse 1.5s ease-in-out infinite;
}

.skeleton-stars {
  width: 160px;
  height: 28px;
  background: var(--bg-elevated);
  border-radius: var(--radius-sm);
  animation: pulse 1.5s ease-in-out infinite;
}

/* Rating Item */
.rating-item {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  padding: var(--space-3) 0;
  border-bottom: 1px solid var(--border);
  transition: all var(--duration-base) var(--ease-out);
}

.rating-item:last-child {
  border-bottom: none;
}

.item-header {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
}

.dim-name {
  font-weight: var(--weight-medium);
  color: var(--text-primary);
  font-size: var(--text-sm);
}

.dim-desc {
  font-size: var(--text-sm);
  color: var(--text-muted);
}

/* Stars Row */
.stars-row {
  display: flex;
  align-items: center;
  gap: var(--space-4);
}

.stars {
  display: flex;
  gap: var(--space-1);
}

.star-btn {
  padding: var(--space-1);
  color: var(--text-muted);
  transition: all var(--duration-fast) var(--ease-out);
}

.star-btn svg {
  width: 28px;
  height: 28px;
  display: block;
}

.star-btn:hover {
  transform: scale(1.15);
}

.star-btn.hovered,
.star-btn.filled {
  color: var(--brand-accent);
}

.star-btn.filled {
  filter: none;
}

.rating-label {
  font-size: var(--text-sm);
  color: var(--brand-accent);
  font-weight: var(--weight-medium);
}

/* Transitions */
.fade-enter-active,
.fade-leave-active {
  transition: opacity var(--duration-fast);
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}

/* Responsive */
@media (min-width: 640px) {
  .rating-item {
    flex-direction: row;
    align-items: center;
    justify-content: space-between;
  }

  .item-header {
    flex: 1;
  }
}
</style>
