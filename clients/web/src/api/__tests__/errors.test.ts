import { beforeEach, describe, expect, it } from 'vitest'

import i18n from '@/i18n'

import { ApiError, getErrorMessage } from '../errors'

describe('API error messages', () => {
  beforeEach(() => {
    i18n.global.locale.value = 'zh-CN'
  })

  it('prefers sanitized backend messages for server ApiError instances', () => {
    expect(
      getErrorMessage(
        new ApiError({
          code: 'A0010201',
          message: 'insufficient permission for this review',
        }),
        '请求失败',
      ),
    ).toBe('insufficient permission for this review')
  })

  it('localizes client-side network errors instead of exposing technical messages', () => {
    expect(
      getErrorMessage(
        new ApiError({
          code: 'NETWORK_ERROR',
          message: 'Failed to fetch',
        }),
        '请求失败',
      ),
    ).toBe('网络连接失败，请检查网络设置')
  })

  it('falls back by code, then by call-site fallback', () => {
    expect(
      getErrorMessage(new ApiError({ code: 'A0010201', message: '' }), '请求失败'),
    ).toBe('访问被拒绝')
    expect(
      getErrorMessage(new ApiError({ code: 'X9999999', message: '' }), '请求失败'),
    ).toBe('请求失败')
  })

  it('does not expose non-API Error.message values', () => {
    expect(getErrorMessage(new Error('internal stack hint'), '请求失败')).toBe('请求失败')
  })
})
