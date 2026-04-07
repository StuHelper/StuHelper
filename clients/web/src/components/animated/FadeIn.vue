<script setup lang="ts">
import { useMotion } from '@vueuse/motion'
import { ref } from 'vue'
import { useMediaQuery } from '@vueuse/core'

interface Props {
  delay?: number
}

const props = withDefaults(defineProps<Props>(), {
  delay: 0
})

const prefersReducedMotion = useMediaQuery('(prefers-reduced-motion: reduce)')

const target = ref<HTMLElement>()

useMotion(target, prefersReducedMotion.value
  ? {
      initial: { opacity: 1, y: 0 },
      enter: { opacity: 1, y: 0 }
    }
  : {
      initial: { opacity: 0, y: 20 },
      enter: {
        opacity: 1,
        y: 0,
        transition: {
          delay: props.delay,
          duration: 300
        }
      }
    }
)
</script>

<template>
  <div ref="target">
    <slot />
  </div>
</template>
