/**
 * 3D 卡片倾斜效果 composable
 * 鼠标移动时卡片跟随倾斜，离开时回弹
 * - 自动检测 prefers-reduced-motion，禁用时返回静态样式
 * - 自动检测触屏设备，仅对有鼠标的设备生效
 */
import { ref, computed, onMounted, onUnmounted, type Ref } from 'vue'

interface TiltOptions {
  /** 最大倾斜角度（度） */
  maxTilt?: number
  /** 3D 透视距离（px） */
  perspective?: number
  /** hover 时的缩放倍数 */
  scale?: number
  /** 回弹过渡时长（ms） */
  speed?: number
}

const IDENTITY_STYLE = {
  transform: 'none',
  transition: 'none',
  willChange: 'auto' as const,
  transformStyle: 'preserve-3d' as const
}

export function use3DTilt(
  elementRef: Ref<HTMLElement | undefined>,
  options: TiltOptions = {}
) {
  const {
    maxTilt = 8,
    perspective = 800,
    scale = 1.02,
    speed = 400
  } = options

  const tiltX = ref(0)
  const tiltY = ref(0)
  const isHovering = ref(false)

  // 检测用户偏好和设备能力
  const prefersReducedMotion = typeof window !== 'undefined'
    && window.matchMedia('(prefers-reduced-motion: reduce)').matches
  const hasHover = typeof window !== 'undefined'
    && window.matchMedia('(hover: hover)').matches

  const disabled = prefersReducedMotion || !hasHover

  function handleMouseMove(e: MouseEvent) {
    if (!elementRef.value) return
    const rect = elementRef.value.getBoundingClientRect()
    const centerX = rect.left + rect.width / 2
    const centerY = rect.top + rect.height / 2
    const mouseX = e.clientX - centerX
    const mouseY = e.clientY - centerY

    tiltX.value = (mouseY / (rect.height / 2)) * -maxTilt
    tiltY.value = (mouseX / (rect.width / 2)) * maxTilt
  }

  function handleMouseEnter() {
    isHovering.value = true
  }

  function handleMouseLeave() {
    isHovering.value = false
    tiltX.value = 0
    tiltY.value = 0
  }

  const style = computed(() => {
    if (disabled) return IDENTITY_STYLE
    return {
      transform: isHovering.value
        ? `perspective(${perspective}px) rotateX(${tiltX.value}deg) rotateY(${tiltY.value}deg) scale3d(${scale}, ${scale}, ${scale})`
        : `perspective(${perspective}px) rotateX(0deg) rotateY(0deg) scale3d(1, 1, 1)`,
      // hover 中跟随鼠标应即时响应，离开时弹簧回弹
      transition: isHovering.value
        ? 'transform 80ms linear'
        : `transform ${speed}ms cubic-bezier(0.34, 1.56, 0.64, 1)`,
      willChange: isHovering.value ? 'transform' : ('auto' as const),
      transformStyle: 'preserve-3d' as const
    }
  })

  // 捕获 mounted 时的 DOM 元素引用，确保 cleanup 可靠
  let mountedEl: HTMLElement | null = null

  onMounted(() => {
    if (disabled) return
    const el = elementRef.value
    if (!el) return
    mountedEl = el
    el.addEventListener('mousemove', handleMouseMove)
    el.addEventListener('mouseenter', handleMouseEnter)
    el.addEventListener('mouseleave', handleMouseLeave)
  })

  onUnmounted(() => {
    const el = mountedEl
    if (!el) return
    el.removeEventListener('mousemove', handleMouseMove)
    el.removeEventListener('mouseenter', handleMouseEnter)
    el.removeEventListener('mouseleave', handleMouseLeave)
    mountedEl = null
  })

  return { tiltX, tiltY, isHovering, style }
}
