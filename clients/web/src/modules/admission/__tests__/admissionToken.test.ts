import { describe, expect, it } from 'vitest'

import { ApiError } from '@/api/errors'

import {
  buildAdmissionReturnURL,
  mapAdmissionApiError,
} from '../admissionToken'

const sameOrigin = 'https://auth.stuhelper.com'

describe('admission token return URL', () => {
  it('keeps admission return URLs same-origin and preserves QQ query', () => {
    expect(
      buildAdmissionReturnURL('/admission/a/ABCD?qq=123', sameOrigin),
    ).toBe('https://auth.stuhelper.com/admission/a/ABCD?qq=123')
  })

  it('rejects protocol-relative off-origin URLs', () => {
    expect(() => buildAdmissionReturnURL('//evil.example', sameOrigin)).toThrow(
      'Admission return URL must stay on the current origin',
    )
  })

  it('rejects non-admission paths', () => {
    expect(() => buildAdmissionReturnURL('/user/qq-binding?qq=123', sameOrigin))
      .toThrow('Admission return URL must target /admission/a/:code')
  })
})

describe('admission token API error mapping', () => {
  it.each([
    ['admission.qq_mismatch', 'qqMismatch'],
    ['admission.token_consumed', 'expired'],
    ['admission.token_expired', 'expired'],
  ] as const)('maps %s to %s', (code, state) => {
    expect(
      mapAdmissionApiError(new ApiError({ code, message: 'admission failed' })),
    ).toBe(state)
  })
})
