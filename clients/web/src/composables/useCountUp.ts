/**
 * 数字增长动画 composable
 */
import { ref, watch, onMounted } from 'vue'

export function useCountUp(
  target: number | (() => number),
  duration = 1000
) {
  const current = ref(0)
  const isAnimating = ref(false)

  const animate = (from: number, to: number) => {
    if (from === to) return

    isAnimating.value = true
    const startTime = performance.now()
    const diff = to - from

    const step = (currentTime: number) => {
      const elapsed = currentTime - startTime
      const progress = Math.min(elapsed / duration, 1)

      // easeOutQuart 缓动函数
      const eased = 1 - Math.pow(1 - progress, 4)
      current.value = Math.round(from + diff * eased)

      if (progress < 1) {
        requestAnimationFrame(step)
      } else {
        isAnimating.value = false
      }
    }

    requestAnimationFrame(step)
  }

  onMounted(() => {
    const targetValue = typeof target === 'function' ? target() : target
    animate(0, targetValue)
  })

  // 监听目标值变化
  if (typeof target === 'function') {
    watch(target, (newVal, oldVal) => {
      animate(oldVal || 0, newVal)
    })
  }

  return {
    current,
    isAnimating,
    animate
  }
}
