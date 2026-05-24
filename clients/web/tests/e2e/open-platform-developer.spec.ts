import { expect, test, type Page } from '@playwright/test'

const user = {
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

async function mockAuth(page: Page) {
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
  await page.route('**/api/v1/user/identity', (route) =>
    route.fulfill({
      contentType: 'application/json',
      body: JSON.stringify({
        success: true,
        data: { verified: true, status: 'verified' },
      }),
    }),
  )
  await page.route('**/api/v1/user/profile', (route) =>
    route.fulfill({
      contentType: 'application/json',
      body: JSON.stringify({
        success: true,
        data: {
          verificationStatus: 'verified',
          schoolName: '测试大学',
          schoolID: 1,
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
}

test.describe('Open Platform developer portal', () => {
  test.beforeEach(async ({ page }) => {
    await mockAuth(page)
  })

  test('developer lists and submits apps', async ({ page }) => {
    let appSubmitted = false
    let secretRotated = false
    let redirectChangeSubmitted = false
    let submittedBody: unknown = null
    let redirectChangeBody: unknown = null

    type MockRedirectURIRequest = {
      id: number
      redirectURIs: string[]
      reason: string
      status: string
      reviewerUserID: number | null
      reviewedAt: string | null
      decisionNote: string | null
      createdAt: string
      updatedAt: string
    }
    type MockAppListItem = {
      app: Record<string, unknown>
      scopes: Array<Record<string, unknown>>
      redirectURIRequests: MockRedirectURIRequest[]
    }

    const appList: MockAppListItem[] = [
      {
        app: {
          id: 7,
          clientID: 'op_existing',
          displayName: 'Campus Connector',
          description: 'Connect campus services',
          homepageURL: 'https://connector.example.com',
          privacyPolicyURL: 'https://connector.example.com/privacy',
          redirectURIs: ['https://connector.example.com/callback'],
          status: 'approved',
          createdAt: '2026-04-01T10:00:00Z',
          updatedAt: '2026-04-02T10:00:00Z',
        },
        scopes: [
          {
            id: 1,
            scope: 'profile.basic.read',
            displayName: '用户基本信息',
            sensitivity: 'low',
            fields: ['用户名', '用户昵称', '头像地址'],
            reason: '显示登录用户',
            status: 'approved',
            reviewerUserID: null,
            reviewedAt: null,
            decisionNote: null,
            createdAt: '2026-04-01T10:00:00Z',
            updatedAt: '2026-04-02T10:00:00Z',
          },
        ],
        redirectURIRequests: [],
      },
    ]

    await page.route('**/api/v1/open-platform/apps/*/secret/rotate', async (route) => {
      secretRotated = true
      expect(route.request().method()).toBe('POST')
      expect(route.request().postDataJSON()).toEqual({
        reason: '例行轮换',
      })
      await route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: {
            app: appList[0].app,
            clientSecret: 'ids_rotated_secret',
          },
        }),
      })
    })

    await page.route('**/api/v1/open-platform/apps/*/redirect-uris', async (route) => {
      redirectChangeSubmitted = true
      expect(route.request().method()).toBe('POST')
      redirectChangeBody = route.request().postDataJSON()
      appList[0].redirectURIRequests = [
        {
          id: 9,
          redirectURIs: ['https://connector.example.com/oauth/callback'],
          reason: '迁移 OAuth 回调',
          status: 'pending',
          reviewerUserID: null,
          reviewedAt: null,
          decisionNote: null,
          createdAt: '2026-05-02T10:00:00Z',
          updatedAt: '2026-05-02T10:00:00Z',
        },
      ]
      await route.fulfill({
        status: 201,
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: appList[0].redirectURIRequests[0],
        }),
      })
    })

    await page.route('**/api/v1/open-platform/apps*', async (route) => {
      if (route.request().method() === 'POST') {
        appSubmitted = true
        submittedBody = route.request().postDataJSON()
        appList.unshift({
          app: {
            id: 8,
            clientID: 'op_new',
            displayName: 'Library Sync',
            description: 'Sync library notices',
            homepageURL: 'https://library.example.com',
            privacyPolicyURL: 'https://library.example.com/privacy',
            redirectURIs: ['https://library.example.com/callback'],
            status: 'pending',
            createdAt: '2026-05-01T10:00:00Z',
            updatedAt: '2026-05-01T10:00:00Z',
          },
          scopes: [
            {
              id: 2,
              scope: 'profile.basic.read',
              displayName: '用户基本信息',
              sensitivity: 'low',
              fields: ['用户名', '用户昵称', '头像地址'],
              reason: '用于显示登录用户',
              status: 'pending',
              reviewerUserID: null,
              reviewedAt: null,
              decisionNote: null,
              createdAt: '2026-05-01T10:00:00Z',
              updatedAt: '2026-05-01T10:00:00Z',
            },
            {
              id: 3,
              scope: 'email.read',
              displayName: '邮箱',
              sensitivity: 'medium',
              fields: ['邮箱地址'],
              reason: '用于发送借阅通知',
              status: 'pending',
              reviewerUserID: null,
              reviewedAt: null,
              decisionNote: null,
              createdAt: '2026-05-01T10:00:00Z',
              updatedAt: '2026-05-01T10:00:00Z',
            },
          ],
          redirectURIRequests: [],
        })
        await route.fulfill({
          status: 201,
          contentType: 'application/json',
          body: JSON.stringify({ success: true, data: { app: appList[0].app } }),
        })
        return
      }

      await route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: {
            list: appList,
            total: appList.length,
          },
        }),
      })
    })

    await page.goto('/developers/apps')
    await page.waitForLoadState('networkidle')

    const existingApp = page.locator('article').filter({ hasText: 'Campus Connector' }).first()
    await expect(existingApp).toBeVisible({ timeout: 10_000 })
    await expect(existingApp).toContainText('op_existing')

    await existingApp.getByRole('button', { name: '轮换密钥' }).click()
    await expect(page.getByRole('dialog', { name: '轮换 client secret' })).toBeVisible()
    await page.getByLabel('操作原因').fill('例行轮换')
    await page.getByRole('button', { name: '确认轮换' }).click()
    await expect.poll(() => secretRotated).toBe(true)
    await expect(page.getByText('ids_rotated_secret')).toBeVisible()

    await existingApp.getByRole('button', { name: '变更回调地址' }).click()
    await page.getByLabel('新回调地址 1').fill('https://connector.example.com/oauth/callback')
    await page.getByLabel('回调地址变更原因').fill('迁移 OAuth 回调')
    await page.getByRole('button', { name: '提交变更' }).click()

    await expect.poll(() => redirectChangeSubmitted).toBe(true)
    expect(redirectChangeBody).toEqual({
      redirectURIs: ['https://connector.example.com/oauth/callback'],
      reason: '迁移 OAuth 回调',
    })
    await expect(page.getByText('https://connector.example.com/oauth/callback')).toBeVisible()

    await page.getByLabel('应用名称').fill('Library Sync')
    await page.getByLabel('应用说明').fill('Sync library notices')
    await page.getByLabel('主页 URL').fill('https://library.example.com')
    await page.getByLabel('隐私政策 URL').fill('https://library.example.com/privacy')
    await page.getByLabel('回调地址 1').fill('https://library.example.com/callback')
    await page.getByLabel('profile.basic.read 用途说明').fill('用于显示登录用户')
    await page.getByRole('checkbox', { name: /邮箱/ }).check()
    await page.getByLabel('email.read 用途说明').fill('用于发送借阅通知')
    await page.getByRole('button', { name: '提交审核' }).click()

    await expect.poll(() => appSubmitted).toBe(true)
    expect(submittedBody).toMatchObject({
      displayName: 'Library Sync',
      homepageURL: 'https://library.example.com',
      privacyPolicyURL: 'https://library.example.com/privacy',
      redirectURIs: ['https://library.example.com/callback'],
      scopes: [
        { scope: 'profile.basic.read', reason: '用于显示登录用户' },
        { scope: 'email.read', reason: '用于发送借阅通知' },
      ],
    })
    await expect(page.getByText('Library Sync')).toBeVisible()
  })
})
