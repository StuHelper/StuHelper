import { afterEach, describe, expect, it, vi } from 'vitest'

import {
  NATIVE_SSO_REDIRECT_DELAY_MS,
  scheduleNativeSSORedirect,
} from '@/auth/callback-redirect'
import {
  DEFAULT_SSO_REDIRECT,
  SSO_REDIRECT_STORAGE_KEY,
} from '@/auth/sso-state'

describe('native SSO callback redirect', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it.each([
    ['the stored internal page', '/pages/course/detail?id=42'],
    ['the profile fallback when storage is empty', DEFAULT_SSO_REDIRECT],
  ])('reLaunches to %s after a successful exchange', (_caseName, storedRedirect) => {
    const removeStorageSync = vi.fn()
    vi.stubGlobal('uni', {
      getStorageSync: vi.fn(() => (
        storedRedirect === DEFAULT_SSO_REDIRECT ? '' : storedRedirect
      )),
      removeStorageSync,
    })
    const relaunch = vi.fn()
    const schedule = vi.fn((callback: () => void) => callback())

    scheduleNativeSSORedirect(relaunch, schedule)

    expect(schedule).toHaveBeenCalledWith(expect.any(Function), NATIVE_SSO_REDIRECT_DELAY_MS)
    expect(relaunch).toHaveBeenCalledWith({ url: storedRedirect })
    expect(removeStorageSync).toHaveBeenCalledWith(SSO_REDIRECT_STORAGE_KEY)
  })
})
