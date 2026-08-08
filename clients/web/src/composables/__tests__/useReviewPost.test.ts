import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { REVIEW_CREATE } from '@stuhelper/shared/constants'

const mockPush = vi.fn()
const mockGetUserSurface = vi.fn()
const mockGetPhoneStatus = vi.fn()
const mockToastError = vi.fn()
const mockGetErrorMessage = vi.fn()
const mockNavigateToExternalURL = vi.fn()
const mockAuthStore = {
  bootstrapCompleted: true,
  bootstrapSession: vi.fn(),
  isAuthenticated: true,
}

vi.mock('vue-router', () => ({
  useRouter: () => ({
    push: mockPush,
    currentRoute: { value: { fullPath: '/courses/reviews' } },
  }),
}))
vi.mock('@/api', () => ({
  api: {
    identity: { getUserSurface: mockGetUserSurface },
    studentVerification: { getPhoneStatus: mockGetPhoneStatus },
  },
}))
vi.mock('@/api/errors', () => ({ getErrorMessage: mockGetErrorMessage }))
vi.mock('@/i18n', () => ({ default: { global: { t: (key: string) => key } } }))
vi.mock('@/stores/auth', () => ({ useAuthStore: () => mockAuthStore }))
vi.mock('@/composables/useToast', () => ({ useToast: () => ({ error: mockToastError }) }))
vi.mock('@/utils/redirect', () => ({
  accountCenterURLWithRedirect: (path: string, redirect: string) =>
    `https://stuhelper.com${path}?redirect=${encodeURIComponent(redirect)}`,
  navigateToExternalURL: mockNavigateToExternalURL,
}))

const { resolveReviewPostBlock, useReviewPost } = await import('../useReviewPost')

const surface = {
  displayName: 'Alice',
  studentVerificationStatus: 'approved',
  phoneBound: true,
  capabilities: [REVIEW_CREATE],
}
const phone = {
  state: 'verified',
  publishingRequirementSatisfied: true,
  revision: 3,
}
const ok = (data: unknown) => ({ data: { data } })

describe('useReviewPost target gates', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
    mockAuthStore.bootstrapCompleted = true
    mockAuthStore.bootstrapSession.mockResolvedValue(true)
    mockAuthStore.isAuthenticated = true
    mockGetUserSurface.mockResolvedValue(ok(surface))
    mockGetPhoneStatus.mockResolvedValue(ok(phone))
  })

  it('redirects anonymous users before reading verification state', async () => {
    mockAuthStore.isAuthenticated = false
    const { ensureCanPostReview } = useReviewPost()

    await expect(ensureCanPostReview()).resolves.toBe(false)
    expect(mockGetUserSurface).not.toHaveBeenCalled()
    expect(mockGetPhoneStatus).not.toHaveBeenCalled()
    expect(mockPush).toHaveBeenCalledWith({
      name: 'login',
      query: { redirect: '/courses/reviews' },
    })
  })

  it('allows posting only when student, phone and capability gates pass', async () => {
    const { ensureCanPostReview } = useReviewPost()
    await expect(ensureCanPostReview()).resolves.toBe(true)
  })

  it('routes a missing student credential to the independent verification center', async () => {
    mockGetUserSurface.mockResolvedValue(ok({ ...surface, studentVerificationStatus: 'none' }))
    const { ensureCanPostReview } = useReviewPost()

    await expect(ensureCanPostReview()).resolves.toBe(false)
    expect(mockNavigateToExternalURL).toHaveBeenCalledWith(
      'https://stuhelper.com/user/student-verification?redirect=%2Fcourses%2Freviews',
    )
  })

  it('routes a missing publishing phone gate to Casdoor-backed phone maintenance', async () => {
    mockGetPhoneStatus.mockResolvedValue(ok({
      state: 'unbound',
      publishingRequirementSatisfied: false,
      revision: 4,
    }))
    const { ensureCanPostReview } = useReviewPost()

    await expect(
      ensureCanPostReview({ redirect: '/courses/reviews/post' }),
    ).resolves.toBe(false)
    expect(mockNavigateToExternalURL).toHaveBeenCalledWith(
      'https://stuhelper.com/user/phone-binding?redirect=%2Fcourses%2Freviews%2Fpost',
    )
  })

  it('does not infer phone control from phoneBound alone', () => {
    expect(resolveReviewPostBlock(surface as never, {
      state: 'syncing',
      publishingRequirementSatisfied: false,
      revision: 5,
    } as never)?.routeName).toBe('phone-binding')
  })

  it('routes a missing capability to home after identity gates pass', async () => {
    mockGetUserSurface.mockResolvedValue(ok({ ...surface, capabilities: [] }))
    const { ensureCanPostReview } = useReviewPost()

    await expect(ensureCanPostReview()).resolves.toBe(false)
    expect(mockPush).toHaveBeenCalledWith({ name: 'home' })
  })

  it('fails closed on a malformed projection', async () => {
    mockGetPhoneStatus.mockResolvedValue(ok({ ...phone, publishingRequirementSatisfied: 'yes' }))
    mockGetErrorMessage.mockReturnValue('common.loadFailed')
    const { ensureCanPostReview } = useReviewPost()

    await expect(ensureCanPostReview()).resolves.toBe(false)
    expect(mockToastError).toHaveBeenCalledWith('common.loadFailed')
  })
})
