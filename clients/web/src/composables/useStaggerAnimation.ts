/**
 * 列表入场动画 composable
 */
import { ref, onMounted } from 'vue'

export function useStaggerAnimation(baseDelay = 50) {
  const isVisible = ref(false)

  onMounted(() => {
    // 延迟触发动画，确保 DOM 已渲染
    requestAnimationFrame(() => {
      isVisible.value = true
    })
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
