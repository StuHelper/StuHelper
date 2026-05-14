// @vitest-environment jsdom

import { mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import ReviewCard from '../ReviewCard.vue'
import type { Review } from '@stuhelper/shared/review'

const mocks = vi.hoisted(() => ({
  authStore: {
    isAuthenticated: true,
    user: {
      capabilities: ['review:list:brief'],
      canAccessAdmin: false,
    },
    login: vi.fn(),
  },
  verificationStore: {
    canViewFullReviews: false,
  },
  routerPush: vi.fn(),
  loadReplies: vi.fn(),
  handleReplySubmit: vi.fn(),
  handleDeleteReply: vi.fn(),
  handleVote: vi.fn(),
}))

vi.mock('vue-router', () => ({
  useRouter: () => ({ push: mocks.routerPush }),
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key,
    locale: { value: 'zh-CN' },
  }),
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => mocks.authStore,
}))

vi.mock('@/stores/verification', () => ({
  useVerificationStore: () => mocks.verificationStore,
}))

vi.mock('@/composables/use3DTilt', () => ({
  use3DTilt: () => ({ style: {} }),
}))

vi.mock('../useReviewVote', () => ({
  useReviewVote: () => ({
    userVote: null,
    likeBounce: false,
    shaking: false,
    displayLikes: 0,
    displayDislikes: 0,
    handleVote: mocks.handleVote,
  }),
}))

vi.mock('../useReviewReply', () => ({
  useReviewReply: () => ({
    replies: [],
    repliesLoading: false,
    repliesError: false,
    replySubmitting: false,
    replyCount: 0,
    replyFormRef: null,
    loadReplies: mocks.loadReplies,
    handleReplySubmit: mocks.handleReplySubmit,
    handleDeleteReply: mocks.handleDeleteReply,
  }),
}))

vi.mock('../useReviewReport', () => ({
  useReviewReport: () => ({
    showReportMenu: false,
    reporting: false,
    reportReasons: [],
    toggleReportMenu: vi.fn(),
    handleReport: vi.fn(),
  }),
}))

vi.mock('../useReviewEdit', () => ({
  useReviewEdit: () => ({
    editing: false,
    editContent: '',
    saving: false,
    startEditing: vi.fn(),
    cancelEditing: vi.fn(),
    handleSaveEdit: vi.fn(),
  }),
}))

vi.mock('../useReviewDelete', () => ({
  useReviewDelete: () => ({
    deleting: false,
    handleDeleteOwn: vi.fn(),
  }),
}))

vi.mock('../useReviewModeration', () => ({
  useReviewModeration: () => ({
    showModerationDialog: false,
    showEditDialog: false,
    handleModerate: vi.fn(),
    handleRestore: vi.fn(),
    handleAdminEdit: vi.fn(),
  }),
}))

const previewLine = '这是一段可以显示的第一行预览'
const hiddenLine = '这是一段无权限用户不应该看到的完整评课正文'

function makeReview(): Review {
  return {
    id: '550e8400-e29b-41d4-a716-446655440001',
    courseID: 20,
    courseName: '软件工程',
    teacherName: '张老师',
    termID: '2025-fall',
    title: '课堂体验',
    content: previewLine,
    ratings: { teaching: 5 },
    likeCount: 3,
    dislikeCount: 1,
    replyCount: 0,
    status: 'published',
    createdAt: '2026-05-12T00:00:00Z',
  }
}

function mountCard() {
  return mount(ReviewCard, {
    props: {
      review: makeReview(),
    },
    global: {
      directives: {
        ripple: {},
      },
      stubs: {
        RouterLink: {
          props: ['to'],
          template: '<a><slot /></a>',
        },
        ReplyCard: true,
        ReplyForm: true,
        ModerationDialog: true,
        AdminEditDialog: true,
      },
    },
  })
}

describe('ReviewCard locked content', () => {
  beforeEach(() => {
    Object.assign(mocks.authStore, {
      isAuthenticated: true,
      user: {
        capabilities: ['review:list:brief'],
        canAccessAdmin: false,
      },
    })
    mocks.verificationStore.canViewFullReviews = false
  })

  it('renders the safe first-line preview in the locked state', () => {
    const wrapper = mountCard()

    expect(wrapper.text()).toContain('review.card.verifyToView')
    expect(wrapper.text()).toContain(previewLine)
    expect(wrapper.text()).not.toContain(hiddenLine)
  })

  it('requires both full-list capability and student verification before rendering body text', () => {
    mocks.authStore.user = {
      capabilities: ['review:list:full'],
      canAccessAdmin: false,
    }

    const capabilityOnly = mountCard()
    expect(capabilityOnly.text()).toContain('review.card.verifyToView')
    expect(capabilityOnly.text()).toContain(previewLine)
    expect(capabilityOnly.text()).not.toContain(hiddenLine)

    mocks.verificationStore.canViewFullReviews = true
    const fullAccess = mountCard()
    expect(fullAccess.text()).toContain(previewLine)
  })
})
