import { beforeEach, describe, expect, it, vi } from 'vitest'

const mockToastError = vi.fn()
const mockGetErrorMessage = vi.fn()

vi.mock('@/composables/useToast', () => ({
  useToast: () => ({
    error: mockToastError,
  }),
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

const { useOptimisticUpdate } = await import('../useOptimisticUpdate')

describe('useOptimisticUpdate', () => {
  beforeEach(() => {
    mockToastError.mockReset()
    mockGetErrorMessage.mockReset()
  })

  it('passes request failures through getErrorMessage with provided fallback', async () => {
    const failure = new Error('stack hint')
    const optimistic = vi.fn()
    const rollback = vi.fn()
    mockGetErrorMessage.mockReturnValue('safe message')

    const { execute } = useOptimisticUpdate()
    const result = await execute({
      optimistic,
      rollback,
      errorMessage: 'fallback message',
      request: async () => {
        throw failure
      },
    })

    expect(optimistic).toHaveBeenCalledTimes(1)
    expect(rollback).toHaveBeenCalledTimes(1)
    expect(mockGetErrorMessage).toHaveBeenCalledWith(
      failure,
      'fallback message',
    )
    expect(mockToastError).toHaveBeenCalledWith('safe message')
    expect(result).toEqual({ success: false, error: failure })
  })

  it('uses the default operation fallback when no custom message is provided', async () => {
    const failure = new Error('stack hint')
    mockGetErrorMessage.mockReturnValue('common.actions.operationFailed')

    const { execute } = useOptimisticUpdate()
    await execute({
      optimistic: vi.fn(),
      rollback: vi.fn(),
      request: async () => {
        throw failure
      },
    })

    expect(mockGetErrorMessage).toHaveBeenCalledWith(
      failure,
      'common.actions.operationFailed',
    )
  })
})
