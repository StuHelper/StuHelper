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

function ok(data: unknown = null) {
  return {
    contentType: 'application/json',
    body: JSON.stringify({ success: true, data }),
  }
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
        data: {
          userID: 12,
          docType: 'MAINLAND_ID',
          realName: '张三',
          verified: true,
          verifyMethod: 'manual',
          reviewedAt: '2026-05-24T04:00:00Z',
          verifiedAt: '2026-05-24T04:00:00Z',
          rejectionReason: null,
          createdAt: '2026-05-24T04:00:00Z',
          updatedAt: '2026-05-24T04:00:00Z',
        },
      }),
    }),
  )
  await page.route('**/api/v1/user/profile', (route) =>
    route.fulfill({
      contentType: 'application/json',
      body: JSON.stringify({
        success: true,
        data: {
          userID: 12,
          verificationStatus: 'verified',
          schoolID: 4111010006,
          studentIDs: ['20260001'],
          activeStudentID: '20260001',
          verificationMethod: 'manual',
          rejectionReason: null,
          reviewedAt: '2026-05-24T04:00:00Z',
          phone: null,
          phoneVerified: false,
          consentGivenAt: '2026-05-24T04:00:00Z',
          verifiedAt: '2026-05-24T04:00:00Z',
          createdAt: '2026-05-24T04:00:00Z',
          updatedAt: '2026-05-24T04:00:00Z',
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
    await expect(main.getByText('实名认证', { exact: true })).toBeVisible()
    await expect(main.getByText('学生认证', { exact: true })).toBeVisible()
    await expect(main.getByText('绑定 QQ', { exact: true })).toBeVisible()
    await expect(main.getByText('绑定手机', { exact: true })).toBeVisible()
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
          data: {
            courseCount: 230,
            reviewCount: 580,
            departmentCount: 8,
            userCount: 42,
          },
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
                termID: '2026-spring',
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
                termID: '2026-spring',
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
                termID: '2026-spring',
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
                credits: 3,
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

  test('invalid user reviews response fails closed and can retry', async ({
    page,
  }) => {
    let loadCount = 0

    await page.route('**/api/v1/course/review/user/reviews*', async (route) => {
      loadCount += 1
      await route.fulfill(
        loadCount === 1
          ? ok({
              list: [
                {
                  id: 'retry-review-1',
                  courseID: 101,
                  courseName: '数据结构与算法',
                  termID: '2026-spring',
                  title: '重试后的评价',
                  content: '畸形响应不应显示。',
                  ratings: { recommendation: 6 },
                  likeCount: 3,
                  dislikeCount: 0,
                  replyCount: 1,
                  status: 'published',
                  createdAt: '2026-04-01T10:00:00Z',
                },
              ],
              total: 1,
              page: 1,
              pageSize: 10,
            })
          : ok({
              list: [
                {
                  id: 'retry-review-1',
                  courseID: 101,
                  courseName: '数据结构与算法',
                  termID: '2026-spring',
                  title: '重试后的评价',
                  content: '异常响应重试后应显示真实评价。',
                  ratings: { recommendation: 4 },
                  likeCount: 3,
                  dislikeCount: 0,
                  replyCount: 1,
                  status: 'published',
                  createdAt: '2026-04-01T10:00:00Z',
                },
              ],
              total: 1,
              page: 1,
              pageSize: 10,
            }),
      )
    })

    await page.goto('/user/reviews')

    const status = page.getByRole('status').filter({ hasText: '加载失败' })
    await expect(status).toBeVisible({ timeout: 10_000 })
    await expect(page.getByText('暂无评价')).toHaveCount(0)

    await status.getByRole('button', { name: '重试' }).click()

    await expect.poll(() => loadCount).toBe(2)
    await expect(page.getByText('重试后的评价')).toBeVisible()
  })

  test('invalid user votes response fails closed and can retry', async ({
    page,
  }) => {
    let loadCount = 0

    await page.route('**/api/v1/course/review/user/votes*', async (route) => {
      loadCount += 1
      await route.fulfill(
        loadCount === 1
          ? ok({
              list: [
                {
                  id: 'retry-vote-1',
                  courseID: 101,
                  courseName: '数据结构与算法',
                  termID: '2026-spring',
                  title: '重试后的点赞评价',
                  content: '畸形响应不应显示。',
                  ratings: { recommendation: 5 },
                  likeCount: 10,
                  dislikeCount: 0,
                  replyCount: 2,
                  status: 'approved',
                  createdAt: '2026-03-25T10:00:00Z',
                },
              ],
              total: 1,
              page: 1,
              pageSize: 10,
            })
          : ok({
              list: [
                {
                  id: 'retry-vote-1',
                  courseID: 101,
                  courseName: '数据结构与算法',
                  termID: '2026-spring',
                  title: '重试后的点赞评价',
                  content: '异常响应重试后应显示真实点赞评价。',
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
            }),
      )
    })

    await page.goto('/user/votes')

    const status = page.getByRole('status').filter({ hasText: '加载失败' })
    await expect(status).toBeVisible({ timeout: 10_000 })
    await expect(page.getByText('暂无点赞')).toHaveCount(0)

    await status.getByRole('button', { name: '重试' }).click()

    await expect.poll(() => loadCount).toBe(2)
    await expect(page.getByText('重试后的点赞评价')).toBeVisible()
  })

  test('invalid user favorites response fails closed and can retry', async ({
    page,
  }) => {
    let loadCount = 0

    await page.route('**/api/v1/course/review/user/favorites*', async (route) => {
      loadCount += 1
      await route.fulfill(
        loadCount === 1
          ? ok({
              list: [
                {
                  id: 101,
                  name: '数据结构与算法',
                  code: 'CS201',
                  credits: '3',
                  departmentID: 1,
                  departmentName: '计算机科学与技术学院',
                  reviewCount: 23,
                  favoritedAt: '2026-03-25T10:00:00Z',
                },
              ],
              total: 1,
              page: 1,
              pageSize: 10,
            })
          : ok({
              list: [
                {
                  id: 101,
                  name: '数据结构与算法',
                  code: 'CS201',
                  credits: 3,
                  departmentID: 1,
                  departmentName: '计算机科学与技术学院',
                  reviewCount: 23,
                  favoritedAt: '2026-03-25T10:00:00Z',
                },
              ],
              total: 1,
              page: 1,
              pageSize: 10,
            }),
      )
    })

    await page.goto('/user/favorites')

    const status = page.getByRole('status').filter({ hasText: '加载失败' })
    await expect(status).toBeVisible({ timeout: 10_000 })
    await expect(page.getByText('暂无收藏')).toHaveCount(0)

    await status.getByRole('button', { name: '重试' }).click()

    await expect.poll(() => loadCount).toBe(2)
    await expect(page.getByText('数据结构与算法').first()).toBeVisible()
  })

  test('invalid authorized apps response fails closed and can retry', async ({
    page,
  }) => {
    let loadCount = 0

    await page.route('**/api/v1/open-platform/consents', async (route) => {
      if (route.request().method() !== 'GET') {
        await route.fallback()
        return
      }

      loadCount += 1
      await route.fulfill(
        loadCount === 1
          ? ok({
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
                    status: 'enabled',
                    createdAt: '2026-04-01T10:00:00Z',
                    updatedAt: '2026-04-01T10:00:00Z',
                  },
                  scopes: [],
                },
              ],
            })
          : ok({
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
                      reason: '展示基础资料',
                    },
                  ],
                },
              ],
            }),
      )
    })
    await page.route(
      '**/api/v1/open-platform/consents/audit-events*',
      (route) =>
        route.fulfill(ok({ list: [], total: 0, page: 1, pageSize: 10 })),
    )

    await page.goto('/user/authorized-apps')

    const status = page.getByRole('status').filter({ hasText: '加载失败' })
    await expect(status).toBeVisible({ timeout: 10_000 })
    await expect(page.getByText('暂无授权应用')).toHaveCount(0)

    await status.getByRole('button', { name: '重试' }).click()

    await expect.poll(() => loadCount).toBe(2)
    await expect(
      page.getByRole('heading', { name: 'Campus Tools' }),
    ).toBeVisible()
  })

  test('invalid authorized app activity response fails closed and can retry', async ({
    page,
  }) => {
    let activityLoadCount = 0

    await page.route('**/api/v1/open-platform/consents', async (route) => {
      if (route.request().method() !== 'GET') {
        await route.fallback()
        return
      }

      await route.fulfill(
        ok({
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
                  scope: 'email.read',
                  displayName: '邮箱',
                  sensitivity: 'medium',
                  fields: ['邮箱'],
                  grantedAt: '2026-04-05T10:00:00Z',
                  grantSource: 'consent_page',
                  reason: '发送邮箱通知',
                },
              ],
            },
          ],
        }),
      )
    })
    await page.route(
      '**/api/v1/open-platform/consents/audit-events*',
      async (route) => {
        activityLoadCount += 1
        await route.fulfill(
          activityLoadCount === 1
            ? ok({
                list: [
                  {
                    id: 101,
                    appID: 42,
                    appDisplayName: 'Campus Tools',
                    clientID: 'campus-client',
                    eventType: 'open_platform.consent.granted',
                    scopes: ['future.scope'],
                    endpoint: '/oidc/userinfo',
                    result: 'success',
                    requestID: 'req-grant',
                    details: {},
                    createdAt: '2026-04-05T10:00:00Z',
                  },
                ],
                total: 1,
              })
            : ok({
                list: [
                  {
                    id: 101,
                    appID: 42,
                    appDisplayName: 'Campus Tools',
                    clientID: 'campus-client',
                    eventType: 'open_platform.consent.granted',
                    scopes: ['email.read'],
                    endpoint: '/oidc/userinfo',
                    result: 'success',
                    requestID: 'req-grant',
                    details: {},
                    createdAt: '2026-04-05T10:00:00Z',
                  },
                ],
                total: 1,
                page: 1,
                pageSize: 10,
              }),
        )
      },
    )

    await page.goto('/user/authorized-apps')

    await expect(
      page.getByRole('heading', { name: 'Campus Tools' }),
    ).toBeVisible({ timeout: 10_000 })
    await expect(
      page.getByText('授权活动加载失败，请重试'),
    ).toBeVisible()
    await expect(page.getByText('暂无授权活动记录')).toHaveCount(0)

    await page.getByRole('button', { name: '重试' }).click()

    await expect.poll(() => activityLoadCount).toBe(2)
    await expect(page.getByText('授权已授予')).toBeVisible()
    await expect(page.getByText('涉及权限：email.read')).toBeVisible()
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
                    reason: '展示基础资料',
                  },
                  {
                    scope: 'email.read',
                    displayName: '邮箱',
                    sensitivity: 'medium',
                    fields: ['邮箱'],
                    grantedAt: '2026-04-05T10:00:00Z',
                    grantSource: 'consent_page',
                    reason: '发送邮箱通知',
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
                    reason: '展示基础资料',
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

  test('invalid notifications response fails closed and can retry', async ({
    page,
  }) => {
    let loadCount = 0

    await page.route(
      '**/api/v1/course/review/user/notifications?*',
      async (route) => {
        loadCount += 1
        await route.fulfill(
          loadCount === 1
            ? ok({
                list: [
                  {
                    id: 'bad-notif-1',
                    type: 'reply',
                    title: '畸形通知',
                    isRead: 'no',
                    createdAt: '2026-04-05T10:00:00Z',
                  },
                ],
                total: 1,
                page: 1,
                pageSize: 20,
              })
            : ok({
                list: [
                  {
                    id: 'retry-notif-1',
                    type: 'reply',
                    title: '重试后的通知',
                    content: '异常响应重试后应显示真实通知。',
                    isRead: false,
                    createdAt: '2026-04-05T10:00:00Z',
                    meta: { courseID: 101, reviewID: 'my-rev-1' },
                  },
                ],
                total: 1,
                page: 1,
                pageSize: 20,
              }),
        )
      },
    )

    await page.goto('/notifications')

    const status = page.getByRole('status').filter({ hasText: '加载失败' })
    await expect(status).toBeVisible({ timeout: 10_000 })
    await expect(page.getByText('暂无通知')).toHaveCount(0)

    await status.getByRole('button', { name: '重试' }).click()

    await expect.poll(() => loadCount).toBe(2)
    await expect(page.getByText('重试后的通知')).toBeVisible()
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

    expect(markReadRequest).toBeNull()
    await expect(page).toHaveURL(/\/about$/)
  })

  test('notification bell receives live SSE notifications and follows the pushed link', async ({
    page,
  }) => {
    let historyQuery: QueryRecord | null = null
    let markReadRequest: { method: string; path: string } | null = null
    let streamOpened = false

    await page.unroute('**/api/v1/course/review/user/notifications/unread-count*')
    await page.route(
      '**/api/v1/course/review/user/notifications/unread-count*',
      (route) =>
        route.fulfill({
          contentType: 'application/json',
          body: JSON.stringify({
            success: true,
            data: { count: streamOpened ? 3 : 2 },
          }),
        }),
    )
    await page.unroute('**/api/v1/course/review/user/notifications/stream')
    await page.route(
      '**/api/v1/course/review/user/notifications/stream',
      (route) => {
        streamOpened = true
        return route.fulfill({
          status: 200,
          contentType: 'text/event-stream',
          headers: { 'Cache-Control': 'no-cache' },
          body: [
            'event: notification',
            'data: {"id":"sse-notif-1","type":"reply","title":"实时提醒","content":"SSE 推送的回复","isRead":false,"sourceUrl":"/about","createdAt":"2026-04-05T10:10:00Z"}',
            '',
            'event: unread_count',
            'data: {"count":3}',
            '',
            '',
          ].join('\n'),
        })
      },
    )

    await page.route('**/api/v1/course/review/user/notifications?*', (route) => {
      historyQuery = captureQuery(route.request().url())
      return route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: { list: [], total: 0, page: 1, pageSize: 5 },
        }),
      })
    })

    await page.route(
      '**/api/v1/course/review/user/notifications/sse-notif-1/read',
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
    await expect.poll(() => streamOpened).toBe(true)
    await expect(bellButton.locator('span').filter({ hasText: '3' })).toBeVisible()

    await bellButton.click()
    const bellPanel = page.getByRole('region', { name: '通知' })
    await expect(bellPanel.getByText('实时提醒')).toBeVisible()
    await expect(bellPanel.getByText('SSE 推送的回复')).toBeVisible()
    await expect.poll(() => historyQuery).toEqual({ page: '1', pageSize: '5' })

    await bellPanel.getByRole('button', { name: /实时提醒/ }).click()

    await expect
      .poll(() => markReadRequest)
      .toEqual({
        method: 'PUT',
        path: '/api/v1/course/review/user/notifications/sse-notif-1/read',
      })
    await expect(page).toHaveURL(/\/about$/)
  })

  test('notifications page inserts live SSE notifications after initial empty load', async ({
    page,
  }) => {
    let releaseStream!: () => void
    const streamRelease = new Promise<void>((resolve) => {
      releaseStream = resolve
    })
    let streamOpened = false
    let historyQuery: QueryRecord | null = null
    let markReadRequest: { method: string; path: string } | null = null

    await page.unroute('**/api/v1/course/review/user/notifications/unread-count*')
    await page.route(
      '**/api/v1/course/review/user/notifications/unread-count*',
      (route) =>
        route.fulfill({
          contentType: 'application/json',
          body: JSON.stringify({
            success: true,
            data: { count: streamOpened ? 1 : 0 },
          }),
        }),
    )
    await page.unroute('**/api/v1/course/review/user/notifications/stream')
    await page.route(
      '**/api/v1/course/review/user/notifications/stream',
      async (route) => {
        streamOpened = true
        await streamRelease
        await route.fulfill({
          status: 200,
          contentType: 'text/event-stream',
          headers: { 'Cache-Control': 'no-cache' },
          body: [
            'event: notification',
            'data: {"id":"sse-page-notif-1","type":"reply","title":"实时页面提醒","content":"通知中心收到 SSE","isRead":false,"sourceUrl":"/about","createdAt":"2026-04-05T10:12:00Z"}',
            '',
            'event: unread_count',
            'data: {"count":1}',
            '',
            '',
          ].join('\n'),
        })
      },
    )

    await page.route('**/api/v1/course/review/user/notifications?*', (route) => {
      historyQuery = captureQuery(route.request().url())
      return route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: { list: [], total: 0, page: 1, pageSize: 20 },
        }),
      })
    })

    await page.route(
      '**/api/v1/course/review/user/notifications/sse-page-notif-1/read',
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

    await expect(page.getByRole('heading', { name: '通知' })).toBeVisible({
      timeout: 10_000,
    })
    await expect.poll(() => historyQuery).toEqual({ page: '1', pageSize: '20' })
    await expect(page.getByText('暂无通知')).toBeVisible()
    await expect.poll(() => streamOpened).toBe(true)

    releaseStream()

    await expect(page.getByText('实时页面提醒')).toBeVisible()
    await expect(page.getByText('通知中心收到 SSE')).toBeVisible()
    await expect(page.getByRole('button', { name: /全部.*已读/ })).toBeVisible()

    await page.getByRole('button', { name: /实时页面提醒/ }).click()

    await expect
      .poll(() => markReadRequest)
      .toEqual({
        method: 'PUT',
        path: '/api/v1/course/review/user/notifications/sse-page-notif-1/read',
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
