import { expect, test, type Page } from './fixtures'

type MockOptions = {
  authenticated?: boolean
  ssoState?: string
}

const now = '2026-05-24T04:00:00Z'

const user = {
  id: 'mobile-user-1',
  name: 'mobile-user',
  displayName: '移动端用户',
  email: 'mobile@example.com',
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

const course = {
  id: 101,
  name: '移动端数据结构',
  code: 'MOBILE-DS-101',
  departmentID: 1,
  departmentName: '移动端学院',
  credits: 3,
  category: '专业核心',
  reviewCount: 2,
}

const term = {
  id: '2026-spring',
  name: '2026 春',
  isCurrent: true,
}

const teacher = {
  teacherID: 10,
  teacherName: '移动端教师',
  departmentName: '移动端学院',
  reviewCount: 2,
  avgRating: 4.8,
}

const dimensions = [
  { key: 'overall', name: '综合体验', description: '整体课程体验' },
  { key: 'workload', name: '作业压力', description: '作业和项目负担' },
]

const review = {
  id: 'mobile-review-1',
  courseID: course.id,
  courseName: course.name,
  teacherID: teacher.teacherID,
  teacherName: teacher.teacherName,
  termID: term.id,
  termName: term.name,
  title: '移动端评课样例',
  content: '这是一条用于 UniAppX H5 端到端测试的评课内容，覆盖课程、教师、评分和列表展示。',
  ratings: { overall: 5, workload: 4 },
  likeCount: 7,
  dislikeCount: 1,
  replyCount: 1,
  authorID: user.id,
  authorName: user.displayName,
  createdAt: now,
  updatedAt: now,
}

const favorite = {
  ...course,
  favoritedAt: now,
}

const notification = {
  id: 'notice-1',
  title: '移动端通知',
  content: '你的评课收到了一条新回复。',
  type: 'reply',
  isRead: false,
  createdAt: now,
}

function json(data: unknown, status = 200) {
  return {
    status,
    contentType: 'application/json',
    body: JSON.stringify(data),
  }
}

function ok(data: unknown = {}) {
  return json({ success: true, data })
}

function list<T>(items: T[]) {
  return ok({ list: items, total: items.length })
}

async function mockUniApi(page: Page, options: MockOptions = {}) {
  const mutations: string[] = []

  await page.addInitScript((ssoState) => {
    localStorage.setItem('stuhelper:uniappx:locale', 'zh-CN')
    if (ssoState) {
      localStorage.setItem('stuhelper:sso-state', ssoState)
    }
  }, options.ssoState ?? '')

  await page.route('**/api/v1/**', async (route) => {
    const request = route.request()
    const method = request.method().toUpperCase()
    const pathname = new URL(request.url()).pathname
    const authenticated = options.authenticated === true

    if (method !== 'GET') {
      mutations.push(`${method} ${pathname}`)
    }

    if (method === 'GET' && pathname === '/api/v1/auth/me') {
      await route.fulfill(ok(authenticated ? user : null))
      return
    }
    if (method === 'POST' && pathname === '/api/v1/auth/logout') {
      await route.fulfill(ok())
      return
    }
    if (method === 'GET' && pathname === '/api/v1/auth/login') {
      await route.fulfill(ok({ url: 'https://sso.example.test/login', state: 'mobile-e2e-state' }))
      return
    }
    if (method === 'POST' && pathname === '/api/v1/auth/exchange-native') {
      await route.fulfill(ok({
        accessToken: 'mobile-native-access-token',
        refreshToken: 'mobile-native-refresh-token',
        sessionID: 'mobile-native-session',
        expiresIn: 900,
      }))
      return
    }

    if (method === 'GET' && pathname === '/api/v1/course/stats') {
      await route.fulfill(ok({ courseCount: 2, departmentCount: 1 }))
      return
    }
    if (method === 'GET' && pathname === '/api/v1/course/terms') {
      await route.fulfill(ok([term]))
      return
    }
    if (method === 'GET' && pathname === '/api/v1/course/courses') {
      await route.fulfill(list([course]))
      return
    }
    if (method === 'GET' && /^\/api\/v1\/course\/courses\/\d+$/.test(pathname)) {
      await route.fulfill(ok(course))
      return
    }

    if (method === 'GET' && pathname === '/api/v1/course/review/stats') {
      await route.fulfill(ok({ reviewCount: 3, departmentCount: 1 }))
      return
    }
    if (method === 'GET' && pathname === '/api/v1/course/review/rankings/hot') {
      await route.fulfill(list([
        {
          courseID: course.id,
          courseName: course.name,
          reviewCount: course.reviewCount,
          avgRating: 4.8,
        },
      ]))
      return
    }
    if (method === 'GET' && pathname === '/api/v1/course/review/rating-dimensions') {
      await route.fulfill(ok(dimensions))
      return
    }
    if (
      method === 'GET' &&
      /^\/api\/v1\/course\/review\/courses\/\d+\/rating-stats$/.test(pathname)
    ) {
      await route.fulfill(ok({
        courseID: course.id,
        courseName: course.name,
        overall: {
          avgRating: 4.8,
          ratingCount: 2,
          dimensions: [
            { key: 'overall', name: '综合体验', avgRating: 4.8, ratingCount: 2 },
            { key: 'workload', name: '作业压力', avgRating: 4.4, ratingCount: 2 },
          ],
        },
        terms: [],
      }))
      return
    }
    if (
      method === 'GET' &&
      /^\/api\/v1\/course\/review\/courses\/\d+\/teachers$/.test(pathname)
    ) {
      await route.fulfill(ok([teacher]))
      return
    }
    if (
      method === 'GET' &&
      /^\/api\/v1\/course\/review\/courses\/\d+\/reviews$/.test(pathname)
    ) {
      await route.fulfill(list([review]))
      return
    }
    if (
      method === 'GET' &&
      /^\/api\/v1\/course\/review\/courses\/\d+\/favorites$/.test(pathname)
    ) {
      await route.fulfill(ok({ favorited: true }))
      return
    }
    if (method === 'GET' && pathname === '/api/v1/course/review/reviews/latest') {
      await route.fulfill(list([review]))
      return
    }
    if (method === 'POST' && /^\/api\/v1\/course\/review\/reviews\/[^/]+\/votes$/.test(pathname)) {
      await route.fulfill(ok())
      return
    }
    if (method === 'POST' && pathname === '/api/v1/course/review/reviews') {
      await route.fulfill(ok({ id: 'created-mobile-review' }))
      return
    }
    if (method === 'GET' && pathname === '/api/v1/course/review/drafts') {
      await route.fulfill(ok(null))
      return
    }
    if (method === 'POST' && pathname === '/api/v1/course/review/drafts') {
      await route.fulfill(ok())
      return
    }
    if (method === 'DELETE' && pathname === '/api/v1/course/review/drafts') {
      await route.fulfill(ok())
      return
    }
    if (
      method === 'GET' &&
      /^\/api\/v1\/course\/review\/teachers\/\d+\/stats$/.test(pathname)
    ) {
      await route.fulfill(ok({
        teacherID: teacher.teacherID,
        teacherName: teacher.teacherName,
        departmentName: teacher.departmentName,
        avgRating: teacher.avgRating,
        courseCount: 1,
        reviewCount: teacher.reviewCount,
        courses: [{ ...course, avgRating: teacher.avgRating }],
      }))
      return
    }

    if (method === 'GET' && pathname === '/api/v1/user/me') {
      await route.fulfill(ok({
        identityStatus: authenticated ? 'approved' : 'unverified',
        verificationStatus: authenticated ? 'approved' : 'unverified',
        phoneBound: authenticated,
      }))
      return
    }
    if (method === 'GET' && pathname === '/api/v1/course/review/user/reviews') {
      await route.fulfill(list([review]))
      return
    }
    if (method === 'GET' && pathname === '/api/v1/course/review/user/votes') {
      await route.fulfill(list([review]))
      return
    }
    if (method === 'GET' && pathname === '/api/v1/course/review/user/favorites') {
      await route.fulfill(list([favorite]))
      return
    }
    if (method === 'GET' && pathname === '/api/v1/course/review/user/notifications') {
      await route.fulfill(list([notification]))
      return
    }
    if (
      method === 'PUT' &&
      /^\/api\/v1\/course\/review\/user\/notifications\/[^/]+\/read$/.test(pathname)
    ) {
      await route.fulfill(ok())
      return
    }
    if (method === 'PUT' && pathname === '/api/v1/course/review/user/notifications/read-all') {
      await route.fulfill(ok())
      return
    }

    await route.fulfill(json({
      success: false,
      code: 'E2E_UNMOCKED_API',
      message: `unmocked UniAppX E2E API request: ${method} ${pathname}`,
    }, 500))
  })

  return mutations
}

async function gotoUniPage(page: Page, url: string) {
  await page.goto(url)
  await page.waitForLoadState('networkidle')
}

async function expectUniPageTitle(page: Page, title: string) {
  await expect(page).toHaveTitle(title)
}

async function expectTabBarIconsAvailable(page: Page) {
  const origin = new URL(page.url()).origin
  const iconNames = ['home', 'course', 'review', 'user']
  for (const name of iconNames) {
    for (const suffix of ['', '-active']) {
      const response = await page.request.get(`${origin}/static/tabbar/${name}${suffix}.png`)
      expect(response.status(), `${name}${suffix}.png should be served`).toBe(200)
      expect(response.headers()['content-type']).toContain('image/png')
    }
  }
}

test.describe('UniAppX H5 surface', () => {
  test('home dashboard renders real feature entrypoints and hot course data', async ({
    page,
  }) => {
    await mockUniApi(page)

    await gotoUniPage(page, '/')

    await expectUniPageTitle(page, 'StuHelper')
    await expect(page.getByText('StuHelper 移动端')).toBeVisible()
    await expect(page.getByText('欢迎使用 StuHelper')).toBeVisible()
    await expect(page.getByText('课程查询')).toBeVisible()
    await expect(page.getByText('评课广场')).toBeVisible()
    await expect(page.getByText('个人中心', { exact: true })).toBeVisible()
    await expect(page.getByText(course.name).first()).toBeVisible()
    await expect(page.getByText('首页', { exact: true })).toBeVisible()
    await expect(page.getByText('我的', { exact: true })).toBeVisible()
    await expectTabBarIconsAvailable(page)
  })

  test('course, review, and teacher browsing pages render API-backed content', async ({
    page,
  }) => {
    await mockUniApi(page)

    await gotoUniPage(page, '/#/pages/course/index')
    await expectUniPageTitle(page, '课程列表')
    await expect(page.getByText(course.name)).toBeVisible()
    await expect(page.getByText(course.code)).toBeVisible()

    await gotoUniPage(page, `/#/pages/course/detail?id=${course.id}`)
    await expectUniPageTitle(page, '课程详情')
    await expect(page.getByText(course.name).first()).toBeVisible()
    await expect(page.getByText(teacher.teacherName).first()).toBeVisible()
    await expect(page.getByText(review.title)).toBeVisible()
    await expect(page.getByText('写评课')).toBeVisible()

    await gotoUniPage(page, '/#/pages/review/index')
    await expectUniPageTitle(page, '评课广场')
    await expect(page.getByText('最新')).toBeVisible()
    await expect(page.getByText('最热')).toBeVisible()
    await expect(page.getByText(review.title)).toBeVisible()

    await gotoUniPage(page, `/#/pages/teacher/profile?id=${teacher.teacherID}`)
    await expectUniPageTitle(page, '教师主页')
    await expect(page.getByText(teacher.teacherName).first()).toBeVisible()
    await expect(page.getByText('授课课程')).toBeVisible()
    await expect(page.getByText(course.name)).toBeVisible()
  })

  test('authenticated review post page loads form data and saves a draft', async ({
    page,
  }) => {
    const mutations = await mockUniApi(page, { authenticated: true })

    await gotoUniPage(page, `/#/pages/review/post?courseID=${course.id}`)

    await expectUniPageTitle(page, '发布评课')
    await expect(page.getByText('发布评课').first()).toBeVisible()
    await expect(page.getByText(course.name)).toBeVisible()
    await expect(page.getByText(term.name).last()).toBeVisible()
    await expect(page.getByText('综合体验')).toBeVisible()

    await page.getByText('保存草稿').click()
    await expect
      .poll(() => mutations.includes('POST /api/v1/course/review/drafts'))
      .toBe(true)
  })

  test('authenticated user center pages render profile data and user lists', async ({
    page,
  }) => {
    const mutations = await mockUniApi(page, { authenticated: true })

    await gotoUniPage(page, '/#/pages/user/index')
    await expectUniPageTitle(page, '个人中心')
    await expect(page.getByText(user.displayName)).toBeVisible()
    await expect(page.getByText('认证概览')).toBeVisible()
    await expect(page.getByText('实名已通过')).toBeVisible()
    await expect(page.getByText('学生认证已通过')).toBeVisible()

    await gotoUniPage(page, '/#/pages/user/reviews')
    await expectUniPageTitle(page, '我的评课')
    await expect(page.getByText(review.title)).toBeVisible()

    await gotoUniPage(page, '/#/pages/user/votes')
    await expectUniPageTitle(page, '我的投票')
    await expect(page.getByText(review.title)).toBeVisible()

    await gotoUniPage(page, '/#/pages/user/favorites')
    await expectUniPageTitle(page, '我的收藏')
    await expect(page.getByText(course.name)).toBeVisible()

    await gotoUniPage(page, '/#/pages/user/notifications')
    await expectUniPageTitle(page, '消息通知')
    await expect(page.getByText(notification.title)).toBeVisible()
    await page.getByText('全部已读').click()
    await expect
      .poll(() => mutations.includes('PUT /api/v1/course/review/user/notifications/read-all'))
      .toBe(true)
  })

  test('auth pages render login and callback error states', async ({ page }) => {
    await mockUniApi(page)

    await gotoUniPage(page, '/#/pages/auth/login')
    await expectUniPageTitle(page, '登录')
    await expect(page.getByText('登录 StuHelper')).toBeVisible()
    await expect(page.getByText('使用校园 SSO 登录')).toBeVisible()

    await gotoUniPage(page, '/#/pages/auth/callback')
    await expectUniPageTitle(page, 'SSO 回调')
    await expect(page.getByText('登录失败')).toBeVisible()
    await expect(page.getByText('回调参数缺失，请重新登录')).toBeVisible()
  })

  test('auth callback exchanges native code and opens the user center', async ({ page }) => {
    const mutations = await mockUniApi(page, {
      authenticated: true,
      ssoState: 'mobile-e2e-state',
    })

    await gotoUniPage(page, '/#/pages/auth/callback?code=mobile-e2e-code&state=mobile-e2e-state')

    await expect
      .poll(() => mutations.includes('POST /api/v1/auth/exchange-native'))
      .toBe(true)
    await expect(page).toHaveURL(/\/#\/pages\/user\/index/)
    await expectUniPageTitle(page, '个人中心')
    await expect(page.getByText(user.displayName)).toBeVisible()
    await expect
      .poll(() => page.evaluate(() => localStorage.getItem('stuhelper:sso-state')))
      .toBeNull()
  })
})
