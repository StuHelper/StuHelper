// @vitest-environment jsdom
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'

const LOCALE_STORAGE_KEY = 'locale'
const ORIGINAL_LANGUAGE = window.navigator.language

function setNavigatorLanguage(language: string) {
  Object.defineProperty(window.navigator, 'language', {
    value: language,
    configurable: true
  })
}

async function createLocaleStore() {
  vi.resetModules()
  setActivePinia(createPinia())

  const { useLocaleStore } = await import('../locale')
  const { default: i18n } = await import('@/i18n')

  return {
    store: useLocaleStore(),
    i18n
  }
}

describe('locale store', () => {
  beforeEach(() => {
    localStorage.clear()
    document.documentElement.removeAttribute('lang')
    setNavigatorLanguage(ORIGINAL_LANGUAGE)
  })

  afterEach(() => {
    vi.restoreAllMocks()
    localStorage.clear()
    setNavigatorLanguage(ORIGINAL_LANGUAGE)
  })

  it('defaults to Chinese without stored preference even when browser language is English', async () => {
    setNavigatorLanguage('en-US')

    const { store, i18n } = await createLocaleStore()

    expect(store.locale).toBe('zh-CN')
    expect(i18n.global.locale.value).toBe('zh-CN')
    expect(document.documentElement.lang).toBe('zh-CN')
  })

  it('keeps an explicit stored English preference across startup', async () => {
    localStorage.setItem(LOCALE_STORAGE_KEY, 'en-US')

    const { store, i18n } = await createLocaleStore()

    expect(store.locale).toBe('en-US')
    expect(i18n.global.locale.value).toBe('en-US')
    expect(document.documentElement.lang).toBe('en-US')
  })

  it('updates the visible i18n locale on the first toggle', async () => {
    localStorage.setItem(LOCALE_STORAGE_KEY, 'en-US')

    const { store, i18n } = await createLocaleStore()

    store.toggleLocale()

    expect(store.locale).toBe('zh-CN')
    expect(i18n.global.locale.value).toBe('zh-CN')
    expect(localStorage.getItem(LOCALE_STORAGE_KEY)).toBe('zh-CN')
    expect(document.documentElement.lang).toBe('zh-CN')
  })
})
