import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { REVIEW_CREATE } from '@stuhelper/shared/constants'

const mockPush = vi.fn()
const mockGetUserSurface = vi.fn()
const mockToastError = vi.fn()
const mockGetErrorMessage = vi.fn()
const mockAuthStore = {
  bootstrapCompleted: true,
  bootstrapSession: vi.fn(),
  isAuthenticated: true,
}

vi.mock('vue-router', () => ({
  useRouter: () => ({
    push: mockPush,
    currentRoute: {
      value: {
        fullPath: '/review',
      },
    },
  }),
}))

vi.mock('@/api', () => ({
  api: {
    identity: {
      getUserSurface: mockGetUserSurface,
    },
  },
}))

vi.mock('@/api/errors', () => ({
  getErrorMessage: mockGetErrorMessage,
}))

vi.mock('@/i18n', () => ({
  default: {
    global: {
      t: (key: string) => key,
    },
  },
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => mockAuthStore,
}))

vi.mock('@/composables/useToast', () => ({
  useToast: () => ({
    error: mockToastError,
  }),
}))

const { resolveReviewPostBlock, useReviewPost } = await import('../useReviewPost')

describe('useReviewPost', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    mockPush.mockReset()
    mockGetUserSurface.mockReset()
    mockToastError.mockReset()
    mockGetErrorMessage.mockReset()
    mockAuthStore.bootstrapCompleted = true
    mockAuthStore.bootstrapSession.mockReset()
    mockAuthStore.bootstrapSession.mockResolvedValue(true)
    mockAuthStore.isAuthenticated = true
  })

  it('redirects anonymous users to login before checking review access', async () => {
    mockAuthStore.isAuthenticated = false

    const { ensureCanPostReview } = useReviewPost()

    await expect(ensureCanPostReview()).resolves.toBe(false)
    expect(mockGetUserSurface).not.toHaveBeenCalled()
    expect(mockPush).toHaveBeenCalledWith({
      name: 'login',
      query: { redirect: '/review' },
    })
  })

  it('bootstraps the session before checking review access', async () => {
    mockAuthStore.bootstrapCompleted = false
    mockGetUserSurface.mockResolvedValue({
      data: {
        data: {
          identityStatus: 'approved',
          verificationStatus: 'approved',
          capabilities: [REVIEW_CREATE],
        },
      },
    })

    const { ensureCanPostReview } = useReviewPost()

    await expect(ensureCanPostReview()).resolves.toBe(true)
    expect(mockAuthStore.bootstrapSession).toHaveBeenCalledTimes(1)
  })

  it('redirects users without identity verification to the identity page', async () => {
    mockGetUserSurface.mockResolvedValue({
      data: {
        data: {
          identityStatus: 'pending',
          verificationStatus: 'approved',
          capabilities: [REVIEW_CREATE],
        },
      },
    })

    const { ensureCanPostReview } = useReviewPost()

    await expect(ensureCanPostReview()).resolves.toBe(false)
    expect(mockToastError).toHaveBeenCalledWith('user.verification.student.identityRequired')
    expect(mockPush).toHaveBeenCalledWith({ name: 'identity-verification' })
  })

  it('redirects users without student verification to the student page', async () => {
    mockGetUserSurface.mockResolvedValue({
      data: {
        data: {
          identityStatus: 'approved',
          verificationStatus: 'pending',
          capabilities: [REVIEW_CREATE],
        },
      },
    })

    const { ensureCanPostReview } = useReviewPost()

    await expect(ensureCanPostReview()).resolves.toBe(false)
    expect(mockToastError).toHaveBeenCalledWith('review.card.verifyToView')
    expect(mockPush).toHaveBeenCalledWith({ name: 'student-verification' })
  })

  it('redirects users without review:create capability away from the post flow', async () => {
    mockGetUserSurface.mockResolvedValue({
      data: {
        data: {
          identityStatus: 'approved',
          verificationStatus: 'approved',
          capabilities: [],
        },
      },
    })

    const { ensureCanPostReview } = useReviewPost()

    await expect(ensureCanPostReview()).resolves.toBe(false)
    expect(mockToastError).toHaveBeenCalledWith('errors.A0010200')
    expect(mockPush).toHaveBeenCalledWith({ name: 'home' })
  })

  it('uses getErrorMessage when user surface lookup fails', async () => {
    const failure = new Error('internal stack hint')
    mockGetUserSurface.mockRejectedValue(failure)
    mockGetErrorMessage.mockReturnValue('common.loadFailed')

    const { ensureCanPostReview } = useReviewPost()

    await expect(ensureCanPostReview()).resolves.toBe(false)
    expect(mockGetErrorMessage).toHaveBeenCalledWith(
      failure,
      'common.loadFailed',
    )
    expect(mockToastError).toHaveBeenCalledWith('common.loadFailed')
  })
})

describe('resolveReviewPostBlock', () => {
  it('returns null only when identity, student verification and capability all pass', () => {
    expect(resolveReviewPostBlock({
      identityStatus: 'approved',
      verificationStatus: 'approved',
      capabilities: [REVIEW_CREATE],
    } as never)).toBeNull()
  })
})
