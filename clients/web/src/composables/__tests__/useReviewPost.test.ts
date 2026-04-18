import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'

const mockPush = vi.fn()
const mockGetUserSurface = vi.fn()
const mockToastError = vi.fn()
const mockGetErrorMessage = vi.fn()
const mockAuthStore = {
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

const { useReviewPost } = await import('../useReviewPost')

describe('useReviewPost', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    mockPush.mockReset()
    mockGetUserSurface.mockReset()
    mockToastError.mockReset()
    mockGetErrorMessage.mockReset()
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
