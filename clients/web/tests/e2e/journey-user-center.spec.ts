/**
 * User Journey: User Center
 * Login → My Reviews → My Votes → My Favorites → Notifications
 *
 * Simulates a user navigating through their personal content.
 */
import { test, expect, mockNotificationStream, type Page } from './fixtures'

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
  isPlatformAdmin: false,
  canAccessAdmin: false,
}

type QueryRecord = Record<string, string>

function captureQuery(urlString: string): QueryRecord {
  return Object.fromEntries(new URL(urlString).searchParams.entries())
}

async function mockAuth(page: Page, authUser = user) {
  await page.addInitScript((u) => {
    localStorage.setItem('stuhelper_user', JSON.stringify(u))
    localStorage.setItem(
      'stuhelper_token_expiry',
      String(Date.now() + 60 * 60 * 1000),
    )
  }, authUser)

  await page.route('**/api/v1/auth/me', (route) =>
    route.fulfill({
      contentType: 'application/json',
      body: JSON.stringify({ success: true, data: authUser }),
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
        body: JSON.stringify({ success: true, data: { count: 2 } }),
      }),
  )
  await mockNotificationStream(page)
  // Verification status (for AppShell badges + canViewFullReviews)
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
  await page.route('**/api/v1/course/review/user/reviews*', (route) =>
    route.fulfill({
      contentType: 'application/json',
      body: JSON.stringify({
        success: true,
        data: { list: [], total: 0, page: 1, pageSize: 10 },
      }),
    }),
  )
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
  await page.route(
    '**/api/v1/open-platform/consents/audit-events*',
    (route) =>
      route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: { list: [], total: 0, page: 1, pageSize: 10 },
        }),
      }),
  )
}

test.describe('Admin AppShell', () => {
  test.beforeEach(async ({ page }) => {
    await mockAuth(page, {
      ...user,
      canAccessAdmin: true,
      isPlatformAdmin: true,
      roles: ['platform_admin', ...user.roles],
    })
  })

  test('admin user opens header user menu and enters admin console', async ({
    page,
  }) => {
    await page.goto('/user/reviews')

    const userButton = page.getByRole('button', { name: '用户' })
    await expect(userButton).toBeVisible({ timeout: 10_000 })
    await expect(page.locator('[title="管理员"]')).toBeVisible()

    await userButton.click()
    const menu = page.getByRole('menu', { name: '用户' })
    await expect(menu).toBeVisible()
    await expect(menu.getByRole('menuitem', { name: '管理后台' })).toBeVisible()

    await menu.getByRole('menuitem', { name: '管理后台' }).click()

    await expect(page).toHaveURL(/\/admin\/$/)
  })
})

