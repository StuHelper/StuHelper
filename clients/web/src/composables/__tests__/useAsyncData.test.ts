import { describe, expect, it, vi } from 'vitest'
import { effectScope } from 'vue'

import { useAsyncData } from '../useAsyncData'

async function flushPromises() {
  await Promise.resolve()
  await Promise.resolve()
}

describe('useAsyncData', () => {
  it('executes immediately by default', async () => {
    const fetcher = vi.fn().mockResolvedValue('ready')

    const scope = effectScope()
    const state = scope.run(() => useAsyncData(fetcher))
    if (!state) {
      throw new Error('failed to create state')
    }

    await flushPromises()

    expect(fetcher).toHaveBeenCalledTimes(1)
    expect(state.data.value).toBe('ready')
    expect(state.loading.value).toBe(false)
    scope.stop()
  })

  it('supports lazy mode with immediate: false', async () => {
    const fetcher = vi.fn().mockResolvedValue('lazy')

    const scope = effectScope()
    const state = scope.run(() =>
      useAsyncData(fetcher, { immediate: false }),
    )
    if (!state) {
      throw new Error('failed to create state')
    }

    expect(fetcher).not.toHaveBeenCalled()
    expect(state.data.value).toBeNull()

    await state.execute()

    expect(fetcher).toHaveBeenCalledTimes(1)
    expect(state.data.value).toBe('lazy')
    scope.stop()
  })
})
