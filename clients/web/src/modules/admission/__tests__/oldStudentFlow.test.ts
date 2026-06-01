// @vitest-environment jsdom

import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

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
    schoolCode: '4111010006',
    schoolName: '已接入大学',
    verificationMethod: 'ldap',
    enabled: true,
    schoolSsoEnabled: true,
    schoolEmailOtpEnabled: true,
  },
  {
    schoolID: 2,
    schoolCode: '4111010002',
    schoolName: '未接入大学',
    verificationMethod: 'manual',
    enabled: true,
  },
]

describe('OldStudentVerificationFlow', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('shows official SSO only for configured schools', async () => {
    const configuredWrapper = mount(OldStudentVerificationFlow, {
      props: {
        currentReturnUrl: 'https://join.stuhelper.com/verify/ABCD?qq=123',
        linked: true,
        schools: [schools[0]!],
      },
    })
    await flushPromises()
    expect(configuredWrapper.find('[data-school-sso-button]').exists()).toBe(true)

    const unconfiguredWrapper = mount(OldStudentVerificationFlow, {
      props: {
        currentReturnUrl: 'https://join.stuhelper.com/verify/ABCD?qq=123',
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
        currentReturnUrl: 'https://join.stuhelper.com/verify/ABCD?qq=123',
        linked: false,
        schools,
      },
    })
    expect(pendingWrapper.find('[data-school-email-otp-form]').exists()).toBe(false)

    const linkedWrapper = mount(OldStudentVerificationFlow, {
      props: {
        currentReturnUrl: 'https://join.stuhelper.com/verify/ABCD?qq=123',
        linked: true,
        schools,
      },
    })
    expect(linkedWrapper.find('[data-school-email-otp-form]').exists()).toBe(true)
  })

  it('fills BUAA student email only after academic name match succeeds', async () => {
    mockAdmissionApi.requestSchoolEmailOTP.mockResolvedValue({
      email: '20250001@buaa.edu.cn',
      studentID: '20250001',
      cooldownSeconds: 60,
    })
    const wrapper = mount(OldStudentVerificationFlow, {
      props: {
        currentReturnUrl: 'https://join.stuhelper.com/verify/ABCD?qq=123',
        linked: true,
        schools: [{
          schoolID: 10006,
          schoolCode: '4111010006',
          schoolName: '北京航空航天大学',
          verificationMethod: 'manual',
          enabled: true,
          schoolEmailOtpEnabled: true,
          schoolEmailIdentityPolicy: {
            type: 'academic_student_email',
            studentIDEmailDomain: 'buaa.edu.cn',
            requireStudentName: true,
          },
        }],
      },
    })

    const emailInput = wrapper.find<HTMLInputElement>('[data-academic-email-input]')
    expect(emailInput.element.value).toBe('')
    await wrapper.find('[data-academic-student-id-input]').setValue('20250001')
    expect(emailInput.element.value).toBe('')
    await wrapper.find('[data-academic-student-name-input]').setValue('张三')
    await wrapper.find('[data-school-email-otp-request]').trigger('click')
    await flushPromises()

    expect(mockAdmissionApi.requestSchoolEmailOTP).toHaveBeenCalledWith({
      schoolCode: '4111010006',
      email: undefined,
      studentID: '20250001',
      studentName: '张三',
    })
    expect(emailInput.element.value).toBe('20250001@buaa.edu.cn')
  })
})

describe('formal admission credential state', () => {
  it('hides freshman submission after an old-student credential exists', () => {
    expect(shouldShowFreshmanSubmission(null)).toBe(true)
    expect(shouldShowFreshmanSubmission({ credentialKind: 'school_sso' })).toBe(false)
    expect(shouldShowFreshmanSubmission({ credentialKind: 'school_email_otp' })).toBe(false)
  })
})
