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
  await page.route('**/api/v1/course/stats', (route) =>
    route.fulfill({
      contentType: 'application/json',
      body: JSON.stringify({
        success: true,
        data: { courseCount: 120, departmentCount: 8 },
      }),
    }),
  )
  await page.route('**/api/v1/course/review/stats', (route) =>
    route.fulfill({
      contentType: 'application/json',
      body: JSON.stringify({
        success: true,
        data: { reviewCount: 580, userCount: 230 },
      }),
    }),
  )
}

test('home page renders shell and brand', async ({ page }) => {
  await mockUnauthenticated(page)

  await page.goto('/')

  await expect(page).toHaveTitle(/StuHelper/i)
  await expect(
    page.getByRole('link', { name: /StuHelper/i }).first(),
  ).toBeVisible()
})
