/**
 * User Journey: Advanced search
 * Navigate to search → fill conditions → execute → view results → click through
 *
 * Simulates a user searching for courses and reviews.
 */
import { test, expect, type Page, type Route } from './fixtures'

const webApiRequests: string[] = []

function recordApiRequest(route: Route) {
  const request = route.request()
  const url = new URL(request.url())
  webApiRequests.push(`${request.method()} ${url.pathname}${url.search}`)
}

function hasWebGetRequest(pathname: string, matches: (url: URL) => boolean) {
  return webApiRequests.some((request) => {
    if (!request.startsWith('GET ')) return false
    const url = new URL(request.slice('GET '.length), 'http://web.e2e')
    return url.pathname === pathname && matches(url)
  })
}

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
    webApiRequests.length = 0
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

  test('invalid reference data shows load failure and retry restores filters', async ({
    page,
  }) => {
    let departmentsRequestCount = 0
    let termsRequestCount = 0

    await page.route('**/api/v1/course/departments*', (route) => {
      departmentsRequestCount += 1
      return route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify(
          departmentsRequestCount === 1
            ? { success: true, data: null }
            : {
                success: true,
                data: [{ id: 1, name: '计算机科学与技术学院' }],
              },
        ),
      })
    })

    await page.route('**/api/v1/course/terms*', (route) => {
      termsRequestCount += 1
      return route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify(
          termsRequestCount === 1
            ? { success: true, data: null }
            : {
                success: true,
                data: [{ id: '2025-fall', name: '2025 秋' }],
              },
        ),
      })
    })

    await page.goto('/search')
    await page.waitForLoadState('networkidle')

    await expect(
      page.getByRole('alert').filter({ hasText: /Load failed|加载失败/i }).first(),
    ).toBeVisible({ timeout: 10_000 })
    await expect(page.locator('#advanced-department')).not.toContainText(
      '计算机科学与技术学院',
    )
    await expect(page.locator('#advanced-term')).not.toContainText('2025 秋')

    await page.getByRole('button', { name: /Retry|重试/i }).click()

    await expect(page.locator('#advanced-department')).toContainText(
      '计算机科学与技术学院',
    )
    await expect(page.locator('#advanced-term')).toContainText('2025 秋')
  })

  test('user searches by course name and sees matching results', async ({
    page,
  }) => {
    await page.route('**/api/v1/course/courses/search*', (route) => {
      recordApiRequest(route)
      return route.fulfill({
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
      })
    })

    await page.route('**/api/v1/course/review/reviews/search*', (route) => {
      recordApiRequest(route)
      return route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: {
            list: [
              {
                id: 'search-rev-1',
                courseID: 101,
                courseName: '数据结构与算法',
                termID: '2026-spring',
                title: '搜索命中的评价',
                content: '包含数据结构关键词的评价内容。',
                ratings: { recommendation: 4 },
                likeCount: 3,
                dislikeCount: 0,
                replyCount: 0,
                status: 'published',
                createdAt: '2026-03-20T10:00:00Z',
              },
            ],
            total: 1,
          },
        }),
      })
    })

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
    const coursesSearchResponse = page.waitForResponse(
      (resp) =>
        resp.url().includes('/api/v1/course/courses/search') &&
        resp.status() === 200,
    )
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
    await Promise.all([coursesSearchResponse, reviewsSearchResponse])

    await expect
      .poll(() =>
        hasWebGetRequest('/api/v1/course/courses/search', (url) =>
          (
            url.searchParams.get('q') === '数据结构' &&
            url.searchParams.get('pageSize') === '50'
          ),
        ),
      )
      .toBe(true)
    await expect
      .poll(() =>
        hasWebGetRequest('/api/v1/course/review/reviews/search', (url) =>
          (
            url.searchParams.get('q') === '数据结构' &&
            url.searchParams.get('pageSize') === '50' &&
            url.searchParams.get('sort') === 'time'
          ),
        ),
      )
      .toBe(true)

    // The matching course name appears in results
    await expect(page.getByText('数据结构与算法').first()).toBeVisible({
      timeout: 10_000,
    })
  })

  test('user searches by department, teacher, and term query params', async ({
    page,
  }) => {
    await page.route('**/api/v1/course/courses?*', (route) => {
      recordApiRequest(route)
      return route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: {
            list: [
              {
                id: 102,
                name: '编译原理',
                code: 'CS301',
                departmentID: 1,
                departmentName: '计算机科学与技术学院',
                reviewCount: 8,
              },
            ],
            total: 1,
          },
        }),
      })
    })

    await page.route('**/api/v1/course/review/reviews/search*', (route) => {
      recordApiRequest(route)
      return route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: {
            list: [
              {
                id: 'search-rev-teacher-term',
                courseID: 102,
                courseName: '编译原理',
                teacherName: '张教授',
                termID: '2025-fall',
                title: '按老师和学期命中的评价',
                content: '这个筛选组合能找到指定老师在指定学期的评价。',
                ratings: { recommendation: 5 },
                likeCount: 4,
                dislikeCount: 0,
                replyCount: 0,
                status: 'published',
                createdAt: '2026-03-22T10:00:00Z',
              },
            ],
            total: 1,
          },
        }),
      })
    })

    await page.goto('/search')
    await page.waitForLoadState('networkidle')

    await page.locator('#advanced-department').selectOption('1')
    await page.locator('#advanced-teacher-name').fill('张教授')
    await page.locator('#advanced-term').selectOption('2025-fall')

    const coursesResponse = page.waitForResponse(
      (resp) =>
        resp.url().includes('/api/v1/course/courses?') &&
        resp.status() === 200,
    )
    const reviewsResponse = page.waitForResponse(
      (resp) =>
        resp.url().includes('/api/v1/course/review/reviews/search') &&
        resp.status() === 200,
    )

    await page
      .getByRole('button', { name: /^Search$|^搜索$/ })
      .first()
      .click()

    await Promise.all([coursesResponse, reviewsResponse])

    await expect
      .poll(() =>
        hasWebGetRequest('/api/v1/course/courses', (url) =>
          (
            url.searchParams.get('departmentID') === '1' &&
            url.searchParams.get('pageSize') === '50'
          ),
        ),
      )
      .toBe(true)
    await expect
      .poll(() =>
        hasWebGetRequest('/api/v1/course/review/reviews/search', (url) =>
          (
            url.searchParams.get('departmentID') === '1' &&
            url.searchParams.get('teacherName') === '张教授' &&
            url.searchParams.get('termID') === '2025-fall' &&
            url.searchParams.get('pageSize') === '50' &&
            url.searchParams.get('sort') === 'time'
          ),
        ),
      )
      .toBe(true)

    await expect(page.getByText('编译原理').first()).toBeVisible({
      timeout: 10_000,
    })
    await expect(page.getByText('按老师和学期命中的评价')).toBeVisible()
  })

  test('search with no results shows empty state', async ({ page }) => {
    await page.route('**/api/v1/course/courses/search*', (route) => {
      recordApiRequest(route)
      return route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: { list: [], total: 0 },
        }),
      })
    })

    await page.route('**/api/v1/course/review/reviews/search*', (route) => {
      recordApiRequest(route)
      return route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: { list: [], total: 0 },
        }),
      })
    })

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

  test('malformed course search response shows load failure instead of empty state', async ({
    page,
  }) => {
    await page.route('**/api/v1/course/courses/search*', (route) => {
      recordApiRequest(route)
      return route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: null,
        }),
      })
    })

    await page.route('**/api/v1/course/review/reviews/search*', (route) => {
      recordApiRequest(route)
      return route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: { list: [], total: 0 },
        }),
      })
    })

    await page.goto('/search')
    await page.waitForLoadState('networkidle')

    const courseNameInput = page
      .locator('label')
      .filter({ hasText: /Course Name|课程名|课程名称/i })
      .first()
      .locator('..')
      .locator('input')
      .first()
    await courseNameInput.fill('畸形课程响应')

    const coursesSearchResponse = page.waitForResponse(
      (resp) =>
        resp.url().includes('/api/v1/course/courses/search') &&
        resp.status() === 200,
    )
    const reviewsSearchResponse = page.waitForResponse(
      (resp) =>
        resp.url().includes('/api/v1/course/review/reviews/search') &&
        resp.status() === 200,
    )

    await page
      .getByRole('button', { name: /^Search$|^搜索$/ })
      .first()
      .click()

    await Promise.all([coursesSearchResponse, reviewsSearchResponse])

    await expect(
      page.getByRole('alert').filter({ hasText: /Load failed|加载失败/i }).first(),
    ).toBeVisible({ timeout: 10_000 })
    await expect(
      page.getByText(/No results found|未找到任何符合条件/i),
    ).toHaveCount(0)
  })
})
