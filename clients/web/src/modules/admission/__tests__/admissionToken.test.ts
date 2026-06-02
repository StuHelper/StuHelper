import { describe, expect, it } from 'vitest'

import { ApiError } from '@/api/errors'

import {
  buildAdmissionReturnURL,
  isFreshmanCameraHandoffLockedError,
  isAdmissionSessionExpiredError,
  isAdmissionTokenConsumedError,
  mapAdmissionApiError,
} from '../admissionToken'

const sameOrigin = 'https://join.stuhelper.com'

describe('admission token return URL', () => {
  it('keeps admission return URLs same-origin and preserves QQ query', () => {
    expect(
      buildAdmissionReturnURL('/verify/ABCD?qq=123', sameOrigin),
    ).toBe('https://join.stuhelper.com/verify/ABCD?qq=123')
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

  it('detects freshman camera handoff lock conflicts by domain message', () => {
    expect(
      isFreshmanCameraHandoffLockedError(
        new ApiError({
          code: 'A0000409',
          message: 'admission camera handoff locked',
        }),
      ),
    ).toBe(true)
    expect(
      isFreshmanCameraHandoffLockedError(
        new ApiError({ code: 'A0000409', message: 'conflict' }),
      ),
    ).toBe(false)
  })
})

describe('admission token API error mapping', () => {
  it.each([
    ['admission.qq_mismatch', 'qqMismatch'],
    ['admission.token_consumed', 'expired'],
    ['admission.token_expired', 'expired'],
    ['admission.token_not_found', 'expired'],
    ['admission.session_not_found', 'expired'],
  ] as const)('maps %s to %s', (code, state) => {
    expect(
      mapAdmissionApiError(new ApiError({ code, message: 'admission failed' })),
    ).toBe(state)
  })

  it('maps structurally equivalent admission errors without relying on instanceof', () => {
    expect(mapAdmissionApiError({ code: 'admission.token_not_found' })).toBe('expired')
    expect(mapAdmissionApiError({
      error: { code: 'admission.session_not_found' },
    })).toBe('expired')
    expect(isAdmissionTokenConsumedError({ code: 'admission.token_consumed' })).toBe(true)
    expect(isAdmissionSessionExpiredError({ code: 'admission.token_expired' })).toBe(true)
  })
})
