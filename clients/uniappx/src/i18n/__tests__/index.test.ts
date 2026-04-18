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
  let warnSpy: ReturnType<typeof vi.spyOn>

  beforeEach(() => {
    storage.clear()
    runtime.uni = createUniStub('en-US')
    warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {})
  })

  afterEach(() => {
    delete runtime.uni
    warnSpy.mockRestore()
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

  it('reports storage read failure once and falls back to system locale', () => {
    runtime.uni = {
      ...createUniStub('en-US'),
      getStorageSync: () => {
        throw new Error('storage unavailable')
      },
    }

    expect(bootstrapLocale()).toBe('en-US')
    expect(bootstrapLocale()).toBe('en-US')
    expect(warnSpy).toHaveBeenCalledTimes(1)
    expect(warnSpy.mock.calls[0]?.[0]).toContain('[uniappx:i18n] failed to read stored locale')
  })

  it('ignores expected tabbar-unavailable errors', () => {
    runtime.uni = {
      ...createUniStub('en-US'),
      setTabBarItem: vi.fn(() => {
        throw new Error('not TabBar page')
      }),
    }

    bootstrapLocale()

    expect(warnSpy).not.toHaveBeenCalled()
  })

  it('reports unexpected tabbar sync errors once', () => {
    runtime.uni = {
      ...createUniStub('en-US'),
      setTabBarItem: vi.fn(() => {
        throw new Error('native bridge crashed')
      }),
    }

    bootstrapLocale()
    bootstrapLocale()

    expect(warnSpy).toHaveBeenCalledTimes(1)
    expect(warnSpy.mock.calls[0]?.[0]).toContain('[uniappx:i18n] failed to sync tab bar label')
  })
})
