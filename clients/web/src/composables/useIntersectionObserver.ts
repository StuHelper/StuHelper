/**
 * 滚动触发动画 composable
 */
import { ref, watch, onUnmounted, type Ref } from 'vue'

export function useIntersectionObserver(
  elementRef: Ref<HTMLElement | null>,
  options: IntersectionObserverInit = {}
) {
  const isVisible = ref(false)
  const hasBeenVisible = ref(false)
  let observer: IntersectionObserver | null = null
  // L-54: 防抖，避免条件渲染导致频繁重建观察器
  let rebuildTimer: ReturnType<typeof setTimeout> | null = null

  const defaultOptions: IntersectionObserverInit = {
    threshold: 0.1,
    rootMargin: '0px',
    ...options
  }

  const cleanup = () => {
    if (rebuildTimer) {
      clearTimeout(rebuildTimer)
      rebuildTimer = null
    }
    if (observer) {
      observer.disconnect()
      observer = null
    }
  }

  const setupObserver = (el: HTMLElement) => {
    if (observer) {
      observer.disconnect()
    }
    observer = new IntersectionObserver((entries) => {
      entries.forEach((entry) => {
        isVisible.value = entry.isIntersecting
        if (entry.isIntersecting) {
          hasBeenVisible.value = true
        }
      })
    }, defaultOptions)
    observer.observe(el)
  }

  // 监听 elementRef 变化，支持条件渲染元素后续可用
  watch(
    () => elementRef.value,
    (el) => {
      if (rebuildTimer) {
        clearTimeout(rebuildTimer)
        rebuildTimer = null
      }
      if (!el) {
        if (observer) {
          observer.disconnect()
          observer = null
        }
        return
      }
      // L-54: 防抖 50ms，避免频繁挂载/卸载时大量创建销毁
      rebuildTimer = setTimeout(() => {
        if (elementRef.value) {
          setupObserver(elementRef.value)
        }
      }, 50)
    },
    { immediate: true }
  )

  onUnmounted(cleanup)

  return {
    isVisible,
    hasBeenVisible
  }
}
