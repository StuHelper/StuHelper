// @vitest-environment jsdom

import { mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import AppUserMenu from '../AppUserMenu.vue'

const mocks = vi.hoisted(() => ({
  authStore: {
    bootstrapCompleted: false,
    isAuthenticated: false,
    user: {
      id: 'user_1',
      name: 'alice',
      displayName: 'Alice',
      isPlatformAdmin: false,
      capabilities: [],
      globalCapabilities: [],
      capabilityGrants: [],
      canAccessAdmin: false,
    },
    logout: vi.fn(),
  },
  verificationStore: {
    identityVerified: false,
    studentVerified: false,
    qqBound: false,
    fetchStatus: vi.fn(),
  },
  routerPush: vi.fn(),
  toastError: vi.fn(),
  identityPortalURL: vi.fn((path: string) => `https://stuhelper.com${path}`),
  navigateToExternalURL: vi.fn(),
}))

vi.mock('vue-router', () => ({
  useRouter: () => ({ push: mocks.routerPush }),
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => mocks.authStore,
}))

vi.mock('@/stores/verification', () => ({
  useVerificationStore: () => mocks.verificationStore,
}))

vi.mock('@/composables/useToast', () => ({
  useToast: () => ({
    error: mocks.toastError,
  }),
}))

vi.mock('@/utils/adminAccess', () => ({
  canShowAdminEntry: () => false,
}))

vi.mock('@/utils/adminUrl', () => ({
  resolveAdminConsoleURL: () => 'http://localhost:5174',
}))

vi.mock('@/utils/redirect', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/utils/redirect')>()
  return {
    ...actual,
    identityPortalURL: mocks.identityPortalURL,
    navigateToExternalURL: mocks.navigateToExternalURL,
  }
})

function mountUserMenu(authState: {
  bootstrapCompleted: boolean
  isAuthenticated: boolean
}) {
  Object.assign(mocks.authStore, authState)
  return mount(AppUserMenu)
}

