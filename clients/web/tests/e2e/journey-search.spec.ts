/**
 * User Journey: Advanced search
 * Navigate to search → fill conditions → execute → view results → click through
 *
 * Simulates a user searching for courses and reviews.
 */
import { test, expect, type Page } from './fixtures'

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

test.describe('User Journey: Search', () => {
  test.beforeEach(async ({ page }) => {
    await mockUnauthenticated(page)

    await page.route('**/api/v1/course/departments*', (route) =>
      route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: [
            { id: 1, name: '计算机科学与技术学院' },
            { id: 2, name: '数学科学学院' },
          ],
        }),
      }),
    )

    await page.route('**/api/v1/course/terms*', (route) =>
      route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: [
            { id: '2025-fall', name: '2025 秋' },
            { id: '2025-spring', name: '2025 春' },
          ],
        }),
      }),
    )
  })

  test('user searches by course name and sees matching results', async ({
    page,
  }) => {
    await page.route('**/api/v1/course/courses/search*', (route) =>
      route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: {
            list: [
              {
                id: 101,
                name: '数据结构与算法',
                code: 'CS201',
                departmentID: 1,
                departmentName: '计算机科学与技术学院',
                reviewCount: 23,
              },
            ],
            total: 1,
          },
        }),
      }),
    )

    await page.route('**/api/v1/course/review/reviews/search*', (route) =>
      route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: {
            list: [
              {
                id: 'search-rev-1',
                courseID: 101,
                courseName: '数据结构与算法',
                title: '搜索命中的评价',
                content: '包含数据结构关键词的评价内容。',
                rating: 4,
                likeCount: 3,
                createdAt: '2026-03-20T10:00:00Z',
              },
            ],
            total: 1,
          },
        }),
      }),
    )

    await page.goto('/search')
    await page.waitForLoadState('networkidle')

    // The page renders SearchPage heading
    await expect(
      page.getByRole('heading', {
        level: 1,
        name: /Advanced Search|高级搜索/i,
      }),
    ).toBeVisible({ timeout: 10_000 })

    // Fill in the course name field by its label
    const courseNameInput = page
      .locator('label')
      .filter({ hasText: /Course Name|课程名|课程名称/i })
      .first()
      .locator('..')
      .locator('input')
      .first()
    await courseNameInput.fill('数据结构')

    // Click the search submit button (the one with "Search" or "搜索" text)
    const reviewsSearchResponse = page.waitForResponse(
      (resp) =>
        resp.url().includes('/api/v1/course/review/reviews/search') &&
        resp.status() === 200,
    )

    await page
      .getByRole('button', { name: /^Search$|^搜索$/ })
      .first()
      .click()

    // Wait for the search to complete and results view to render.
    await reviewsSearchResponse

    // The matching course name appears in results
    await expect(page.getByText('数据结构与算法').first()).toBeVisible({
      timeout: 10_000,
    })
  })

  test('search with no results shows empty state', async ({ page }) => {
    await page.route('**/api/v1/course/courses/search*', (route) =>
      route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: { list: [], total: 0 },
        }),
      }),
    )

    await page.route('**/api/v1/course/review/reviews/search*', (route) =>
      route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: { list: [], total: 0 },
        }),
      }),
    )

    await page.goto('/search')
    await page.waitForLoadState('networkidle')

    const courseNameInput = page
      .locator('label')
      .filter({ hasText: /Course Name|课程名|课程名称/i })
      .first()
      .locator('..')
      .locator('input')
      .first()
    await courseNameInput.fill('不存在的课程名xyz')

    const reviewsSearchResponse = page.waitForResponse(
      (resp) =>
        resp.url().includes('/api/v1/course/review/reviews/search') &&
        resp.status() === 200,
    )

    await page
      .getByRole('button', { name: /^Search$|^搜索$/ })
      .first()
      .click()

    await reviewsSearchResponse

    // Should show "no results" empty state from review.search.noResults i18n
    await expect(
      page.getByText(/No results found|未找到任何符合条件/i).first(),
    ).toBeVisible({ timeout: 10_000 })
  })
})
