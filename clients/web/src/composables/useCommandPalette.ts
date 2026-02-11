/**
 * 命令面板状态管理
 * Cmd+K / Ctrl+K 触发全局搜索
 */
import { ref, onMounted, onUnmounted } from 'vue'

const isOpen = ref(false)
const searchQuery = ref('')

export function useCommandPalette() {
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
    if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
      e.preventDefault()
      toggle()
    }
    if (e.key === 'Escape' && isOpen.value) {
      e.preventDefault()
      close()
    }
  }

  onMounted(() => {
    document.addEventListener('keydown', handleKeydown)
  })

  onUnmounted(() => {
    document.removeEventListener('keydown', handleKeydown)
  })

  return {
    isOpen,
    searchQuery,
    open,
    close,
    toggle
  }
}
