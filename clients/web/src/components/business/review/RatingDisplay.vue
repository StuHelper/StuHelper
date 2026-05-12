<template>
  <div class="inline-flex items-center gap-2" :class="{ 'show-label': showLabel }">
    <div class="flex gap-0.5">
      <span
        class="transition-colors duration-fast ease-out"
        :style="{ color: ratingColor }"
      >
        <svg
          class="block"
          :class="iconSizeClass"
          viewBox="0 0 512 512"
          fill="currentColor"
          aria-hidden="true"
        >
          <path :d="facePath" />
        </svg>
      </span>
    </div>
    <span v-if="showLabel && label" class="text-text-secondary text-sm">{{ label }}</span>
    <span v-if="showCount && count !== undefined" class="text-text-muted text-xs">{{ t('review.rating.countSuffix', { count }) }}</span>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { getRatingColor } from '@/design-system/rating'
import { getRatingFacePath, normalizeRatingLevel } from '@/modules/review/ratingFaces'

const { t } = useI18n()

const props = withDefaults(defineProps<{
  value: number
  label?: string
  showLabel?: boolean
  showCount?: boolean
  count?: number
  size?: 'sm' | 'md' | 'lg'
}>(), {
  size: 'md',
  showLabel: false,
  showCount: false
})

const normalizedValue = computed(() => normalizeRatingLevel(props.value))
const ratingColor = computed(() => getRatingColor(normalizedValue.value))
const facePath = computed(() => getRatingFacePath(normalizedValue.value))

const iconSizeClass = computed(() => {
  switch (props.size) {
    case 'sm': return 'size-3.5'
    case 'lg': return 'size-6'
    default: return 'size-[18px]'
  }
})

</script>
