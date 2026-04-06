/**
 * 数字增长动画 composable
 */
import { ref, watch, onMounted, onUnmounted } from 'vue'

export function useCountUp(
  target: number | (() => number),
  duration = 1000
) {
  const current = ref(0)
  const isAnimating = ref(false)
  let rafId = 0

  const animate = (from: number, to: number) => {
    if (from === to) return

    // 取消上一次未完成的动画
    if (rafId) {
      cancelAnimationFrame(rafId)
      rafId = 0
    }

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
        rafId = requestAnimationFrame(step)
      } else {
        rafId = 0
        isAnimating.value = false
      }
    }

    rafId = requestAnimationFrame(step)
  }

  // watch 移入 onMounted 内，确保仅在组件挂载后才监听变化
  onMounted(() => {
    const targetValue = typeof target === 'function' ? target() : target
    animate(0, targetValue)

    if (typeof target === 'function') {
      watch(target, (newVal, oldVal) => {
        animate(oldVal || 0, newVal)
      })
    }
  })

  onUnmounted(() => {
    if (rafId) {
      cancelAnimationFrame(rafId)
      rafId = 0
    }
  })

  return {
    current,
    isAnimating,
    animate
  }
}
