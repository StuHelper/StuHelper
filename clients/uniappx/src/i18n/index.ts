import { ref } from 'vue'
import { enUSMessages } from './en-US'
import { zhCNMessages } from './zh-CN'

export type SupportedLocale = 'en-US' | 'zh-CN'
type TranslationParams = Record<string, boolean | number | string | null | undefined>
type TranslationDictionary = Record<string, string>
type UniRuntime = {
  getLocale?: () => string
  getStorageSync: (key: string) => unknown
  getSystemInfoSync: () => { language?: string | null | undefined }
  setNavigationBarTitle: (options: { title: string }) => void
  setStorageSync: (key: string, value: string) => void
  setTabBarItem: (options: { index: number; text: string }) => void
}

const STORAGE_KEY = 'stuhelper:uniappx:locale'
const DEFAULT_LOCALE: SupportedLocale = 'zh-CN'
const messages: Record<SupportedLocale, TranslationDictionary> = {
  'en-US': enUSMessages,
  'zh-CN': zhCNMessages,
}

export const activeLocale = ref<SupportedLocale>(DEFAULT_LOCALE)

function getUniRuntime(): UniRuntime | undefined {
  return (globalThis as typeof globalThis & { uni?: UniRuntime }).uni
}

function normalizeLocale(value?: string | null): SupportedLocale {
  if (!value) return DEFAULT_LOCALE
  return value.toLowerCase().startsWith('en') ? 'en-US' : 'zh-CN'
}

function readStoredLocale(): SupportedLocale | null {
  const runtime = getUniRuntime()
  if (!runtime) return null
  try {
    const value = runtime.getStorageSync(STORAGE_KEY)
    return typeof value === 'string' && value ? normalizeLocale(value) : null
  } catch {
    return null
  }
}

function readSystemLocale(): SupportedLocale {
  const runtime = getUniRuntime()
  if (!runtime) return DEFAULT_LOCALE
  try {
    if (typeof runtime.getLocale === 'function') {
      return normalizeLocale(runtime.getLocale())
    }
  } catch {
    // ignore system locale access failure
  }

  try {
    const system = runtime.getSystemInfoSync()
    return normalizeLocale(system.language)
  } catch {
    return DEFAULT_LOCALE
  }
}

function formatMessage(template: string, params?: TranslationParams): string {
  if (!params) return template
  return template.replace(/\{(\w+)\}/g, (_match, key) => {
    const value = params[key]
    return value == null ? '' : String(value)
  })
}

export function getLocale(): SupportedLocale {
  return activeLocale.value
}

export function setLocale(locale: SupportedLocale) {
  activeLocale.value = locale
  const runtime = getUniRuntime()
  if (!runtime) return
  try {
    runtime.setStorageSync(STORAGE_KEY, locale)
  } catch {
    // ignore storage failure
  }
  syncAppChrome()
}

export function bootstrapLocale(): SupportedLocale {
  const locale = readStoredLocale() ?? readSystemLocale()
  activeLocale.value = locale
  syncAppChrome()
  return locale
}

export function translate(key: string, params?: TranslationParams): string {
  const message = messages[activeLocale.value][key] ?? messages[DEFAULT_LOCALE][key] ?? key
  return formatMessage(message, params)
}

export function isEnglishLocale(): boolean {
  return activeLocale.value === 'en-US'
}

export function setPageTitle(key: string, params?: TranslationParams) {
  const runtime = getUniRuntime()
  if (!runtime) return
  try {
    runtime.setNavigationBarTitle({ title: translate(key, params) })
  } catch {
    // ignore title update failure
  }
}

export function syncAppChrome() {
  const runtime = getUniRuntime()
  if (!runtime) return

  const tabs = [
    translate('common.tabs.home'),
    translate('common.tabs.course'),
    translate('common.tabs.review'),
    translate('common.tabs.user'),
  ]

  tabs.forEach((text, index) => {
    try {
      runtime.setTabBarItem({ index, text })
    } catch {
      // ignore when tabbar is unavailable
    }
  })
}
