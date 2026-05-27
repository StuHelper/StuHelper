// @vitest-environment jsdom

import { mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import AppHeader from '../AppHeader.vue'

const mocks = vi.hoisted(() => ({
  authStore: {
    bootstrapCompleted: false,
    isAuthenticated: false,
  },
  route: {
    name: 'home' as string | undefined,
    path: '/',
    fullPath: '/',
    params: {} as Record<string, string>,
  },
  routerPush: vi.fn(),
  ensureCanPostReview: vi.fn(),
  rememberReviewPostCourse: vi.fn(),
}))

vi.mock('vue-router', () => ({
  useRoute: () => mocks.route,
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

vi.mock('@/composables/useReviewPost', () => ({
  useReviewPost: () => ({
    ensureCanPostReview: mocks.ensureCanPostReview,
  }),
}))

vi.mock('@/modules/review/reviewPostNavigation', () => ({
  rememberReviewPostCourse: mocks.rememberReviewPostCourse,
}))

vi.mock('@/components/common/InlineSearch.vue', () => ({
  default: {
    name: 'InlineSearch',
    template: '<div data-test="inline-search" />',
  },
}))

vi.mock('@/components/common/NotificationBell.vue', () => ({
  default: {
    name: 'NotificationBell',
    template: '<div data-test="notification-bell" />',
  },
}))

vi.mock('@/components/common/LocaleSwitcher.vue', () => ({
  default: {
    name: 'LocaleSwitcher',
    template: '<div data-test="locale-switcher" />',
  },
}))

vi.mock('../AppUserMenu.vue', () => ({
  default: {
    name: 'AppUserMenu',
    template: '<div data-test="app-user-menu" />',
  },
}))

function mountHeader(authState: {
  bootstrapCompleted: boolean
  isAuthenticated: boolean
}) {
  Object.assign(mocks.authStore, authState)
  return mount(AppHeader, {
    global: {
      directives: {
        ripple: {},
      },
      stubs: {
        RouterLink: {
          props: ['to'],
          template: '<a data-test="router-link" :data-to="to"><slot /></a>',
        },
      },
    },
  })
}

describe('AppHeader', () => {
  beforeEach(() => {
    vi.unstubAllEnvs()
    Object.assign(mocks.authStore, {
      bootstrapCompleted: false,
      isAuthenticated: false,
    })
    Object.assign(mocks.route, {
      name: 'home',
      path: '/',
      fullPath: '/',
      params: {},
    })
    mocks.routerPush.mockReset()
    mocks.ensureCanPostReview.mockResolvedValue(true)
    mocks.rememberReviewPostCourse.mockReset()
  })

  afterEach(() => {
    vi.unstubAllEnvs()
  })

  it('does not mount authenticated header actions while cached auth is unresolved', () => {
    const wrapper = mountHeader({
      bootstrapCompleted: false,
      isAuthenticated: true,
    })

    expect(wrapper.find('[data-test="notification-bell"]').exists()).toBe(false)
    expect(wrapper.find('[data-test="app-user-menu"]').exists()).toBe(false)
    expect(wrapper.text()).not.toContain('nav.login')
  })

  it('mounts authenticated header actions after session bootstrap resolves authenticated', () => {
    const wrapper = mountHeader({
      bootstrapCompleted: true,
      isAuthenticated: true,
    })

    expect(wrapper.find('[data-test="notification-bell"]').exists()).toBe(true)
    expect(wrapper.find('[data-test="app-user-menu"]').exists()).toBe(true)
    expect(wrapper.text()).not.toContain('nav.login')
  })

  it('shows the login entry after session bootstrap resolves anonymous', () => {
    const wrapper = mountHeader({
      bootstrapCompleted: true,
      isAuthenticated: false,
    })

    expect(wrapper.find('[data-test="notification-bell"]').exists()).toBe(false)
    expect(wrapper.find('[data-test="app-user-menu"]').exists()).toBe(false)
    expect(wrapper.text()).toContain('nav.login')
  })

  it('uses identity portal navigation on the configured identity host', () => {
    vi.stubEnv('VITE_IDENTITY_URL', window.location.origin)
    Object.assign(mocks.route, {
      name: 'login',
      path: '/login',
      fullPath: '/login?redirect=/developers/apps',
      params: {},
    })

    const wrapper = mountHeader({
      bootstrapCompleted: true,
      isAuthenticated: false,
    })

    expect(wrapper.text()).toContain('routes.openPlatformDeveloperApps')
    expect(wrapper.text()).toContain('routes.userAuthorizedApps')
    expect(wrapper.text()).toContain('routes.identityVerification')
    expect(wrapper.text()).not.toContain('nav.courses')
    expect(wrapper.text()).not.toContain('nav.teacher')
    expect(wrapper.text()).not.toContain('nav.login')
  })

  it('uses courses as the only course-review top navigation entry', () => {
    const wrapper = mountHeader({
      bootstrapCompleted: true,
      isAuthenticated: false,
    })

    expect(wrapper.text()).toContain('nav.courses')
    expect(wrapper.text()).not.toContain('nav.review')
  })

  it('shows the header course search only on the course hub', () => {
    Object.assign(mocks.route, {
      path: '/courses',
      fullPath: '/courses',
      params: {},
    })
    const courseHubWrapper = mountHeader({
      bootstrapCompleted: true,
      isAuthenticated: false,
    })
    expect(courseHubWrapper.find('[data-test="inline-search"]').exists()).toBe(true)

    Object.assign(mocks.route, {
      path: '/courses/list',
      fullPath: '/courses/list',
      params: {},
    })
    const courseListWrapper = mountHeader({
      bootstrapCompleted: true,
      isAuthenticated: false,
    })
    expect(courseListWrapper.find('[data-test="inline-search"]').exists()).toBe(false)
  })
})
