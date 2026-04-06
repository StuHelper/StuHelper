<script setup lang="ts">
/**
 * 滚动揭示动画组件
 * 元素进入视口时触发入场动画，支持多方向和自定义参数
 * - prefers-reduced-motion 时跳过动画，内容立即可见
 */
import { ref, computed, onMounted, onUnmounted } from 'vue'

interface Props {
  /** 动画延迟（ms） */
  delay?: number
  /** 动画时长（ms） */
  duration?: number
  /** 位移距离（px） */
  distance?: number
  /** 动画方向 */
  direction?: 'up' | 'down' | 'left' | 'right'
  /** 仅触发一次 */
  once?: boolean
  /** IntersectionObserver 阈值 */
  threshold?: number
}

const props = withDefaults(defineProps<Props>(), {
  delay: 0,
  duration: 500,
  distance: 24,
  direction: 'up',
  once: true,
  threshold: 0.15
})

const prefersReducedMotion = typeof window !== 'undefined'
  && window.matchMedia('(prefers-reduced-motion: reduce)').matches

const elementRef = ref<HTMLElement>()
const isVisible = ref(prefersReducedMotion)
let observer: IntersectionObserver | null = null

const offsets: Record<Props['direction'] & string, { x: number; y: number }> = {
  up: { x: 0, y: 1 },
  down: { x: 0, y: -1 },
  left: { x: 1, y: 0 },
  right: { x: -1, y: 0 }
}

const dir = computed(() => offsets[props.direction] ?? offsets.up)

onMounted(() => {
  if (prefersReducedMotion || !elementRef.value) return
  observer = new IntersectionObserver(
    (entries) => {
      for (const entry of entries) {
        if (entry.isIntersecting) {
          isVisible.value = true
          if (props.once && observer) {
            observer.unobserve(entry.target)
          }
        } else if (!props.once) {
          isVisible.value = false
        }
      }
    },
    { threshold: props.threshold }
  )
  observer.observe(elementRef.value)
})

onUnmounted(() => {
  observer?.disconnect()
})
</script>

<template>
  <div
    ref="elementRef"
    :style="prefersReducedMotion ? undefined : {
      opacity: isVisible ? 1 : 0,
      transform: isVisible
        ? 'translate3d(0, 0, 0)'
        : `translate3d(${dir.x * distance}px, ${dir.y * distance}px, 0)`,
      transition: `opacity ${duration}ms var(--ease-out) ${delay}ms, transform ${duration}ms var(--ease-out) ${delay}ms`,
      willChange: isVisible ? 'auto' : 'opacity, transform'
    }"
  >
    <slot />
  </div>
</template>
