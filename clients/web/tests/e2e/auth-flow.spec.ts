import { test, expect, mockNotificationStream, type Page } from './fixtures'

// ---- Auth mock helpers (inlined to avoid dual @playwright/test resolution) ----

const BASIC_USER = {
  id: 'u1',
  name: 'alice',
  displayName: 'Alice',
  email: 'alice@example.com',
  roles: ['user'],
  capabilities: ['review:list:brief'],
  globalCapabilities: ['review:list:brief'],
  capabilityGrants: [],
  isPlatformAdmin: false,
  canAccessAdmin: false,
}

const VERIFIED_STUDENT = {
  id: 'u2',
  name: 'bob',
  displayName: 'Bob',
  email: 'bob@example.com',
  roles: ['verified_student'],
  capabilities: [
    'review:list:full',
    'review:create',
    'review:edit:own',
    'review:delete:own',
  ],
  globalCapabilities: [
    'review:list:full',
    'review:create',
    'review:edit:own',
    'review:delete:own',
  ],
  capabilityGrants: [],
  isPlatformAdmin: false,
  canAccessAdmin: false,
}

async function mockAuthenticated(
  page: Page,
  user: typeof BASIC_USER = BASIC_USER,
) {
  await page.addInitScript((u) => {
    localStorage.setItem('stuhelper_user', JSON.stringify(u))
    localStorage.setItem(
      'stuhelper_token_expiry',
      String(Date.now() + 60 * 60 * 1000),
    )
  }, user)

  await page.route('**/api/v1/auth/me', (route) =>
    route.fulfill({
      contentType: 'application/json',
      body: JSON.stringify({ success: true, data: user }),
    }),
  )
  await page.route('**/api/v1/auth/refresh', (route) =>
    route.fulfill({
      contentType: 'application/json',
      body: JSON.stringify({ success: true, data: { expiresIn: 3600 } }),
    }),
  )
  await page.route('**/api/v1/user/identity', (route) =>
    route.fulfill({
      contentType: 'application/json',
      body: JSON.stringify({
        success: true,
        data: { verified: user.roles.includes('verified_student'), status: 'verified' },
      }),
    }),
  )
  await page.route('**/api/v1/user/profile', (route) =>
    route.fulfill({
      contentType: 'application/json',
      body: JSON.stringify({
        success: true,
        data: {
          verificationStatus: user.roles.includes('verified_student')
            ? 'verified'
            : 'unverified',
          schoolName: user.roles.includes('verified_student') ? '测试大学' : null,
          schoolID: user.roles.includes('verified_student') ? 1 : null,
        },
      }),
    }),
  )
  await page.route('**/api/v1/user/qq-binding', (route) =>
    route.fulfill({
      status: 404,
      contentType: 'application/json',
      body: JSON.stringify({
        success: false,
        error: { code: 'A0040404', message: 'not bound' },
      }),
    }),
  )
  await page.route(
    '**/api/v1/course/review/user/notifications/unread-count*',
    (route) =>
      route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify({ success: true, data: { count: 0 } }),
      }),
  )
  await mockNotificationStream(page)
  await page.route('**/api/v1/course/review/user/votes*', (route) =>
    route.fulfill({
      contentType: 'application/json',
      body: JSON.stringify({
        success: true,
        data: { list: [], total: 0, page: 1, pageSize: 10 },
      }),
    }),
  )
  await page.route('**/api/v1/course/review/user/favorites*', (route) =>
    route.fulfill({
      contentType: 'application/json',
      body: JSON.stringify({
        success: true,
        data: { list: [], total: 0, page: 1, pageSize: 10 },
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
        data: {
          courseCount: 120,
          reviewCount: 580,
          departmentCount: 8,
          userCount: 230,
        },
      }),
    }),
  )
  await page.route('**/api/v1/course/review/rankings/hot*', (route) =>
    route.fulfill({
      contentType: 'application/json',
      body: JSON.stringify({ success: true, data: { list: [] } }),
    }),
  )
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

