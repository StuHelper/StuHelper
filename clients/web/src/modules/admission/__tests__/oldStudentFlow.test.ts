// @vitest-environment jsdom

import { flushPromises, mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'

import {
  shouldShowFreshmanSubmission,
  type AdmissionSchoolOption,
} from '../oldStudentAdmission'
import OldStudentVerificationFlow from '../views/OldStudentVerificationFlow.vue'

const mockAdmissionApi = vi.hoisted(() => ({
  requestSchoolEmailOTP: vi.fn(),
  verifySchoolEmailOTP: vi.fn(),
}))

vi.mock('../api', () => ({
  admissionApi: mockAdmissionApi,
}))

const schools: AdmissionSchoolOption[] = [
  {
    schoolID: 1,
    schoolName: '已接入大学',
    verificationMethod: 'ldap',
    enabled: true,
    schoolSsoEnabled: true,
  },
  {
    schoolID: 2,
    schoolName: '未接入大学',
    verificationMethod: 'manual',
    enabled: true,
  },
]

describe('OldStudentVerificationFlow', () => {
  it('shows official SSO only for configured schools', async () => {
    const configuredWrapper = mount(OldStudentVerificationFlow, {
      props: {
        currentReturnUrl: 'https://auth.stuhelper.com/admission/a/ABCD?qq=123',
        linked: true,
        schools: [schools[0]!],
      },
    })
    await flushPromises()
    expect(configuredWrapper.find('[data-school-sso-button]').exists()).toBe(true)

    const unconfiguredWrapper = mount(OldStudentVerificationFlow, {
      props: {
        currentReturnUrl: 'https://auth.stuhelper.com/admission/a/ABCD?qq=123',
        linked: true,
        schools: [schools[1]!],
      },
    })
    await flushPromises()
    expect(unconfiguredWrapper.find('[data-school-sso-button]').exists()).toBe(false)
  })

  it('shows email OTP form only after admission is linked', async () => {
    const pendingWrapper = mount(OldStudentVerificationFlow, {
      props: {
        currentReturnUrl: 'https://auth.stuhelper.com/admission/a/ABCD?qq=123',
        linked: false,
        schools,
      },
    })
    expect(pendingWrapper.find('[data-school-email-otp-form]').exists()).toBe(false)

    const linkedWrapper = mount(OldStudentVerificationFlow, {
      props: {
        currentReturnUrl: 'https://auth.stuhelper.com/admission/a/ABCD?qq=123',
        linked: true,
        schools,
      },
    })
    expect(linkedWrapper.find('[data-school-email-otp-form]').exists()).toBe(true)
  })
})

describe('formal admission credential state', () => {
  it('hides freshman submission after an old-student credential exists', () => {
    expect(shouldShowFreshmanSubmission(null)).toBe(true)
    expect(shouldShowFreshmanSubmission({ credentialKind: 'school_sso' })).toBe(false)
    expect(shouldShowFreshmanSubmission({ credentialKind: 'school_email_otp' })).toBe(false)
  })
})
