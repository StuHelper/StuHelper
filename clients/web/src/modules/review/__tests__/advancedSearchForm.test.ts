// @vitest-environment jsdom

import { flushPromises, mount } from '@vue/test-utils'
import { reactive } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const mockCourseApi = vi.hoisted(() => ({
  getDepartments: vi.fn(),
  getTerms: vi.fn(),
  searchCourses: vi.fn(),
}))

const mockReviewApi = vi.hoisted(() => ({
  searchReviewsPage: vi.fn(),
}))

const mockRouteContainer = vi.hoisted(() => ({
  route: null as null | {
    query: Record<string, string>
  },
}))

const mockRouter = vi.hoisted(() => ({
  push: vi.fn(),
}))

const mockToastError = vi.hoisted(() => vi.fn())
const mockScrollTo = vi.hoisted(() => vi.fn())

const translations: Record<string, string> = {
  'common.actions.back': '返回',
  'common.actions.loadMore': '加载更多',
  'common.actions.loading': '加载中...',
  'common.actions.retry': '重试',
  'common.loadFailed': '加载失败',
  'review.course.reviewUnit': '条测评',
  'review.search.allDepartments': '全部',
  'review.search.anySemester': '任意',
  'review.search.atLeastOne': '请至少输入一个搜索条件！',
  'review.search.courseCode': '课程代码',
  'review.search.courseCodeHelper': '学校内部课程编号',
  'review.search.courseConditions': '按课程条件搜索',
  'review.search.courseName': '课程名称',
  'review.search.courseNameHelper': '支持中英文及拼音搜索',
  'review.search.coursesWithReviews': '有测评的课程',
  'review.search.coursesWithoutReviews': '无测评课程',
  'review.search.department': '开课院系',
  'review.search.noResults': '没有找到结果，请换个条件试试。',
  'review.search.results': '搜索结果',
  'review.search.reviewConditions': '按测评条件搜索',
  'review.search.reviewResults': '测评结果',
  'review.search.searchBtn': '搜索',
  'review.search.semester': '学期',
  'review.search.subtitle': '自由组合以下搜索条件以搜索测评',
  'review.search.teacherName': '教师姓名',
  'review.search.teacherNameHelper': '支持模糊搜索',
  'review.search.title': '高级搜索',
}

vi.mock('@/api', () => ({
  api: {
    course: mockCourseApi,
    review: mockReviewApi,
  },
}))

vi.mock('vue-router', () => {
  mockRouteContainer.route = reactive({
    query: {},
  })

  return {
    useRoute: () => mockRouteContainer.route,
    useRouter: () => mockRouter,
  }
})
vi.mock('vue-i18n', () => {
  const t = (key: string, params?: Record<string, unknown>) => {
    const message = translations[key] ?? key
    if (!params) return message
    return Object.entries(params).reduce(
      (current, [name, value]) => current.replace(`{${name}}`, String(value)),
      message,
    )
  }

  return {
    createI18n: () => ({
      global: {
        t,
        te: (key: string) => key in translations,
      },
      install: vi.fn(),
    }),
    useI18n: () => ({
      t,
    }),
  }
})

vi.mock('@/api/errors', () => ({
  getErrorMessage: (_error: unknown, fallback: string) => fallback,
}))

vi.mock('@/composables/useToast', () => ({
  useToast: () => ({
    error: mockToastError,
  }),
}))

vi.mock('@/composables/usePageMeta', () => ({
  updatePageMeta: vi.fn(),
}))

vi.mock('@/components/business/review/ReviewCard.vue', () => ({
  default: {
    name: 'ReviewCard',
    props: ['review'],
    emits: ['moderated'],
    template: '<article data-review-card>{{ review.title }}</article>',
  },
}))

const { default: SearchPage } = await import('../views/SearchPage.vue')

function mountSearchPage() {
  return mount(SearchPage, {
    global: {
      stubs: {
        RouterLink: true,
        'router-link': true,
      },
    },
  })
}

describe('SearchPage advanced form', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockRouteContainer.route = reactive({
      query: {},
    })
    window.scrollTo = mockScrollTo
    mockCourseApi.getDepartments.mockResolvedValue({
      data: {
        data: [
          {
            id: 1,
            name: '数学科学学院',
            category: 'department',
          },
        ],
      },
    })
    mockCourseApi.getTerms.mockResolvedValue({
      data: {
        data: [
          {
            id: '2025-1',
            name: '2025 春',
            isCurrent: true,
          },
        ],
      },
    })
    mockCourseApi.searchCourses.mockResolvedValue({
      data: {
        data: {
          list: [
            {
              id: 8,
              departmentID: 1,
              departmentName: '数学科学学院',
              code: 'MA1001',
              name: '高等数学A',
              credits: 5,
              reviewCount: 2,
            },
          ],
          total: 1,
        },
      },
    })
    mockReviewApi.searchReviewsPage.mockResolvedValue({
      list: [
        {
          id: 101,
          title: '高等数学A',
        },
      ],
      total: 1,
    })
  })

  it('submits search criteria from the form and renders results', async () => {
    const wrapper = mountSearchPage()
    await flushPromises()

    await wrapper.get('#advanced-course-name').setValue('高等数学')
    await wrapper.get('form').trigger('submit')
    await flushPromises()

    expect(mockCourseApi.searchCourses).toHaveBeenCalledWith(
      '高等数学',
      { page: 1, pageSize: 50 },
      { signal: expect.any(AbortSignal) },
    )
    expect(mockReviewApi.searchReviewsPage).toHaveBeenCalledWith(
      {
        q: '高等数学',
        departmentID: undefined,
        teacherName: undefined,
        termID: undefined,
        page: 1,
        pageSize: 50,
        sort: 'time',
      },
      { signal: expect.any(AbortSignal) },
    )
    expect(wrapper.text()).toContain('搜索结果')
    expect(wrapper.text()).toContain('高等数学A')
    expect(wrapper.find('form').exists()).toBe(false)
  })

  it('refreshes only the loaded review pages after moderation', async () => {
    const wrapper = mountSearchPage()
    await flushPromises()

    await wrapper.get('#advanced-course-name').setValue('高等数学')
    await wrapper.get('form').trigger('submit')
    await flushPromises()
    expect(wrapper.text()).toContain('高等数学A')
    const scrollCountBeforeModeration = mockScrollTo.mock.calls.length

    mockReviewApi.searchReviewsPage.mockResolvedValueOnce({
      list: [{ id: 101, title: '审核后的高等数学A' }],
      total: 1,
    })
    wrapper.findComponent({ name: 'ReviewCard' }).vm.$emit('moderated')
    await flushPromises()

    expect(mockReviewApi.searchReviewsPage).toHaveBeenCalledTimes(2)
    expect(mockCourseApi.searchCourses).toHaveBeenCalledTimes(1)
    expect(mockScrollTo).toHaveBeenCalledTimes(scrollCountBeforeModeration)
    expect(wrapper.text()).toContain('审核后的高等数学A')
  })
})
