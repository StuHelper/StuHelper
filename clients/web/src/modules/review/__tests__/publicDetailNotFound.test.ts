// @vitest-environment jsdom

import { flushPromises, mount } from '@vue/test-utils'
import { reactive } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const mockCourseApi = vi.hoisted(() => ({
  getCourse: vi.fn(),
}))

const mockRatingApi = vi.hoisted(() => ({
  getCourseStats: vi.fn(),
  getCourseTeachers: vi.fn(),
  getRatingTrend: vi.fn(),
  getTeacherStats: vi.fn(),
}))

const mockReviewApi = vi.hoisted(() => ({
  getLatestReviewsPage: vi.fn(),
  getReviewsPage: vi.fn(),
}))

const mockRouteContainer = vi.hoisted(() => ({
  route: null as null | {
    fullPath: string
    matched: Array<{ name: string }>
    params: { id: string }
    path: string
    query: Record<string, string>
  },
}))

const mockRouter = vi.hoisted(() => ({
  push: vi.fn(),
  replace: vi.fn(),
  resolve: vi.fn(() => ({ fullPath: '/courses/reviews/post' })),
}))

const mockToastError = vi.hoisted(() => vi.fn())

const translations: Record<string, string> = {
  'common.actions.loading': '加载中...',
  'common.actions.loadMore': '加载更多',
  'common.actions.retry': '重试',
  'common.loadFailed': '加载失败',
  'review.course.notFound': '课程不存在或已被移除。',
  'review.courseDetail': '课程详情',
  'teaching.profile.notFound': '未找到教师信息',
  'teaching.profile.reviewsLoadFailed': '评课加载失败',
}

vi.mock('@/api', () => ({
  api: {
    course: mockCourseApi,
    rating: mockRatingApi,
    review: mockReviewApi,
  },
}))

