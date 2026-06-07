// @vitest-environment jsdom

import { flushPromises, mount } from '@vue/test-utils'
import { reactive } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const mockResourceApi = vi.hoisted(() => ({
  deleteResource: vi.fn(),
  getDownloadURL: vi.fn(),
  getResource: vi.fn(),
}))

const mockRouteContainer = vi.hoisted(() => ({
  route: null as null | {
    params: { id: string }
    query: Record<string, string>
  },
}))

const mockRouterPush = vi.hoisted(() => vi.fn())
const mockAuthStore = vi.hoisted(() => ({
  user: null as null | { id: string },
}))

const translations: Record<string, string> = {
  'resource.detail.back': '返回资料列表',
  'resource.detail.loadFailed': '资料详情加载失败，请稍后重试。',
  'resource.detail.noDescription': '暂无描述',
  'resource.detail.notFound': '资料不存在或已被移除。',
  'resource.detail.titleFallback': '资料详情',
  'resource.list.retry': '重新加载',
}

vi.mock('@/api', () => ({
  api: {
    resource: mockResourceApi,
  },
}))

vi.mock('vue-router', () => {
  mockRouteContainer.route = reactive({
    params: {
      id: '999999',
    },
    query: {},
  })

  return {
    RouterLink: {
      name: 'RouterLink',
      props: ['to'],
      template: '<a data-router-link><slot /></a>',
    },
    useRoute: () => mockRouteContainer.route,
    useRouter: () => ({
      push: mockRouterPush,
    }),
  }
})

vi.mock('vue-i18n', () => {
  const t = (key: string) => translations[key] ?? key

  return {
    createI18n: () => ({
      global: {
        t,
        te: (key: string) => key in translations,
      },
      install: vi.fn(),
    }),
    useI18n: () => ({
      locale: {
        value: 'zh-CN',
      },
      t,
    }),
  }
})

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => mockAuthStore,
}))

const { default: ResourceDetailPage } = await import('../views/ResourceDetailPage.vue')

function mountPage() {
  return mount(ResourceDetailPage)
}

describe('ResourceDetailPage', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockAuthStore.user = null
    mockRouteContainer.route = reactive({
      params: {
        id: '999999',
      },
      query: {},
    })
  })

  it('shows a stable not-found state for missing resources', async () => {
    mockResourceApi.getResource.mockRejectedValueOnce({
      code: 'A0000404',
      message: 'resource not found',
      status: 404,
    })

    const wrapper = mountPage()
    await flushPromises()

    expect(mockResourceApi.getResource).toHaveBeenCalledWith('999999')
    expect(wrapper.get('[role="alert"]').text()).toContain('资料不存在或已被移除。')
    expect(wrapper.text()).not.toContain('resource not found')
    expect(wrapper.find('[data-resource-detail-retry-button]').exists()).toBe(false)
  })

  it('keeps retry available for transient resource detail failures', async () => {
    mockResourceApi.getResource.mockRejectedValueOnce(new Error('backend unavailable'))

    const wrapper = mountPage()
    await flushPromises()

    expect(wrapper.get('[role="alert"]').text()).toContain('资料详情加载失败，请稍后重试。')
    expect(wrapper.get('[data-resource-detail-retry-button]').text()).toContain('重新加载')
  })
})