async function mockSSOAuth(page: Page) {
  await mockUnauthenticated(page)

  await page.route('**/api/v1/auth/login*', (route) =>
    route.fulfill({
      contentType: 'application/json',
      body: JSON.stringify({
        success: true,
        data: {
          url: 'http://localhost:8085/login/oauth/authorize?client_id=stuhelper-web&state=sso-state',
          state: 'sso-state',
        },
      }),
    }),
  )

  await page.route('http://localhost:8085/**', (route) =>
    route.fulfill({
      contentType: 'text/html',
      body: '<!doctype html><title>Mock SSO</title><main>Mock SSO</main>',
    }),
  )
}

// ---- Tests ----

test.describe('Auth Flow', () => {
  test('unauthenticated user sees login page', async ({ page }) => {
    await mockUnauthenticated(page)
    await page.goto('/login')
    await expect(page).toHaveURL(/\/login/)
  })

  test('auth-protected route redirects to login with redirect param', async ({
    page,
  }) => {
    await mockUnauthenticated(page)
    await page.goto('/user/reviews')
    await expect(page).toHaveURL(/\/login/)
    // Verify redirect query param preserves the original URL
    const url = new URL(page.url())
    expect(url.searchParams.get('redirect')).toBe('/user/reviews')
  })

  test('unauthenticated review post route redirects before loading drafts', async ({
    page,
  }) => {
    await mockUnauthenticated(page)
    let draftRequestCount = 0
    await page.route('**/api/v1/course/review/drafts**', (route) => {
      draftRequestCount += 1
      return route.fulfill({
        status: 500,
        contentType: 'application/json',
        body: JSON.stringify({
          success: false,
          error: { code: 'E0050000', message: 'unexpected draft access' },
        }),
      })
    })

    await page.goto('/courses/reviews/post?from=e2e')
    await expect(page).toHaveURL(/\/login/)
    const url = new URL(page.url())
    expect(url.searchParams.get('redirect')).toBe('/courses/reviews/post?from=e2e')
    await page.waitForLoadState('networkidle')
    expect(draftRequestCount).toBe(0)
  })

  test('authenticated user is redirected away from login', async ({ page }) => {
    await mockAuthenticated(page, BASIC_USER)
    await page.goto('/login')
    await expect(page).toHaveURL('/')
  })

  test('authenticated user can force SSO reauthentication from login route', async ({
    page,
  }) => {
    await mockAuthenticated(page, BASIC_USER)

    let loginRequestURL: URL | null = null
    await page.route('**/api/v1/auth/login*', (route) => {
      loginRequestURL = new URL(route.request().url())
      return route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: {
            url: 'http://localhost:8085/login/oauth/authorize?client_id=stuhelper-web&state=reauth-state',
            state: 'reauth-state',
          },
        }),
      })
    })
    await page.route('http://localhost:8085/**', (route) =>
      route.fulfill({
        contentType: 'text/html',
        body: '<!doctype html><title>Mock SSO Reauth</title><main>Mock SSO Reauth</main>',
      }),
    )

    await page.goto('/login?reauth=1&redirect=/user/reviews')
    const appOrigin = new URL(page.url()).origin
    await expect(page).toHaveURL(/\/login\?/)

    await Promise.all([
      page.waitForURL(/http:\/\/localhost:8085\/login\/oauth\/authorize/),
      page.getByRole('button', { name: /Login with SSO|使用 SSO 登录/ }).click(),
    ])

    expect(loginRequestURL).not.toBeNull()
    expect(loginRequestURL!.searchParams.get('app')).toBe('web')
    expect(loginRequestURL!.searchParams.get('prompt')).toBe('login')
    expect(loginRequestURL!.searchParams.get('max_age')).toBe('0')
    expect(loginRequestURL!.searchParams.get('redirect')).toBe(
      `${appOrigin}/user/reviews`,
    )
    const ssoURL = new URL(page.url())
    expect(ssoURL.origin).toBe('http://localhost:8085')
    await expect(page.getByText('Mock SSO Reauth')).toBeVisible()
  })

  test('login button starts SSO and preserves redirect target', async ({
    page,
  }) => {
    await mockSSOAuth(page)
    let loginRequestURL: URL | null = null
    await page.route('**/api/v1/auth/login*', (route) => {
      loginRequestURL = new URL(route.request().url())
      return route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: {
            url: 'http://localhost:8085/login/oauth/authorize?client_id=stuhelper-web&state=sso-state',
            state: 'sso-state',
          },
        }),
      })
    })

    await page.goto('/login?redirect=/user/reviews')
    const appOrigin = new URL(page.url()).origin
    await expect(page.getByRole('button', { name: /Login with SSO|使用 SSO 登录/ })).toBeVisible()

    await Promise.all([
      page.waitForURL(/http:\/\/localhost:8085\/login\/oauth\/authorize/),
      page.getByRole('button', { name: /Login with SSO|使用 SSO 登录/ }).click(),
    ])

    expect(loginRequestURL).not.toBeNull()
    expect(loginRequestURL!.searchParams.get('app')).toBe('web')
    expect(loginRequestURL!.searchParams.get('redirect')).toBe(`${appOrigin}/user/reviews`)
    const ssoURL = new URL(page.url())
    expect(ssoURL.origin).toBe('http://localhost:8085')
    expect(ssoURL.searchParams.get('client_id')).toBe('stuhelper-web')
    expect(ssoURL.searchParams.get('state')).toBe('sso-state')
  })

  test('authenticated user can access user center', async ({ page }) => {
    await mockAuthenticated(page, VERIFIED_STUDENT)

    await page.route('**/api/v1/course/review/user/reviews*', (route) =>
      route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: { list: [], total: 0, page: 1, pageSize: 20 },
        }),
      }),
    )

    await page.goto('/user/reviews')
    await expect(page).toHaveURL(/\/user\/reviews/)
    await page.waitForLoadState('networkidle')
    const main = page.locator('main, [role="main"], #app').first()
    await expect(main).toBeVisible()
  })

  test('user center shortcut redirects to the reviews tab', async ({ page }) => {
    await mockAuthenticated(page, VERIFIED_STUDENT)

    await page.route('**/api/v1/course/review/user/reviews*', (route) =>
      route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: { list: [], total: 0, page: 1, pageSize: 20 },
        }),
      }),
    )

    await page.goto('/user')
    await expect(page).toHaveURL(/\/user\/reviews/)
    await expect(
      page.locator('main, [role="main"], #app').first(),
    ).toBeVisible()
  })

  test('user without review creation capability is redirected away from post page', async ({
    page,
  }) => {
    await mockAuthenticated(page, BASIC_USER)

    await page.goto('/courses/reviews/post')
    await expect(page).toHaveURL('/')
    await expect(
      page.getByRole('button', { name: /发布评价|Submit Review|提交评价/ }),
    ).toHaveCount(0)
  })

  test('user center shows review data from mock', async ({ page }) => {
    await mockAuthenticated(page, VERIFIED_STUDENT)

    await page.route('**/api/v1/course/review/user/reviews*', (route) =>
      route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: {
            list: [
              {
                id: 'rev-e2e-1',
                courseID: 42,
                courseName: 'E2E Test Course',
                termID: '2026-spring',
                title: 'My E2E Review Title',
                content: 'Content body.',
                ratings: { recommendation: 5 },
                likeCount: 3,
                dislikeCount: 0,
                replyCount: 0,
                status: 'published',
                createdAt: '2026-04-01T00:00:00Z',
              },
            ],
            total: 1,
            page: 1,
            pageSize: 20,
          },
        }),
      }),
    )

    await page.goto('/user/reviews')
    await page.waitForLoadState('networkidle')
    await expect(page.getByText('My E2E Review Title')).toBeVisible({
      timeout: 10_000,
    })
  })
})
