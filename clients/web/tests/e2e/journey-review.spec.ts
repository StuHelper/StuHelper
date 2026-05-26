/**
 * User Journey: Review lifecycle
 * Login → Find course → Post review → See it appear → Vote → Reply → Favorite
 *
 * Simulates a verified student interacting with course reviews.
 */
import { test, expect, mockNotificationStream, type Page } from './fixtures'

const verifiedStudent = {
  id: 'u2',
  name: 'bob',
  displayName: 'Bob',
  email: 'bob@example.com',
  roles: ['verified_student'],
  capabilities: [
    'review:list:full',
    'review:create',
    'review:edit:own',
    'review:delete:own',
  ],
  globalCapabilities: [
    'review:list:full',
    'review:create',
    'review:edit:own',
    'review:delete:own',
  ],
  capabilityGrants: [],
  isPlatformAdmin: false,
  canAccessAdmin: false,
}

async function mockAuth(page: Page) {
  await page.addInitScript((u) => {
    localStorage.setItem('stuhelper_user', JSON.stringify(u))
    localStorage.setItem(
      'stuhelper_token_expiry',
      String(Date.now() + 60 * 60 * 1000),
    )
  }, verifiedStudent)

  await page.route('**/api/v1/auth/me', (route) =>
    route.fulfill({
      contentType: 'application/json',
      body: JSON.stringify({ success: true, data: verifiedStudent }),
    }),
  )
  await page.route('**/api/v1/auth/refresh', (route) =>
    route.fulfill({
      contentType: 'application/json',
      body: JSON.stringify({ success: true, data: { expiresIn: 3600 } }),
    }),
  )
  await page.route('**/api/v1/user/identity', (route) =>
    route.fulfill({
      contentType: 'application/json',
      body: JSON.stringify({
        success: true,
        data: { verified: true, status: 'verified' },
      }),
    }),
  )
  await page.route('**/api/v1/user/profile', (route) =>
    route.fulfill({
      contentType: 'application/json',
      body: JSON.stringify({
        success: true,
        data: {
          verificationStatus: 'verified',
          schoolName: '测试大学',
          schoolID: 1,
        },
      }),
    }),
  )
  await page.route('**/api/v1/user/qq-binding', (route) =>
    route.fulfill({
      status: 404,
      contentType: 'application/json',
      body: JSON.stringify({
        success: false,
        error: { code: 'A0040404', message: 'not bound' },
      }),
    }),
  )
  await page.route(
    '**/api/v1/course/review/user/notifications/unread-count*',
    (route) =>
      route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify({ success: true, data: { count: 0 } }),
      }),
  )
  await mockNotificationStream(page)
  await page.route('**/api/v1/course/review/courses/*/favorites', (route) =>
    route.fulfill({
      contentType: 'application/json',
      body: JSON.stringify({ success: true, data: { favorited: false } }),
    }),
  )
}

const existingReview = {
  id: 'rev-existing',
  courseID: 101,
  courseName: '数据结构与算法',
  teacherName: '张教授',
  teacherID: 10,
  termID: '2025-fall',
  termName: '2025 秋',
  title: '已有评价标题',
  content: '这是一条已存在的课程评价内容，用于测试投票和回复功能。',
  ratings: { recommendation: 4, content_quality: 4, workload: 3, grading: 4 },
  likeCount: 5,
  dislikeCount: 1,
  replyCount: 0,
  status: 'published',
  createdAt: '2026-03-20T10:00:00Z',
  authorName: 'student_a',
  authorDisplayName: '匿名用户',
}

function makeReply(id: string, content: string) {
  return {
    id,
    reviewID: 'rev-existing',
    parentID: null,
    content,
    likeCount: 0,
    status: 'published',
    isOwner: true,
    createdAt: '2026-03-20T10:05:00Z',
    updatedAt: '2026-03-20T10:05:00Z',
  }
}

