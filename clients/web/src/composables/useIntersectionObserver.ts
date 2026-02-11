/**
 * 滚动触发动画 composable
 */
import { ref, onMounted, onUnmounted, type Ref } from 'vue'

export function useIntersectionObserver(
  elementRef: Ref<HTMLElement | null>,
  options: IntersectionObserverInit = {}
) {
  const isVisible = ref(false)
  const hasBeenVisible = ref(false)
  let observer: IntersectionObserver | null = null

  const defaultOptions: IntersectionObserverInit = {
    threshold: 0.1,
    rootMargin: '0px',
    ...options
  }

  onMounted(() => {
    if (!elementRef.value) return

    observer = new IntersectionObserver((entries) => {
      entries.forEach((entry) => {
        isVisible.value = entry.isIntersecting
        if (entry.isIntersecting) {
          hasBeenVisible.value = true
        }
      })
    }, defaultOptions)

    observer.observe(elementRef.value)
  })

  onUnmounted(() => {
    observer?.disconnect()
  })

  return {
    isVisible,
    hasBeenVisible
  }
}
