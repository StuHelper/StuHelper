/**
 * Toast 通知系统
 * 轻量级消息提示
 */
import { ref, onScopeDispose, type Ref } from 'vue'

export type ToastType = 'success' | 'error' | 'info' | 'warning'

export interface ToastItem {
  id: number
  type: ToastType
  message: string
  duration: number
}

const toasts = ref<ToastItem[]>([])
const timers = new Map<number, ReturnType<typeof setTimeout>>()
// H-51: 使用安全的自增 ID，避免全局变量在多实例场景下冲突
let nextID = 0

export interface UseToastReturn {
  toasts: Ref<ToastItem[]>
  show: (message: string, type?: ToastType, duration?: number) => number
  remove: (id: number) => void
  success: (message: string, duration?: number) => number
  error: (message: string, duration?: number) => number
  info: (message: string, duration?: number) => number
  warning: (message: string, duration?: number) => number
  clearAll: () => void
}

export function useToast(): UseToastReturn {
  // M-73: 跟踪当前实例创建的 toast ID，unmount 时清理
  const instanceTimerIDs: number[] = []

  function show(message: string, type: ToastType = 'info', duration = 3000) {
    const id = ++nextID
    toasts.value.push({ id, type, message, duration })

    if (duration > 0) {
      const timer = setTimeout(() => {
        timers.delete(id)
        remove(id)
      }, duration)
      timers.set(id, timer)
      instanceTimerIDs.push(id)
    }

    return id
  }

  function remove(id: number) {
    // H-51: 移除 toast 时同步清理 timer 条目
    const timer = timers.get(id)
    if (timer) {
      clearTimeout(timer)
      timers.delete(id)
    }
    const idx = toasts.value.findIndex((t) => t.id === id)
    if (idx !== -1) toasts.value.splice(idx, 1)
  }

  // M-73: 清理所有活跃 timer
  function clearAll() {
    for (const [id, timer] of timers) {
      clearTimeout(timer)
      timers.delete(id)
    }
    toasts.value = []
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

  // M-73: scope 销毁时清理当前实例创建的 timer
  onScopeDispose(() => {
    for (const id of instanceTimerIDs) {
      const timer = timers.get(id)
      if (timer) {
        clearTimeout(timer)
        timers.delete(id)
      }
    }
  })

  return {
    toasts,
    show,
    remove,
    success,
    error,
    info,
    warning,
    clearAll
  }
}
