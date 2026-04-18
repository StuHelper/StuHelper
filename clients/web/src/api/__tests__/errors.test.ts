import { beforeEach, describe, expect, it } from 'vitest'

import i18n from '@/i18n'

import { ApiError, classifyApiError, getErrorMessage, httpStatusToDefaultCode } from '../errors'

describe('API error messages', () => {
  beforeEach(() => {
    i18n.global.locale.value = 'zh-CN'
  })

  it('prefers localized code messages over backend raw messages', () => {
    expect(
      getErrorMessage(
        new ApiError({
          code: 'A0010201',
          message: 'insufficient permission for this review',
        }),
        '请求失败',
      ),
    ).toBe('访问被拒绝')
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

  it('classifies network/api/unknown errors through a shared helper', () => {
    expect(classifyApiError(
      new ApiError({ code: 'NETWORK_ERROR', message: 'fetch failed' }),
      {
        networkType: 'network',
        apiType: 'server',
        unknownType: 'unknown',
        fallbackMessage: '请求失败',
      },
    )).toEqual({
      type: 'network',
      message: '网络连接失败，请检查网络设置',
    })

    expect(classifyApiError(
      new ApiError({ code: 'A0010201', message: 'forbidden' }),
      {
        networkType: 'network',
        apiType: 'server',
        unknownType: 'unknown',
        fallbackMessage: '请求失败',
      },
    )).toEqual({
      type: 'server',
      message: '访问被拒绝',
    })

    expect(classifyApiError(
      new Error('boom'),
      {
        networkType: 'network',
        apiType: 'server',
        unknownType: 'unknown',
        fallbackMessage: '请求失败',
      },
    )).toEqual({
      type: 'unknown',
      message: 'boom',
    })
  })

  it('maps bare HTTP statuses through shared default error codes', () => {
    expect(httpStatusToDefaultCode(401)).toBe('A0010100')
    expect(httpStatusToDefaultCode(429)).toBe('A0000429')
    expect(httpStatusToDefaultCode(599)).toBe('B0000001')
  })
})
