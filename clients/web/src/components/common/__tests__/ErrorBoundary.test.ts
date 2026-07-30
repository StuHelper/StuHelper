// @vitest-environment jsdom

import { flushPromises, mount } from '@vue/test-utils'
import { defineComponent, h } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const mocks = vi.hoisted(() => ({
  reportFrontendError: vi.fn(),
  route: {
    fullPath: '/courses/20',
  },
}))

vi.mock('@/utils/observability', () => ({
  reportFrontendError: mocks.reportFrontendError,
}))

vi.mock('vue-router', () => ({
  useRoute: () => mocks.route,
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}))

const { default: ErrorBoundary } =
  await import('../ErrorBoundary.vue')

describe('ErrorBoundary component telemetry', () => {
  beforeEach(() => {
    mocks.reportFrontendError.mockReset()
  })

  it('reports a kind-only Vue error before owning the fallback UI', async () => {
    const consoleError = vi
      .spyOn(console, 'error')
      .mockImplementation(() => undefined)
    const CrashingChild = defineComponent({
      name: 'CrashingChild',
      setup() {
        throw new Error('component-private-detail')
      },
      render: () => h('div'),
    })

    const wrapper = mount(ErrorBoundary, {
      slots: {
        default: CrashingChild,
      },
    })
    await flushPromises()

    expect(mocks.reportFrontendError).toHaveBeenCalledTimes(1)
    expect(mocks.reportFrontendError).toHaveBeenCalledWith('vue-error')
    expect(wrapper.text()).toContain('errors.boundary.title')
    expect(wrapper.text()).toContain('errors.boundary.description')

    wrapper.unmount()
    consoleError.mockRestore()
  })
})
