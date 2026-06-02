// @vitest-environment jsdom

import { flushPromises, mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { ApiError } from '@/api/errors'
import type { AdmissionSession } from '@stuhelper/shared/api'

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
}))

const mockVerificationStore = vi.hoisted(() => ({
  fetchSchools: vi.fn(),
  schools: [],
}))

const mockWaitForAdmissionProjection = vi.hoisted(() => vi.fn())
const mountedWrappers: Array<{ unmount(): void }> = []

const mockRoute = vi.hoisted(() => ({
  fullPath: '/verify/ABCD?qq=123',
  params: { code: 'ABCD' },
  query: { qq: '123' },
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
    mockAuth.bootstrapSession.mockResolvedValue(true)
    mockRoute.fullPath = '/verify/ABCD?qq=123'
    mockRoute.params = { code: 'ABCD' }
    mockRoute.query = { qq: '123' }
    mockWaitForAdmissionProjection.mockResolvedValue(false)
    mockVerificationStore.schools = []
    mockVerificationStore.fetchSchools.mockResolvedValue(undefined)
  })

  afterEach(() => {
    while (mountedWrappers.length > 0) {
      mountedWrappers.pop()?.unmount()
    }
  })

  it('blocks login and link actions when token QQ does not match query QQ', async () => {
    mockAdmissionApi.getAdmissionSession.mockRejectedValueOnce(
      new ApiError({ code: 'admission.qq_mismatch', message: 'mismatch' }),
    )

    const wrapper = await mountAdmissionPage()
    await settleAdmissionPage(wrapper)
    expect(wrapper.find('[data-state="qqMismatch"]').exists()).toBe(true)
    expect(wrapper.text()).not.toContain('登录')
    expect(wrapper.text()).not.toContain('开始认证')
    expect(mockAuth.login).not.toHaveBeenCalled()
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

  it('maps material_submitted session status to pendingReview', async () => {
    mockAdmissionApi.getAdmissionSession.mockResolvedValueOnce(
      sessionWithStatus('material_submitted'),
    )

    const wrapper = await mountAdmissionPage()
    await settleAdmissionPage(wrapper)

    expect(wrapper.find('[data-state="pendingReview"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('等待管理员审核')
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
    expect(mockAdmissionApi.linkAdmissionSession).toHaveBeenCalledWith('ABCD', '123')
    expect(mockAdmissionApi.getAdmissionMe).toHaveBeenCalledTimes(1)
    expect(mockVerificationStore.fetchSchools).toHaveBeenCalledTimes(1)
  })

  it('defaults linked users to old-student verification when a school supports it', async () => {
    mockVerificationStore.schools = [{
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

    mockRoute.fullPath = '/verify/EFGH?qq=456'
    mockRoute.params = { code: 'EFGH' }
    mockRoute.query = { qq: '456' }
    window.dispatchEvent(new Event('focus'))
    await flushPromises()

    firstLoad.resolve(sessionWithStatus('linked'))
    await settleAdmissionPage(wrapper)

    expect(mockAdmissionApi.getAdmissionSession).toHaveBeenCalledTimes(2)
    expect(mockAdmissionApi.getAdmissionSession).toHaveBeenNthCalledWith(1, 'ABCD', '123')
    expect(mockAdmissionApi.getAdmissionSession).toHaveBeenNthCalledWith(2, 'EFGH', '456')
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
    await wrapper.get('button.primary-button').trigger('click')
    await settleAdmissionPage(wrapper)

    expect(wrapper.find('[data-state="linked"]').exists()).toBe(true)
    expect(wrapper.find('[data-linked-resource-error]').text()).toContain(
      'admission me unavailable',
    )
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
    await wrapper.get('button.primary-button').trigger('click')
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
  const wrapper = mount(component.default)
  mountedWrappers.push(wrapper)
  return wrapper
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
