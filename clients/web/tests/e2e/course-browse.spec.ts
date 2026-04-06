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

test.describe('Course Browse Flow', () => {
  test.beforeEach(async ({ page }) => {
    await mockUnauthenticated(page)

    await page.route('**/api/v1/course/departments*', (route) =>
      route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: [
            { id: 1, name: '计算机科学与技术学院', code: 'CS' },
            { id: 2, name: '数学科学学院', code: 'MATH' },
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
            { id: '2025-fall', name: '2025 秋', isCurrent: true },
            { id: '2025-spring', name: '2025 春', isCurrent: false },
          ],
        }),
      }),
    )

    await page.route('**/api/v1/course/categories*', (route) =>
      route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify({ success: true, data: [] }),
      }),
    )

    await page.route('**/api/v1/course/courses?*', (route) =>
      route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: {
            list: [
              {
                id: 1,
                name: '高等数学A',
                code: 'MATH101',
                departmentName: '数学科学学院',
                reviewCount: 15,
              },
              {
                id: 2,
                name: '数据结构',
                code: 'CS201',
                departmentName: '计算机科学与技术学院',
                reviewCount: 23,
              },
            ],
            total: 2,
          },
        }),
      }),
    )
  })

  test('course list page loads and shows course names', async ({ page }) => {
    await page.goto('/courses')
    await page.waitForLoadState('networkidle')

    // Assert actual rendered content from mocked data
    await expect(page.getByText('高等数学A')).toBeVisible({ timeout: 10_000 })
    await expect(page.getByText('数据结构')).toBeVisible()
  })

  test('course detail page loads and shows course info', async ({ page }) => {
    await page.route('**/api/v1/course/courses/1', (route) =>
      route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: {
            id: 1,
            name: '高等数学A',
            code: 'MATH101',
            departmentName: '数学科学学院',
            credits: 4,
          },
        }),
      }),
    )

    await page.route('**/api/v1/course/review/courses/1/reviews*', (route) =>
      route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify({ success: true, data: { list: [], total: 0 } }),
      }),
    )

    await page.route(
      '**/api/v1/course/review/courses/1/rating-stats*',
      (route) =>
        route.fulfill({
          contentType: 'application/json',
          body: JSON.stringify({
            success: true,
            data: { averageRating: 4.2, totalReviews: 15 },
          }),
        }),
    )

    await page.route(
      '**/api/v1/course/review/courses/1/teachers*',
      (route) =>
        route.fulfill({
          contentType: 'application/json',
          body: JSON.stringify({ success: true, data: [] }),
        }),
    )

    await page.goto('/courses/1')
    await page.waitForLoadState('networkidle')

    // Assert course name and department rendered from mocked data
    await expect(page.getByText('高等数学A')).toBeVisible({ timeout: 10_000 })
    await expect(page.getByText('数学科学学院')).toBeVisible()
  })
})
