// @vitest-environment jsdom

import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const mockUpdatePageMeta = vi.fn()
const mockToastError = vi.fn()

const mockAuth = vi.hoisted(() => ({
  bootstrapSession: vi.fn(),
  isAuthenticated: false,
  loading: false,
  login: vi.fn(),
  signup: vi.fn(),
}))

const mockVerificationStore = vi.hoisted(() => ({
  fetchSchools: vi.fn(),
  fetchStatus: vi.fn(),
  qqBound: false,
  studentVerified: false,
}))

const mockAdmissionApi = vi.hoisted(() => ({
  getAdmissionMe: vi.fn(),
  getAdmissionSession: vi.fn(),
  linkAdmissionSession: vi.fn(),
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => mockAuth,
}))

vi.mock('@/stores/verification', () => ({
  useVerificationStore: () => mockVerificationStore,
}))

vi.mock('@/composables/usePageMeta', () => ({
  updatePageMeta: mockUpdatePageMeta,
}))

vi.mock('@/composables/useToast', () => ({
  useToast: () => ({
    error: mockToastError,
  }),
}))

vi.mock('@/modules/admission/api', () => ({
  admissionApi: mockAdmissionApi,
}))

const { default: JoinStartPage } = await import('../views/JoinStartPage.vue')

describe('JoinStartPage', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockAuth.bootstrapSession.mockResolvedValue(false)
    mockAuth.isAuthenticated = false
    mockAuth.loading = false
    mockVerificationStore.fetchStatus.mockResolvedValue(undefined)
    mockVerificationStore.fetchSchools.mockResolvedValue(undefined)
    mockVerificationStore.qqBound = false
    mockVerificationStore.studentVerified = false
  })

  it('shows login actions when no StuHelper session is available', async () => {
    const wrapper = mount(JoinStartPage)
    await flushPromises()

    expect(wrapper.find('[data-state="needsLogin"]').exists()).toBe(true)
    expect(mockVerificationStore.fetchStatus).not.toHaveBeenCalled()

    await wrapper.find('button.join-start-primary-button').trigger('click')
    expect(mockAuth.login).toHaveBeenCalledWith(window.location.href)

    await wrapper.find('button.join-start-secondary-button').trigger('click')
    expect(mockAuth.signup).toHaveBeenCalledWith(window.location.href)
  })

  it('shows student verification first for authenticated users without student status', async () => {
    mockAuth.bootstrapSession.mockResolvedValue(true)
    mockAuth.isAuthenticated = true
    mockVerificationStore.studentVerified = false
    mockVerificationStore.qqBound = false

    const wrapper = mount(JoinStartPage)
    await flushPromises()

    expect(wrapper.find('[data-state="needsStudentVerification"]').exists()).toBe(true)
    expect(wrapper.get('[data-open-student-verification]').attributes('href')).toBe(
      'http://localhost:3000/user/student-verification?redirect=http%3A%2F%2Flocalhost%3A3000%2Fstart',
    )
    expect(mockVerificationStore.fetchStatus).toHaveBeenCalledTimes(1)
    expect(mockVerificationStore.fetchSchools).not.toHaveBeenCalled()
    expect(mockAdmissionApi.getAdmissionSession).not.toHaveBeenCalled()
    expect(mockAdmissionApi.getAdmissionMe).not.toHaveBeenCalled()
    expect(mockAdmissionApi.linkAdmissionSession).not.toHaveBeenCalled()
  })

  it('shows QQ binding after student verification is complete', async () => {
    mockAuth.bootstrapSession.mockResolvedValue(true)
    mockAuth.isAuthenticated = true
    mockVerificationStore.studentVerified = true
    mockVerificationStore.qqBound = false

    const wrapper = mount(JoinStartPage)
    await flushPromises()

    expect(wrapper.find('[data-state="needsQQBinding"]').exists()).toBe(true)
    expect(wrapper.get('[data-open-qq-binding]').attributes('href')).toBe(
      'http://localhost:3000/user/qq-binding?redirect=http%3A%2F%2Flocalhost%3A3000%2Fstart',
    )
    expect(mockVerificationStore.fetchStatus).toHaveBeenCalledTimes(1)
    expect(mockVerificationStore.fetchSchools).not.toHaveBeenCalled()
  })

  it('shows ready when both student verification and QQ binding are complete', async () => {
    mockAuth.bootstrapSession.mockResolvedValue(true)
    mockAuth.isAuthenticated = true
    mockVerificationStore.studentVerified = true
    mockVerificationStore.qqBound = true

    const wrapper = mount(JoinStartPage)
    await flushPromises()

    expect(wrapper.find('[data-state="ready"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('账号已准备好')
  })

  it('can refresh readiness after returning from an independent module', async () => {
    mockAuth.bootstrapSession.mockResolvedValue(true)
    mockAuth.isAuthenticated = true
    mockVerificationStore.studentVerified = false
    mockVerificationStore.qqBound = false

    const wrapper = mount(JoinStartPage)
    await flushPromises()
    expect(wrapper.find('[data-state="needsStudentVerification"]').exists()).toBe(true)

    mockVerificationStore.studentVerified = true
    await wrapper.find('button.join-start-secondary-button').trigger('click')
    await flushPromises()

    expect(wrapper.find('[data-state="needsQQBinding"]').exists()).toBe(true)
    expect(mockVerificationStore.fetchStatus).toHaveBeenCalledTimes(2)
  })

  it('shows a retry state when readiness loading fails', async () => {
    mockAuth.bootstrapSession.mockResolvedValue(true)
    mockAuth.isAuthenticated = true
    mockVerificationStore.fetchStatus.mockRejectedValue(new Error('network'))

    const wrapper = mount(JoinStartPage)
    await flushPromises()

    expect(wrapper.find('[data-state="loadFailed"]').exists()).toBe(true)
    expect(mockToastError).toHaveBeenCalledWith('账号认证状态加载失败，请稍后重试')
  })
})
