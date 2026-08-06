// @vitest-environment jsdom

import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import enUSUser from '@/i18n/locales/en-US/user'
import zhCNUser from '@/i18n/locales/zh-CN/user'

const mockRouterPush = vi.fn()
const mockRouterReplace = vi.fn()
const mockToastSuccess = vi.fn()
const mockToastError = vi.fn()
const mockRoute = vi.hoisted(() => ({
  query: {} as Record<string, string>,
}))

const mockVerificationApi = vi.hoisted(() => ({
  listSchools: vi.fn(),
  listCredentials: vi.fn(),
  getEligibility: vi.fn(),
  createApplication: vi.fn(),
  getApplication: vi.fn(),
  cancelApplication: vi.fn(),
  verifyRealName: vi.fn(),
  verifySchoolSSO: vi.fn(),
  requestStudentEmailOTP: vi.fn(),
  verifyStudentEmailOTP: vi.fn(),
  createInboundEmailChallenge: vi.fn(),
  getInboundEmailChallenge: vi.fn(),
  revokeCredential: vi.fn(),
  suggestSchool: vi.fn(),
}))

vi.mock('@/api', () => ({
  api: { studentVerification: mockVerificationApi },
}))

vi.mock('vue-router', async () => {
  const actual = await vi.importActual<typeof import('vue-router')>('vue-router')
  return {
    ...actual,
    useRoute: () => mockRoute,
    useRouter: () => ({ push: mockRouterPush, replace: mockRouterReplace }),
  }
})

vi.mock('vue-i18n', () => ({
  createI18n: () => ({
    global: { t: (key: string) => key, te: () => false },
    install: vi.fn(),
  }),
  useI18n: () => ({ t: (key: string, params?: Record<string, unknown>) => `${key}${params ? JSON.stringify(params) : ''}` }),
}))

vi.mock('@/composables/useToast', () => ({
  useToast: () => ({ success: mockToastSuccess, error: mockToastError }),
}))

const { default: StudentVerificationPage } = await import('../views/StudentVerificationPage.vue')

const privacyNotice = {
  version: '2026-08-05',
  title: '学生认证隐私说明',
  summary: '仅处理本次校验必要的数据。',
  dataCategories: ['学号', '姓名'],
  retentionSummary: '按认证策略保存最小凭据。',
}

const school = {
  code: '4111010006',
  name: '北京航空航天大学',
  location: '北京',
  methods: [
    {
      method: 'real_name_identity_check' as const,
      displayName: '实名信息校验',
      description: '一次性实名信息校验',
      availability: 'available' as const,
      formFields: [],
      privacyNotice,
    },
    {
      method: 'student_email_outbound_otp' as const,
      displayName: '学校邮箱接收验证码',
      description: '发送到规范学号邮箱',
      availability: 'available' as const,
      formFields: [],
      privacyNotice,
    },
    {
      method: 'manual_material_review' as const,
      displayName: '人工材料审核',
      description: '提交学校材料',
      availability: 'available' as const,
      formFields: [
        { key: 'department', label: '学院', inputType: 'text' as const, required: true },
        { key: 'studentID', label: '学号', inputType: 'text' as const, required: true },
        { key: 'name', label: '姓名', inputType: 'text' as const, required: true },
        { key: 'email', label: '邮箱', inputType: 'email' as const, required: true },
      ],
      privacyNotice,
    },
  ],
}

function ok<T>(data: T) {
  return Promise.resolve({ data: { data } })
}

function application(status: 'created' | 'approved' = 'created') {
  return {
    id: '11111111-1111-4111-8111-111111111111',
    school: { code: school.code, name: school.name },
    status,
    revision: status === 'approved' ? 2 : 1,
    nextActions: status === 'approved' ? ['return_to_consumer'] : ['choose_method'],
    credential: status === 'approved' ? {
      id: '22222222-2222-4222-8222-222222222222',
      schoolCode: school.code,
      schoolName: school.name,
      method: 'real_name_identity_check',
      status: 'active',
      credentialClass: 'formal_student',
      subjectDisplay: '20****01',
      verifiedAt: '2026-08-05T10:00:00Z',
      revision: 1,
    } : null,
    createdAt: '2026-08-05T09:00:00Z',
    updatedAt: '2026-08-05T10:00:00Z',
    expiresAt: '2026-08-06T09:00:00Z',
  }
}

async function mountPage() {
  const wrapper = mount(StudentVerificationPage, {
    global: {
      stubs: {
        RouterLink: { template: '<a><slot /></a>' },
        ManualReviewEvidence: { template: '<div data-manual-review-stub />' },
      },
    },
  })
  await flushPromises()
  return wrapper
}

