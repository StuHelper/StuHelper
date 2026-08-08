// @vitest-environment jsdom

import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const mocks = vi.hoisted(() => ({
  auth: {
    isAuthenticated: true,
    user: {
      id: 'user-1',
      name: 'alice',
      displayName: 'Alice',
      email: 'alice@example.edu',
      avatar: '',
    },
  },
  verification: {
    userSurface: null as null | { phone?: string | null },
    phoneStatus: null as null | { maskedPhone?: string | null },
    qqBinding: null as null | { qqID: string },
    studentVerified: false,
    phoneVerified: false,
    fetchStatus: vi.fn(),
  },
}))

vi.mock('@/stores/auth', () => ({ useAuthStore: () => mocks.auth }))
vi.mock('@/stores/verification', () => ({ useVerificationStore: () => mocks.verification }))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))

const { default: AccountProfilePage } = await import('../views/AccountProfilePage.vue')

function mountPage() {
  return mount(AccountProfilePage, {
    global: { stubs: { RouterLink: { template: '<a><slot /></a>' } } },
  })
}

describe('AccountProfilePage current verification projections', () => {
  beforeEach(() => {
    mocks.auth.isAuthenticated = true
    mocks.verification.userSurface = null
    mocks.verification.phoneStatus = null
    mocks.verification.qqBinding = null
    mocks.verification.studentVerified = false
    mocks.verification.phoneVerified = false
    mocks.verification.fetchStatus.mockReset()
  })

  it('does not render negative gate conclusions before a successful response', async () => {
    let resolveStatus!: () => void
    mocks.verification.fetchStatus.mockReturnValue(new Promise<void>((resolve) => {
      resolveStatus = resolve
    }))

    const wrapper = mountPage()
    expect(wrapper.find('[data-account-profile-status-loading]').exists()).toBe(true)
    expect(wrapper.text()).toContain('alice@example.edu')
    expect(wrapper.text()).not.toContain('user.accountProfile.status.unverified')
    expect(wrapper.text()).not.toContain('user.accountProfile.status.unbound')

    resolveStatus()
    await flushPromises()

    expect(wrapper.find('[data-account-profile-status-loading]').exists()).toBe(false)
    expect(wrapper.text()).toContain('user.verification.student.unverified')
    expect(wrapper.text()).toContain('user.accountProfile.status.unverified')
    expect(wrapper.text()).toContain('user.accountProfile.status.unbound')
  })

  it('keeps account facts on dependency failure and recovers through retry', async () => {
    const warn = vi.spyOn(console, 'warn').mockImplementation(() => undefined)
    mocks.verification.fetchStatus
      .mockRejectedValueOnce(new Error('temporary status outage'))
      .mockImplementationOnce(async () => {
        mocks.verification.userSurface = { phone: '138****0000' }
        mocks.verification.phoneStatus = { maskedPhone: '138****0000' }
        mocks.verification.qqBinding = { qqID: '10001' }
        mocks.verification.studentVerified = true
        mocks.verification.phoneVerified = true
      })

    const wrapper = mountPage()
    await flushPromises()
    expect(wrapper.find('[data-account-profile-status-error]').exists()).toBe(true)
    expect(wrapper.text()).toContain('alice@example.edu')
    expect(wrapper.text()).not.toContain('user.accountProfile.status.unverified')

    await wrapper.find('[data-account-profile-status-retry]').trigger('click')
    await flushPromises()

    expect(mocks.verification.fetchStatus).toHaveBeenCalledTimes(2)
    expect(wrapper.find('[data-account-profile-status-error]').exists()).toBe(false)
    expect(wrapper.text()).toContain('138****0000')
    expect(wrapper.text()).toContain('10001')
    expect(wrapper.text()).toContain('user.verification.student.verified')
    expect(wrapper.text()).toContain('user.accountProfile.status.verified')
    warn.mockRestore()
  })
})
