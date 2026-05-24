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

test('home page renders shell and brand', async ({ page }) => {
  await mockUnauthenticated(page)

  await page.goto('/')

  await expect(page).toHaveTitle(/StuHelper/i)
  await expect(
    page.getByRole('link', { name: /StuHelper/i }).first(),
  ).toBeVisible()
})
