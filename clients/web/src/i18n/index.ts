/**
 * vue-i18n 配置
 * 支持中文 (zh-CN) 和英文 (en-US)
 */
import { createI18n } from 'vue-i18n'
import zhCN from './locales/zh-CN'
import enUS from './locales/en-US'

// 支持的语言列表
export const SUPPORTED_LOCALES = ['zh-CN', 'en-US'] as const
export type SupportedLocale = (typeof SUPPORTED_LOCALES)[number]

// 默认语言
export const DEFAULT_LOCALE: SupportedLocale = 'zh-CN'

// localStorage key（与 stores/locale.ts 共享）
export const LOCALE_STORAGE_KEY = 'locale'

// 语言检测：localStorage → navigator.language → 默认
function detectLocale(): SupportedLocale {
  // 1. 从 localStorage 读取
  const stored = localStorage.getItem(LOCALE_STORAGE_KEY)
  if (stored && SUPPORTED_LOCALES.includes(stored as SupportedLocale)) {
    return stored as SupportedLocale
  }

  // 2. 从浏览器语言检测
  const browserLang = navigator.language
  if (browserLang.startsWith('zh')) {
    return 'zh-CN'
  }
  if (browserLang.startsWith('en')) {
    return 'en-US'
  }

  // 3. 返回默认语言
  return DEFAULT_LOCALE
}

// 创建 i18n 实例
const i18n = createI18n({
  legacy: false, // 使用 Composition API 模式
  locale: detectLocale(),
  fallbackLocale: DEFAULT_LOCALE,
  messages: {
    'zh-CN': zhCN,
    'en-US': enUS
  }
})

export default i18n
