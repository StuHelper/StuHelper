/**
 * 列表入场动画 composable
 */
import { ref, onMounted, onUnmounted } from 'vue'

export function useStaggerAnimation(baseDelay = 50) {
  const isVisible = ref(false)
  let rafId: number | null = null

  onMounted(() => {
    // 延迟触发动画，确保 DOM 已渲染
    rafId = requestAnimationFrame(() => {
      isVisible.value = true
      rafId = null
    })
  })

  onUnmounted(() => {
    if (rafId !== null) {
      cancelAnimationFrame(rafId)
    }
  })

  const getDelay = (index: number) => `${index * baseDelay}ms`

  const getStyle = (index: number) => ({
    animationDelay: getDelay(index)
  })

  return {
    isVisible,
    getDelay,
    getStyle
  }
}
