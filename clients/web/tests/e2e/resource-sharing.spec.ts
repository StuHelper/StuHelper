import {
  allowExpectedCriticalResourceFailure,
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

const sampleResource = {
  id: 42,
  ownerUserID: 'seed-user',
  title: '高等数学A 期末复习讲义',
  description: '覆盖极限、导数和积分。',
  category: '讲义',
  visibility: 'public',
  tags: ['期末', '高数'],
  bindings: [{ type: 'course', value: '8' }],
  latestVersion: {
    id: 9,
    versionNo: 1,
    mountID: 1,
    objectKey: 'resources/demo/math-final.pdf',
    filename: 'math-final.pdf',
    contentType: 'application/pdf',
    sizeBytes: 204800,
    createdAt: '2026-05-01T08:00:00Z',
  },
  createdAt: '2026-05-01T08:00:00Z',
  updatedAt: '2026-05-02T08:00:00Z',
}

async function openResourceListFromNavigation(page: Page) {
  const primaryResourceLink = page.getByRole('link', { name: /^资源$/ }).first()
  try {
    await primaryResourceLink.click({ timeout: 2_000 })
    return
  } catch {
    await page.getByRole('button', { name: /菜单|Menu/i }).click()
    await page.getByRole('link', { name: /^资源$/ }).click()
  }
}

test.describe('Resource sharing', () => {
  test.beforeEach(async ({ page }) => {
    await mockUnauthenticated(page)
  })

  test('resource list can be reached from navigation and filters resources', async ({
    page,
  }) => {
    let requestedURL = ''

    await page.route('**/api/v1/course/stats', (route) =>
      route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: { courseCount: 2, departmentCount: 1 },
        }),
      }),
    )
    await page.route('**/api/v1/course/review/stats', (route) =>
      route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: {
            courseCount: 2,
            reviewCount: 3,
            departmentCount: 1,
            userCount: 4,
          },
        }),
      }),
    )
    await page.route('**/api/v1/resources?*', (route) => {
      requestedURL = route.request().url()
      return route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: { items: [sampleResource], total: 1, page: 1, pageSize: 24 },
        }),
      })
    })

    await page.goto('/')
    await openResourceListFromNavigation(page)

    await expect(page).toHaveURL(/\/resources$/)
    await expect(
      page.getByRole('heading', { name: '资料共享' }),
    ).toBeVisible()
    await expect(page.getByText('高等数学A 期末复习讲义')).toBeVisible()

    await page.getByRole('searchbox', { name: /搜索资料/ }).fill('高数')
    await page.getByLabel('标签').fill('期末')
    await page.getByLabel('绑定类型').fill('course')
    await page.getByLabel('绑定值').fill('8')
    await page.getByRole('button', { name: '搜索资料' }).click()

    await expect
      .poll(() => new URL(requestedURL).searchParams.get('query'))
      .toBe('高数')
    const requestURL = new URL(requestedURL)
    expect(requestURL.searchParams.get('tag')).toBe('期末')
    expect(requestURL.searchParams.get('bindingType')).toBe('course')
    expect(requestURL.searchParams.get('bindingValue')).toBe('8')

    await page.getByRole('link', { name: /高等数学A 期末复习讲义/ }).click()
    await expect(page).toHaveURL(/\/resources\/42$/)
  })

  test('resource list loads additional pages when more resources exist', async ({
    page,
  }) => {
    const requestedPages: string[] = []
    const makeResource = (id: number) => ({
      ...sampleResource,
      id,
      title: `资料 ${id}`,
      latestVersion: {
        ...sampleResource.latestVersion,
        id,
        filename: `resource-${id}.pdf`,
      },
    })

    await page.route('**/api/v1/resources?*', (route) => {
      const url = new URL(route.request().url())
      const requestedPage = url.searchParams.get('page') ?? '1'
      requestedPages.push(requestedPage)
      const items =
        requestedPage === '2'
          ? [makeResource(25)]
          : Array.from({ length: 24 }, (_, index) => makeResource(index + 1))
      return route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: { items, total: 25, page: Number(requestedPage), pageSize: 24 },
        }),
      })
    })

    await page.goto('/resources')

    await expect(page.getByRole('link', { name: /资料 1\b/ })).toBeVisible()
    await expect(page.getByRole('link', { name: /资料 25\b/ })).toHaveCount(0)
    await page.getByRole('button', { name: '加载更多' }).click()

    await expect(page.getByRole('link', { name: /资料 25\b/ })).toBeVisible()
    await expect.poll(() => requestedPages).toEqual(['1', '2'])
    await expect(page.getByRole('button', { name: '加载更多' })).toHaveCount(0)
  })

  test('resource detail displays metadata and opens download URL', async ({
    page,
  }) => {
    allowExpectedCriticalResourceFailure(
      page,
      /\/resource-downloads\/math-final\.pdf$/,
    )
    await page.route('**/api/v1/resources/42', (route) =>
      route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify({ success: true, data: sampleResource }),
      }),
    )
    await page.route('**/api/v1/resources/42/download-url', (route) =>
      route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: { url: '/resource-downloads/math-final.pdf' },
        }),
      }),
    )
    let downloadDocumentRequests = 0
    await page.route('**/resource-downloads/math-final.pdf', (route) => {
      downloadDocumentRequests += 1
      return route.fulfill({
        contentType: 'application/pdf',
        body: 'demo',
      })
    })

    await page.goto('/resources/42')

    await expect(
      page.getByRole('heading', { name: '高等数学A 期末复习讲义' }),
    ).toBeVisible()
    await expect(page.getByText('math-final.pdf')).toBeVisible()
    await expect(page.getByText('application/pdf')).toBeVisible()
    await expect(page.getByText('course: 8')).toBeVisible()

    const downloadRequest = page.waitForRequest(
      '**/resource-downloads/math-final.pdf',
    )
    await page.getByRole('button', { name: '下载资料' }).click()
    await downloadRequest
    await expect.poll(() => downloadDocumentRequests).toBe(1)
  })

  test('resource detail reloads when navigating between resource ids in the SPA', async ({
    page,
  }) => {
    const nextResource = {
      ...sampleResource,
      id: 43,
      title: '线性代数习题集',
      description: '矩阵、行列式与线性空间练习。',
      latestVersion: {
        ...sampleResource.latestVersion,
        id: 10,
        filename: 'linear-algebra.pdf',
        objectKey: 'resources/demo/linear-algebra.pdf',
      },
      updatedAt: '2026-05-03T08:00:00Z',
    }
    let firstResourceRequests = 0
    let nextResourceRequests = 0

    await page.route('**/api/v1/resources/42', (route) => {
      firstResourceRequests += 1
      return route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify({ success: true, data: sampleResource }),
      })
    })
    await page.route('**/api/v1/resources/43', (route) => {
      nextResourceRequests += 1
      return route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify({ success: true, data: nextResource }),
      })
    })

    await page.goto('/resources/42')

    await expect(
      page.getByRole('heading', { name: '高等数学A 期末复习讲义' }),
    ).toBeVisible()
    await expect(page.getByText('math-final.pdf')).toBeVisible()

    await page.evaluate(() => {
      window.history.pushState(null, '', '/resources/43')
      window.dispatchEvent(new PopStateEvent('popstate'))
    })

    await expect(page).toHaveURL(/\/resources\/43$/)
    await expect(
      page.getByRole('heading', { name: '线性代数习题集' }),
    ).toBeVisible()
    await expect(page.getByText('linear-algebra.pdf')).toBeVisible()
    await expect(page.getByText('math-final.pdf')).toHaveCount(0)
    await expect.poll(() => firstResourceRequests).toBe(1)
    await expect.poll(() => nextResourceRequests).toBe(1)
  })

  test('resource detail rejects unsafe download URLs before navigation', async ({
    page,
  }) => {
    await page.route('**/api/v1/resources/42', (route) =>
      route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify({ success: true, data: sampleResource }),
      }),
    )
    await page.route('**/api/v1/resources/42/download-url', (route) =>
      route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: { url: 'javascript:alert(1)' },
        }),
      }),
    )

    await page.goto('/resources/42')

    await expect(
      page.getByRole('heading', { name: '高等数学A 期末复习讲义' }),
    ).toBeVisible()
    const detailURL = page.url()

    await page.getByRole('button', { name: '下载资料' }).click()

    await expect(page.getByRole('alert')).toContainText('下载链接获取失败')
    await expect.poll(() => page.url()).toBe(detailURL)
  })
})