test.describe('User Journey: Review Lifecycle', () => {
  test.beforeEach(async ({ page }) => {
    await mockAuth(page)
  })

  test('authenticated user votes on a review (like → dislike toggle)', async ({
    page,
  }) => {
    let lastVotePayload: Record<string, unknown> | null = null

    // Mock course detail
    await page.route('**/api/v1/course/courses/101', (route) =>
      route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: {
            id: 101,
            name: '数据结构与算法',
            code: 'CS201',
            departmentName: '计算机科学与技术学院',
          },
        }),
      }),
    )

    await page.route(
      '**/api/v1/course/review/courses/101/reviews*',
      (route) =>
        route.fulfill({
          contentType: 'application/json',
          body: JSON.stringify({
            success: true,
            data: {
              list: [existingReview],
              total: 1,
              page: 1,
              pageSize: 20,
            },
          }),
        }),
    )

    await page.route(
      '**/api/v1/course/review/courses/101/rating-stats*',
      (route) =>
        route.fulfill({
          contentType: 'application/json',
          body: JSON.stringify({
            success: true,
            data: {
              courseID: 101,
              overall: { termName: '总体', dimensions: [] },
              byTerm: [],
              allDimensionKeys: [],
            },
          }),
        }),
    )

    await page.route(
      '**/api/v1/course/review/courses/101/teachers*',
      (route) =>
        route.fulfill({
          contentType: 'application/json',
          body: JSON.stringify({ success: true, data: [] }),
        }),
    )

    await page.route(
      '**/api/v1/course/review/courses/101/rating-trend*',
      (route) =>
        route.fulfill({
          contentType: 'application/json',
          body: JSON.stringify({ success: true, data: { trend: [] } }),
        }),
    )

    await page.route('**/api/v1/course/terms*', (route) =>
      route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: [{ id: '2025-fall', name: '2025 秋' }],
        }),
      }),
    )

    // Capture vote payloads
    await page.route(
      '**/api/v1/course/review/reviews/rev-existing/votes',
      async (route) => {
        lastVotePayload = route.request().postDataJSON() as Record<
          string,
          unknown
        >
        const vt = lastVotePayload?.voteType as string
        await route.fulfill({
          contentType: 'application/json',
          body: JSON.stringify({
            success: true,
            data: { voteType: vt },
          }),
        })
      },
    )

    await page.goto('/courses/101/reviews')
    await page.waitForLoadState('networkidle')

    // Review title visible
    await expect(page.getByText('已有评价标题')).toBeVisible({
      timeout: 10_000,
    })

    // Click like button
    await page.getByTestId('review-like-rev-existing').click()
    await expect
      .poll(() => lastVotePayload?.voteType as string | undefined)
      .toBe('like')

    // Click dislike button (toggle)
    await page.getByTestId('review-dislike-rev-existing').click()
    await expect
      .poll(() => lastVotePayload?.voteType as string | undefined)
      .toBe('dislike')
  })

  test('authenticated user posts a reply to a review', async ({ page }) => {
    let replyPayload: Record<string, unknown> | null = null

    // Same course detail mocks
    await page.route('**/api/v1/course/courses/101', (route) =>
      route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: {
            id: 101,
            name: '数据结构与算法',
            departmentName: '计算机科学与技术学院',
          },
        }),
      }),
    )

    await page.route(
      '**/api/v1/course/review/courses/101/reviews*',
      (route) =>
        route.fulfill({
          contentType: 'application/json',
          body: JSON.stringify({
            success: true,
            data: {
              list: [existingReview],
              total: 1,
              page: 1,
              pageSize: 20,
            },
          }),
        }),
    )

    await page.route(
      '**/api/v1/course/review/courses/101/rating-stats*',
      (route) =>
        route.fulfill({
          contentType: 'application/json',
          body: JSON.stringify({
            success: true,
            data: {
              courseID: 101,
              overall: { termName: '总体', dimensions: [] },
              byTerm: [],
              allDimensionKeys: [],
            },
          }),
        }),
    )
    await page.route(
      '**/api/v1/course/review/courses/101/teachers*',
      (route) =>
        route.fulfill({
          contentType: 'application/json',
          body: JSON.stringify({ success: true, data: [] }),
        }),
    )
    await page.route(
      '**/api/v1/course/review/courses/101/rating-trend*',
      (route) =>
        route.fulfill({
          contentType: 'application/json',
          body: JSON.stringify({ success: true, data: { trend: [] } }),
        }),
    )
    await page.route('**/api/v1/course/terms*', (route) =>
      route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: [{ id: '2025-fall', name: '2025 秋' }],
        }),
      }),
    )

    // Mock replies
    await page.route(
      '**/api/v1/course/review/reviews/rev-existing/replies',
      async (route) => {
        if (route.request().method() === 'POST') {
          replyPayload = route.request().postDataJSON() as Record<
            string,
            unknown
          >
          await route.fulfill({
            status: 201,
            contentType: 'application/json',
            body: JSON.stringify({
              success: true,
              data: makeReply('reply-new', String(replyPayload?.content ?? '')),
            }),
          })
        } else {
          await route.fulfill({
            contentType: 'application/json',
            body: JSON.stringify({
              success: true,
              data: { list: [], total: 0 },
            }),
          })
        }
      },
    )

    await page.goto('/courses/101/reviews')
    await page.waitForLoadState('networkidle')

    // Wait for reviews to render
    await expect(page.getByText('已有评价标题')).toBeVisible({
      timeout: 10_000,
    })

    // Click comment/reply button to expand reply area (MessageCircle icon)
    const replyToggle = page
      .locator('button')
      .filter({ has: page.locator('svg.lucide-message-circle') })
      .first()
    await replyToggle.click()

    // Type reply
    const replyTextarea = page.locator('textarea').first()
    await replyTextarea.waitFor({ state: 'visible', timeout: 5_000 })
    await replyTextarea.fill('很赞同这个评价，确实讲得很好！')

    // Submit reply (click send button)
    const sendBtn = page
      .locator('button')
      .filter({ hasText: /send|发送/i })
      .first()
    await sendBtn.click()

    // Verify reply was submitted with correct content
    await expect
      .poll(() => replyPayload?.content as string | undefined)
      .toBe('很赞同这个评价，确实讲得很好！')
  })
})
