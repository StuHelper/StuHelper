/**
 * 命令面板状态管理
 * Cmd+K / Ctrl+K 触发全局搜索
 */
import { onScopeDispose, ref } from 'vue'

const isOpen = ref(false)
const searchQuery = ref('')

// 单例 + 引用计数，确保全局监听器不泄漏
let listenerCount = 0
let keydownHandler: ((e: KeyboardEvent) => void) | null = null

function open() {
  isOpen.value = true
  searchQuery.value = ''
}

function close() {
  isOpen.value = false
  searchQuery.value = ''
}

function toggle() {
  if (isOpen.value) close()
  else open()
}

function handleKeydown(e: KeyboardEvent) {
  if (e.defaultPrevented) return
  const key = e.key.toLowerCase()

  const target = e.target as HTMLElement | null
  const isEditable = target instanceof HTMLInputElement
    || target instanceof HTMLTextAreaElement
    || target?.isContentEditable === true

  if ((e.metaKey || e.ctrlKey) && key === 'k') {
    if (isEditable) return
    e.preventDefault()
    toggle()
  }
  if (key === 'escape' && isOpen.value) {
    e.preventDefault()
    close()
  }
}

function addListener() {
  if (listenerCount === 0) {
    keydownHandler = handleKeydown
    document.addEventListener('keydown', keydownHandler)
  }
  listenerCount++
}

function removeListener() {
  listenerCount--
  if (listenerCount <= 0 && keydownHandler) {
    document.removeEventListener('keydown', keydownHandler)
    keydownHandler = null
    listenerCount = 0
  }
}

export function useCommandPalette() {
  addListener()
  onScopeDispose(removeListener)

  return {
    isOpen,
    searchQuery,
    open,
    close,
    toggle
  }
}
