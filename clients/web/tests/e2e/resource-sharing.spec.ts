import {
  allowExpectedCriticalResourceFailure,
  mockNotificationStream,
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

const authenticatedUser = {
  id: 'seed-user',
  name: 'seed',
  displayName: 'Seed User',
  email: 'seed@example.com',
  roles: ['verified_student'],
  capabilities: [],
  globalCapabilities: [],
  capabilityGrants: [],
  isPlatformAdmin: false,
  canAccessAdmin: false,
}

function ok(data: unknown = null) {
  return {
    contentType: 'application/json',
    body: JSON.stringify({ success: true, data }),
  }
}

function requireSubmittedPayload(
  payload: Record<string, unknown> | null,
): Record<string, unknown> {
  expect(payload).not.toBeNull()
  return payload as Record<string, unknown>
}

async function mockAuthenticated(page: Page) {
  await page.unroute('**/api/v1/auth/me')
  await page.addInitScript((user) => {
    localStorage.setItem('stuhelper_user', JSON.stringify(user))
    localStorage.setItem(
      'stuhelper_token_expiry',
      String(Date.now() + 60 * 60 * 1000),
    )
  }, authenticatedUser)

  await page.route('**/api/v1/auth/me', (route) =>
    route.fulfill(ok(authenticatedUser)),
  )
  await page.route('**/api/v1/auth/refresh', (route) =>
    route.fulfill(ok({ expiresIn: 3600 })),
  )
  await page.route(
    '**/api/v1/course/review/user/notifications/unread-count*',
    (route) => route.fulfill(ok({ count: 0 })),
  )
  await page.route('**/api/v1/user/identity', (route) =>
    route.fulfill(ok({
      userID: 1,
      realName: 'Seed User',
      verified: true,
      verifyMethod: 'manual',
      reviewedAt: '2026-05-01T08:00:00Z',
      verifiedAt: '2026-05-01T08:00:00Z',
      rejectionReason: null,
      createdAt: '2026-05-01T08:00:00Z',
      updatedAt: '2026-05-01T08:00:00Z',
    })),
  )
  await page.route('**/api/v1/user/profile', (route) =>
    route.fulfill(ok({
      userID: 1,
      schoolID: null,
      studentIDs: [],
      activeStudentID: null,
      verificationStatus: 'verified',
      verificationMethod: 'manual',
      rejectionReason: null,
      reviewedAt: '2026-05-01T08:00:00Z',
      phone: null,
      phoneVerified: false,
      consentGivenAt: null,
      verifiedAt: '2026-05-01T08:00:00Z',
      createdAt: '2026-05-01T08:00:00Z',
      updatedAt: '2026-05-01T08:00:00Z',
    })),
  )
  await page.route('**/api/v1/user/qq-binding', (route) =>
    route.fulfill({
      status: 404,
      contentType: 'application/json',
      body: JSON.stringify({
        success: false,
        error: { code: 'A0020001', message: 'QQ binding not found' },
      }),
    }),
  )
  await mockNotificationStream(page)
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

  test('resource authoring routes require sign-in', async ({ page }) => {
    await page.goto('/resources/new')
    await expect(page).toHaveURL((url) =>
      url.pathname === '/login' && url.searchParams.get('redirect') === '/resources/new',
    )

    await page.goto('/resources/mine')
    await expect(page).toHaveURL((url) =>
      url.pathname === '/login' && url.searchParams.get('redirect') === '/resources/mine',
    )

    await page.goto('/resources/42/edit')
    await expect(page).toHaveURL((url) =>
      url.pathname === '/login' && url.searchParams.get('redirect') === '/resources/42/edit',
    )
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
    await page.route('**/api/v1/resources/42', (route) =>
      route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify({ success: true, data: sampleResource }),
      }),
    )

    await page.goto('/')
    await openResourceListFromNavigation(page)

    await expect(page).toHaveURL(/\/resources$/)
    await expect(
      page.getByRole('heading', { name: '资料共享' }),
    ).toBeVisible()
    await expect(page.getByRole('link', { name: '发布资料' })).toBeVisible()
    await expect(page.getByRole('link', { name: '我的资料' })).toBeVisible()
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
    await expect(page).toHaveURL((url) =>
      url.pathname === '/resources' &&
      url.searchParams.get('query') === '高数' &&
      url.searchParams.get('tag') === '期末' &&
      url.searchParams.get('bindingType') === 'course' &&
      url.searchParams.get('bindingValue') === '8',
    )

    await page.getByRole('link', { name: /高等数学A 期末复习讲义/ }).click()
    await expect(page).toHaveURL((url) =>
      url.pathname === '/resources/42' &&
      url.searchParams.get('query') === '高数' &&
      url.searchParams.get('tag') === '期末' &&
      url.searchParams.get('bindingType') === 'course' &&
      url.searchParams.get('bindingValue') === '8',
    )
    await expect(page.getByRole('heading', { name: '高等数学A 期末复习讲义' })).toBeVisible()

    await page.getByRole('link', { name: '返回资料列表' }).click()
    await expect(page).toHaveURL((url) =>
      url.pathname === '/resources' &&
      url.searchParams.get('query') === '高数' &&
      url.searchParams.get('tag') === '期末' &&
      url.searchParams.get('bindingType') === 'course' &&
      url.searchParams.get('bindingValue') === '8',
    )
    await expect(page.getByRole('searchbox', { name: /搜索资料/ })).toHaveValue('高数')
    await expect(page.getByLabel('标签')).toHaveValue('期末')
    await expect(page.getByLabel('绑定类型')).toHaveValue('course')
    await expect(page.getByLabel('绑定值')).toHaveValue('8')
  })

  test('resource list restores filters from a shared URL', async ({
    page,
  }) => {
    let requestedURL = ''

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

    await page.goto('/resources?query=高数&tag=期末&bindingType=course&bindingValue=8')

    await expect(page.getByRole('heading', { name: '资料共享' })).toBeVisible()
    await expect(page.getByRole('searchbox', { name: /搜索资料/ })).toHaveValue('高数')
    await expect(page.getByLabel('标签')).toHaveValue('期末')
    await expect(page.getByLabel('绑定类型')).toHaveValue('course')
    await expect(page.getByLabel('绑定值')).toHaveValue('8')
    await expect(page.getByText('高等数学A 期末复习讲义')).toBeVisible()

    await expect
      .poll(() => new URL(requestedURL).searchParams.get('query'))
      .toBe('高数')
    const requestURL = new URL(requestedURL)
    expect(requestURL.searchParams.get('tag')).toBe('期末')
    expect(requestURL.searchParams.get('bindingType')).toBe('course')
    expect(requestURL.searchParams.get('bindingValue')).toBe('8')

    await page.getByRole('button', { name: '清空筛选' }).click()
    await expect(page).toHaveURL(/\/resources$/)
    await expect(page.getByRole('searchbox', { name: /搜索资料/ })).toHaveValue('')
    await expect(page.getByLabel('标签')).toHaveValue('')
    await expect(page.getByLabel('绑定类型')).toHaveValue('')
    await expect(page.getByLabel('绑定值')).toHaveValue('')
  })

  test('resource list ignores stale search failures after filters are cleared', async ({
    page,
  }) => {
    let releaseSearch!: () => void
    let markSearchRequested!: () => void
    const searchRequested = new Promise<void>((resolve) => {
      markSearchRequested = resolve
    })
    const searchRelease = new Promise<void>((resolve) => {
      releaseSearch = resolve
    })

    await page.route('**/api/v1/resources?*', async (route) => {
      const url = new URL(route.request().url())
      const query = url.searchParams.get('query')

      if (query === '旧搜索') {
        markSearchRequested()
        await searchRelease
        return route.fulfill({
          contentType: 'application/json',
          body: JSON.stringify({
            success: true,
            data: {
              items: [
                {
                  ...sampleResource,
                  latestVersion: {
                    ...sampleResource.latestVersion,
                    sizeBytes: '204800',
                  },
                },
              ],
              total: 1,
              page: 1,
              pageSize: 24,
            },
          }),
        })
      }

      return route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: { items: [sampleResource], total: 1, page: 1, pageSize: 24 },
        }),
      })
    })

    await page.goto('/resources')
    await expect(
      page.getByRole('link', { name: /高等数学A 期末复习讲义/ }),
    ).toBeVisible({ timeout: 10_000 })

    await page.getByRole('searchbox', { name: /搜索资料/ }).fill('旧搜索')
    await page.getByRole('button', { name: '搜索资料' }).click()
    await searchRequested

    await page.getByRole('button', { name: '清空筛选' }).click()
    await expect(
      page.getByRole('link', { name: /高等数学A 期末复习讲义/ }),
    ).toBeVisible({ timeout: 10_000 })

    releaseSearch()

    await expect(
      page.getByRole('link', { name: /高等数学A 期末复习讲义/ }),
    ).toBeVisible()
    await expect(page.getByRole('alert')).toHaveCount(0)
    await expect(page.getByText('资料列表加载失败，请稍后重试。')).toHaveCount(0)
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

  test('authenticated user can publish a resource', async ({ page }) => {
    await mockAuthenticated(page)

    const createdResource = {
      ...sampleResource,
      id: 99,
      title: '线性代数课堂笔记',
      description: '矩阵与线性空间。',
      category: '笔记',
      visibility: 'private',
      tags: ['线代', '笔记'],
      bindings: [{ type: 'course', value: '12' }],
      latestVersion: {
        ...sampleResource.latestVersion,
        id: 99,
        filename: 'linear-notes.txt',
        contentType: 'text/plain; charset=utf-8',
      },
    }
    let submittedPayload: Record<string, unknown> | null = null

    await page.route('**/api/v1/resources', async (route) => {
      if (route.request().method() !== 'POST') {
        return route.fulfill(ok({ items: [], total: 0, page: 1, pageSize: 24 }))
      }
      submittedPayload = route.request().postDataJSON() as Record<string, unknown>
      return route.fulfill({ status: 201, ...ok(createdResource) })
    })
    await page.route('**/api/v1/resources/99', (route) =>
      route.fulfill(ok(createdResource)),
    )

    await page.goto('/resources/new')

    await expect(page.getByRole('heading', { name: '发布资料' })).toBeVisible()
    await page.getByLabel('标题').fill('线性代数课堂笔记')
    await page.getByLabel('描述').fill('矩阵与线性空间。')
    await page.getByLabel('分类').fill('笔记')
    await page.getByLabel('可见性').selectOption('private')
    await page.getByLabel('标签').fill('线代, 笔记')
    await page.getByLabel('绑定关系').fill('course: 12')
    await page.getByLabel('文件').setInputFiles({
      name: 'linear-notes.txt',
      mimeType: 'text/plain',
      buffer: Buffer.from('linear algebra notes'),
    })
    await page.getByRole('button', { name: '发布' }).click()

    await expect(page).toHaveURL(/\/resources\/99$/)
    await expect
      .poll(() => submittedPayload?.title)
      .toBe('线性代数课堂笔记')
    const payload = requireSubmittedPayload(submittedPayload)
    expect(payload.visibility).toBe('private')
    expect(payload.tags).toEqual(['线代', '笔记'])
    expect(payload.bindings).toEqual([{ type: 'course', value: '12' }])
    expect(payload.filename).toBe('linear-notes.txt')
    expect(String(payload.dataBase64)).toContain('base64,')
  })

  test('my resources page includes private resources and persists filters', async ({
    page,
  }) => {
    await mockAuthenticated(page)

    const privateResource = {
      ...sampleResource,
      id: 43,
      title: '私有实验报告',
      visibility: 'private',
      latestVersion: {
        ...sampleResource.latestVersion,
        id: 43,
        filename: 'lab-report.pdf',
      },
    }
    let requestedURL = ''

    await page.route('**/api/v1/resources/mine?*', (route) => {
      requestedURL = route.request().url()
      return route.fulfill(ok({
        items: [sampleResource, privateResource],
        total: 2,
        page: 1,
        pageSize: 24,
      }))
    })

    await page.goto('/resources/mine')

    await expect(page.getByRole('heading', { name: '我的资料' })).toBeVisible()
    await expect(page.getByText('高等数学A 期末复习讲义')).toBeVisible()
    await expect(page.getByText('私有实验报告')).toBeVisible()
    await expect(page.getByText('私有').first()).toBeVisible()

    await page.getByRole('searchbox', { name: /搜索资料/ }).fill('实验')
    await page.getByLabel('可见性').selectOption('private')
    await page.getByRole('button', { name: '搜索资料' }).click()

    await expect
      .poll(() => new URL(requestedURL).searchParams.get('query'))
      .toBe('实验')
    expect(new URL(requestedURL).searchParams.get('visibility')).toBe('private')
    await expect(page).toHaveURL((url) =>
      url.pathname === '/resources/mine' &&
      url.searchParams.get('query') === '实验' &&
      url.searchParams.get('visibility') === 'private',
    )
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
    await expect(page).toHaveTitle(/高等数学A 期末复习讲义 - StuHelper/)
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

  test('authenticated owner can edit resource metadata', async ({ page }) => {
    await mockAuthenticated(page)

    let currentResource = { ...sampleResource }
    let submittedPayload: Record<string, unknown> | null = null

    await page.route('**/api/v1/resources/42', (route) => {
      if (route.request().method() === 'PATCH') {
        submittedPayload = route.request().postDataJSON() as Record<string, unknown>
        currentResource = {
          ...currentResource,
          title: String(submittedPayload.title),
          description: String(submittedPayload.description),
          category: String(submittedPayload.category),
          visibility: submittedPayload.visibility as 'public' | 'private',
          tags: submittedPayload.tags as string[],
          bindings: submittedPayload.bindings as typeof sampleResource.bindings,
        }
        return route.fulfill(ok(currentResource))
      }
      return route.fulfill(ok(currentResource))
    })

    await page.goto('/resources/42/edit')

    await expect(page.getByRole('heading', { name: '编辑资料' })).toBeVisible()
    await expect(page.getByLabel('标题')).toHaveValue('高等数学A 期末复习讲义')
    await page.getByLabel('标题').fill('高等数学A 期末复习讲义 v2')
    await page.getByLabel('描述').fill('更新后的复习重点。')
    await page.getByLabel('分类').fill('复习资料')
    await page.getByLabel('可见性').selectOption('private')
    await page.getByLabel('标签').fill('高数, 期末, 重点')
    await page.getByLabel('绑定关系').fill('course: 8')
    await page.getByRole('button', { name: '保存修改' }).click()

    await expect(page).toHaveURL(/\/resources\/42$/)
    await expect
      .poll(() => submittedPayload?.title)
      .toBe('高等数学A 期末复习讲义 v2')
    const payload = requireSubmittedPayload(submittedPayload)
    expect(payload.visibility).toBe('private')
    expect(payload.tags).toEqual(['高数', '期末', '重点'])
    await expect(
      page.getByRole('heading', { name: '高等数学A 期末复习讲义 v2' }),
    ).toBeVisible()
  })

  test('authenticated owner can delete a resource from detail', async ({ page }) => {
    await mockAuthenticated(page)

    let deleted = false
    await page.route('**/api/v1/resources/42', (route) => {
      if (route.request().method() === 'DELETE') {
        deleted = true
        return route.fulfill(ok({ message: 'resource deleted' }))
      }
      return route.fulfill(ok(sampleResource))
    })
    await page.route('**/api/v1/resources/mine?*', (route) =>
      route.fulfill(ok({ items: [], total: 0, page: 1, pageSize: 24 })),
    )
    page.on('dialog', (dialog) => dialog.accept())

    await page.goto('/resources/42')

    await expect(
      page.getByRole('heading', { name: '高等数学A 期末复习讲义' }),
    ).toBeVisible()
    await expect(page.getByRole('link', { name: '编辑' })).toBeVisible()
    await page.getByRole('button', { name: '删除' }).click()

    await expect.poll(() => deleted).toBe(true)
    await expect(page).toHaveURL(/\/resources\/mine$/)
  })

  test('resource detail preserves int64 path ids when loading and downloading', async ({
    page,
  }) => {
    const bigResourceID = '9007199254740993'
    const bigResource = {
      ...sampleResource,
      title: '资源 ID 精度测试资料',
      latestVersion: {
        ...sampleResource.latestVersion,
        filename: 'big-resource.pdf',
      },
    }
    const detailRequests: string[] = []
    const downloadURLRequests: string[] = []
    const unexpectedResourceRequests: string[] = []
    let downloadDocumentRequests = 0

    allowExpectedCriticalResourceFailure(
      page,
      /\/resource-downloads\/big-resource\.pdf$/,
    )
    await page.route('**/api/v1/resources/**', (route) => {
      const url = new URL(route.request().url())
      if (url.pathname === `/api/v1/resources/${bigResourceID}`) {
        detailRequests.push(url.pathname)
        return route.fulfill({
          contentType: 'application/json',
          body: JSON.stringify({ success: true, data: bigResource }),
        })
      }
      if (url.pathname === `/api/v1/resources/${bigResourceID}/download-url`) {
        downloadURLRequests.push(url.pathname)
        return route.fulfill({
          contentType: 'application/json',
          body: JSON.stringify({
            success: true,
            data: { url: '/resource-downloads/big-resource.pdf' },
          }),
        })
      }
      unexpectedResourceRequests.push(url.pathname)
      return route.fulfill({
        status: 404,
        contentType: 'application/json',
        body: JSON.stringify({
          success: false,
          error: { code: 'NOT_FOUND', message: 'not found' },
        }),
      })
    })
    await page.route('**/resource-downloads/big-resource.pdf', (route) => {
      downloadDocumentRequests += 1
      return route.fulfill({
        contentType: 'application/pdf',
        body: 'demo',
      })
    })

    await page.goto(`/resources/${bigResourceID}`)

    await expect(
      page.getByRole('heading', { name: '资源 ID 精度测试资料' }),
    ).toBeVisible()
    await expect.poll(() => detailRequests).toEqual([
      `/api/v1/resources/${bigResourceID}`,
    ])

    const downloadRequest = page.waitForRequest(
      '**/resource-downloads/big-resource.pdf',
    )
    await page.getByRole('button', { name: '下载资料' }).click()
    await downloadRequest

    await expect.poll(() => downloadURLRequests).toEqual([
      `/api/v1/resources/${bigResourceID}/download-url`,
    ])
    await expect.poll(() => downloadDocumentRequests).toBe(1)
    expect(unexpectedResourceRequests).toEqual([])
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
    await expect(page).toHaveTitle(/线性代数习题集 - StuHelper/)
    await expect(page.getByText('linear-algebra.pdf')).toBeVisible()
    await expect(page.getByText('math-final.pdf')).toHaveCount(0)
    await expect.poll(() => firstResourceRequests).toBe(1)
    await expect.poll(() => nextResourceRequests).toBe(1)
  })

  test('resource detail ignores stale download URLs after navigating to another resource', async ({
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
    let releaseDownload!: () => void
    let markDownloadRequested!: () => void
    const downloadRequested = new Promise<void>((resolve) => {
      markDownloadRequested = resolve
    })
    const downloadRelease = new Promise<void>((resolve) => {
      releaseDownload = resolve
    })
    let staleDownloadDocumentRequests = 0

    await page.route('**/api/v1/resources/42', (route) =>
      route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify({ success: true, data: sampleResource }),
      }),
    )
    await page.route('**/api/v1/resources/43', (route) =>
      route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify({ success: true, data: nextResource }),
      }),
    )
    await page.route('**/api/v1/resources/42/download-url', async (route) => {
      markDownloadRequested()
      await downloadRelease
      return route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: { url: '/resource-downloads/math-final.pdf' },
        }),
      })
    })
    await page.route('**/resource-downloads/math-final.pdf', (route) => {
      staleDownloadDocumentRequests += 1
      return route.fulfill({
        contentType: 'application/pdf',
        body: 'stale',
      })
    })

    await page.goto('/resources/42')
    await expect(
      page.getByRole('heading', { name: '高等数学A 期末复习讲义' }),
    ).toBeVisible()

    await page.getByRole('button', { name: '下载资料' }).click()
    await downloadRequested

    await page.evaluate(() => {
      window.history.pushState(null, '', '/resources/43')
      window.dispatchEvent(new PopStateEvent('popstate'))
    })
    await expect(
      page.getByRole('heading', { name: '线性代数习题集' }),
    ).toBeVisible()

    releaseDownload()
    await page.waitForTimeout(500)

    await expect(page).toHaveURL(/\/resources\/43$/)
    await expect(page.getByText('linear-algebra.pdf')).toBeVisible()
    await expect(page.getByText('math-final.pdf')).toHaveCount(0)
    await expect.poll(() => staleDownloadDocumentRequests).toBe(0)
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
