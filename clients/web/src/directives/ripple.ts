/**
 * 涟漪效果指令
 */
import type { Directive } from 'vue'

// M-10: 使用 WeakMap 跟踪每个元素的事件处理器，确保 unmounted 时正确清理
const rippleHandlers = new WeakMap<HTMLElement, (event: MouseEvent) => void>()

const createRipple = (event: MouseEvent) => {
  const button = event.currentTarget as HTMLElement
  const rect = button.getBoundingClientRect()

  const circle = document.createElement('span')
  const diameter = Math.max(rect.width, rect.height)
  const radius = diameter / 2

  circle.style.width = circle.style.height = `${diameter}px`
  circle.style.left = `${event.clientX - rect.left - radius}px`
  circle.style.top = `${event.clientY - rect.top - radius}px`
  circle.classList.add('ripple')

  const existingRipple = button.querySelector('.ripple')
  if (existingRipple) {
    existingRipple.remove()
  }

  button.appendChild(circle)

  // 动画结束后移除
  circle.addEventListener('animationend', () => {
    circle.remove()
  }, { once: true })
}

// M-122: 使用 WeakSet 跟踪样式注入状态，支持多 Vue 实例
const injectedDocuments = new WeakSet<Document>()

function injectRippleStyle() {
  if (injectedDocuments.has(document)) return
  injectedDocuments.add(document)
  const style = document.createElement('style')
  style.textContent = `
    .ripple {
      position: absolute;
      border-radius: 50%;
      transform: scale(0);
      animation: ripple 0.6s linear;
      background: rgba(255, 255, 255, 0.3);
      pointer-events: none;
    }
    @keyframes ripple {
      to { transform: scale(4); opacity: 0; }
    }
  `
  document.head.appendChild(style)
}

export const vRipple: Directive = {
  mounted(el: HTMLElement) {
    injectRippleStyle()
    // 仅在未设置 position 时添加，避免覆盖已有布局
    const computed = getComputedStyle(el)
    if (computed.position === 'static') {
      el.style.position = 'relative'
    }
    el.style.overflow = 'hidden'
    // M-10: 保存处理器引用，确保 unmounted 时能正确移除
    rippleHandlers.set(el, createRipple)
    el.addEventListener('click', createRipple)
  },
  unmounted(el: HTMLElement) {
    const handler = rippleHandlers.get(el)
    if (handler) {
      el.removeEventListener('click', handler)
      rippleHandlers.delete(el)
    }
    // M-10: 清理残留的 ripple 元素
    el.querySelectorAll('.ripple').forEach(r => r.remove())
  }
}
