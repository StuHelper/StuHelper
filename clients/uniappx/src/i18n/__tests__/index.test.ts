import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import {
  bootstrapLocale,
  setLocale,
  setPageTitle,
  translate,
} from '../index'

type UniStub = {
  getLocale: () => string
  getStorageSync: (key: string) => string | undefined
  getSystemInfoSync: () => { language: string }
  setNavigationBarTitle: ReturnType<typeof vi.fn>
  setStorageSync: (key: string, value: string) => void
  setTabBarItem: ReturnType<typeof vi.fn>
}

const storage = new Map<string, string>()
const runtime = globalThis as typeof globalThis & { uni?: UniStub }

function createUniStub(systemLocale = 'en-US'): UniStub {
  return {
    getLocale: () => systemLocale,
    getStorageSync: (key: string) => storage.get(key),
    getSystemInfoSync: () => ({ language: systemLocale }),
    setNavigationBarTitle: vi.fn(),
    setStorageSync: (key: string, value: string) => {
      storage.set(key, value)
    },
    setTabBarItem: vi.fn(),
  }
}

describe('uniappx i18n', () => {
  beforeEach(() => {
    storage.clear()
    runtime.uni = createUniStub('en-US')
  })

  afterEach(() => {
    delete runtime.uni
    setLocale('zh-CN')
  })

  it('prefers stored locale over system locale', () => {
    runtime.uni?.setStorageSync('stuhelper:uniappx:locale', 'zh-CN')

    expect(bootstrapLocale()).toBe('zh-CN')
    expect(translate('common.tabs.home')).toBe('首页')
  })

  it('syncs translated tab labels during bootstrap', () => {
    bootstrapLocale()

    expect(runtime.uni?.setTabBarItem).toHaveBeenNthCalledWith(1, { index: 0, text: 'Home' })
    expect(runtime.uni?.setTabBarItem).toHaveBeenNthCalledWith(2, { index: 1, text: 'Courses' })
    expect(runtime.uni?.setTabBarItem).toHaveBeenNthCalledWith(3, { index: 2, text: 'Reviews' })
    expect(runtime.uni?.setTabBarItem).toHaveBeenNthCalledWith(4, { index: 3, text: 'Profile' })
  })

  it('sets localized page titles', () => {
    bootstrapLocale()
    setPageTitle('common.pageTitles.courseList')

    expect(runtime.uni?.setNavigationBarTitle).toHaveBeenCalledWith({ title: 'Courses' })
  })
})