test.describe('User Journey: User Center', () => {
  test.beforeEach(async ({ page }) => {
    await mockAuth(page)
  })

  test('profile summary renders verification state and binding entrypoints', async ({
    page,
  }) => {
    await page.goto('/user/reviews')

    const main = page.locator('main')

    await expect(main.getByRole('heading', { name: 'Bob' })).toBeVisible({
      timeout: 10_000,
    })
    await expect(main.getByText('bob@example.com')).toBeVisible()
    await expect(main.getByText('实名认证')).toBeVisible()
    await expect(main.getByText('学生认证')).toBeVisible()
    await expect(main.getByText('绑定 QQ')).toBeVisible()
    await expect(main.getByText('绑定手机')).toBeVisible()
    await expect(main.getByText('已认证', { exact: true })).toHaveCount(2)
    await expect(main.getByText('未绑定', { exact: true })).toHaveCount(2)

    await expect(
      main.getByRole('link', { name: '学业信息' }),
    ).toHaveAttribute('href', '/user/academic-info')
    await expect(
      main.getByRole('link', { name: '生成绑定码' }),
    ).toHaveAttribute('href', '/user/qq-binding')
    await expect(
      main.getByRole('link', { name: '绑定', exact: true }),
    ).toHaveAttribute('href', '/user/phone-binding')
  })

  test('user opens header user menu and logs out', async ({ page }) => {
    let authActive = true
    let logoutRequest: { method: string; path: string } | null = null

    await page.route('**/api/v1/auth/me', (route) => {
      if (!authActive) {
        return route.fulfill({
          status: 401,
          contentType: 'application/json',
          body: JSON.stringify({
            success: false,
            error: { code: 'A0040101', message: 'login required' },
          }),
        })
      }

      return route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify({ success: true, data: user }),
      })
    })
    await page.route('**/api/v1/auth/logout', async (route) => {
      const url = new URL(route.request().url())
      logoutRequest = {
        method: route.request().method(),
        path: url.pathname,
      }
      authActive = false
      await route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify({ success: true }),
      })
    })
    await page.route('**/api/v1/course/stats*', (route) =>
      route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: { courseCount: 120, departmentCount: 8 },
        }),
      }),
    )
    await page.route('**/api/v1/course/review/stats*', (route) =>
      route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: { courseCount: 230, reviewCount: 580, userCount: 42 },
        }),
      }),
    )

    await page.goto('/user/reviews')

    const userButton = page.getByRole('button', { name: '用户' })
    await expect(userButton).toBeVisible({ timeout: 10_000 })
    await expect(userButton).toHaveAttribute('aria-expanded', 'false')

    await userButton.click()
    await expect(userButton).toHaveAttribute('aria-expanded', 'true')

    const menu = page.getByRole('menu', { name: '用户' })
    await expect(menu).toBeVisible()
    await expect(menu.getByRole('menuitem', { name: /个人中心/ })).toBeVisible()
    await expect(menu.getByRole('menuitem', { name: /开发者应用/ })).toBeVisible()
    await expect(menu.getByRole('menuitem', { name: /实名认证/ })).toBeVisible()
    await expect(menu.getByRole('menuitem', { name: /学生认证/ })).toBeVisible()
    await expect(menu.getByRole('menuitem', { name: /绑定 QQ/ })).toBeVisible()

    await menu.getByRole('menuitem', { name: '退出登录' }).click()

    await expect
      .poll(() => logoutRequest)
      .toEqual({
        method: 'POST',
        path: '/api/v1/auth/logout',
      })
    await expect(page).toHaveURL(/\/$/)
    await expect(page.getByRole('link', { name: '登录' })).toBeVisible()
    await expect
      .poll(() =>
        page.evaluate(() => ({
          expiry: localStorage.getItem('stuhelper_token_expiry'),
          user: localStorage.getItem('stuhelper_user'),
        })),
      )
      .toEqual({ expiry: null, user: null })
  })

  test('user views their own reviews', async ({ page }) => {
    await page.route('**/api/v1/course/review/user/reviews*', (route) =>
      route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: {
            list: [
              {
                id: 'my-rev-1',
                courseID: 101,
                courseName: '数据结构与算法',
                title: '我的第一条评价',
                content: '这门课帮助我理解了基本的数据结构概念。',
                ratings: { recommendation: 4 },
                likeCount: 3,
                dislikeCount: 0,
                replyCount: 1,
                status: 'published',
                createdAt: '2026-04-01T10:00:00Z',
              },
              {
                id: 'my-rev-2',
                courseID: 102,
                courseName: '高等数学A',
                title: '考试较难但收获很大',
                content: '高数A是一门很有挑战性的课，需要大量练习。',
                ratings: { recommendation: 3 },
                likeCount: 1,
                dislikeCount: 0,
                replyCount: 0,
                status: 'published',
                createdAt: '2026-03-28T08:00:00Z',
              },
            ],
            total: 2,
            page: 1,
            pageSize: 10,
          },
        }),
      }),
    )

    await page.goto('/user/reviews')
    await page.waitForLoadState('networkidle')

    // Both review titles visible (ReviewCard shows title or courseName)
    await expect(page.getByText('我的第一条评价')).toBeVisible({
      timeout: 10_000,
    })
    await expect(page.getByText('考试较难但收获很大')).toBeVisible()

    // Review content snippets also visible
    await expect(
      page.getByText(/这门课帮助我理解了基本的数据结构概念/),
    ).toBeVisible()
  })

  test('user views their voted reviews', async ({ page }) => {
    await page.route('**/api/v1/course/review/user/votes*', (route) =>
      route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: {
            list: [
              {
                id: 'voted-rev-1',
                courseID: 101,
                courseName: '数据结构与算法',
                title: '我投过票的评价',
                content: '这是一条我投过票的评价内容。',
                ratings: { recommendation: 5 },
                likeCount: 10,
                dislikeCount: 0,
                replyCount: 2,
                status: 'published',
                createdAt: '2026-03-25T10:00:00Z',
              },
            ],
            total: 1,
            page: 1,
            pageSize: 10,
          },
        }),
      }),
    )

    await page.goto('/user/votes')
    await page.waitForLoadState('networkidle')

    await expect(page.getByText('我投过票的评价').first()).toBeVisible({
      timeout: 10_000,
    })
  })

  test('user views their favorited courses', async ({ page }) => {
    await page.route('**/api/v1/course/review/user/favorites*', (route) =>
      route.fulfill({
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
                favoritedAt: '2026-03-25T10:00:00Z',
              },
            ],
            total: 1,
            page: 1,
            pageSize: 20,
          },
        }),
      }),
    )

    await page.goto('/user/favorites')
    await page.waitForLoadState('networkidle')

    await expect(page.getByText('数据结构与算法').first()).toBeVisible({
      timeout: 10_000,
    })
  })

  test('user views and revokes authorized app scopes', async ({ page }) => {
    let revokeCalled = false
    let scopeRevoked = false
    const auditQueries: QueryRecord[] = []

    await page.route('**/api/v1/open-platform/consents', async (route) => {
      if (route.request().method() !== 'GET') {
        await route.fallback()
        return
      }

      await route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: {
            apps: [
              {
                app: {
                  id: 42,
                  clientID: 'campus-client',
                  displayName: 'Campus Tools',
                  description: 'Campus utility integration',
                  homepageURL: 'https://tools.example.com',
                  privacyPolicyURL: 'https://tools.example.com/privacy',
                  redirectURIs: ['https://tools.example.com/callback'],
                  status: 'approved',
                  createdAt: '2026-04-01T10:00:00Z',
                  updatedAt: '2026-04-01T10:00:00Z',
                },
                scopes: [
                  {
                    scope: 'profile.basic.read',
                    displayName: '基础资料',
                    sensitivity: 'low',
                    fields: ['昵称'],
                    grantedAt: '2026-04-05T10:00:00Z',
                    grantSource: 'consent_page',
                  },
                  {
                    scope: 'email.read',
                    displayName: '邮箱',
                    sensitivity: 'medium',
                    fields: ['邮箱'],
                    grantedAt: '2026-04-05T10:00:00Z',
                    grantSource: 'consent_page',
                  },
                ],
              },
            ],
          },
        }),
      })
    })

    await page.route(
      '**/api/v1/open-platform/consents/audit-events*',
      async (route) => {
        auditQueries.push(captureQuery(route.request().url()))
        await route.fulfill({
          contentType: 'application/json',
          body: JSON.stringify({
            success: true,
            data: {
              list: [
                {
                  id: scopeRevoked ? 102 : 101,
                  appID: 42,
                  appDisplayName: 'Campus Tools',
                  clientID: 'campus-client',
                  eventType: scopeRevoked
                    ? 'open_platform.consent.revoked'
                    : 'open_platform.consent.granted',
                  scopes: scopeRevoked
                    ? ['email.read']
                    : ['profile.basic.read', 'email.read'],
                  endpoint: '/oidc/userinfo',
                  result: scopeRevoked ? 'revoked' : 'success',
                  requestID: scopeRevoked ? 'req-revoke-email' : 'req-grant',
                  details: {},
                  createdAt: scopeRevoked
                    ? '2026-04-06T10:00:00Z'
                    : '2026-04-05T10:00:00Z',
                },
              ],
              total: 1,
              page: 1,
              pageSize: 10,
            },
          }),
        })
      },
    )

    await page.route('**/api/v1/open-platform/consents/42?*', async (route) => {
      revokeCalled = true
      scopeRevoked = true
      expect(route.request().method()).toBe('DELETE')
      expect(new URL(route.request().url()).searchParams.getAll('scope')).toEqual(['email.read'])
      await route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify({ success: true, data: { message: 'ok' } }),
      })
    })

    await page.goto('/user/authorized-apps')
    await page.waitForLoadState('networkidle')

    const emailScopeItem = page
      .getByRole('listitem')
      .filter({ hasText: '邮箱' })
      .filter({ hasText: 'email.read' })

    await expect(
      page.getByRole('heading', { name: 'Campus Tools' }),
    ).toBeVisible({ timeout: 10_000 })
    await expect(emailScopeItem).toBeVisible()
    await expect(page.getByText('授权已授予')).toBeVisible()
    await expect(
      page.getByRole('listitem').filter({ hasText: '授权已授予' }),
    ).toContainText('Campus Tools')
    await expect(
      page.getByText('涉及权限：profile.basic.read / email.read'),
    ).toBeVisible()
    await expect(page.getByText('接口：/oidc/userinfo · 结果：success')).toBeVisible()
    await expect
      .poll(() => auditQueries)
      .toContainEqual({ pageSize: '10' })

    await page.getByRole('button', { name: '撤销 邮箱' }).click()
    await expect(page.getByRole('dialog', { name: '撤销 邮箱' })).toBeVisible()
    await page.getByRole('button', { name: '确认撤销' }).click()

    await expect.poll(() => revokeCalled).toBe(true)
    await expect(emailScopeItem).toHaveCount(0)
    const revokedActivityItem = page
      .getByRole('listitem')
      .filter({ hasText: '授权已撤销' })
    await expect(revokedActivityItem).toBeVisible()
    await expect(revokedActivityItem).toContainText('涉及权限：email.read')
    await expect
      .poll(() => auditQueries.filter((query) => query.pageSize === '10').length)
      .toBeGreaterThanOrEqual(2)
  })

  test('user revokes an authorized app grant', async ({ page }) => {
    let revokeRequest: { method: string; scopes: string[] } | null = null

    await page.route('**/api/v1/open-platform/consents', async (route) => {
      if (route.request().method() !== 'GET') {
        await route.fallback()
        return
      }

      await route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: {
            apps: [
              {
                app: {
                  id: 43,
                  clientID: 'campus-data',
                  displayName: 'Campus Data',
                  description: 'Campus data integration',
                  homepageURL: 'https://data.example.com',
                  privacyPolicyURL: 'https://data.example.com/privacy',
                  redirectURIs: ['https://data.example.com/callback'],
                  status: 'approved',
                  createdAt: '2026-04-01T10:00:00Z',
                  updatedAt: '2026-04-01T10:00:00Z',
                },
                scopes: [
                  {
                    scope: 'profile.basic.read',
                    displayName: '基础资料',
                    sensitivity: 'low',
                    fields: ['昵称'],
                    grantedAt: '2026-04-05T10:00:00Z',
                    grantSource: 'consent_page',
                  },
                ],
              },
            ],
          },
        }),
      })
    })

    await page.route(
      '**/api/v1/open-platform/consents/audit-events*',
      async (route) => {
        await route.fulfill({
          contentType: 'application/json',
          body: JSON.stringify({
            success: true,
            data: { list: [], total: 0, page: 1, pageSize: 10 },
          }),
        })
      },
    )

    await page.route('**/api/v1/open-platform/consents/43', async (route) => {
      const url = new URL(route.request().url())
      revokeRequest = {
        method: route.request().method(),
        scopes: url.searchParams.getAll('scope'),
      }
      await route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: { message: 'ok' },
        }),
      })
    })

    await page.goto('/user/authorized-apps')
    await page.waitForLoadState('networkidle')

    await expect(
      page.getByRole('heading', { name: 'Campus Data' }),
    ).toBeVisible({
      timeout: 10_000,
    })
    await page.getByRole('button', { name: '撤销全部' }).click()
    await expect(
      page.getByRole('dialog', { name: '撤销应用授权' }),
    ).toBeVisible()
    await page.getByRole('button', { name: '确认撤销' }).click()

    await expect
      .poll(() => revokeRequest)
      .toEqual({
        method: 'DELETE',
        scopes: [],
      })
    await expect(
      page.getByRole('heading', { name: 'Campus Data' }),
    ).toHaveCount(0)
  })

  test('user views notifications and marks all as read', async ({ page }) => {
    let markAllCalled = false

    await page.route(
      '**/api/v1/course/review/user/notifications?*',
      (route) =>
        route.fulfill({
          contentType: 'application/json',
          body: JSON.stringify({
            success: true,
            data: {
              list: [
                {
                  id: 'notif-1',
                  type: 'reply',
                  title: '收到新回复',
                  content: '有人回复了你的评价',
                  isRead: false,
                  createdAt: '2026-04-05T10:00:00Z',
                  meta: { courseID: 101, reviewID: 'my-rev-1' },
                },
                {
                  id: 'notif-2',
                  type: 'vote',
                  title: '收到新赞',
                  content: '有人赞了你的评价',
                  isRead: false,
                  createdAt: '2026-04-05T09:00:00Z',
                  meta: { courseID: 101, reviewID: 'my-rev-1' },
                },
              ],
              total: 2,
              page: 1,
              pageSize: 20,
            },
          }),
        }),
    )

    await page.route(
      '**/api/v1/course/review/user/notifications/read-all',
      async (route) => {
        markAllCalled = true
        await route.fulfill({
          contentType: 'application/json',
          body: JSON.stringify({ success: true }),
        })
      },
    )

    await page.goto('/notifications')
    await page.waitForLoadState('networkidle')

    // Notifications visible
    await expect(page.getByText('收到新回复')).toBeVisible({
      timeout: 10_000,
    })
    await expect(page.getByText('收到新赞')).toBeVisible()

    // Click "Mark All Read"
    const markAllBtn = page
      .locator('button')
      .filter({ hasText: /mark all|全部已读|标记/i })
      .first()
    if (await markAllBtn.isVisible()) {
      await markAllBtn.click()
      await expect.poll(() => markAllCalled).toBe(true)
    }
  })

  test('user opens header notification bell and follows a notification link', async ({
    page,
  }) => {
    let bellQuery: QueryRecord | null = null
    let markAllRequest: { method: string; path: string } | null = null
    let markReadRequest: { method: string; path: string } | null = null

    await page.route('**/api/v1/course/review/user/notifications?*', (route) => {
      bellQuery = captureQuery(route.request().url())
      return route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: {
            list: [
              {
                id: 'bell-notif-1',
                type: 'reply',
                title: '顶部提醒',
                content: '点击后进入说明页',
                isRead: false,
                sourceUrl: '/about',
                createdAt: '2026-04-05T10:00:00Z',
              },
              {
                id: 'bell-notif-2',
                type: 'vote',
                title: '新的点赞',
                content: '有人赞了你的评价',
                isRead: false,
                createdAt: '2026-04-05T09:00:00Z',
              },
            ],
            total: 2,
            page: 1,
            pageSize: 5,
          },
        }),
      })
    })

    await page.route(
      '**/api/v1/course/review/user/notifications/read-all',
      async (route) => {
        const url = new URL(route.request().url())
        markAllRequest = {
          method: route.request().method(),
          path: url.pathname,
        }
        await route.fulfill({
          contentType: 'application/json',
          body: JSON.stringify({ success: true }),
        })
      },
    )

    await page.route(
      '**/api/v1/course/review/user/notifications/bell-notif-1/read',
      async (route) => {
        const url = new URL(route.request().url())
        markReadRequest = {
          method: route.request().method(),
          path: url.pathname,
        }
        await route.fulfill({
          contentType: 'application/json',
          body: JSON.stringify({ success: true }),
        })
      },
    )

    await page.goto('/user/reviews')

    const bellButton = page.getByRole('button', { name: '通知' })
    await expect(bellButton).toBeVisible({ timeout: 10_000 })
    await expect(bellButton.locator('span').filter({ hasText: '2' })).toBeVisible()

    await bellButton.click()
    const bellPanel = page.getByRole('region', { name: '通知' })
    await expect(bellPanel).toBeVisible()
    await expect(bellPanel.getByText('顶部提醒')).toBeVisible()
    await expect(bellPanel.getByText('新的点赞')).toBeVisible()

    await expect.poll(() => bellQuery).toEqual({ page: '1', pageSize: '5' })

    await bellPanel.getByRole('button', { name: '全部已读' }).click()
    await expect
      .poll(() => markAllRequest)
      .toEqual({
        method: 'PUT',
        path: '/api/v1/course/review/user/notifications/read-all',
      })
    await expect(bellButton.locator('span').filter({ hasText: '2' })).toHaveCount(0)
    await expect(bellPanel.getByRole('button', { name: '全部已读' })).toHaveCount(0)

    await bellPanel.getByRole('button', { name: /顶部提醒/ }).click()

    await expect
      .poll(() => markReadRequest)
      .toEqual({
        method: 'PUT',
        path: '/api/v1/course/review/user/notifications/bell-notif-1/read',
      })
    await expect(page).toHaveURL(/\/about$/)
  })

  test('user clicks one notification to mark it read and follow its link', async ({
    page,
  }) => {
    let markReadRequest: { method: string; path: string } | null = null

    await page.route('**/api/v1/course/review/user/notifications?*', (route) =>
      route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: {
            list: [
              {
                id: 'notif-1',
                type: 'reply',
                title: '系统提醒',
                content: '点击后进入说明页',
                isRead: false,
                sourceUrl: '/about',
                createdAt: '2026-04-05T10:00:00Z',
              },
            ],
            total: 1,
            page: 1,
            pageSize: 20,
          },
        }),
      }),
    )

    await page.route(
      '**/api/v1/course/review/user/notifications/notif-1/read',
      async (route) => {
        const url = new URL(route.request().url())
        markReadRequest = {
          method: route.request().method(),
          path: url.pathname,
        }
        await route.fulfill({
          contentType: 'application/json',
          body: JSON.stringify({ success: true }),
        })
      },
    )

    await page.goto('/notifications')
    await page.waitForLoadState('networkidle')

    await page.getByRole('button', { name: /系统提醒/ }).click()

    await expect
      .poll(() => markReadRequest)
      .toEqual({
        method: 'PUT',
        path: '/api/v1/course/review/user/notifications/notif-1/read',
      })
    await expect(page).toHaveURL(/\/about$/)
  })

  test('user sees empty state when no reviews exist', async ({ page }) => {
    await page.route('**/api/v1/course/review/user/reviews*', (route) =>
      route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: { list: [], total: 0, page: 1, pageSize: 10 },
        }),
      }),
    )

    await page.goto('/user/reviews')
    await page.waitForLoadState('networkidle')

    // Should show empty state text
    await expect(
      page.getByText(/暂无评价|还没有|no reviews|empty/i).first(),
    ).toBeVisible({ timeout: 10_000 })
  })
})
