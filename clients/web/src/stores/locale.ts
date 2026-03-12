/**
 * 语言状态管理
 */
import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import i18n, { SUPPORTED_LOCALES, DEFAULT_LOCALE, LOCALE_STORAGE_KEY, type SupportedLocale } from '@/i18n'
import { updateLocaleMeta } from '@/composables/usePageMeta'

export const useLocaleStore = defineStore('locale', () => {
  // M-44: 安全读取 localStorage，隐私模式下降级
  let stored: string | null = null
  try {
    stored = localStorage.getItem(LOCALE_STORAGE_KEY)
  } catch {
    stored = null
  }

  // L-18: 无效 locale 降级时输出警告
  if (stored && !SUPPORTED_LOCALES.includes(stored as SupportedLocale)) {
    if (import.meta.env.DEV) {
      console.warn(`[Locale] Invalid stored locale "${stored}", falling back to "${DEFAULT_LOCALE}"`)
    }
    stored = null
  }

  const locale = ref<SupportedLocale>(
    stored ? (stored as SupportedLocale) : DEFAULT_LOCALE
  )

  // 是否为中文
  const isZhCN = computed(() => locale.value === 'zh-CN')

  // 是否为英文
  const isEnUS = computed(() => locale.value === 'en-US')

  // 根据语言动态更新 meta 标签
  function updateMetaTags(loc: SupportedLocale) {
    const { t } = i18n.global
    updateLocaleMeta(loc, t('common.meta.description'), t('common.meta.ogTitle'))
  }

  // 初始化时同步 HTML lang 属性和 meta 标签
  updateMetaTags(locale.value)

  // 切换语言
  function setLocale(newLocale: SupportedLocale) {
    if (!SUPPORTED_LOCALES.includes(newLocale)) {
      return
    }

    locale.value = newLocale
    try {
      localStorage.setItem(LOCALE_STORAGE_KEY, newLocale)
    } catch {
      // M-44: 隐私模式或存储已满，降级为内存存储
    }
    i18n.global.locale.value = newLocale
    updateMetaTags(newLocale)
  }

  // 切换语言（在中英文之间切换）
  function toggleLocale() {
    setLocale(locale.value === 'zh-CN' ? 'en-US' : 'zh-CN')
  }

  return {
    locale,
    isZhCN,
    isEnUS,
    setLocale,
    toggleLocale
  }
})
