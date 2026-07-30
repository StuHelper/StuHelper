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
    identity: null as null | {
      verified: boolean
      reviewedAt: string | null
    },
    profile: null as null | {
      phone: string
      phoneVerified: boolean
      verificationStatus: string
    },
    qqBinding: null as null | {
      qqID: string
    },
    fetchStatus: vi.fn(),
  },
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => mocks.auth,
}))

vi.mock('@/stores/verification', () => ({
  useVerificationStore: () => mocks.verification,
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}))

const { default: AccountProfilePage } =
  await import('../views/AccountProfilePage.vue')

function mountPage() {
  return mount(AccountProfilePage, {
    global: {
      stubs: {
        RouterLink: {
          template: '<a><slot /></a>',
        },
      },
    },
  })
}

describe('AccountProfilePage verification status loading', () => {
  beforeEach(() => {
    mocks.auth.isAuthenticated = true
    mocks.verification.identity = null
    mocks.verification.profile = null
    mocks.verification.qqBinding = null
    mocks.verification.fetchStatus.mockReset()
  })

  it('does not render negative verification conclusions before a successful response', async () => {
    let resolveStatus!: () => void
    mocks.verification.fetchStatus.mockReturnValue(
      new Promise<void>((resolve) => {
        resolveStatus = resolve
      }),
    )

    const wrapper = mountPage()

    expect(wrapper.find('[data-account-profile-status-loading]').exists()).toBe(true)
    expect(wrapper.text()).toContain('alice@example.edu')
    expect(wrapper.text()).not.toContain('user.accountProfile.status.unverified')
    expect(wrapper.text()).not.toContain('user.accountProfile.status.unbound')
    expect(wrapper.text()).not.toContain('user.verification.identity.unverified')
    expect(wrapper.text()).not.toContain('user.verification.student.unverified')
    expect(wrapper.text()).not.toContain('user.accountProfile.missing.phone')
    expect(wrapper.text()).not.toContain('user.accountProfile.missing.qq')

    resolveStatus()
    await flushPromises()

    expect(wrapper.find('[data-account-profile-status-loading]').exists()).toBe(false)
    expect(wrapper.text()).toContain('user.accountProfile.status.unverified')
    expect(wrapper.text()).toContain('user.accountProfile.status.unbound')
    expect(wrapper.text()).toContain('user.verification.identity.unverified')
    expect(wrapper.text()).toContain('user.verification.student.unverified')

    wrapper.unmount()
  })

  it('keeps reliable account facts on failure and recovers through retry', async () => {
    const warn = vi.spyOn(console, 'warn').mockImplementation(() => undefined)
    mocks.verification.fetchStatus
      .mockRejectedValueOnce(new Error('temporary status outage'))
      .mockImplementationOnce(async () => {
        mocks.verification.identity = {
          verified: true,
          reviewedAt: '2026-07-31T00:00:00Z',
        }
        mocks.verification.profile = {
          phone: '138****0000',
          phoneVerified: true,
          verificationStatus: 'verified',
        }
        mocks.verification.qqBinding = {
          qqID: '10001',
        }
      })

    const wrapper = mountPage()
    await flushPromises()

    expect(wrapper.find('[data-account-profile-status-error]').exists()).toBe(true)
    expect(wrapper.text()).toContain('common.loadFailed')
    expect(wrapper.text()).toContain('alice@example.edu')
    expect(wrapper.text()).not.toContain('user.accountProfile.status.unverified')
    expect(wrapper.text()).not.toContain('user.accountProfile.status.unbound')
    expect(wrapper.text()).not.toContain('user.verification.identity.unverified')
    expect(wrapper.text()).not.toContain('user.verification.student.unverified')

    await wrapper.find('[data-account-profile-status-retry]').trigger('click')
    await flushPromises()

    expect(mocks.verification.fetchStatus).toHaveBeenCalledTimes(2)
    expect(wrapper.find('[data-account-profile-status-error]').exists()).toBe(false)
    expect(wrapper.text()).toContain('138****0000')
    expect(wrapper.text()).toContain('10001')
    expect(wrapper.text()).toContain('user.verification.identity.verified')
    expect(wrapper.text()).toContain('user.verification.student.verified')

    wrapper.unmount()
    warn.mockRestore()
  })
})
