import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { Review } from '@stuhelper/shared/review'

const mocks = vi.hoisted(() => ({
  editReview: vi.fn(),
  error: vi.fn(),
  success: vi.fn(),
  updateReview: vi.fn(),
}))

vi.mock('@/api', () => ({
  api: {
    admin: {
      editReview: mocks.editReview,
      updateReview: mocks.updateReview,
    },
  },
}))

vi.mock('@/composables/useToast', () => ({
  useToast: () => ({
    error: mocks.error,
    success: mocks.success,
  }),
}))

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
    }),
  }
})

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({ user: null }),
}))

vi.mock('@/utils/adminAccess', () => ({
  canEditReviewContent: () => true,
  canManageReviews: () => true,
}))

import { useReviewModeration } from '@/components/business/review/useReviewModeration'
import { useReviewAdmin } from '../useReviewAdmin'

interface Deferred<T> {
  promise: Promise<T>
  reject: (reason?: unknown) => void
  resolve: (value: T) => void
}

function deferred<T>(): Deferred<T> {
  let resolve!: (value: T) => void
  let reject!: (reason?: unknown) => void
  const promise = new Promise<T>((resolvePromise, rejectPromise) => {
    resolve = resolvePromise
    reject = rejectPromise
  })
  return { promise, reject, resolve }
}

function makeReview(): Review {
  return {
    id: '550e8400-e29b-41d4-a716-446655440001',
    courseID: 20,
    courseName: '软件工程',
    teacherName: '张老师',
    termID: '2025-fall',
    title: '课堂体验',
    content: '课程内容',
    ratings: { teaching: 5 },
    likeCount: 3,
    dislikeCount: 1,
    replyCount: 0,
    status: 'published',
    createdAt: '2026-05-12T00:00:00Z',
  }
}

describe('review administration submission lifecycle', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('keeps the card moderation dialog open while pending and closes it after success', async () => {
    const request = deferred<unknown>()
    mocks.updateReview.mockReturnValueOnce(request.promise)
    const composable = useReviewModeration(
      makeReview,
      key => key,
      vi.fn(),
    )
    composable.showModerationDialog.value = true

    const submission = composable.handleModerate('需要复核')

    expect(composable.moderationSubmitting.value).toBe(true)
    expect(composable.showModerationDialog.value).toBe(true)

    request.resolve({})
    await submission

    expect(composable.moderationSubmitting.value).toBe(false)
    expect(composable.showModerationDialog.value).toBe(false)
    expect(mocks.success).toHaveBeenCalledOnce()
  })

  it('preserves the card edit dialog after a failed request and permits retry', async () => {
    mocks.editReview.mockRejectedValueOnce(new Error('network unavailable'))
    const composable = useReviewModeration(
      makeReview,
      key => key,
      vi.fn(),
    )
    composable.showEditDialog.value = true

    await composable.handleAdminEdit({
      title: '新标题',
      content: '修订后的内容',
      reason: '纠错',
    })

    expect(composable.editSubmitting.value).toBe(false)
    expect(composable.showEditDialog.value).toBe(true)
    expect(mocks.error).toHaveBeenCalledOnce()
  })

  it('uses the same pending and success semantics on the course detail page', async () => {
    const request = deferred<unknown>()
    mocks.updateReview.mockReturnValueOnce(request.promise)
    const composable = useReviewAdmin(vi.fn())
    composable.openModeration(makeReview())

    const submission = composable.handleModerate('不符合规范')

    expect(composable.moderationSubmitting.value).toBe(true)
    expect(composable.showModerationDialog.value).toBe(true)

    request.resolve({})
    await submission

    expect(composable.moderationSubmitting.value).toBe(false)
    expect(composable.showModerationDialog.value).toBe(false)
  })

  it('keeps the course detail edit dialog and its model after failure', async () => {
    mocks.editReview.mockRejectedValueOnce(new Error('network unavailable'))
    const composable = useReviewAdmin(vi.fn())
    const review = makeReview()
    composable.openEdit(review)

    await composable.handleAdminEdit({
      title: '新标题',
      content: '修订后的内容',
      reason: '纠错',
    })

    expect(composable.editSubmitting.value).toBe(false)
    expect(composable.showEditDialog.value).toBe(true)
    expect(composable.editingReview.value).toStrictEqual(review)
    expect(mocks.error).toHaveBeenCalledOnce()
  })
})
