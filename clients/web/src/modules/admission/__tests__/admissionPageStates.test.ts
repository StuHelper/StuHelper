// @vitest-environment jsdom

import { flushPromises, mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { ApiError } from '@/api/errors'
import type { AdmissionSession } from '@stuhelper/shared/api'

const mockAdmissionApi = vi.hoisted(() => ({
  getAdmissionMe: vi.fn(),
  getAdmissionSession: vi.fn(),
  linkAdmissionSession: vi.fn(),
}))

const mockAuth = vi.hoisted(() => ({
  fetchUser: vi.fn(),
  isAuthenticated: true,
  login: vi.fn(),
  signup: vi.fn(),
}))

const mockVerificationStore = vi.hoisted(() => ({
  fetchSchools: vi.fn(),
  schools: [],
}))

const mockWaitForAdmissionProjection = vi.hoisted(() => vi.fn())

const mockRoute = vi.hoisted(() => ({
  fullPath: '/admission/a/ABCD?qq=123',
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
    mockRoute.fullPath = '/admission/a/ABCD?qq=123'
    mockRoute.params = { code: 'ABCD' }
    mockRoute.query = { qq: '123' }
    mockWaitForAdmissionProjection.mockResolvedValue(false)
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
    })
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
  return mount(component.default)
}
