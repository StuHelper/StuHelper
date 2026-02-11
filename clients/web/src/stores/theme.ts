/**
 * 主题状态管理
 * 支持亮色/暗色/跟随系统三种模式
 */
import { defineStore } from 'pinia'
import { ref, computed, watch, onScopeDispose } from 'vue'

export type ThemeMode = 'light' | 'dark' | 'system'

const STORAGE_KEY = 'theme-mode'

export const useThemeStore = defineStore('theme', () => {
  const mode = ref<ThemeMode>(
    (localStorage.getItem(STORAGE_KEY) as ThemeMode) || 'system'
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
    localStorage.setItem(STORAGE_KEY, newMode)
  }

  function applyTheme() {
    const root = document.documentElement
    if (mode.value === 'system') {
      root.removeAttribute('data-theme')
    } else {
      root.setAttribute('data-theme', mode.value)
    }
  }

  // 监听系统主题变化
  const mql = window.matchMedia('(prefers-color-scheme: dark)')
  const onMqlChange = (e: MediaQueryListEvent) => {
    systemDark.value = e.matches
  }
  mql.addEventListener('change', onMqlChange)
  onScopeDispose(() => {
    mql.removeEventListener('change', onMqlChange)
  })

  // 监听 mode 变化并应用
  watch(mode, applyTheme, { immediate: true })
  watch(systemDark, () => {
    if (mode.value === 'system') applyTheme()
  })

  return {
    mode,
    isDark,
    resolvedTheme,
    setMode
  }
})
