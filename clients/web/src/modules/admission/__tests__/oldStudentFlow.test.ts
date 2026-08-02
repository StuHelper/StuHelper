// @vitest-environment jsdom

import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { ApiError } from '@/api/errors'
import {
  buildSchoolSSOLoginPath,
  shouldShowFreshmanSubmission,
  type AdmissionSchoolOption,
} from '../oldStudentAdmission'
import OldStudentVerificationFlow from '../views/OldStudentVerificationFlow.vue'

const mockAdmissionApi = vi.hoisted(() => ({
  matchSchoolEmailAcademicStudent: vi.fn(),
  requestSchoolEmailOTP: vi.fn(),
  verifySchoolEmailOTP: vi.fn(),
}))

vi.mock('../api', () => ({
  admissionApi: mockAdmissionApi,
}))

const schools: AdmissionSchoolOption[] = [
  {
    schoolID: 4111010006,
    schoolCode: '4111010006',
    schoolName: '已接入大学',
    verificationMethod: 'ldap',
    enabled: true,
    schoolSsoEnabled: true,
    schoolEmailOtpEnabled: true,
  },
  {
    schoolID: 4111010001,
    schoolCode: '4111010002',
    schoolName: '未接入大学',
    verificationMethod: 'manual',
    enabled: true,
  },
]

function createDeferred<T>() {
  let resolve!: (value: T) => void
  let reject!: (reason?: unknown) => void
  const promise = new Promise<T>((promiseResolve, promiseReject) => {
    resolve = promiseResolve
    reject = promiseReject
  })
  return { promise, reject, resolve }
}

