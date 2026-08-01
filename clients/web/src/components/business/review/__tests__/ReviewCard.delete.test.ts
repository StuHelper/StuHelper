// @vitest-environment jsdom

import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { Review } from '@stuhelper/shared/review'

const mocks = vi.hoisted(() => ({
  authStore: {
    isAuthenticated: true,
    user: {
      capabilities: ['review:list:full'],
      canAccessAdmin: false,
    },
    login: vi.fn(),
  },
  verificationStore: {
    canViewFullReviews: true,
  },
  deleteReview: vi.fn(),
  toastSuccess: vi.fn(),
  toastError: vi.fn(),
  loadReplies: vi.fn(),
}))

vi.mock('@/api', () => ({
  api: {
    review: {
      deleteReview: mocks.deleteReview,
    },
  },
}))

vi.mock('@/composables/useToast', () => ({
  useToast: () => ({
    success: mocks.toastSuccess,
    error: mocks.toastError,
  }),
}))

vi.mock('vue-i18n', () => ({
  createI18n: () => ({
    global: {
      t: (key: string) => key,
      te: () => false,
    },
    install: vi.fn(),
  }),
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
    handleVote: vi.fn(),
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
    handleReplySubmit: vi.fn(),
    handleDeleteReply: vi.fn(),
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

vi.mock('../useReviewModeration', () => ({
  useReviewModeration: () => ({
    showModerationDialog: false,
    showEditDialog: false,
    moderationSubmitting: false,
    editSubmitting: false,
    handleModerate: vi.fn(),
    handleRestore: vi.fn(),
    handleAdminEdit: vi.fn(),
  }),
}))

const { default: ReviewCard } = await import('../ReviewCard.vue')
const { useReviewDelete } = await import('../useReviewDelete')

const review: Review = {
  id: '550e8400-e29b-41d4-a716-446655440035',
  courseID: 20,
  courseName: '软件工程',
  teacherName: '张老师',
  termID: '2025-fall',
  title: '课堂体验',
  content: '这是一条只允许确认后删除的评课。',
  ratings: { teaching: 5 },
  likeCount: 3,
  dislikeCount: 1,
  replyCount: 0,
  status: 'published',
  createdAt: '2026-07-31T00:00:00Z',
}

function deferred() {
  let resolve!: () => void
  const promise = new Promise<void>((done) => {
    resolve = done
  })
  return { promise, resolve }
}

function mountCard() {
  return mount(ReviewCard, {
    attachTo: document.body,
    props: {
      review,
      isOwnReview: true,
    },
    global: {
      directives: {
        ripple: {},
      },
      stubs: {
        RouterLink: {
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

describe('ReviewCard own-review deletion', () => {
  beforeEach(() => {
    document.body.innerHTML = ''
    mocks.deleteReview.mockReset()
    mocks.toastSuccess.mockReset()
    mocks.toastError.mockReset()
  })

  it('requires explicit confirmation and restores focus on cancellation', async () => {
    const wrapper = mountCard()
    const requestButton = wrapper.get(
      `[data-testid="review-delete-${review.id}"]`,
    )

    await requestButton.trigger('click')

    expect(mocks.deleteReview).not.toHaveBeenCalled()
    expect(
      wrapper.get(`[data-testid="review-delete-confirm-${review.id}"]`).attributes('role'),
    ).toBe('group')
    const cancelButton = wrapper.get(
      `[data-testid="review-delete-cancel-${review.id}"]`,
    )
    expect(document.activeElement).toBe(cancelButton.element)

    await cancelButton.trigger('click')

    expect(mocks.deleteReview).not.toHaveBeenCalled()
    expect(
      wrapper.find(`[data-testid="review-delete-confirm-${review.id}"]`).exists(),
    ).toBe(false)
    expect(document.activeElement).toBe(
      wrapper.get(`[data-testid="review-delete-${review.id}"]`).element,
    )

    wrapper.unmount()
  })

  it('issues one delete only after confirmation and emits the deleted id', async () => {
    const pending = deferred()
    mocks.deleteReview.mockReturnValue(pending.promise)
    const wrapper = mountCard()

    await wrapper
      .get(`[data-testid="review-delete-${review.id}"]`)
      .trigger('click')
    await wrapper
      .get(`[data-testid="review-delete-confirm-action-${review.id}"]`)
      .trigger('click')

    expect(mocks.deleteReview).toHaveBeenCalledTimes(1)
    expect(mocks.deleteReview).toHaveBeenCalledWith(review.id)
    expect(
      wrapper.get(`[data-testid="review-delete-${review.id}"]`).attributes('disabled'),
    ).toBeDefined()

    pending.resolve()
    await flushPromises()

    expect(wrapper.emitted('deleted')).toEqual([[review.id]])
    expect(mocks.toastSuccess).toHaveBeenCalledWith(
      'review.review.deleteSuccess',
    )
    expect(mocks.toastError).not.toHaveBeenCalled()

    wrapper.unmount()
  })

  it('keeps the delete request single-flight at the composable boundary', async () => {
    const pending = deferred()
    mocks.deleteReview.mockReturnValue(pending.promise)
    const emitDeleted = vi.fn()
    const flow = useReviewDelete(
      () => review,
      (key: string) => key,
      emitDeleted,
    )

    const first = flow.handleDeleteOwn()
    const duplicate = flow.handleDeleteOwn()

    expect(flow.deleting.value).toBe(true)
    expect(mocks.deleteReview).toHaveBeenCalledTimes(1)

    pending.resolve()
    await Promise.all([first, duplicate])

    expect(flow.deleting.value).toBe(false)
    expect(emitDeleted).toHaveBeenCalledTimes(1)
    expect(emitDeleted).toHaveBeenCalledWith(review.id)
  })
})
