/**
 * 语言状态管理
 */
import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { SUPPORTED_LOCALES, DEFAULT_LOCALE, type SupportedLocale } from '@/i18n'

const LOCALE_STORAGE_KEY = 'locale'

export const useLocaleStore = defineStore('locale', () => {
  // 当前语言
  const locale = ref<SupportedLocale>(
    (localStorage.getItem(LOCALE_STORAGE_KEY) as SupportedLocale) || DEFAULT_LOCALE
  )

  // 是否为中文
  const isZhCN = computed(() => locale.value === 'zh-CN')

  // 是否为英文
  const isEnUS = computed(() => locale.value === 'en-US')

  // 切换语言
  function setLocale(newLocale: SupportedLocale) {
    if (!SUPPORTED_LOCALES.includes(newLocale)) {
      console.warn(`Unsupported locale: ${newLocale}`)
      return
    }

    locale.value = newLocale
    localStorage.setItem(LOCALE_STORAGE_KEY, newLocale)

    // 更新 vue-i18n 的语言
    const { locale: i18nLocale } = useI18n()
    i18nLocale.value = newLocale
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
