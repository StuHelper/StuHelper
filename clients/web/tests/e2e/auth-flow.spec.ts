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

async function mockPhoneOtpLogin(page: Page) {
  await mockUnauthenticated(page)

  await page.route('**/api/v1/auth/phone/request-otp', (route) =>
    route.fulfill({
      contentType: 'application/json',
      body: JSON.stringify({
        success: true,
        data: {
          message: 'otp sent',
          cooldown: 60,
        },
      }),
    }),
  )

  await page.route('**/api/v1/auth/phone/verify-otp', (route) =>
    route.fulfill({
      contentType: 'application/json',
      body: JSON.stringify({
        success: true,
        data: {
          user: VERIFIED_STUDENT,
          expiresIn: 3600,
        },
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

  test('phone otp login renders six code boxes and auto-submits after completion', async ({
    page,
  }) => {
    await mockPhoneOtpLogin(page)
    await page.goto('/login')

    const phoneInput = page.getByPlaceholder('Enter phone number')
    await phoneInput.fill('13800138000')

    const sendButton = page.getByRole('button', { name: /Send Code|获取验证码/ })
    await expect(sendButton).toBeEnabled()
    await sendButton.click()

    const codeInputs = page.locator('div[role="group"] input')
    await expect(codeInputs).toHaveCount(6)

    for (let index = 0; index < 6; index += 1) {
      await codeInputs.nth(index).fill(String(index + 1))
    }

    await expect(page).toHaveURL('/')
  })

  test('phone otp login keeps send button beside phone input on small screens', async ({
    page,
  }) => {
    await page.setViewportSize({ width: 320, height: 812 })
    await mockPhoneOtpLogin(page)
    await page.goto('/login')

    const phoneInput = page.getByPlaceholder('Enter phone number')
    await phoneInput.fill('13800138000')

    const sendButton = page.getByRole('button', { name: /Send Code|获取验证码/ })
    await expect(sendButton).toBeVisible()

    const phoneBox = await phoneInput.boundingBox()
    const buttonBox = await sendButton.boundingBox()
    expect(phoneBox).not.toBeNull()
    expect(buttonBox).not.toBeNull()
    expect(Math.abs(phoneBox!.y - buttonBox!.y)).toBeLessThan(2)
    expect(buttonBox!.x).toBeGreaterThan(phoneBox!.x + phoneBox!.width)

    await sendButton.click()

    const codeInputs = page.locator('div[role="group"] input')
    await expect(codeInputs).toHaveCount(6)
    const firstCodeBox = await codeInputs.first().boundingBox()
    const lastCodeBox = await codeInputs.last().boundingBox()
    expect(firstCodeBox).not.toBeNull()
    expect(lastCodeBox).not.toBeNull()
    expect(firstCodeBox!.width).toBeGreaterThanOrEqual(36)
    expect(lastCodeBox!.x + lastCodeBox!.width).toBeLessThanOrEqual(320)
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