describe('StudentVerificationPage', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    sessionStorage.clear()
    mockRoute.query = {}
    vi.spyOn(window, 'confirm').mockReturnValue(true)
    mockVerificationApi.listSchools.mockImplementation(() => ok([school]))
    mockVerificationApi.listCredentials.mockImplementation(() => ok([]))
    mockVerificationApi.getEligibility.mockImplementation(() => ok({ eligible: false }))
    mockVerificationApi.createApplication.mockImplementation(() => ok(application()))
    mockVerificationApi.getApplication.mockImplementation(() => ok(application()))
  })

  it('keeps the new platform copy complete in Chinese and English', () => {
    expect(Object.keys(enUSUser.verification.student.platform).sort()).toEqual(
      Object.keys(zhCNUser.verification.student.platform).sort(),
    )
    expect(Object.keys(enUSUser.verification.student.platform.methods).sort()).toEqual(
      Object.keys(zhCNUser.verification.student.platform.methods).sort(),
    )
  })

  it('loads independent student credentials and capability-driven school methods', async () => {
    mockVerificationApi.listCredentials.mockImplementation(() => ok([{
      id: '22222222-2222-4222-8222-222222222222',
      schoolCode: school.code,
      schoolName: school.name,
      method: 'student_email_outbound_otp',
      status: 'active',
      credentialClass: 'formal_student',
      subjectDisplay: '20****01',
      verifiedAt: '2026-08-05T10:00:00Z',
      expiresAt: null,
      revision: 1,
    }]))

    const wrapper = await mountPage()

    expect(wrapper.get('[data-active-credentials]').text()).toContain(school.name)
    await wrapper.get('[data-verification-school-option]').trigger('click')
    expect(wrapper.findAll('[data-verification-method]')).toHaveLength(3)
    expect(mockVerificationApi.createApplication).not.toHaveBeenCalled()
  })

  it('opens a requested available method after the user selects a school', async () => {
    mockRoute.query = { method: 'real_name_identity_check', redirect: '/open-platform' }
    const wrapper = await mountPage()

    await wrapper.get('[data-verification-school-option]').trigger('click')
    await flushPromises()

    expect(mockVerificationApi.createApplication).toHaveBeenCalledWith({
      schoolCode: school.code,
    })
    expect(wrapper.find('[data-verification-method-form]').exists()).toBe(true)
    expect(mockRouterReplace).toHaveBeenCalledWith({
      query: { redirect: '/open-platform' },
    })
  })

  it('submits real-name evidence only after explicit privacy consent', async () => {
    mockVerificationApi.verifyRealName.mockImplementation(() => ok(application('approved')))
    const wrapper = await mountPage()

    await wrapper.get('[data-verification-school-option]').trigger('click')
    await wrapper.get('[data-verification-method="real_name_identity_check"]').trigger('click')
    await flushPromises()

    await wrapper.get('[data-verification-student-id]').setValue('20990001')
    await wrapper.get('[data-verification-name]').setValue('测试用户')
    await wrapper.get('[data-verification-document-number]').setValue('11010519491231002X')
    expect(wrapper.get<HTMLButtonElement>('[data-verification-submit]').element.disabled).toBe(true)

    await wrapper.get('[data-verification-consent]').setValue(true)
    await wrapper.get('[data-verification-method-form]').trigger('submit')
    await flushPromises()

    expect(mockVerificationApi.verifyRealName).toHaveBeenCalledWith(
      application().id,
      {
        studentID: '20990001',
        name: '测试用户',
        documentNumber: '11010519491231002X',
        privacyNoticeVersion: privacyNotice.version,
        sensitiveDataConsent: true,
      },
    )
    expect(wrapper.find('[data-verification-complete]').exists()).toBe(true)
  })

  it('requests email OTP without accepting a user-supplied email address', async () => {
    mockVerificationApi.requestStudentEmailOTP.mockImplementation(() => ok({
      applicationID: application().id,
      maskedEmail: '20****01@buaa.edu.cn',
      expiresAt: '2026-08-05T10:10:00Z',
      resendAvailableAt: '2026-08-05T10:01:00Z',
      remainingAttempts: 5,
    }))
    const wrapper = await mountPage()

    await wrapper.get('[data-verification-school-option]').trigger('click')
    await wrapper.get('[data-verification-method="student_email_outbound_otp"]').trigger('click')
    await flushPromises()
    await wrapper.get('[data-verification-student-id]').setValue('20990001')
    await wrapper.get('[data-verification-name]').setValue('测试用户')
    await wrapper.get('[data-verification-consent]').setValue(true)
    await wrapper.get('[data-verification-method-form]').trigger('submit')
    await flushPromises()

    expect(mockVerificationApi.requestStudentEmailOTP).toHaveBeenCalledWith(
      application().id,
      {
        studentID: '20990001',
        name: '测试用户',
        privacyNoticeVersion: privacyNotice.version,
        sensitiveDataConsent: true,
      },
    )
    expect(wrapper.find('input[type="email"]').exists()).toBe(false)
  })

  it('cancels a recoverable application through the new lifecycle endpoint', async () => {
    mockVerificationApi.cancelApplication.mockImplementation(() => ok({
      ...application(),
      status: 'cancelled',
      terminalCode: 'user_cancelled',
      revision: 2,
    }))
    const wrapper = await mountPage()

    await wrapper.get('[data-verification-school-option]').trigger('click')
    await wrapper.get('[data-verification-method="real_name_identity_check"]').trigger('click')
    await flushPromises()
    await wrapper.get('[data-verification-cancel]').trigger('click')
    await flushPromises()

    expect(mockVerificationApi.cancelApplication).toHaveBeenCalledWith(application().id)
  })
})
