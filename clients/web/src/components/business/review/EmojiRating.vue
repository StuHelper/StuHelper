<template>
  <div class="inline-flex items-center gap-1.5">
    <span
      class="inline-flex items-center justify-center transition-transform duration-200"
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
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { getRatingColor } from '@/design-system/rating'
import { getRatingFacePath, normalizeRatingLevel } from '@/modules/review/ratingFaces'

const props = withDefaults(defineProps<{
  value: number
  size?: 'sm' | 'md' | 'lg'
}>(), {
  size: 'md',
})

const normalizedValue = computed(() => normalizeRatingLevel(props.value))
const ratingColor = computed(() => getRatingColor(normalizedValue.value))
const facePath = computed(() => getRatingFacePath(normalizedValue.value))

const iconSizeClass = computed(() => {
  switch (props.size) {
    case 'sm': return 'size-4'
    case 'lg': return 'size-7'
    default: return 'size-5'
  }
})
</script>
