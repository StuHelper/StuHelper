import {
  allowExpectedCriticalResourceFailure,
  allowExpectedConsoleError,
  test,
  expect,
  type Page,
} from './fixtures'

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

test.describe('Search Page', () => {
  test('search page loads with search UI', async ({ page }) => {
    await mockUnauthenticated(page)

    await page.route('**/api/v1/course/review/reviews/search*', (route) =>
      route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: { list: [], total: 0 },
        }),
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
    await page.route('**/api/v1/course/departments*', (route) =>
      route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify({ success: true, data: [] }),
      }),
    )

    await page.goto('/search')
    await expect(page).toHaveURL(/\/search/)
    await page.waitForLoadState('networkidle')
    // Search page should have a search input or search-related UI
    const main = page.locator('main, [role="main"], #app').first()
    await expect(main).toBeVisible()
  })
})

test.describe('Static Pages', () => {
  test('about page renders title and sections', async ({ page }) => {
    await mockUnauthenticated(page)
    await page.goto('/about')
    await expect(page).toHaveURL(/\/about/)
    // Assert i18n-rendered title: "About StuHelper" (en) or "关于 StuHelper" (zh)
    await expect(
      page.getByRole('heading', { name: /About StuHelper|关于/i }),
    ).toBeVisible({ timeout: 10_000 })
  })

  test('privacy page renders title', async ({ page }) => {
    await mockUnauthenticated(page)
    await page.goto('/privacy')
    await expect(page).toHaveURL(/\/privacy/)
    await expect(
      page.getByRole('heading', { name: /Privacy Policy|隐私/i }),
    ).toBeVisible({ timeout: 10_000 })
  })

  test('terms page renders title', async ({ page }) => {
    await mockUnauthenticated(page)
    await page.goto('/terms')
    await expect(page).toHaveURL(/\/terms/)
    await expect(
      page.getByRole('heading', { name: /Terms of Service|服务条款|使用条款/i }),
    ).toBeVisible({ timeout: 10_000 })
  })

  test('unknown route shows 404 heading and description', async ({ page }) => {
    await mockUnauthenticated(page)
    await page.goto('/this-route-does-not-exist')
    await page.waitForLoadState('networkidle')
    // NotFoundPage renders h1 with "Page Not Found" or "页面不存在"
    await expect(
      page.getByRole('heading', { name: /Page Not Found|页面不存在/i }),
    ).toBeVisible({ timeout: 10_000 })
    // Also check description text
    await expect(
      page.getByText(/removed|地址有误/i),
    ).toBeVisible()
  })

  test('chunk load failure retries once and renders static load error page', async ({ page }) => {
    await mockUnauthenticated(page)

    const searchPageChunkPattern =
      /\/src\/modules\/review\/views\/SearchPage\.vue(?:\?|$)/
    let failedChunkRequests = 0
    allowExpectedCriticalResourceFailure(page, searchPageChunkPattern)
    allowExpectedConsoleError(page, /^Failed to load resource: net::ERR_FAILED$/)
    allowExpectedConsoleError(
      page,
      /^\[App\] bootstrap failed: TypeError: Failed to fetch dynamically imported module: .*\/src\/modules\/review\/views\/SearchPage\.vue/,
    )
    await page.route('**/src/modules/review/views/SearchPage.vue*', (route) => {
      failedChunkRequests += 1
      return route.abort('failed')
    })

    await page.goto('/search')

    await expect(
      page.getByRole('heading', { name: /Load Failed|加载失败/i }),
    ).toBeVisible({ timeout: 10_000 })
    await expect(
      page.getByText(/Please refresh and try again|请刷新重试/i),
    ).toBeVisible()
    await expect.poll(() => failedChunkRequests).toBeGreaterThanOrEqual(2)
    await expect
      .poll(() =>
        page.evaluate(() =>
          sessionStorage.getItem('stuhelper_chunk_reload_attempted'),
        ),
      )
      .toBeNull()
  })
})