vi.mock('vue-router', () => {
  mockRouteContainer.route = reactive({
    fullPath: '/courses/999999',
    matched: [],
    params: {
      id: '999999',
    },
    path: '/courses/999999',
    query: {},
  })

  return {
    useRoute: () => mockRouteContainer.route,
    useRouter: () => mockRouter,
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

vi.mock('@/composables/useToast', () => ({
  useToast: () => ({
    error: mockToastError,
  }),
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({
    isAuthenticated: false,
    login: vi.fn(),
    user: null,
  }),
}))

vi.mock('@/stores/verification', () => ({
  useVerificationStore: () => ({
    canViewFullReviews: false,
  }),
}))

vi.mock('@/utils/adminAccess', () => ({
  canListFullReviews: () => false,
}))

vi.mock('@/composables/useReviewPost', () => ({
  useReviewPost: () => ({
    ensureCanPostReview: vi.fn(),
  }),
}))

vi.mock('@/modules/review/reviewPostNavigation', () => ({
  rememberReviewPostCourse: vi.fn(),
}))

vi.mock('@/modules/review/useReviewAdmin', () => ({
  useReviewAdmin: () => ({
    canManageReviews: false,
    editingReview: null,
    handleAdminEdit: vi.fn(),
    handleModerate: vi.fn(),
    handleRestore: vi.fn(),
    moderatingReviewID: null,
    openEdit: vi.fn(),
    openModeration: vi.fn(),
    showEditDialog: false,
    showModerationDialog: false,
  }),
}))

vi.mock('@/modules/review/useReviewReplies', () => ({
  useReviewReplies: () => ({
    expandedReviewID: null,
    handleDeleteReply: vi.fn(),
    handleReplySubmit: vi.fn(),
    loadReplies: vi.fn(),
    replies: [],
    repliesError: false,
    repliesLoading: false,
    replyCountMap: {},
    replyFormRef: null,
    replySubmitting: false,
    toggleExpand: vi.fn(),
  }),
}))

vi.mock('@/modules/review/useReviewVoting', () => ({
  useReviewVoting: () => ({
    displayDislikeCount: vi.fn(() => 0),
    displayLikeCount: vi.fn(() => 0),
    handleVote: vi.fn(),
    reviewVotes: {},
  }),
}))

vi.mock('@/modules/course/theme/CourseThemeProvider.vue', () => ({
  default: {
    name: 'CourseThemeProvider',
    template: '<div><slot /></div>',
  },
}))

vi.mock('@/components/business/review/EmojiRating.vue', () => ({
  default: {
    name: 'EmojiRating',
    template: '<div data-stubbed-child />',
  },
}))

vi.mock('@/components/business/review/ControversialBadge.vue', () => ({
  default: {
    name: 'ControversialBadge',
    template: '<div data-stubbed-child />',
  },
}))

vi.mock('@/components/business/review/FavoriteButton.vue', () => ({
  default: {
    name: 'FavoriteButton',
    template: '<div data-stubbed-child />',
  },
}))

vi.mock('@/components/business/review/LockedReviewContent.vue', () => ({
  default: {
    name: 'LockedReviewContent',
    template: '<div data-stubbed-child />',
  },
}))

vi.mock('@/components/business/review/ReplyCard.vue', () => ({
  default: {
    name: 'ReplyCard',
    template: '<div data-stubbed-child />',
  },
}))

vi.mock('@/components/business/review/ReplyForm.vue', () => ({
  default: {
    name: 'ReplyForm',
    template: '<div data-stubbed-child />',
  },
}))

vi.mock('@/components/business/review/ReplyLoginPrompt.vue', () => ({
  default: {
    name: 'ReplyLoginPrompt',
    template: '<div data-stubbed-child />',
  },
}))

vi.mock('@/components/business/review/ModerationDialog.vue', () => ({
  default: {
    name: 'ModerationDialog',
    template: '<div data-stubbed-child />',
  },
}))

vi.mock('@/components/business/review/AdminEditDialog.vue', () => ({
  default: {
    name: 'AdminEditDialog',
    template: '<div data-stubbed-child />',
  },
}))

vi.mock('@/components/common/RatingCircle.vue', () => ({
  default: {
    name: 'RatingCircle',
    template: '<div data-stubbed-child><slot /></div>',
  },
}))

vi.mock('@/components/business/review/ReviewCard.vue', () => ({
  default: {
    name: 'ReviewCard',
    props: ['review'],
    template: '<div data-review-card>{{ review.title }}</div>',
  },
}))

const { default: CourseDetailPage } = await import('../views/CourseDetailPage.vue')
const { default: TeacherProfilePage } = await import('../views/TeacherProfilePage.vue')

function setRoute(path: string, id = '999999') {
  mockRouteContainer.route = reactive({
    fullPath: path,
    matched: [],
    params: {
      id,
    },
    path,
    query: {},
  })
}

function mountPage(component: unknown) {
  return mount(component, {
    global: {
      stubs: {
        RouterLink: true,
        'router-link': true,
      },
    },
  })
}

function resolveCourseSideRequests() {
  mockRatingApi.getCourseStats.mockResolvedValue({
    data: {
      data: {
        allDimensionKeys: [],
        byTerm: [],
        courseID: 999999,
        overall: {
          dimensions: [],
          termName: '总体',
        },
      },
    },
  })
  mockRatingApi.getCourseTeachers.mockResolvedValue({
    data: {
      data: [],
    },
  })
  mockRatingApi.getRatingTrend.mockResolvedValue({
    data: {
      data: {
        trend: [],
      },
    },
  })
  mockReviewApi.getReviewsPage.mockResolvedValue({
    list: [],
    total: 0,
  })
}

describe('public detail not-found states', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    setRoute('/courses/999999')
    resolveCourseSideRequests()
    mockReviewApi.getLatestReviewsPage.mockResolvedValue({
      list: [],
      total: 0,
    })
  })

  it('shows a stable course not-found state without retry affordance', async () => {
    mockCourseApi.getCourse.mockRejectedValueOnce({
      code: 'A0100001',
      message: 'course not found',
      status: 404,
    })

    const wrapper = mountPage(CourseDetailPage)
    await flushPromises()

    expect(mockCourseApi.getCourse).toHaveBeenCalledWith(999999)
    expect(mockRatingApi.getCourseStats).not.toHaveBeenCalled()
    expect(mockReviewApi.getReviewsPage).not.toHaveBeenCalled()
    expect(mockRatingApi.getCourseTeachers).not.toHaveBeenCalled()
    expect(mockRatingApi.getRatingTrend).not.toHaveBeenCalled()
    expect(wrapper.get('[role="alert"]').text()).toContain('课程不存在或已被移除。')
    expect(wrapper.text()).not.toContain('加载失败')
    expect(wrapper.find('[data-course-detail-retry-button]').exists()).toBe(false)
  })

  it('shows course not-found for invalid course ids without routing away', async () => {
    setRoute('/courses/not-a-number', 'not-a-number')

    const wrapper = mountPage(CourseDetailPage)
    await flushPromises()

    expect(mockRouter.replace).not.toHaveBeenCalled()
    expect(mockCourseApi.getCourse).not.toHaveBeenCalled()
    expect(mockRatingApi.getCourseStats).not.toHaveBeenCalled()
    expect(mockReviewApi.getReviewsPage).not.toHaveBeenCalled()
    expect(wrapper.get('[role="alert"]').text()).toContain('课程不存在或已被移除。')
    expect(wrapper.find('[data-course-detail-retry-button]').exists()).toBe(false)
  })

  it('keeps course retry available for transient detail failures', async () => {
    mockCourseApi.getCourse.mockRejectedValueOnce(new Error('backend unavailable'))

    const wrapper = mountPage(CourseDetailPage)
    await flushPromises()

    expect(wrapper.get('[role="alert"]').text()).toContain('加载失败')
    expect(wrapper.get('[data-course-detail-retry-button]').text()).toContain('重试')
  })

  it('shows a stable teacher not-found state without retry affordance', async () => {
    setRoute('/teachers/999999')
    mockRatingApi.getTeacherStats.mockRejectedValueOnce({
      code: 'A0100003',
      message: 'teacher not found',
      status: 404,
    })

    const wrapper = mountPage(TeacherProfilePage)
    await flushPromises()

    expect(mockRatingApi.getTeacherStats).toHaveBeenCalledWith(999999)
    expect(wrapper.get('[role="alert"]').text()).toContain('未找到教师信息')
    expect(wrapper.text()).not.toContain('加载失败')
    expect(wrapper.find('[data-teacher-profile-retry-button]').exists()).toBe(false)
    expect(mockReviewApi.getLatestReviewsPage).not.toHaveBeenCalled()
  })

  it('shows teacher not-found for invalid teacher ids without routing away', async () => {
    setRoute('/teachers/not-a-number', 'not-a-number')

    const wrapper = mountPage(TeacherProfilePage)
    await flushPromises()

    expect(mockRouter.replace).not.toHaveBeenCalled()
    expect(mockRatingApi.getTeacherStats).not.toHaveBeenCalled()
    expect(mockReviewApi.getLatestReviewsPage).not.toHaveBeenCalled()
    expect(wrapper.get('[role="alert"]').text()).toContain('未找到教师信息')
    expect(wrapper.find('[data-teacher-profile-retry-button]').exists()).toBe(false)
  })

  it('keeps teacher retry available for transient detail failures', async () => {
    setRoute('/teachers/999999')
    mockRatingApi.getTeacherStats.mockRejectedValueOnce(new Error('backend unavailable'))

    const wrapper = mountPage(TeacherProfilePage)
    await flushPromises()

    expect(wrapper.get('[role="alert"]').text()).toContain('加载失败')
    expect(wrapper.get('[data-teacher-profile-retry-button]').text()).toContain('重试')
  })

  it('keeps loaded teacher reviews visible when loading the next page fails', async () => {
    setRoute('/teachers/42', '42')
    mockRatingApi.getTeacherStats.mockResolvedValueOnce({
      data: {
        data: {
          teacherID: 42,
          teacherName: '测试教师',
          departmentName: '测试学院',
          avgRating: 4.5,
          courseCount: 0,
          reviewCount: 2,
          courses: [],
          ratingTrend: [],
        },
      },
    })
    mockReviewApi.getLatestReviewsPage
      .mockResolvedValueOnce({ list: [{ id: 'review-1', title: '已加载评课' }], total: 2 })
      .mockRejectedValueOnce(new Error('temporary outage'))

    const wrapper = mountPage(TeacherProfilePage)
    await flushPromises()
    expect(wrapper.get('[data-review-card]').text()).toBe('已加载评课')

    const loadMore = wrapper.findAll('button').find(button => button.text().includes('加载更多'))
    expect(loadMore).toBeDefined()
    await loadMore!.trigger('click')
    await flushPromises()

    expect(wrapper.get('[data-review-card]').text()).toBe('已加载评课')
    expect(wrapper.get('[role="alert"]').text()).toContain('评课加载失败')
    expect(mockReviewApi.getLatestReviewsPage).toHaveBeenCalledTimes(2)

    mockReviewApi.getLatestReviewsPage.mockResolvedValueOnce({
      list: [{ id: 'review-2', title: '重试加载的评课' }],
      total: 2,
    })
    const retry = wrapper.findAll('button').find(button => button.text().includes('重试'))
    expect(retry).toBeDefined()
    await retry!.trigger('click')
    await flushPromises()

    expect(wrapper.findAll('[data-review-card]').map(card => card.text())).toEqual([
      '已加载评课',
      '重试加载的评课',
    ])
    expect(mockReviewApi.getLatestReviewsPage).toHaveBeenCalledTimes(3)
  })
})
