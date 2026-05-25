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
const reportedDiagnostics = new Set<string>()
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

function emitLocaleDiagnostic(key: string, message: string, error: unknown) {
  if (reportedDiagnostics.has(key)) return
  reportedDiagnostics.add(key)
  console.warn(`[uniappx:i18n] ${message}`, error)
}

function shouldIgnoreTabBarSyncError(error: unknown): boolean {
  const message = error instanceof Error ? error.message : typeof error === 'string' ? error : ''
  const normalized = message.toLowerCase()
  return normalized.includes('tabbar')
    || normalized.includes('not tabbar page')
    || normalized.includes('tab bar')
    || normalized.includes('not support')
}

function isH5Runtime(): boolean {
  return typeof window !== 'undefined' && typeof document !== 'undefined' && typeof plus === 'undefined'
}

function readStoredLocale(): SupportedLocale | null {
  const runtime = getUniRuntime()
  if (!runtime) return null
  try {
    const value = runtime.getStorageSync(STORAGE_KEY)
    return typeof value === 'string' && value ? normalizeLocale(value) : null
  } catch (error) {
    emitLocaleDiagnostic('i18n_locale_storage_read_error', 'failed to read stored locale, falling back to system locale', error)
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
  } catch (error) {
    emitLocaleDiagnostic('i18n_locale_runtime_error', 'failed to read runtime locale, falling back to system info', error)
  }

  try {
    const system = runtime.getSystemInfoSync()
    return normalizeLocale(system.language)
  } catch (error) {
    emitLocaleDiagnostic('i18n_locale_system_info_error', 'failed to read system locale, falling back to default locale', error)
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
  } catch (error) {
    emitLocaleDiagnostic('i18n_locale_storage_write_error', 'failed to persist locale, keeping in-memory locale only', error)
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
  } catch (error) {
    emitLocaleDiagnostic('i18n_nav_title_error', 'failed to update navigation title', error)
  }
}

export function syncAppChrome() {
  const runtime = getUniRuntime()
  if (!runtime) return
  if (isH5Runtime()) return

  const tabs = [
    translate('common.tabs.home'),
    translate('common.tabs.course'),
    translate('common.tabs.review'),
    translate('common.tabs.user'),
  ]

  tabs.forEach((text, index) => {
    try {
      runtime.setTabBarItem({ index, text })
    } catch (error) {
      if (shouldIgnoreTabBarSyncError(error)) return
      emitLocaleDiagnostic('i18n_tabbar_sync_error', `failed to sync tab bar label at index ${index}`, error)
    }
  })
}
