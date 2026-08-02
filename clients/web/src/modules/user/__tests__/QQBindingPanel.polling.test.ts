// @vitest-environment jsdom

import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { ApiError } from '@/api/errors'

const mocks = vi.hoisted(() => ({
  createQQBindingCode: vi.fn(),
  fetchQQBinding: vi.fn(),
  fetchStatus: vi.fn(),
  push: vi.fn(),
  toastError: vi.fn(),
  toastInfo: vi.fn(),
  toastSuccess: vi.fn(),
  store: {
    qqBinding: null as null | { qqID: string; boundAt: string },
    qqBindingCode: null as null | { code: string; expiresAt: string },
    studentVerified: false,
  },
}))

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({ t: (key: string) => key }),
  }
})

vi.mock('vue-router', () => ({
  useRouter: () => ({ push: mocks.push }),
}))

vi.mock('@/composables/useToast', () => ({
  useToast: () => ({
    error: mocks.toastError,
    info: mocks.toastInfo,
    success: mocks.toastSuccess,
  }),
}))

vi.mock('@/stores/verification', () => ({
  useVerificationStore: () => ({
    ...mocks.store,
    createQQBindingCode: mocks.createQQBindingCode,
    fetchQQBinding: mocks.fetchQQBinding,
    fetchStatus: mocks.fetchStatus,
    get qqBinding() {
      return mocks.store.qqBinding
    },
    get qqBindingCode() {
      return mocks.store.qqBindingCode
    },
  }),
}))

const { default: QQBindingPanel } = await import('../views/QQBindingPanel.vue')

describe('QQBindingPanel polling lifecycle', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-08-02T00:00:00Z'))
    mocks.store.qqBinding = null
    mocks.store.qqBindingCode = null
    mocks.fetchStatus.mockResolvedValue(undefined)
    mocks.fetchQQBinding.mockResolvedValue(null)
    setVisibility('visible')
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('stops automatically at the code deadline and disables stale copying', async () => {
    mocks.store.qqBindingCode = {
      code: 'ABC123',
      expiresAt: '2026-08-02T00:00:06Z',
    }
    const wrapper = mount(QQBindingPanel, {
      props: { loadOnMount: false, standalone: false },
    })

    await flushPromises()
    await vi.advanceTimersByTimeAsync(3_000)
    expect(mocks.fetchQQBinding).toHaveBeenCalledTimes(1)

    await vi.advanceTimersByTimeAsync(3_000)
    expect(wrapper.find('[data-qq-binding-code-expired]').exists()).toBe(true)
    expect(wrapper.get<HTMLButtonElement>('[data-qq-binding-copy]').element.disabled).toBe(true)

    await vi.advanceTimersByTimeAsync(60_000)
    expect(mocks.fetchQQBinding).toHaveBeenCalledTimes(1)
    wrapper.unmount()
  })

  it('pauses network polling while hidden and resumes against the original deadline', async () => {
    mocks.store.qqBindingCode = {
      code: 'ABC123',
      expiresAt: '2026-08-02T00:00:12Z',
    }
    const wrapper = mount(QQBindingPanel, {
      props: { loadOnMount: false, standalone: false },
    })
    await flushPromises()

    setVisibility('hidden')
    document.dispatchEvent(new Event('visibilitychange'))
    await vi.advanceTimersByTimeAsync(6_000)
    expect(mocks.fetchQQBinding).not.toHaveBeenCalled()

    setVisibility('visible')
    document.dispatchEvent(new Event('visibilitychange'))
    await vi.advanceTimersByTimeAsync(3_000)
    expect(mocks.fetchQQBinding).toHaveBeenCalledTimes(1)

    await vi.advanceTimersByTimeAsync(3_000)
    expect(wrapper.find('[data-qq-binding-code-expired]').exists()).toBe(true)
    wrapper.unmount()
  })

  it('stops after a successful binding or an authentication terminal error', async () => {
    mocks.store.qqBindingCode = {
      code: 'ABC123',
      expiresAt: '2026-08-02T00:10:00Z',
    }
    mocks.fetchQQBinding.mockImplementationOnce(async () => {
      mocks.store.qqBinding = { qqID: '123456', boundAt: '2026-08-02T00:00:01Z' }
      return mocks.store.qqBinding
    })
    const boundWrapper = mount(QQBindingPanel, {
      props: { loadOnMount: false, standalone: false },
    })
    await flushPromises()
    await vi.advanceTimersByTimeAsync(3_000)
    await flushPromises()
    expect(boundWrapper.emitted('bound')).toHaveLength(1)
    await vi.advanceTimersByTimeAsync(30_000)
    expect(mocks.fetchQQBinding).toHaveBeenCalledTimes(1)
    boundWrapper.unmount()

    mocks.store.qqBinding = null
    mocks.store.qqBindingCode = {
      code: 'DEF456',
      expiresAt: '2026-08-02T00:10:00Z',
    }
    mocks.fetchQQBinding.mockReset()
    mocks.fetchQQBinding.mockRejectedValueOnce(
      new ApiError({ code: 'A0010001', message: 'unauthorized', status: 401 }),
    )
    const authWrapper = mount(QQBindingPanel, {
      props: { loadOnMount: false, standalone: false },
    })
    await flushPromises()
    await vi.advanceTimersByTimeAsync(3_000)
    await flushPromises()
    await vi.advanceTimersByTimeAsync(30_000)
    expect(mocks.fetchQQBinding).toHaveBeenCalledTimes(1)
    authWrapper.unmount()
  })
})

function setVisibility(value: DocumentVisibilityState): void {
  Object.defineProperty(document, 'visibilityState', {
    configurable: true,
    value,
  })
}
