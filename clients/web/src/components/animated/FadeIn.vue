<script setup lang="ts">
import { useMotion } from '@vueuse/motion'
import { ref } from 'vue'

interface Props {
  delay?: number
}

const props = withDefaults(defineProps<Props>(), {
  delay: 0
})

const target = ref<HTMLElement>()

useMotion(target, {
  initial: { opacity: 0, y: 20 },
  enter: {
    opacity: 1,
    y: 0,
    transition: {
      delay: props.delay,
      duration: 300
    }
  }
})
</script>

<template>
  <div ref="target">
    <slot />
  </div>
</template>
