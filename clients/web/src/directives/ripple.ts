/**
 * 涟漪效果指令
 */
import type { Directive } from 'vue'

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

export const vRipple: Directive = {
  mounted(el: HTMLElement) {
    el.style.position = 'relative'
    el.style.overflow = 'hidden'
    el.addEventListener('click', createRipple)
  },
  unmounted(el: HTMLElement) {
    el.removeEventListener('click', createRipple)
  }
}

// 需要在全局样式中添加以下 CSS
// .ripple {
//   position: absolute;
//   border-radius: 50%;
//   transform: scale(0);
//   animation: ripple 0.6s linear;
//   background: rgba(255, 255, 255, 0.3);
//   pointer-events: none;
// }
// @keyframes ripple {
//   to {
//     transform: scale(4);
//     opacity: 0;
//   }
// }
