import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { effectScope } from 'vue'
import { useToast } from '@/composables/useToast'

describe('useToast', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    useToast().clearAll()
  })

  afterEach(() => {
    useToast().clearAll()
    vi.useRealTimers()
  })

  it('auto-dismisses a global toast after the creating scope is disposed', () => {
    const creatorScope = effectScope()

    creatorScope.run(() => {
      useToast().success('posted!', 3000)
    })

    const observer = useToast()
    expect(observer.toasts.value.map(toast => toast.message)).toEqual(['posted!'])

    creatorScope.stop()
    vi.advanceTimersByTime(2999)
    expect(observer.toasts.value.map(toast => toast.message)).toEqual(['posted!'])

    vi.advanceTimersByTime(1)
    expect(observer.toasts.value).toEqual([])
  })

  it('keeps explicit removal and persistent-toast cleanup deterministic', () => {
    const toast = useToast()
    const timedID = toast.info('timed', 3000)
    toast.warning('persistent', 0)

    toast.remove(timedID)
    vi.advanceTimersByTime(3000)
    expect(toast.toasts.value.map(item => item.message)).toEqual(['persistent'])

    toast.clearAll()
    expect(toast.toasts.value).toEqual([])
  })
})
