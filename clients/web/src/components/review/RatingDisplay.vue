<template>
  <div class="rating-display" :class="[`size-${size}`, { 'show-label': showLabel }]">
    <div class="stars">
      <span
        v-for="star in 5"
        :key="star"
        class="star"
        :class="{ filled: star <= Math.round(value) }"
      >
        <svg viewBox="0 0 24 24" fill="currentColor">
          <path d="M12 2l3.09 6.26L22 9.27l-5 4.87 1.18 6.88L12 17.77l-6.18 3.25L7 14.14 2 9.27l6.91-1.01L12 2z"/>
        </svg>
      </span>
    </div>
    <span v-if="showValue" class="value">{{ value.toFixed(1) }}</span>
    <span v-if="showLabel && label" class="label">{{ label }}</span>
    <span v-if="showCount && count !== undefined" class="count">{{ t('review.rating.countSuffix', { count }) }}</span>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

withDefaults(defineProps<{
  value: number
  label?: string
  showLabel?: boolean
  showValue?: boolean
  showCount?: boolean
  count?: number
  size?: 'sm' | 'md' | 'lg'
}>(), {
  size: 'md',
  showValue: false,
  showLabel: false,
  showCount: false
})
</script>

<style scoped>
.rating-display {
  display: inline-flex;
  align-items: center;
  gap: var(--space-2);
}

.stars {
  display: flex;
  gap: 2px;
}

.star {
  color: var(--text-muted);
  transition: color var(--duration-fast) var(--ease-out);
}

.star.filled {
  color: var(--accent);
}

.star svg {
  display: block;
}

/* Sizes */
.size-sm .star svg { width: 14px; height: 14px; }
.size-md .star svg { width: 18px; height: 18px; }
.size-lg .star svg { width: 24px; height: 24px; }

.value {
  font-weight: 600;
  color: var(--accent);
}

.size-sm .value { font-size: var(--text-sm); }
.size-md .value { font-size: var(--text-base); }
.size-lg .value { font-size: var(--text-lg); }

.label {
  color: var(--text-secondary);
  font-size: var(--text-sm);
}

.count {
  color: var(--text-muted);
  font-size: var(--text-xs);
}
</style>
