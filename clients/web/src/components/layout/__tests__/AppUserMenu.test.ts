// @vitest-environment jsdom

import { mount } from '@vue/test-utils'
import { nextTick, reactive } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'
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

function mountUserMenu(authState: {
  bootstrapCompleted: boolean
  isAuthenticated: boolean
}) {
  Object.assign(mocks.authStore, authState)
  return mount(AppUserMenu)
}

describe('AppUserMenu', () => {
  beforeEach(() => {
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

  it('fetches verification status when auth resolves after mount', async () => {
    const state = reactive({
      ...mocks.authStore,
      bootstrapCompleted: false,
      isAuthenticated: true,
    }) as typeof mocks.authStore
    mocks.authStore = state

    mount(AppUserMenu)
    expect(mocks.verificationStore.fetchStatus).not.toHaveBeenCalled()

    state.bootstrapCompleted = true
    await nextTick()

    expect(mocks.verificationStore.fetchStatus).toHaveBeenCalledTimes(1)
  })
})