describe('OldStudentVerificationFlow', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('shows official SSO only for configured schools', async () => {
    const configuredWrapper = mount(OldStudentVerificationFlow, {
      props: {
        currentReturnUrl: 'https://join.stuhelper.com/verify/ABCD',
        linked: true,
        schools: [schools[0]!],
      },
    })
    await flushPromises()
    expect(configuredWrapper.find('[data-school-sso-button]').exists()).toBe(true)

    const unconfiguredWrapper = mount(OldStudentVerificationFlow, {
      props: {
        currentReturnUrl: 'https://join.stuhelper.com/verify/ABCD',
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
        currentReturnUrl: 'https://join.stuhelper.com/verify/ABCD',
        linked: false,
        schools,
      },
    })
    expect(pendingWrapper.find('[data-school-email-otp-form]').exists()).toBe(false)

    const linkedWrapper = mount(OldStudentVerificationFlow, {
      props: {
        currentReturnUrl: 'https://join.stuhelper.com/verify/ABCD',
        linked: true,
        schools,
      },
    })
    expect(linkedWrapper.find('[data-school-email-otp-form]').exists()).toBe(true)
  })

  it('adds admission session context to school SSO login URLs', () => {
    expect(
      buildSchoolSSOLoginPath(
        '4111010006',
        'https://join.stuhelper.com/verify/ABCD',
        'session-1',
      ),
    ).toBe(
      '/api/v1/admission/school-sso/4111010006/login?return=https%3A%2F%2Fjoin.stuhelper.com%2Fverify%2FABCD&admissionSessionID=session-1',
    )
  })

  it('fills BUAA student email only after academic name match succeeds', async () => {
    vi.useFakeTimers()
    mockAdmissionApi.matchSchoolEmailAcademicStudent.mockResolvedValue({
      matched: true,
      email: '20250001@buaa.edu.cn',
      studentID: '20250001',
      message: '学号和姓名已匹配。',
    })
    mockAdmissionApi.requestSchoolEmailOTP.mockResolvedValue({
      email: '20250001@buaa.edu.cn',
      studentID: '20250001',
      cooldownSeconds: 60,
    })
    const wrapper = mount(OldStudentVerificationFlow, {
      props: {
        currentReturnUrl: 'https://join.stuhelper.com/verify/ABCD',
        admissionSessionId: 'session-1',
        linked: true,
        schools: [{
          schoolID: 4111010006,
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
    await vi.advanceTimersByTimeAsync(300)
    await flushPromises()
    expect(mockAdmissionApi.matchSchoolEmailAcademicStudent).toHaveBeenCalledWith({
      schoolCode: '4111010006',
      admissionSessionID: 'session-1',
      studentID: '20250001',
      studentName: '张三',
    })
    expect(emailInput.element.value).toBe('20250001@buaa.edu.cn')
    expect(wrapper.get('[data-academic-match-status]').attributes('role')).toBe(
      'status',
    )

    await wrapper.find('[data-school-email-otp-request]').trigger('click')
    await flushPromises()

    expect(mockAdmissionApi.requestSchoolEmailOTP).toHaveBeenCalledWith({
      schoolCode: '4111010006',
      admissionSessionID: 'session-1',
      email: undefined,
      studentID: '20250001',
      studentName: '张三',
    })
    expect(emailInput.element.value).toBe('20250001@buaa.edu.cn')
    expect(wrapper.findAll('[role="status"]')).toHaveLength(2)
    wrapper.unmount()
    vi.useRealTimers()
  })

  it('keeps resend disabled for the server-provided OTP cooldown', async () => {
    vi.useFakeTimers()
    mockAdmissionApi.requestSchoolEmailOTP.mockResolvedValueOnce({
      email: 'student@example.edu',
      cooldownSeconds: 60,
    })
    const wrapper = mount(OldStudentVerificationFlow, {
      props: {
        currentReturnUrl: 'https://join.stuhelper.com/verify/ABCD',
        linked: true,
        schools,
      },
    })

    await wrapper.find('[data-academic-email-input]').setValue('student@example.edu')
    const requestButton = wrapper.get<HTMLButtonElement>('[data-school-email-otp-request]')
    await requestButton.trigger('click')
    await flushPromises()

    expect(requestButton.element.disabled).toBe(true)
    expect(requestButton.text()).toBe('重新发送（60s）')

    await vi.advanceTimersByTimeAsync(59_000)
    expect(requestButton.element.disabled).toBe(true)
    expect(requestButton.text()).toBe('重新发送（1s）')

    await vi.advanceTimersByTimeAsync(1_000)
    expect(requestButton.element.disabled).toBe(false)
    expect(requestButton.text()).toBe('发送验证码')
    wrapper.unmount()
  })

  it('localizes structured OTP cooldown errors instead of echoing server English', async () => {
    mockAdmissionApi.requestSchoolEmailOTP.mockRejectedValueOnce(
      new ApiError({
        code: 'A0000429',
        message: 'please wait before requesting a new code',
        status: 429,
      }),
    )
    const wrapper = mount(OldStudentVerificationFlow, {
      props: {
        currentReturnUrl: 'https://join.stuhelper.com/verify/ABCD',
        linked: true,
        schools,
      },
    })

    await wrapper.find('[data-academic-email-input]').setValue('student@example.edu')
    await wrapper.find('[data-school-email-otp-request]').trigger('click')
    await flushPromises()

    expect(wrapper.get('[role="alert"]').text()).toBe('请求过于频繁，请稍后重试')
    expect(wrapper.text()).not.toContain('please wait before requesting a new code')
  })

  it('requires academic match before sending BUAA student email OTP', async () => {
    const wrapper = mount(OldStudentVerificationFlow, {
      props: {
        currentReturnUrl: 'https://join.stuhelper.com/verify/ABCD',
        admissionSessionId: 'session-1',
        linked: true,
        schools: [{
          schoolID: 4111010006,
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

    await wrapper.find('[data-school-email-otp-request]').trigger('click')
    expect(wrapper.text()).toContain('请先输入学号和姓名')
    expect(mockAdmissionApi.requestSchoolEmailOTP).not.toHaveBeenCalled()
  })

  it('emits expired when the linked session has timed out during email OTP', async () => {
    mockAdmissionApi.requestSchoolEmailOTP.mockRejectedValueOnce(
      new ApiError({ code: 'admission.token_expired', message: 'expired' }),
    )
    const wrapper = mount(OldStudentVerificationFlow, {
      props: {
        currentReturnUrl: 'https://join.stuhelper.com/verify/ABCD',
        linked: true,
        schools,
      },
    })

    await wrapper.find('[data-academic-email-input]').setValue('student@example.edu')
    await wrapper.find('[data-school-email-otp-request]').trigger('click')
    await flushPromises()

    expect(wrapper.emitted('expired')).toHaveLength(1)
    expect(wrapper.text()).not.toContain('验证码发送失败')
  })

  it('does not treat consumed token errors from child requests as expired links', async () => {
    mockAdmissionApi.requestSchoolEmailOTP.mockRejectedValueOnce(
      new ApiError({ code: 'admission.token_consumed', message: 'consumed' }),
    )
    const wrapper = mount(OldStudentVerificationFlow, {
      props: {
        currentReturnUrl: 'https://join.stuhelper.com/verify/ABCD',
        linked: true,
        schools,
      },
    })

    await wrapper.find('[data-academic-email-input]').setValue('student@example.edu')
    await wrapper.find('[data-school-email-otp-request]').trigger('click')
    await flushPromises()

    expect(wrapper.emitted('expired')).toBeUndefined()
    expect(wrapper.text()).not.toContain('consumed')
    expect(wrapper.get('[role="alert"]').text()).toBe('验证码发送失败。')
  })

  it('keeps email OTP verification disabled until email and code are ready', async () => {
    const wrapper = mount(OldStudentVerificationFlow, {
      props: {
        currentReturnUrl: 'https://join.stuhelper.com/verify/ABCD',
        linked: true,
        schools,
      },
    })
    const verifyButton = wrapper.find<HTMLButtonElement>('[data-school-email-otp-verify]')

    expect(verifyButton.element.disabled).toBe(true)
    await wrapper.find('[data-academic-email-input]').setValue('student@example.edu')
    expect(verifyButton.element.disabled).toBe(true)
    await wrapper.findAll('input').at(-1)!.setValue('123456')

    expect(verifyButton.element.disabled).toBe(false)
  })

  it('prevents duplicate email OTP verification submits while one request is pending', async () => {
    const verifyDeferred = createDeferred({
      status: 'verified',
      projectionPending: false,
      credentialKind: 'school_email_otp',
    })
    mockAdmissionApi.verifySchoolEmailOTP.mockReturnValue(verifyDeferred.promise)
    const wrapper = mount(OldStudentVerificationFlow, {
      props: {
        currentReturnUrl: 'https://join.stuhelper.com/verify/ABCD',
        admissionSessionId: 'session-verify',
        linked: true,
        schools,
      },
    })
    await wrapper.find('[data-academic-email-input]').setValue('student@example.edu')
    await wrapper.findAll('input').at(-1)!.setValue('123456')

    await wrapper.find('[data-school-email-otp-form]').trigger('submit')
    await wrapper.find('[data-school-email-otp-form]').trigger('submit')
    await flushPromises()

    expect(
      wrapper.get('[data-school-email-otp-form]').attributes('aria-busy'),
    ).toBe('true')
    expect(mockAdmissionApi.verifySchoolEmailOTP).toHaveBeenCalledTimes(1)
    expect(mockAdmissionApi.verifySchoolEmailOTP).toHaveBeenCalledWith({
      schoolCode: '4111010006',
      admissionSessionID: 'session-verify',
      email: 'student@example.edu',
      code: '123456',
    })

    verifyDeferred.resolve({
      status: 'verified',
      projectionPending: false,
      credentialKind: 'school_email_otp',
    })
    await flushPromises()

    expect(
      wrapper.get('[data-school-email-otp-form]').attributes('aria-busy'),
    ).toBe('false')
    expect(wrapper.emitted('verified')).toHaveLength(1)
  })
})

describe('formal admission credential state', () => {
  it('hides freshman submission after an old-student credential exists', () => {
    expect(shouldShowFreshmanSubmission(null)).toBe(true)
    expect(shouldShowFreshmanSubmission({ credentialKind: 'school_sso' })).toBe(false)
    expect(shouldShowFreshmanSubmission({ credentialKind: 'school_email_otp' })).toBe(false)
  })
})
