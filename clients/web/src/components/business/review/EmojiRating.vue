<template>
  <div class="inline-flex items-center gap-1.5">
    <span
      class="inline-flex items-center justify-center transition-transform duration-200"
      :style="{ color: ratingColor }"
    >
      <component :is="ratingIcon" :size="iconSize" :stroke-width="2.5" />
    </span>
    <span
      v-if="showValue"
      class="text-xs font-mono font-semibold"
      :style="{ color: ratingColor }"
    >
      {{ value }}
    </span>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { Angry, Frown, Meh, Smile, SmilePlus } from 'lucide-vue-next'
import { getRatingColor } from '@/design-system/rating'

const props = withDefaults(defineProps<{
  value: number
  showValue?: boolean
  size?: 'sm' | 'md' | 'lg'
}>(), {
  showValue: false,
  size: 'md',
})

const ratingColor = computed(() => getRatingColor(props.value))

const ratingIcon = computed(() => {
  switch (props.value) {
    case 1: return Angry
    case 2: return Frown
    case 3: return Meh
    case 4: return Smile
    case 5: return SmilePlus
    default: return Meh
  }
})

const iconSize = computed(() => {
  switch (props.size) {
    case 'sm': return 16
    case 'lg': return 28
    default: return 20
  }
})
</script>
