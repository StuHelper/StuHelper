import { test, expect, type Page } from '@playwright/test'

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
  await page.route(
    '**/api/v1/course/review/user/notifications/unread-count*',
    (route) =>
      route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify({ success: true, data: { count: 0 } }),
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

  test('authenticated user is redirected away from login', async ({ page }) => {
    await mockAuthenticated(page, BASIC_USER)
    await page.goto('/login')
    await expect(page).toHaveURL('/')
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
    await expect(page.getByRole('button', { name: /Login with SSO|使用 SSO 登录/ })).toBeVisible()

    await Promise.all([
      page.waitForURL(/http:\/\/localhost:8085\/login\/oauth\/authorize/),
      page.getByRole('button', { name: /Login with SSO|使用 SSO 登录/ }).click(),
    ])

    expect(loginRequestURL).not.toBeNull()
    expect(loginRequestURL!.searchParams.get('app')).toBe('web')
    expect(loginRequestURL!.searchParams.get('redirect')).toBe('http://127.0.0.1:3000/user/reviews')
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
