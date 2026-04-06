/**
 * 主题状态管理
 * 支持亮色/暗色/跟随系统三种模式
 */
import { defineStore } from 'pinia'
import { ref, computed, watch, onScopeDispose } from 'vue'

export type ThemeMode = 'light' | 'dark' | 'system'

const STORAGE_KEY = 'theme-mode'
const VALID_MODES: ThemeMode[] = ['light', 'dark', 'system']

// 安全读取 localStorage，隐私模式下降级
function safeGetStorageItem(key: string): string | null {
  try {
    return localStorage.getItem(key)
  } catch {
    return null
  }
}

// 安全写入 localStorage
function safeSetStorageItem(key: string, value: string): void {
  try {
    localStorage.setItem(key, value)
  } catch {
    // 隐私模式或存储已满，降级为内存存储
  }
}

export const useThemeStore = defineStore('theme', () => {
  // 安全读取，校验值有效性
  const stored = safeGetStorageItem(STORAGE_KEY)
  const mode = ref<ThemeMode>(
    stored && VALID_MODES.includes(stored as ThemeMode)
      ? (stored as ThemeMode)
      : 'system'
  )

  const systemDark = ref(
    window.matchMedia('(prefers-color-scheme: dark)').matches
  )

  const isDark = computed(() => {
    if (mode.value === 'system') return systemDark.value
    return mode.value === 'dark'
  })

  const resolvedTheme = computed(() => (isDark.value ? 'dark' : 'light'))

  function setMode(newMode: ThemeMode) {
    mode.value = newMode
    safeSetStorageItem(STORAGE_KEY, newMode)
  }

  // 合并为单一 watcher 监听 computed isDark，避免竞态
  function applyTheme() {
    const root = document.documentElement
    if (mode.value === 'system') {
      root.removeAttribute('data-theme')
    } else {
      root.setAttribute('data-theme', mode.value)
    }
    root.classList.toggle('dark', isDark.value)
  }

  // 监听系统主题变化，使用具名函数引用确保 add/remove 一致
  const mql = window.matchMedia('(prefers-color-scheme: dark)')
  const onMqlChange = (e: MediaQueryListEvent) => {
    systemDark.value = e.matches
  }
  mql.addEventListener('change', onMqlChange)
  onScopeDispose(() => {
    mql.removeEventListener('change', onMqlChange)
  })

  // 监听 isDark 而非分别监听 mode 和 systemDark
  watch(isDark, applyTheme, { immediate: true })
  // mode 变化时也需要更新 data-theme 属性
  watch(mode, applyTheme)

  return {
    mode,
    isDark,
    resolvedTheme,
    setMode
  }
})
