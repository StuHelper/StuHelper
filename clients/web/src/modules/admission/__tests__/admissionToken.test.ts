// @vitest-environment jsdom

import { beforeEach, describe, expect, it } from 'vitest'

import { ApiError } from '@/api/errors'

import {
  buildAdmissionReturnURL,
  forgetLinkedAdmissionSession,
  isAdmissionSessionExpiredError,
  isAdmissionTokenConsumedError,
  mapAdmissionApiError,
  readLinkedAdmissionSessionID,
  rememberLinkedAdmissionSession,
} from '../admissionToken'

const sameOrigin = 'https://join.stuhelper.com'

describe('admission token return URL', () => {
  beforeEach(() => {
    sessionStorage.clear()
  })

  it('keeps admission return URLs same-origin without query parameters', () => {
    expect(
      buildAdmissionReturnURL('/verify/ABCD', sameOrigin),
    ).toBe('https://join.stuhelper.com/verify/ABCD')
    expect(
      buildAdmissionReturnURL('/verify/ABCD/', sameOrigin),
    ).toBe('https://join.stuhelper.com/verify/ABCD/')
  })

  it('normalizes admission return URLs by discarding source query and hash', () => {
    expect(
      buildAdmissionReturnURL('/verify/ABCD?from=qq&return=https://evil.example#step', sameOrigin),
    ).toBe('https://join.stuhelper.com/verify/ABCD')
  })

  it('rejects protocol-relative off-origin URLs', () => {
    expect(() => buildAdmissionReturnURL('//evil.example', sameOrigin)).toThrow(
      'Admission return URL must stay on the current origin',
    )
  })

  it('rejects non-admission paths', () => {
    expect(() => buildAdmissionReturnURL('/user/qq-binding?qq=123', sameOrigin))
      .toThrow('Admission return URL must target /verify/:code')
  })

  it('rejects multi-segment verify paths', () => {
    expect(() => buildAdmissionReturnURL('/verify/ABCD/extra?qq=123', sameOrigin))
      .toThrow('Admission return URL must target /verify/:code')
  })

  it('detects consumed-token errors for authenticated resume logic', () => {
    expect(
      isAdmissionTokenConsumedError(
        new ApiError({ code: 'admission.token_consumed', message: 'consumed' }),
      ),
    ).toBe(true)
    expect(
      isAdmissionTokenConsumedError(
        new ApiError({ code: 'admission.token_expired', message: 'expired' }),
      ),
    ).toBe(false)
  })

  it('detects expired admission session errors for child flows', () => {
    expect(
      isAdmissionSessionExpiredError(
        new ApiError({ code: 'admission.token_consumed', message: 'consumed' }),
      ),
    ).toBe(false)
    expect(
      isAdmissionSessionExpiredError(
        new ApiError({ code: 'admission.token_expired', message: 'expired' }),
      ),
    ).toBe(true)
    expect(
      isAdmissionSessionExpiredError(
        new ApiError({ code: 'admission.token_not_found', message: 'missing' }),
      ),
    ).toBe(true)
    expect(
      isAdmissionSessionExpiredError(
        new ApiError({ code: 'admission.session_not_found', message: 'missing' }),
      ),
    ).toBe(true)
    expect(
      isAdmissionSessionExpiredError(
        new ApiError({ code: 'A0000409', message: 'conflict' }),
      ),
    ).toBe(false)
  })

  it('remembers linked admission sessions for consumed token reloads', () => {
    expect(readLinkedAdmissionSessionID('ABCD')).toBeNull()

    rememberLinkedAdmissionSession('ABCD', 'session-1')

    expect(readLinkedAdmissionSessionID('ABCD')).toBe('session-1')

    forgetLinkedAdmissionSession('ABCD')

    expect(readLinkedAdmissionSessionID('ABCD')).toBeNull()
  })
})

describe('admission token API error mapping', () => {
  it.each([
    ['admission.qq_mismatch', 'qqMismatch'],
    ['admission.token_consumed', 'expired'],
    ['admission.token_expired', 'expired'],
    ['admission.token_not_found', 'invalid'],
    ['admission.session_not_found', 'invalid'],
  ] as const)('maps %s to %s', (code, state) => {
    expect(
      mapAdmissionApiError(new ApiError({ code, message: 'admission failed' })),
    ).toBe(state)
  })

  it('maps structurally equivalent admission errors without relying on instanceof', () => {
    expect(mapAdmissionApiError({ code: 'admission.token_not_found' })).toBe('invalid')
    expect(mapAdmissionApiError({
      error: { code: 'admission.session_not_found' },
    })).toBe('invalid')
    expect(isAdmissionTokenConsumedError({ code: 'admission.token_consumed' })).toBe(true)
    expect(isAdmissionSessionExpiredError({ code: 'admission.token_expired' })).toBe(true)
  })

  it('maps legacy qq binding conflict responses to qq mismatch', () => {
    expect(
      mapAdmissionApiError(new ApiError({
        code: 'A0000409',
        message: 'qq binding conflict',
      })),
    ).toBe('qqMismatch')
  })
})
