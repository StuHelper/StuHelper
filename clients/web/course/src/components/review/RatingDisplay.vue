<template>
  <div class="inline-flex items-center gap-2" :class="{ 'show-label': showLabel }">
    <div class="flex gap-0.5">
      <span
        v-for="star in 5"
        :key="star"
        class="star transition-colors duration-fast ease-out"
        :class="[`size-${size}`, star <= Math.round(value) ? '!text-accent' : 'text-text-muted']"
      >
        <Star :size="starSize" fill="currentColor" />
      </span>
    </div>
    <span v-if="showValue" class="font-semibold text-accent" :class="valueSizeClass">{{ value.toFixed(1) }}</span>
    <span v-if="showLabel && label" class="text-text-secondary text-sm">{{ label }}</span>
    <span v-if="showCount && count !== undefined" class="text-text-muted text-xs">{{ t('review.rating.countSuffix', { count }) }}</span>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { Star } from 'lucide-vue-next'

const { t } = useI18n()

const props = withDefaults(defineProps<{
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

const starSize = computed(() => {
  switch (props.size) {
    case 'sm': return 14
    case 'lg': return 24
    default: return 18
  }
})

const valueSizeClass = computed(() => {
  switch (props.size) {
    case 'sm': return 'text-sm'
    case 'lg': return 'text-lg'
    default: return 'text-base'
  }
})
</script>
