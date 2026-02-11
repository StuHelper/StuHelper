/**
 * Toast 通知系统
 * 轻量级消息提示
 */
import { ref } from 'vue'

export type ToastType = 'success' | 'error' | 'info' | 'warning'

export interface ToastItem {
  id: number
  type: ToastType
  message: string
  duration: number
}

const toasts = ref<ToastItem[]>([])
let nextID = 0

export function useToast() {
  function show(message: string, type: ToastType = 'info', duration = 3000) {
    const id = ++nextID
    toasts.value.push({ id, type, message, duration })

    if (duration > 0) {
      setTimeout(() => remove(id), duration)
    }

    return id
  }

  function remove(id: number) {
    const idx = toasts.value.findIndex((t) => t.id === id)
    if (idx !== -1) toasts.value.splice(idx, 1)
  }

  function success(message: string, duration?: number) {
    return show(message, 'success', duration)
  }

  function error(message: string, duration?: number) {
    return show(message, 'error', duration)
  }

  function info(message: string, duration?: number) {
    return show(message, 'info', duration)
  }

  function warning(message: string, duration?: number) {
    return show(message, 'warning', duration)
  }

  return {
    toasts,
    show,
    remove,
    success,
    error,
    info,
    warning
  }
}
