<script setup lang="ts">
import { ref, watch, onMounted, onUnmounted } from 'vue'

interface Props {
  target: number
  duration?: number
}

const props = withDefaults(defineProps<Props>(), {
  duration: 2000
})

const current = ref(0)
const elementRef = ref<HTMLElement>()
let observer: IntersectionObserver | null = null
let animationId: number | null = null
let hasAnimated = false

const easeOutQuad = (t: number): number => t * (2 - t)

const animate = () => {
  if (hasAnimated) return
  hasAnimated = true

  const startTime = performance.now()
  const startValue = 0
  const endValue = props.target

  const step = (currentTime: number) => {
    const elapsed = currentTime - startTime
    const progress = Math.min(elapsed / props.duration, 1)
    const easedProgress = easeOutQuad(progress)

    current.value = Math.round(startValue + (endValue - startValue) * easedProgress)

    if (progress < 1) {
      animationId = requestAnimationFrame(step)
    }
  }

  animationId = requestAnimationFrame(step)
}

onMounted(() => {
  if (!elementRef.value) return

  observer = new IntersectionObserver(
    (entries) => {
      if (entries[0].isIntersecting) {
        animate()
      }
    },
    { threshold: 0.1 }
  )

  observer.observe(elementRef.value)
})

onUnmounted(() => {
  if (observer) observer.disconnect()
  if (animationId) cancelAnimationFrame(animationId)
})

watch(() => props.target, () => {
  hasAnimated = false
  current.value = 0
  animate()
})
</script>

<template>
  <span ref="elementRef" class="count-up">{{ current }}</span>
</template>

<style scoped>
.count-up {
  font-variant-numeric: tabular-nums;
}
</style>