describe('AppUserMenu', () => {
  beforeEach(() => {
    vi.unstubAllEnvs()
    Object.assign(mocks.authStore, {
      bootstrapCompleted: false,
      isAuthenticated: false,
    })
    mocks.authStore.logout.mockResolvedValue({ ok: true })
    mocks.verificationStore.fetchStatus.mockReset()
    Object.assign(mocks.verificationStore, {
      identityVerified: false,
      studentVerified: false,
      qqBound: false,
    })
    mocks.verificationStore.fetchStatus.mockResolvedValue(undefined)
    mocks.routerPush.mockReset()
    mocks.toastError.mockReset()
    mocks.identityPortalURL.mockClear()
    mocks.navigateToExternalURL.mockClear()
  })

  afterEach(() => {
    vi.unstubAllEnvs()
  })

  it('does not fetch verification status while cached auth is unresolved', () => {
    mountUserMenu({
      bootstrapCompleted: false,
      isAuthenticated: true,
    })

    expect(mocks.verificationStore.fetchStatus).not.toHaveBeenCalled()
  })

  it('fetches verification status after session bootstrap resolves authenticated', () => {
    mountUserMenu({
      bootstrapCompleted: true,
      isAuthenticated: true,
    })

    expect(mocks.verificationStore.fetchStatus).toHaveBeenCalledTimes(1)
  })

  it('does not fetch verification status for resolved anonymous state', () => {
    mountUserMenu({
      bootstrapCompleted: true,
      isAuthenticated: false,
    })

    expect(mocks.verificationStore.fetchStatus).not.toHaveBeenCalled()
  })

  it('keeps the profile menu item on the main user center outside the identity host', async () => {
    const wrapper = mountUserMenu({
      bootstrapCompleted: true,
      isAuthenticated: true,
    })

    await wrapper.get('[aria-haspopup="menu"]').trigger('click')
    await wrapper
      .findAll('[role="menuitem"]')
      .find((item) => item.text().includes('nav.profile'))
      ?.trigger('click')

    expect(mocks.routerPush).toHaveBeenCalledWith('/user/reviews')
  })

  it('keeps the profile menu item on the main user center even if legacy identity env is present', async () => {
    vi.stubEnv('VITE_IDENTITY_URL', window.location.origin)
    const wrapper = mountUserMenu({
      bootstrapCompleted: true,
      isAuthenticated: true,
    })

    await wrapper.get('[aria-haspopup="menu"]').trigger('click')
    await wrapper
      .findAll('[role="menuitem"]')
      .find((item) => item.text().includes('nav.profile'))
      ?.trigger('click')

    expect(mocks.routerPush).toHaveBeenCalledWith('/user/reviews')
  })

  it('opens account security on the account center from the main host', async () => {
    const wrapper = mountUserMenu({
      bootstrapCompleted: true,
      isAuthenticated: true,
    })

    await wrapper.get('[aria-haspopup="menu"]').trigger('click')
    await wrapper
      .findAll('[role="menuitem"]')
      .find((item) => item.text().includes('nav.accountSecurity'))
      ?.trigger('click')

    expect(mocks.identityPortalURL).toHaveBeenCalledWith('/account/security')
    expect(mocks.navigateToExternalURL).toHaveBeenCalledWith(
      'https://stuhelper.com/account/security',
    )
    expect(mocks.routerPush).not.toHaveBeenCalled()
  })

  it('opens account profile on the account center from the main host', async () => {
    const wrapper = mountUserMenu({
      bootstrapCompleted: true,
      isAuthenticated: true,
    })

    await wrapper.get('[aria-haspopup="menu"]').trigger('click')
    await wrapper
      .findAll('[role="menuitem"]')
      .find((item) => item.text().includes('nav.accountProfile'))
      ?.trigger('click')

    expect(mocks.identityPortalURL).toHaveBeenCalledWith('/account/profile')
    expect(mocks.navigateToExternalURL).toHaveBeenCalledWith(
      'https://stuhelper.com/account/profile',
    )
    expect(mocks.routerPush).not.toHaveBeenCalled()
  })

  it('opens account menu entries on the account center even if legacy identity env is present', async () => {
    vi.stubEnv('VITE_IDENTITY_URL', window.location.origin)
    const wrapper = mountUserMenu({
      bootstrapCompleted: true,
      isAuthenticated: true,
    })

    await wrapper.get('[aria-haspopup="menu"]').trigger('click')
    await wrapper
      .findAll('[role="menuitem"]')
      .find((item) => item.text().includes('nav.accountProfile'))
      ?.trigger('click')

    expect(mocks.routerPush).not.toHaveBeenCalled()
    expect(mocks.navigateToExternalURL).toHaveBeenCalledWith(
      'https://stuhelper.com/account/profile',
    )
  })

  it('opens developer apps on the account center from the main host', async () => {
    const wrapper = mountUserMenu({
      bootstrapCompleted: true,
      isAuthenticated: true,
    })

    await wrapper.get('[aria-haspopup="menu"]').trigger('click')
    await wrapper
      .findAll('[role="menuitem"]')
      .find((item) => item.text().includes('nav.developerApps'))
      ?.trigger('click')

    expect(mocks.identityPortalURL).toHaveBeenCalledWith('/developers/apps')
    expect(mocks.navigateToExternalURL).toHaveBeenCalledWith(
      'https://stuhelper.com/developers/apps',
    )
  })

  it('returns to the main site after logout even if legacy identity env is present', async () => {
    vi.stubEnv('VITE_IDENTITY_URL', window.location.origin)
    const wrapper = mountUserMenu({
      bootstrapCompleted: true,
      isAuthenticated: true,
    })

    await wrapper.get('[aria-haspopup="menu"]').trigger('click')
    await wrapper
      .findAll('[role="menuitem"]')
      .find((item) => item.text().includes('nav.logout'))
      ?.trigger('click')

    expect(mocks.routerPush).toHaveBeenCalledWith('/')
  })
})
