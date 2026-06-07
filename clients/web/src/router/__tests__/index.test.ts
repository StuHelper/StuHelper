import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

const mocks = vi.hoisted(() => {
  let beforeEachGuard: ((to: Record<string, unknown>) => unknown) | undefined

  return {
    authStore: {
      bootstrapCompleted: false,
      bootstrapSession: vi.fn(),
      clearSession: vi.fn(),
      isAuthenticated: false,
      refreshSession: vi.fn(),
      user: null as { capabilities?: string[] } | null,
    },
    getBeforeEachGuard: () => beforeEachGuard,
    hasStoredSessionHint: vi.fn(),
    isTokenExpired: vi.fn(),
    setBeforeEachGuard: (guard: (to: Record<string, unknown>) => unknown) => {
      beforeEachGuard = guard
    },
    updatePageMeta: vi.fn(),
  }
})

vi.mock('vue-router', () => ({
  createRouter: vi.fn(() => ({
    afterEach: vi.fn(),
    beforeEach: vi.fn((guard) => {
      mocks.setBeforeEachGuard(guard)
    }),
    onError: vi.fn(),
    replace: vi.fn(),
  })),
  createWebHistory: vi.fn(() => ({})),
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => mocks.authStore,
}))

vi.mock('@/utils/auth', () => ({
  isTokenExpired: mocks.isTokenExpired,
}))

vi.mock('@/utils/sessionHint', () => ({
  hasStoredSessionHint: mocks.hasStoredSessionHint,
}))

vi.mock('@/composables/usePageMeta', () => ({
  updatePageMeta: mocks.updatePageMeta,
}))

vi.mock('@/i18n', () => ({
  default: {
    global: {
      t: (key: string) => key,
    },
  },
}))

vi.mock('@/modules/errors/views/ChunkErrorPage.vue', () => ({
  default: { name: 'ChunkErrorPage' },
}))

vi.mock('@/modules/errors/views/NotFoundPage.vue', () => ({
  default: { name: 'NotFoundPage' },
}))

await import('../index')

function protectedRoute() {
  const meta = { requiresAuth: true, titleKey: 'routes.userReviews' }

  return {
    fullPath: '/user/reviews',
    matched: [{ meta }],
    meta,
    name: 'user-reviews',
    path: '/user/reviews',
    query: {},
  }
}

function publicRoute(path: string, name = 'home') {
  const meta = { titleKey: 'routes.home' }

  return {
    fullPath: path,
    hash: '',
    matched: [{ meta }],
    meta,
    name,
    path,
    query: {},
  }
}

describe('router auth guard', () => {
  beforeEach(() => {
    mocks.authStore.bootstrapCompleted = false
    mocks.authStore.bootstrapSession.mockReset()
    mocks.authStore.bootstrapSession.mockResolvedValue(false)
    mocks.authStore.clearSession.mockReset()
    mocks.authStore.isAuthenticated = false
    mocks.authStore.refreshSession.mockReset()
    mocks.authStore.user = null
    mocks.hasStoredSessionHint.mockReset()
    mocks.hasStoredSessionHint.mockReturnValue(true)
    mocks.isTokenExpired.mockReset()
    mocks.isTokenExpired.mockReturnValue(false)
    mocks.updatePageMeta.mockReset()
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('cancels protected navigation when session bootstrap remains unresolved', async () => {
    const guard = mocks.getBeforeEachGuard()

    expect(guard).toBeDefined()
    await expect(guard?.(protectedRoute())).resolves.toBe(false)
    expect(mocks.authStore.bootstrapSession).toHaveBeenCalledTimes(1)
  })

  it('renders not found for main-site routes on the join admission host', async () => {
    vi.stubGlobal('window', { location: { hostname: 'join.localhost' } })
    const guard = mocks.getBeforeEachGuard()

    await expect(guard?.(publicRoute('/courses', 'course-hub'))).resolves.toEqual({
      hash: '',
      name: 'not-found',
      params: { pathMatch: ['courses'] },
      query: {},
      replace: true,
    })
  })

  it('allows admission verification routes on the join admission host', async () => {
    vi.stubGlobal('window', { location: { hostname: 'join.localhost' } })
    const guard = mocks.getBeforeEachGuard()

    await expect(guard?.(publicRoute('/verify/LOCALJOINSMOKE', 'admission-token'))).resolves.toBe(true)
  })

  it('does not refresh expired local auth state on public routes', async () => {
    const guard = mocks.getBeforeEachGuard()
    mocks.authStore.isAuthenticated = true
    mocks.hasStoredSessionHint.mockReturnValue(true)
    mocks.isTokenExpired.mockReturnValue(true)

    await expect(guard?.(publicRoute('/about', 'about'))).resolves.toBe(true)
    expect(mocks.authStore.refreshSession).not.toHaveBeenCalled()
  })
})
