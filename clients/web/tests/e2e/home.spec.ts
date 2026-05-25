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

async function mockCourseDetail(page: Page, courseId: number) {
  await page.route(`**/api/v1/course/courses/${courseId}`, (route) =>
    route.fulfill({
      contentType: 'application/json',
      body: JSON.stringify({
        success: true,
        data: {
          id: courseId,
          name: '线性代数',
          code: 'MATH077',
          departmentName: '数学科学学院',
          credits: 3,
        },
      }),
    }),
  )

  await page.route(
    `**/api/v1/course/review/courses/${courseId}/reviews*`,
    (route) =>
      route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify({ success: true, data: { list: [], total: 0 } }),
      }),
  )

  await page.route(
    `**/api/v1/course/review/courses/${courseId}/rating-stats*`,
    (route) =>
      route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: { averageRating: 4.8, totalReviews: 12 },
        }),
      }),
  )

  await page.route(
    `**/api/v1/course/review/courses/${courseId}/teachers*`,
    (route) =>
      route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify({ success: true, data: [] }),
      }),
  )

  await page.route(
    `**/api/v1/course/review/courses/${courseId}/rating-trend*`,
    (route) =>
      route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify({ success: true, data: { trend: [] } }),
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

test('locale switcher updates rendered language and persists preference', async ({
  page,
}) => {
  await mockUnauthenticated(page)

  await page.goto('/')

  await expect(page.locator('html')).toHaveAttribute('lang', 'zh-CN')
  await expect(page.getByText('StuHelper 评课社区')).toBeVisible()

  await page.getByRole('button', { name: '切换到英文' }).click()

  await expect(page.locator('html')).toHaveAttribute('lang', 'en-US')
  await expect(
    page.getByText('StuHelper Course Review Community'),
  ).toBeVisible()
  await expect(
    page.getByRole('button', { name: 'Switch to Chinese' }),
  ).toBeVisible()
  await expect
    .poll(() => page.evaluate(() => localStorage.getItem('locale')))
    .toBe('en-US')

  await page.reload()

  await expect(page.locator('html')).toHaveAttribute('lang', 'en-US')
  await expect(
    page.getByText('StuHelper Course Review Community'),
  ).toBeVisible()

  await page.getByRole('button', { name: 'Switch to Chinese' }).click()

  await expect(page.locator('html')).toHaveAttribute('lang', 'zh-CN')
  await expect(page.getByText('StuHelper 评课社区')).toBeVisible()
  await expect
    .poll(() => page.evaluate(() => localStorage.getItem('locale')))
    .toBe('zh-CN')
})

test('command palette searches courses from keyboard and opens course detail', async ({
  page,
}) => {
  await mockUnauthenticated(page)
  await mockCourseDetail(page, 77)

  const searchRequests: URL[] = []
  await page.route('**/api/v1/course/courses/search*', (route) => {
    searchRequests.push(new URL(route.request().url()))
    return route.fulfill({
      contentType: 'application/json',
      body: JSON.stringify({
        success: true,
        data: {
          list: [
            {
              id: 77,
              name: '线性代数',
              code: 'MATH077',
              departmentName: '数学科学学院',
              credits: 3,
            },
          ],
          total: 1,
        },
      }),
    })
  })

  await page.goto('/')
  await expect(
    page.getByRole('link', { name: /StuHelper/i }).first(),
  ).toBeVisible()

  await page.keyboard.press('Control+K')
  const dialog = page.getByRole('dialog', {
    name: /搜索课程名称、教师|Search course name, teacher/i,
  })
  await expect(dialog).toBeVisible()

  await dialog.getByRole('combobox').fill('math')
  const result = dialog.getByRole('option', { name: /线性代数/ })
  await expect(result).toBeVisible({ timeout: 10_000 })

  const searchRequest = searchRequests.at(-1)
  expect(searchRequest?.searchParams.get('q')).toBe('math')
  expect(searchRequest?.searchParams.get('pageSize')).toBe('10')

  await result.click()

  await expect(page).toHaveURL(/\/courses\/77$/)
  await expect(page.getByText('线性代数').first()).toBeVisible({
    timeout: 10_000,
  })
  await expect(dialog).toBeHidden()
})
