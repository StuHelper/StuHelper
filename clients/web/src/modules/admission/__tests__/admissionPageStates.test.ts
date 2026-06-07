// @vitest-environment jsdom

import { flushPromises, mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { ApiError } from '@/api/errors'
import type { AdmissionSession } from '@stuhelper/shared/api'

import { rememberLinkedAdmissionSession } from '../admissionToken'

const mockAdmissionApi = vi.hoisted(() => ({
  getAdmissionMe: vi.fn(),
  getAdmissionSession: vi.fn(),
  linkAdmissionSession: vi.fn(),
}))

const mockAuth = vi.hoisted(() => ({
  bootstrapSession: vi.fn(),
  fetchUser: vi.fn(),
  isAuthenticated: true,
  login: vi.fn(),
  reauthenticate: vi.fn(),
  signup: vi.fn(),
  switchAccount: vi.fn(),
  user: {
    displayName: 'Alice Zhang',
    name: 'alice',
  },
}))

const mockVerificationStore = vi.hoisted(() => ({
  fetchQQBinding: vi.fn(),
  fetchSchools: vi.fn(),
  qqBinding: null as null | { qqID: string },
  schools: [],
}))

const mockToast = vi.hoisted(() => ({
  error: vi.fn(),
  success: vi.fn(),
}))

const mockHasStoredSessionHint = vi.hoisted(() => vi.fn())
const mockWaitForAdmissionProjection = vi.hoisted(() => vi.fn())
const mountedWrappers: Array<{ unmount(): void }> = []

const mockRoute = vi.hoisted(() => ({
  fullPath: '/verify/ABCD',
  params: { code: 'ABCD' },
  query: {},
}))

vi.mock('../api', () => ({
  admissionApi: mockAdmissionApi,
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => mockAuth,
}))

vi.mock('@/stores/verification', () => ({
  useVerificationStore: () => mockVerificationStore,
}))

vi.mock('@/composables/useToast', () => ({
  useToast: () => mockToast,
}))

vi.mock('@/utils/sessionHint', () => ({
  hasStoredSessionHint: mockHasStoredSessionHint,
}))

vi.mock('../projectionRefresh', () => ({
  waitForAdmissionProjection: mockWaitForAdmissionProjection,
}))

vi.mock('vue-router', () => ({
  useRoute: () => mockRoute,
}))

describe('AdmissionPage edge states', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockAuth.isAuthenticated = true
    mockAuth.user = {
      displayName: 'Alice Zhang',
      name: 'alice',
    }
    mockAuth.bootstrapSession.mockResolvedValue(true)
    mockRoute.fullPath = '/verify/ABCD'
    mockRoute.params = { code: 'ABCD' }
    mockRoute.query = {}
    sessionStorage.clear()
    mockHasStoredSessionHint.mockReturnValue(true)
    mockWaitForAdmissionProjection.mockResolvedValue(false)
    mockVerificationStore.qqBinding = null
    mockVerificationStore.fetchQQBinding.mockResolvedValue(null)
    mockVerificationStore.schools = []
    mockVerificationStore.fetchSchools.mockResolvedValue(undefined)
    Object.defineProperty(navigator, 'clipboard', {
      configurable: true,
      value: {
        writeText: vi.fn().mockResolvedValue(undefined),
      },
    })
  })

  afterEach(() => {
    while (mountedWrappers.length > 0) {
      mountedWrappers.pop()?.unmount()
    }
    document.body.innerHTML = ''
    vi.useRealTimers()
  })

  it('blocks login and link actions when the token belongs to another account', async () => {
    mockAdmissionApi.getAdmissionSession.mockRejectedValueOnce(
      new ApiError({ code: 'admission.qq_mismatch', message: 'mismatch' }),
    )

    const wrapper = await mountAdmissionPage()
    await settleAdmissionPage(wrapper)
    expect(wrapper.find('[data-state="qqMismatch"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('QQ 账号不匹配')
    expect(wrapper.text()).toContain('当前登录的 StuHelper 账号已绑定其他 QQ')
    expect(wrapper.text()).not.toContain('开始认证')
    expect(mockAuth.login).not.toHaveBeenCalled()
    expect(mockAuth.switchAccount).not.toHaveBeenCalled()
    expect(mockAdmissionApi.linkAdmissionSession).not.toHaveBeenCalled()
  })

  it('shows expired state without submission controls', async () => {
    mockAdmissionApi.getAdmissionSession.mockResolvedValueOnce(
      sessionWithStatus('expired_kicked'),
    )

    const wrapper = await mountAdmissionPage()
    await settleAdmissionPage(wrapper)

    expect(wrapper.find('[data-state="expired"]').exists()).toBe(true)
    expect(wrapper.find('[data-admission-freshman-flow]').exists()).toBe(false)
    expect(wrapper.find('[data-school-email-otp-form]').exists()).toBe(false)
  })

  it('shows expired state for cancelled admission sessions', async () => {
    mockAdmissionApi.getAdmissionSession.mockResolvedValueOnce(
      sessionWithStatus('cancelled'),
    )

    const wrapper = await mountAdmissionPage()
    await settleAdmissionPage(wrapper)

    expect(wrapper.find('[data-state="expired"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('链接已失效')
    expect(wrapper.find('[data-admission-freshman-flow]').exists()).toBe(false)
    expect(wrapper.find('[data-school-email-otp-form]').exists()).toBe(false)
  })

  it('shows admission progress and the link deadline before account binding', async () => {
    mockAdmissionApi.getAdmissionSession.mockResolvedValueOnce(
      sessionWithStatus('joined_muted'),
    )

    const wrapper = await mountAdmissionPage()
    await settleAdmissionPage(wrapper)

    expect(wrapper.find('[data-admission-progress]').exists()).toBe(true)
    expect(wrapper.find('[data-admission-progress-current]').text()).toContain('账号绑定')
    expect(wrapper.find('[data-admission-active-deadline]').text()).toContain('绑定账号截止')
    expect(wrapper.find('[data-admission-mute-deadline]').text()).toContain(
      '学生认证通过后会提前解除',
    )
  })

  it('shows QQ mismatch before link confirmation when the signed-in account is already bound to another QQ', async () => {
    mockAdmissionApi.getAdmissionSession.mockResolvedValueOnce({
      ...sessionWithStatus('joined_muted'),
      qqID: '990060607888',
    })
    mockVerificationStore.fetchQQBinding.mockResolvedValueOnce({
      qqID: '990060607003',
    })

    const wrapper = await mountAdmissionPage()
    await settleAdmissionPage(wrapper)

    expect(wrapper.find('[data-state="qqMismatch"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('这条认证链接属于 QQ 990060607888')
    expect(wrapper.text()).toContain('当前登录的 StuHelper 账号已绑定其他 QQ')
    expect(wrapper.text()).not.toContain('开始认证')
    expect(mockAdmissionApi.linkAdmissionSession).not.toHaveBeenCalled()
  })

  it('shows admission progress and the submission deadline after account binding', async () => {
    mockAdmissionApi.getAdmissionSession.mockResolvedValueOnce(
      sessionWithStatus('linked'),
    )
    mockAdmissionApi.getAdmissionMe.mockResolvedValueOnce({
      projectionPending: false,
      session: sessionWithStatus('linked'),
      status: 'linked',
    })

    const wrapper = await mountAdmissionPage()
    await settleAdmissionPage(wrapper)

    expect(wrapper.find('[data-admission-progress-current]').text()).toContain('学生认证')
    expect(wrapper.find('[data-admission-active-deadline]').text()).toContain('提交认证截止')
  })

  it('shows the manual review deadline while waiting for freshman review', async () => {
    mockAdmissionApi.getAdmissionSession.mockResolvedValueOnce({
      ...sessionWithStatus('material_submitted'),
      manualReviewDeadlineAt: '2026-05-06T00:00:00Z',
    })

    const wrapper = await mountAdmissionPage()
    await settleAdmissionPage(wrapper)

    expect(wrapper.find('[data-admission-progress-current]').text()).toContain('审核同步')
    expect(wrapper.find('[data-admission-active-deadline]').text()).toContain('审核处理截止')
  })

  it('shows an expired-link state for missing admission tokens', async () => {
    mockAdmissionApi.getAdmissionSession.mockRejectedValueOnce(
      { code: 'admission.session_not_found', message: 'missing' },
    )

    const wrapper = await mountAdmissionPage()
    await settleAdmissionPage(wrapper)

    expect(wrapper.find('[data-state="expired"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('链接已失效')
    expect(wrapper.text()).toContain('重新生成认证链接 <QQ号>')
    expect(wrapper.text()).not.toContain('认证链接暂时无法打开')
  })

  it('copies the admission reissue command from expired links', async () => {
    mockAdmissionApi.getAdmissionSession.mockRejectedValueOnce(
      { code: 'admission.session_not_found', message: 'missing' },
    )

    const wrapper = await mountAdmissionPage()
    await settleAdmissionPage(wrapper)
    await wrapper.find('[data-admission-copy-reissue-command]').trigger('click')
    await flushPromises()

    expect(navigator.clipboard.writeText).toHaveBeenCalledWith(
      '重新生成认证链接 <QQ号>',
    )
    expect(mockToast.success).toHaveBeenCalledWith('重新生成指令已复制')
  })

  it('maps material_submitted session status to pendingReview', async () => {
    mockAdmissionApi.getAdmissionSession.mockResolvedValueOnce(
      sessionWithStatus('material_submitted'),
    )

    const wrapper = await mountAdmissionPage()
    await settleAdmissionPage(wrapper)

    expect(wrapper.find('[data-state="pendingReview"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('等待管理员审核')
  })

  it('polls pending review state and advances when a freshman review is approved', async () => {
    vi.useFakeTimers()
    mockAdmissionApi.getAdmissionSession.mockResolvedValueOnce(
      sessionWithStatus('material_submitted'),
    )
    mockAdmissionApi.getAdmissionMe.mockResolvedValueOnce({
      projectionPending: true,
      session: {
        ...sessionWithStatus('verified'),
        projectionPending: true,
      },
      status: 'verified',
    })

    const wrapper = await mountAdmissionPage()
    await flushPromises()
    await vi.advanceTimersByTimeAsync(0)
    await flushPromises()
    await nextTick()

    expect(wrapper.find('[data-state="pendingReview"]').exists()).toBe(true)
    expect(mockAdmissionApi.getAdmissionMe).not.toHaveBeenCalled()

    await vi.advanceTimersByTimeAsync(5000)
    await flushPromises()
    await nextTick()

    expect(mockAdmissionApi.getAdmissionMe).toHaveBeenCalledTimes(1)
    expect(mockAdmissionApi.getAdmissionMe).toHaveBeenCalledWith('session-material_submitted')
    expect(wrapper.find('[data-state="projectionPending"]').exists()).toBe(true)
    expect(mockWaitForAdmissionProjection).toHaveBeenCalledWith({
      refreshAuth: mockAuth.fetchUser,
      signal: expect.any(AbortSignal),
    })
  })

  it('resumes the authenticated admission flow when the URL token was already consumed', async () => {
    mockAdmissionApi.getAdmissionSession.mockRejectedValueOnce(
      new ApiError({ code: 'admission.token_consumed', message: 'consumed' }),
    )
    mockAdmissionApi.linkAdmissionSession.mockResolvedValueOnce(
      sessionWithStatus('linked'),
    )
    mockAdmissionApi.getAdmissionMe.mockResolvedValueOnce({
      projectionPending: false,
      session: sessionWithStatus('linked'),
      status: 'linked',
    })

    const wrapper = await mountAdmissionPage()
    await settleAdmissionPage(wrapper)

    expect(wrapper.find('[data-state="linked"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('选择认证方式')
    expect(mockAdmissionApi.linkAdmissionSession).toHaveBeenCalledWith('ABCD')
    expect(mockAdmissionApi.getAdmissionMe).toHaveBeenCalledTimes(1)
    expect(mockAdmissionApi.getAdmissionMe).toHaveBeenCalledWith('session-linked')
    expect(mockVerificationStore.fetchSchools).toHaveBeenCalledTimes(1)
  })

  it('uses the remembered linked session before reloading a consumed admission token', async () => {
    rememberLinkedAdmissionSession('ABCD', 'session-linked')
    mockAdmissionApi.getAdmissionMe.mockResolvedValueOnce({
      projectionPending: false,
      session: sessionWithStatus('material_submitted'),
      status: 'material_submitted',
    })

    const wrapper = await mountAdmissionPage()
    await settleAdmissionPage(wrapper)

    expect(mockAuth.bootstrapSession).toHaveBeenCalledWith({ force: true })
    expect(mockAdmissionApi.getAdmissionSession).not.toHaveBeenCalled()
    expect(mockAdmissionApi.linkAdmissionSession).not.toHaveBeenCalled()
    expect(mockAdmissionApi.getAdmissionMe).toHaveBeenCalledTimes(1)
    expect(mockAdmissionApi.getAdmissionMe).toHaveBeenCalledWith('session-linked')
    expect(wrapper.find('[data-state="pendingReview"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('等待管理员审核')
  })

  it('defaults linked users to old-student verification when a school supports it', async () => {
    mockVerificationStore.schools = [{
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
    }]
    mockAdmissionApi.getAdmissionSession.mockResolvedValueOnce(
      sessionWithStatus('linked'),
    )
    mockAdmissionApi.getAdmissionMe.mockResolvedValueOnce({
      projectionPending: false,
      session: sessionWithStatus('linked'),
      status: 'linked',
    })

    const wrapper = await mountAdmissionPage()
    await settleAdmissionPage(wrapper)

    expect(wrapper.find('[data-admission-old-student-flow]').exists()).toBe(true)
    expect(wrapper.find('[data-school-email-otp-form]').exists()).toBe(true)
    expect(wrapper.find('[data-admission-freshman-flow]').exists()).toBe(false)

    const freshmanTab = wrapper.findAll('.flow-tab').find((button) => (
      button.text() === '新生认证'
    ))
    expect(freshmanTab).toBeTruthy()
    await freshmanTab!.trigger('click')
    await settleAdmissionPage(wrapper)

    expect(wrapper.find('[data-admission-freshman-flow]').exists()).toBe(true)
  })

  it('asks for login without offering signup when a consumed token is reopened logged out', async () => {
    mockAuth.isAuthenticated = false
    mockAuth.bootstrapSession.mockResolvedValueOnce(false)
    mockAdmissionApi.getAdmissionSession.mockRejectedValueOnce(
      new ApiError({ code: 'admission.token_consumed', message: 'consumed' }),
    )

    const wrapper = await mountAdmissionPage()
    await settleAdmissionPage(wrapper)

    expect(wrapper.find('[data-state="needsLogin"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('该链接已绑定 StuHelper 账号')
    expect(wrapper.text()).not.toContain('注册')
    expect(mockAdmissionApi.linkAdmissionSession).not.toHaveBeenCalled()
  })

  it('rejects a consumed token when the current login is not the originally bound account', async () => {
    mockAdmissionApi.getAdmissionSession.mockRejectedValueOnce(
      new ApiError({ code: 'admission.token_consumed', message: 'consumed' }),
    )
    mockAdmissionApi.linkAdmissionSession.mockRejectedValueOnce(
      new ApiError({ code: 'admission.token_consumed', message: 'consumed' }),
    )

    const wrapper = await mountAdmissionPage()
    await settleAdmissionPage(wrapper)

    expect(wrapper.find('[data-state="accountMismatch"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('账号不匹配')
    expect(wrapper.text()).toContain('重新生成认证链接 <QQ号>')
    expect(wrapper.text()).toContain('首次打开时登录的 StuHelper 账号')
    expect(mockAdmissionApi.getAdmissionMe).not.toHaveBeenCalled()
  })

  it('starts bounded auth refresh when approval waits for role projection', async () => {
    mockAdmissionApi.getAdmissionSession.mockResolvedValueOnce({
      ...sessionWithStatus('verified'),
      projectionPending: true,
    })

    const wrapper = await mountAdmissionPage()
    await settleAdmissionPage(wrapper)

    expect(wrapper.find('[data-state="projectionPending"]').exists()).toBe(true)
    expect(wrapper.find('[data-projection-timeout]').exists()).toBe(true)
    expect(mockWaitForAdmissionProjection).toHaveBeenCalledWith({
      refreshAuth: mockAuth.fetchUser,
      signal: expect.any(AbortSignal),
    })
  })

  it('uses admission me projection state after refreshing linked resources', async () => {
    mockAdmissionApi.getAdmissionSession.mockResolvedValueOnce(
      sessionWithStatus('linked'),
    )
    mockAdmissionApi.getAdmissionMe.mockResolvedValueOnce({
      projectionPending: true,
      session: {
        ...sessionWithStatus('linked'),
        projectionPending: true,
      },
      status: 'linked',
    })

    const wrapper = await mountAdmissionPage()
    await settleAdmissionPage(wrapper)

    expect(wrapper.find('[data-state="projectionPending"]').exists()).toBe(true)
    expect(wrapper.find('[data-projection-timeout]').exists()).toBe(true)
    expect(mockVerificationStore.fetchSchools).toHaveBeenCalledTimes(1)
    expect(mockWaitForAdmissionProjection).toHaveBeenCalledWith({
      refreshAuth: mockAuth.fetchUser,
      signal: expect.any(AbortSignal),
    })
  })

  it('refreshes browser session before showing the logged-out admission state', async () => {
    mockAuth.isAuthenticated = false
    mockAuth.bootstrapSession.mockResolvedValueOnce(true)
    mockAdmissionApi.getAdmissionSession.mockResolvedValueOnce(
      sessionWithStatus('joined_muted'),
    )

    const wrapper = await mountAdmissionPage()
    await settleAdmissionPage(wrapper)

    expect(mockAuth.bootstrapSession).toHaveBeenCalledWith({ force: true })
    expect(wrapper.find('[data-state="ready"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('开始认证')
  })

  it('shows login without probing auth state when no local session hint exists', async () => {
    mockAuth.isAuthenticated = false
    mockHasStoredSessionHint.mockReturnValue(false)
    mockAdmissionApi.getAdmissionSession.mockResolvedValueOnce(
      sessionWithStatus('joined_muted'),
    )

    const wrapper = await mountAdmissionPage()
    await settleAdmissionPage(wrapper)

    expect(mockAuth.bootstrapSession).not.toHaveBeenCalled()
    expect(wrapper.find('[data-state="needsLogin"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('登录 StuHelper')
  })

  it('refreshes the admission state when returning from SSO through browser cache', async () => {
    mockAuth.isAuthenticated = false
    mockAuth.bootstrapSession
      .mockResolvedValueOnce(false)
      .mockResolvedValueOnce(true)
    mockAdmissionApi.getAdmissionSession.mockResolvedValue(
      sessionWithStatus('joined_muted'),
    )

    const wrapper = await mountAdmissionPage()
    await settleAdmissionPage(wrapper)
    expect(wrapper.find('[data-state="needsLogin"]').exists()).toBe(true)

    window.dispatchEvent(new Event('pageshow'))
    await settleAdmissionPage(wrapper)

    expect(mockAdmissionApi.getAdmissionSession).toHaveBeenCalledTimes(2)
    expect(mockAuth.bootstrapSession).toHaveBeenCalledTimes(2)
    expect(wrapper.find('[data-state="ready"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('开始认证')
  })

  it('requires manually typing the current qq before binding the StuHelper account', async () => {
    mockAdmissionApi.getAdmissionSession.mockResolvedValueOnce(
      sessionWithStatus('joined_muted'),
    )
    mockAdmissionApi.linkAdmissionSession.mockResolvedValueOnce(
      sessionWithStatus('linked'),
    )
    mockAdmissionApi.getAdmissionMe.mockResolvedValueOnce({
      projectionPending: false,
      session: sessionWithStatus('linked'),
      status: 'linked',
    })

    const wrapper = await mountAdmissionPage()
    await settleAdmissionPage(wrapper)
    await openBindConfirmationDialog(wrapper)

    const dialog = getDialogElement<HTMLElement>('[data-admission-bind-confirmation-dialog]')
    expect(dialog.textContent).toContain('[Alice Zhang]')
    expect(dialog.textContent).toContain('[123]')

    const submit = getDialogElement<HTMLButtonElement>(
      '[data-admission-bind-confirmation-submit]',
    )
    expect(submit.disabled).toBe(true)

    await typeBindConfirmationQQ('124')
    getDialogElement<HTMLInputElement>(
      '[data-admission-bind-confirmation-input]',
    ).dispatchEvent(new Event('blur', { bubbles: true }))
    await nextTick()

    expect(getDialogElement<HTMLElement>(
      '[data-admission-bind-confirmation-error]',
    ).textContent).toContain('输入的 QQ 号与本次入群 QQ 不一致')
    expect(submit.disabled).toBe(true)
    expect(mockAdmissionApi.linkAdmissionSession).not.toHaveBeenCalled()

    await typeBindConfirmationQQ('123')
    expect(submit.disabled).toBe(false)
    submit.click()
    await settleAdmissionPage(wrapper)

    expect(mockAdmissionApi.linkAdmissionSession).toHaveBeenCalledWith('ABCD')
    expect(wrapper.find('[data-state="linked"]').exists()).toBe(true)
  })

  it('accepts copied qq confirmation input with surrounding whitespace', async () => {
    mockAdmissionApi.getAdmissionSession.mockResolvedValueOnce(
      sessionWithStatus('joined_muted'),
    )
    mockAdmissionApi.linkAdmissionSession.mockResolvedValueOnce(
      sessionWithStatus('linked'),
    )
    mockAdmissionApi.getAdmissionMe.mockResolvedValueOnce({
      projectionPending: false,
      session: sessionWithStatus('linked'),
      status: 'linked',
    })

    const wrapper = await mountAdmissionPage()
    await settleAdmissionPage(wrapper)
    await confirmCurrentQQBinding(wrapper, ' 123 ')
    await settleAdmissionPage(wrapper)

    expect(mockAdmissionApi.linkAdmissionSession).toHaveBeenCalledWith('ABCD')
    expect(wrapper.find('[data-state="linked"]').exists()).toBe(true)
  })

  it('refreshes the admission state when the SSO tab returns focus without pageshow', async () => {
    mockAuth.isAuthenticated = false
    mockAuth.bootstrapSession
      .mockResolvedValueOnce(false)
      .mockResolvedValueOnce(true)
    mockAdmissionApi.getAdmissionSession.mockResolvedValue(
      sessionWithStatus('joined_muted'),
    )

    const wrapper = await mountAdmissionPage()
    await settleAdmissionPage(wrapper)
    expect(wrapper.find('[data-state="needsLogin"]').exists()).toBe(true)

    window.dispatchEvent(new Event('focus'))
    await settleAdmissionPage(wrapper)

    expect(mockAdmissionApi.getAdmissionSession).toHaveBeenCalledTimes(2)
    expect(mockAuth.bootstrapSession).toHaveBeenCalledTimes(2)
    expect(wrapper.find('[data-state="ready"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('开始认证')
  })

  it('ignores stale admission loads and reloads the current route token', async () => {
    const firstLoad = createDeferred<AdmissionSession>()
    mockAdmissionApi.getAdmissionSession
      .mockReturnValueOnce(firstLoad.promise)
      .mockResolvedValueOnce(sessionWithStatus('joined_muted'))

    const wrapper = await mountAdmissionPage()
    await flushPromises()

    mockRoute.fullPath = '/verify/EFGH'
    mockRoute.params = { code: 'EFGH' }
    mockRoute.query = {}
    window.dispatchEvent(new Event('focus'))
    await flushPromises()

    firstLoad.resolve(sessionWithStatus('linked'))
    await settleAdmissionPage(wrapper)

    expect(mockAdmissionApi.getAdmissionSession).toHaveBeenCalledTimes(2)
    expect(mockAdmissionApi.getAdmissionSession).toHaveBeenNthCalledWith(1, 'ABCD')
    expect(mockAdmissionApi.getAdmissionSession).toHaveBeenNthCalledWith(2, 'EFGH')
    expect(mockAdmissionApi.getAdmissionMe).not.toHaveBeenCalled()
    expect(mockVerificationStore.fetchSchools).not.toHaveBeenCalled()
    expect(wrapper.find('[data-state="ready"]').exists()).toBe(true)
  })

  it('keeps the linked admission page open when post-link resources fail to refresh', async () => {
    mockAdmissionApi.getAdmissionSession.mockResolvedValueOnce(
      sessionWithStatus('joined_muted'),
    )
    mockAdmissionApi.linkAdmissionSession.mockResolvedValueOnce(
      sessionWithStatus('linked'),
    )
    mockAdmissionApi.getAdmissionMe.mockRejectedValueOnce(new Error('admission me unavailable'))

    const wrapper = await mountAdmissionPage()
    await settleAdmissionPage(wrapper)
    await confirmCurrentQQBinding(wrapper)
    await settleAdmissionPage(wrapper)

    expect(wrapper.find('[data-state="linked"]').exists()).toBe(true)
    expect(wrapper.find('[data-linked-resource-error]').text()).toContain(
      'admission me unavailable',
    )
    expect(wrapper.find('[data-state="error"]').exists()).toBe(false)
  })

  it('moves to expired when linked resources report a cancelled admission session', async () => {
    mockAdmissionApi.getAdmissionSession.mockResolvedValueOnce(
      sessionWithStatus('joined_muted'),
    )
    mockAdmissionApi.linkAdmissionSession.mockResolvedValueOnce(
      sessionWithStatus('linked'),
    )
    mockAdmissionApi.getAdmissionMe.mockResolvedValueOnce({
      projectionPending: false,
      session: sessionWithStatus('cancelled'),
      status: 'cancelled',
    })

    const wrapper = await mountAdmissionPage()
    await settleAdmissionPage(wrapper)
    await confirmCurrentQQBinding(wrapper)
    await settleAdmissionPage(wrapper)

    expect(wrapper.find('[data-state="expired"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('链接已失效')
    expect(wrapper.find('[data-state="error"]').exists()).toBe(false)
  })

  it('shows account mismatch when the token is consumed during explicit linking', async () => {
    mockAdmissionApi.getAdmissionSession.mockResolvedValueOnce(
      sessionWithStatus('joined_muted'),
    )
    mockAdmissionApi.linkAdmissionSession.mockRejectedValueOnce(
      new ApiError({ code: 'admission.token_consumed', message: 'consumed' }),
    )

    const wrapper = await mountAdmissionPage()
    await settleAdmissionPage(wrapper)
    await confirmCurrentQQBinding(wrapper)
    await settleAdmissionPage(wrapper)

    expect(wrapper.find('[data-state="accountMismatch"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('账号不匹配')
    expect(wrapper.text()).toContain('重新生成认证链接 123')
    expect(wrapper.text()).not.toContain('链接已失效')
  })

  it('shows qq mismatch when explicit linking finds another qq already bound', async () => {
    mockAdmissionApi.getAdmissionSession.mockResolvedValueOnce(
      sessionWithStatus('joined_muted'),
    )
    mockAdmissionApi.linkAdmissionSession.mockRejectedValueOnce(
      new ApiError({ code: 'admission.qq_mismatch', message: 'admission qq mismatch' }),
    )

    const wrapper = await mountAdmissionPage()
    await settleAdmissionPage(wrapper)
    await confirmCurrentQQBinding(wrapper)
    await settleAdmissionPage(wrapper)

    expect(wrapper.find('[data-state="qqMismatch"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('QQ 账号不匹配')
    expect(wrapper.text()).toContain('当前登录的 StuHelper 账号已绑定其他 QQ')
    await wrapper.get('button.primary-button').trigger('click')
    expect(mockAuth.switchAccount).toHaveBeenCalledWith('http://localhost:3000/verify/ABCD')
    expect(mockAuth.reauthenticate).not.toHaveBeenCalled()
    expect(wrapper.text()).toContain('重新生成认证链接 123')
    expect(wrapper.find('[data-state="error"]').exists()).toBe(false)
  })

  it('moves to expired when linked resources report an expired admission session', async () => {
    mockAdmissionApi.getAdmissionSession.mockResolvedValueOnce(
      sessionWithStatus('joined_muted'),
    )
    mockAdmissionApi.linkAdmissionSession.mockResolvedValueOnce(
      sessionWithStatus('linked'),
    )
    mockAdmissionApi.getAdmissionMe.mockRejectedValueOnce(
      new ApiError({ code: 'admission.token_expired', message: 'expired' }),
    )

    const wrapper = await mountAdmissionPage()
    await settleAdmissionPage(wrapper)
    await confirmCurrentQQBinding(wrapper)
    await settleAdmissionPage(wrapper)

    expect(wrapper.find('[data-state="expired"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('链接已失效')
  })
})

function sessionWithStatus(status: AdmissionSession['status']): AdmissionSession {
  return {
    guildID: '100',
    id: `session-${status}`,
    initialMuteUntil: '2026-05-04T01:00:00Z',
    linkWaitDeadlineAt: '2026-05-04T01:00:00Z',
    platform: 'qq',
    projectionPending: false,
    qqID: '123',
    status,
    submissionWaitDeadlineAt: '2026-05-05T00:00:00Z',
    tokenExpiresAt: '2026-05-04T01:00:00Z',
  }
}

async function settleAdmissionPage(
  wrapper: Awaited<ReturnType<typeof mountAdmissionPage>>,
): Promise<void> {
  await flushPromises()
  await flushPromises()
  await new Promise((resolve) => setTimeout(resolve, 0))
  await flushPromises()
  await wrapper.vm.$forceUpdate()
  await nextTick()
}

async function mountAdmissionPage() {
  const component = await import('../views/AdmissionPage.vue')
  const target = document.createElement('div')
  document.body.appendChild(target)
  const wrapper = mount(component.default, {
    attachTo: target,
  })
  mountedWrappers.push({
    unmount() {
      wrapper.unmount()
      target.remove()
    },
  })
  return wrapper
}

async function openBindConfirmationDialog(
  wrapper: Awaited<ReturnType<typeof mountAdmissionPage>>,
): Promise<void> {
  await wrapper.get('[data-admission-open-bind-confirmation]').trigger('click')
  await settleAdmissionPage(wrapper)
}

async function confirmCurrentQQBinding(
  wrapper: Awaited<ReturnType<typeof mountAdmissionPage>>,
  qq = '123',
): Promise<void> {
  await openBindConfirmationDialog(wrapper)
  await typeBindConfirmationQQ(qq)
  getDialogElement<HTMLButtonElement>('[data-admission-bind-confirmation-submit]').click()
  await settleAdmissionPage(wrapper)
}

async function typeBindConfirmationQQ(qq: string): Promise<void> {
  const input = getDialogElement<HTMLInputElement>('[data-admission-bind-confirmation-input]')
  input.value = qq
  input.dispatchEvent(new Event('input', { bubbles: true }))
  await nextTick()
  await flushPromises()
}

function getDialogElement<T extends Element>(selector: string): T {
  const element = document.body.querySelector<T>(selector)
  if (!element) {
    throw new Error(`Missing dialog element: ${selector}`)
  }
  return element
}

function createDeferred<T>() {
  let resolve!: (value: T) => void
  let reject!: (reason?: unknown) => void
  const promise = new Promise<T>((promiseResolve, promiseReject) => {
    resolve = promiseResolve
    reject = promiseReject
  })
  return { promise, reject, resolve }
}
