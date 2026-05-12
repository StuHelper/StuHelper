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

export function normalizeLocale(locale: string | null | undefined): SupportedLocale | null {
  if (!locale) {
    return null
  }

  return SUPPORTED_LOCALES.includes(locale as SupportedLocale)
    ? (locale as SupportedLocale)
    : null
}

export function readStoredLocale(): SupportedLocale | null {
  if (typeof localStorage === 'undefined') {
    return null
  }

  try {
    const stored = localStorage.getItem(LOCALE_STORAGE_KEY)
    return normalizeLocale(stored)
  } catch (_error) { void _error;
    return null
  }
}

// 语言检测：显式存储 → 默认中文
export function detectLocale(): SupportedLocale {
  return readStoredLocale() ?? DEFAULT_LOCALE
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
