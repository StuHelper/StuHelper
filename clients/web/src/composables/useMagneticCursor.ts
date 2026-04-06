/**
 * 磁性光标效果 composable
 * 鼠标靠近元素时，元素向鼠标方向轻微吸引
 * - 自动检测 prefers-reduced-motion，禁用时返回静态样式
 * - 自动检测触屏设备，仅对有鼠标的设备生效
 */
import { ref, computed, onMounted, onUnmounted, type Ref } from 'vue'

interface MagneticOptions {
  /** 吸引强度 0-1 */
  strength?: number
  /** 生效半径（px） */
  radius?: number
}

const IDENTITY_STYLE = { transform: 'none', transition: 'none' }

export function useMagneticCursor(
  elementRef: Ref<HTMLElement | undefined>,
  options: MagneticOptions = {}
) {
  const { strength = 0.3, radius = 100 } = options

  const offsetX = ref(0)
  const offsetY = ref(0)
  const isNear = ref(false)

  const prefersReducedMotion = typeof window !== 'undefined'
    && window.matchMedia('(prefers-reduced-motion: reduce)').matches
  const hasHover = typeof window !== 'undefined'
    && window.matchMedia('(hover: hover)').matches

  const disabled = prefersReducedMotion || !hasHover

  let rafId: number | null = null
  let mountedEl: HTMLElement | null = null

  function handleMouseMove(e: MouseEvent) {
    if (!elementRef.value) return
    if (rafId !== null) return
    rafId = requestAnimationFrame(() => {
      rafId = null
      if (!elementRef.value) return

      const rect = elementRef.value.getBoundingClientRect()
      const centerX = rect.left + rect.width / 2
      const centerY = rect.top + rect.height / 2
      const dx = e.clientX - centerX
      const dy = e.clientY - centerY
      const distance = Math.sqrt(dx * dx + dy * dy)

      if (distance < radius) {
        isNear.value = true
        const pull = 1 - distance / radius
        offsetX.value = dx * pull * strength
        offsetY.value = dy * pull * strength
      } else if (isNear.value) {
        isNear.value = false
        offsetX.value = 0
        offsetY.value = 0
      }
    })
  }

  function handleMouseLeave() {
    isNear.value = false
    offsetX.value = 0
    offsetY.value = 0
  }

  const style = computed(() => {
    if (disabled) return IDENTITY_STYLE
    return {
      transform: `translate(${offsetX.value}px, ${offsetY.value}px)`,
      transition: isNear.value
        ? 'transform 0.15s cubic-bezier(0.34, 1.56, 0.64, 1)'
        : 'transform 0.4s cubic-bezier(0.16, 1, 0.3, 1)'
    }
  })

  onMounted(() => {
    if (disabled) return
    mountedEl = elementRef.value ?? null
    document.addEventListener('mousemove', handleMouseMove, { passive: true })
    mountedEl?.addEventListener('mouseleave', handleMouseLeave)
  })

  onUnmounted(() => {
    document.removeEventListener('mousemove', handleMouseMove)
    mountedEl?.removeEventListener('mouseleave', handleMouseLeave)
    if (rafId !== null) cancelAnimationFrame(rafId)
    mountedEl = null
  })

  return { offsetX, offsetY, isNear, style }
}
