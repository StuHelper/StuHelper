/**
 * User Journey: Full browsing flow
 * Home → Explore → Course List → Course Detail → View Reviews → Teacher Profile
 *
 * Simulates an unauthenticated user discovering the platform.
 */
import { test, expect, type Page } from '@playwright/test'

async function mockUnauthenticated(page: Page) {
  await page.route('**/api/v1/auth/me', (route) =>
    route.fulfill({
      status: 401,
      contentType: 'application/json',
      body: JSON.stringify({
        success: false,
        error: { code: 'A0010100', message: 'login required' },
      }),
    }),
  )
}

// Shared mock data
const departments = [
  { id: 1, name: '计算机科学与技术学院', code: 'CS', category: '工科' },
  { id: 2, name: '数学科学学院', code: 'MATH', category: '理科' },
]

const courses = [
  {
    id: 101,
    name: '数据结构与算法',
    code: 'CS201',
    departmentName: '计算机科学与技术学院',
    departmentID: 1,
    credits: 4,
    reviewCount: 23,
  },
  {
    id: 102,
    name: '高等数学A',
    code: 'MATH101',
    departmentName: '数学科学学院',
    departmentID: 2,
    credits: 5,
    reviewCount: 45,
  },
]

const sampleReviews = [
  {
    id: 'rev-001',
    courseID: 101,
    courseName: '数据结构与算法',
    teacherName: '张教授',
    teacherID: 10,
    termID: '2025-fall',
    termName: '2025 秋',
    title: '干货满满，强烈推荐',
    content: '课程内容设计非常好，从基础到进阶循序渐进。张教授讲课清晰，PPT质量很高。作业难度适中，考试公平。',
    ratings: { recommendation: 5, content_quality: 5, workload: 3, grading: 4 },
    likeCount: 12,
    dislikeCount: 1,
    replyCount: 3,
    status: 'published',
    createdAt: '2026-03-15T10:00:00Z',
    authorName: 'student_a',
    authorDisplayName: '匿名用户',
  },
  {
    id: 'rev-002',
    courseID: 101,
    courseName: '数据结构与算法',
    teacherName: '张教授',
    teacherID: 10,
    termID: '2025-fall',
    termName: '2025 秋',
    title: '内容不错但作业偏多',
    content: '课程本身质量很高，但每周都有编程作业，工作量比较大。不过学到的东西确实很扎实。',
    ratings: { recommendation: 4, content_quality: 4, workload: 2, grading: 3 },
    likeCount: 5,
    dislikeCount: 2,
    replyCount: 1,
    status: 'published',
    createdAt: '2026-03-10T08:00:00Z',
    authorName: 'student_b',
    authorDisplayName: '匿名用户',
  },
]

function setupCommonMocks(page: Page) {
  return Promise.all([
    page.route('**/api/v1/course/departments*', (route) =>
      route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify({ success: true, data: departments }),
      }),
    ),
    page.route('**/api/v1/course/terms*', (route) =>
      route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: [
            { id: '2025-fall', name: '2025 秋', isCurrent: true },
            { id: '2025-spring', name: '2025 春', isCurrent: false },
          ],
        }),
      }),
    ),
    page.route('**/api/v1/course/categories*', (route) =>
      route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify({ success: true, data: [] }),
      }),
    ),
    page.route('**/api/v1/course/courses?*', (route) =>
      route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: { list: courses, total: courses.length },
        }),
      }),
    ),
    page.route('**/api/v1/course/stats', (route) =>
      route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: { courseCount: 120, departmentCount: 8 },
        }),
      }),
    ),
    page.route('**/api/v1/course/review/stats', (route) =>
      route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: { reviewCount: 580, userCount: 230 },
        }),
      }),
    ),
  ])
}

test.describe('User Journey: Browse Platform', () => {
  test.beforeEach(async ({ page }) => {
    await mockUnauthenticated(page)
    await setupCommonMocks(page)
  })

  test('visitor lands on home, sees stats, and navigates to courses', async ({
    page,
  }) => {
    await page.goto('/')
    await expect(page).toHaveTitle(/StuHelper/i)

    // StuHelper brand visible in navbar
    await expect(
      page.getByRole('link', { name: /StuHelper/i }).first(),
    ).toBeVisible()

    // Navigate to course review community
    await page.goto('/courses')
    await page.waitForLoadState('networkidle')

    // Course names from the community hub mock data appear
    await expect(page.getByText('数据结构与算法')).toBeVisible({
      timeout: 10_000,
    })
    await expect(page.getByText('高等数学A')).toBeVisible()

    // Navigate to course catalog
    await page.goto('/courses/list')
    await page.waitForLoadState('networkidle')

    // Course names from mock data appear
    await expect(page.getByText('数据结构与算法')).toBeVisible({
      timeout: 10_000,
    })
    await expect(page.getByText('高等数学A')).toBeVisible()
  })

  test('visitor browses course list and clicks into course detail with reviews', async ({
    page,
  }) => {
    // Setup course detail mocks
    await page.route('**/api/v1/course/courses/101', (route) =>
      route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify({ success: true, data: courses[0] }),
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
              list: sampleReviews,
              total: 2,
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
              overall: 4.3,
              totalReviews: 2,
              dimensions: {
                recommendation: 4.5,
                content_quality: 4.5,
                workload: 2.5,
                grading: 3.5,
              },
            },
          }),
        }),
    )
    await page.route(
      '**/api/v1/course/review/courses/101/teachers*',
      (route) =>
        route.fulfill({
          contentType: 'application/json',
          body: JSON.stringify({
            success: true,
            data: [{ teacherID: 10, teacherName: '张教授', reviewCount: 2 }],
          }),
        }),
    )
    await page.route('**/api/v1/course/review/courses/101/rating-trend*', (route) =>
      route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify({ success: true, data: [] }),
      }),
    )

    // Navigate to course list and click the first course
    await page.goto('/courses/101/reviews')
    await page.waitForLoadState('networkidle')

    // Course name visible
    await expect(page.getByText('数据结构与算法').first()).toBeVisible({
      timeout: 10_000,
    })

    // Review titles rendered
    await expect(page.getByText('干货满满，强烈推荐')).toBeVisible()
    await expect(page.getByText('内容不错但作业偏多')).toBeVisible()

    // Review content (partial match)
    await expect(page.getByText(/课程内容设计非常好/).first()).toBeVisible()

    // Teacher chip visible
    await expect(page.getByText('张教授').first()).toBeVisible()
  })

  test('visitor views teacher profile page', async ({ page }) => {
    await page.route('**/api/v1/course/review/teachers/10/stats*', (route) =>
      route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: {
            teacherID: 10,
            teacherName: '张教授',
            departmentName: '计算机科学与技术学院',
            overallRating: 4.5,
            courseCount: 3,
            reviewCount: 28,
            courses: [
              { id: 101, name: '数据结构与算法', reviewCount: 23 },
            ],
            ratingTrend: [
              { termID: '2025-spring', termName: '2025 春', rating: 4.3 },
              { termID: '2025-fall', termName: '2025 秋', rating: 4.5 },
            ],
          },
        }),
      }),
    )

    await page.goto('/teachers/10')
    await page.waitForLoadState('networkidle')

    // Teacher name visible
    await expect(page.getByText('张教授').first()).toBeVisible({
      timeout: 10_000,
    })

    // Department visible
    await expect(
      page.getByText('计算机科学与技术学院').first(),
    ).toBeVisible()

    // Course in teacher's course list
    await expect(page.getByText('数据结构与算法').first()).toBeVisible()
  })
})
