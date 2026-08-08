import { expect, test, type Page } from './fixtures'

type MockOptions = {
  authenticated?: boolean
  courseDetailFailures?: number
  courseResponseDelayMs?: number
  courseSupplementFailure?: boolean
  mutationDelayMs?: number
  paginatedCourses?: boolean
  paginatedReviews?: boolean
  paginatedUserLists?: boolean
  reviewPage2Failures?: number
  reviewDraft?: Record<string, unknown> | null
  reviewSortRace?: boolean
  reviewUserVote?: 'like' | 'dislike'
  ssoState?: string
  teacherDetailFailures?: number
  teacherResponseDelayMs?: number
  phoneVerified?: boolean
  userSurface?: {
    displayName?: string
    studentVerificationStatus: 'none' | 'approved'
    phoneBound: boolean
    phone?: string | null
    capabilities?: string[]
  }
}

type MockUniApiResult = {
  mutations: string[]
  mutationBodies: Array<{
    body: unknown
    method: string
    path: string
  }>
  requests: string[]
}

const now = '2026-05-24T04:00:00Z'
const webVerificationOrigin = (
  process.env.UNIAPPX_E2E_WEB_URL ?? 'https://web.example.test'
).replace(/\/$/, '')

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

function buildCourse(id: number, index: number) {
  return {
    ...course,
    id,
    name: `${course.name} ${index}`,
    code: `MOBILE-DS-${String(index).padStart(3, '0')}`,
    reviewCount: course.reviewCount + index,
  }
}

const secondCourse = {
  ...course,
  id: 202,
  name: '移动端算法设计',
  code: 'MOBILE-ALG-202',
  reviewCount: 5,
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

function buildReview(id: string, index: number) {
  return {
    ...review,
    id,
    title: `${review.title} ${index}`,
    likeCount: review.likeCount + index,
    dislikeCount: review.dislikeCount + index,
  }
}

const secondPageReview = buildReview('mobile-review-page-2-1', 21)
const secondUserReview = {
  ...buildReview('mobile-user-review-page-2-1', 31),
  courseID: secondCourse.id,
  courseName: secondCourse.name,
  title: '移动端二页我的评课',
}
const secondUserVote = {
  ...buildReview('mobile-user-vote-page-2-1', 32),
  courseID: secondCourse.id,
  courseName: secondCourse.name,
  title: '移动端二页我的投票',
}

const initialReply = {
  id: 'mobile-reply-1',
  reviewID: review.id,
  content: '已有移动端回复',
  authorID: 'mobile-reply-author',
  authorName: '回复用户',
  createdAt: now,
  updatedAt: now,
}

const favorite = {
  ...course,
  favoritedAt: now,
}

const secondFavorite = {
  ...secondCourse,
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

const secondNotification = {
  ...notification,
  id: 'notice-2',
  title: '移动端二页通知',
  content: '这是第二页加载出来的通知。',
  isRead: false,
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

function paginatedList<T>(items: T[], total: number) {
  return ok({ list: items, total })
}

async function mockUniApi(page: Page, options: MockOptions = {}): Promise<MockUniApiResult> {
  const mutations: string[] = []
  const mutationBodies: MockUniApiResult['mutationBodies'] = []
  const requests: string[] = []
  const authenticated = options.authenticated === true
  const replyItems = [initialReply]
  let remainingReviewPage2Failures = options.reviewPage2Failures ?? 0
  let remainingCourseDetailFailures = options.courseDetailFailures ?? 0
  let remainingTeacherDetailFailures = options.teacherDetailFailures ?? 0
  const responseReview = options.reviewUserVote
    ? { ...review, userVote: options.reviewUserVote }
    : review
  const paginatedReviewPage = [
    responseReview,
    ...Array.from({ length: 19 }, (_, index) => buildReview(`mobile-review-page-1-${index + 2}`, index + 2)),
  ]
  const paginatedCoursePage = [
    course,
    ...Array.from({ length: 19 }, (_, index) => buildCourse(index + 102, index + 2)),
  ]
  const paginatedUserReviewPage = [
    review,
    ...Array.from({ length: 19 }, (_, index) => (
      buildReview(`mobile-user-review-page-1-${index + 2}`, index + 2)
    )),
  ]
  const paginatedUserVotePage = [
    review,
    ...Array.from({ length: 19 }, (_, index) => (
      buildReview(`mobile-user-vote-page-1-${index + 2}`, index + 2)
    )),
  ]
  const paginatedFavoritePage = [
    favorite,
    ...Array.from({ length: 19 }, (_, index) => ({
      ...buildCourse(index + 102, index + 2),
      favoritedAt: now,
    })),
  ]
  const paginatedNotificationPage = [
    notification,
    ...Array.from({ length: 19 }, (_, index) => ({
      ...notification,
      id: `notice-page-1-${index + 2}`,
      title: `移动端通知 ${index + 2}`,
    })),
  ]
  const phoneVerified = options.phoneVerified ?? authenticated
  const userSurface = options.userSurface ?? {
    displayName: user.displayName,
    studentVerificationStatus: authenticated ? 'approved' : 'none',
    phoneBound: phoneVerified,
    phone: phoneVerified ? '138****5678' : null,
    capabilities: user.capabilities,
  }
  const phoneStatus = {
    state: phoneVerified ? 'verified' : 'unbound',
    maskedPhone: phoneVerified ? '138****5678' : null,
    method: phoneVerified ? 'sms_possession' : null,
    verifiedAt: phoneVerified ? now : null,
    expiresAt: null,
    publishingRequirementSatisfied: phoneVerified,
    revision: 1,
  }

  await page.addInitScript(({ isAuthenticated, ssoState }) => {
    localStorage.setItem('stuhelper:uniappx:locale', 'zh-CN')
    if (ssoState) {
      localStorage.setItem('stuhelper:sso-state', ssoState)
    }
    if (isAuthenticated) {
      document.cookie = 'csrf_token=mobile-e2e-csrf; path=/'
      localStorage.setItem('stuhelper:csrf-token', 'mobile-e2e-csrf')
    }
  }, {
    isAuthenticated: authenticated,
    ssoState: options.ssoState ?? '',
  })

  await page.route('**/api/v1/**', async (route) => {
    const request = route.request()
    const method = request.method().toUpperCase()
    const url = new URL(request.url())
    const pathname = url.pathname

    requests.push(`${method} ${pathname}${url.search}`)

    if (method !== 'GET') {
      mutations.push(`${method} ${pathname}`)
      let body: unknown = null
      try {
        body = request.postDataJSON()
      } catch (_error) {
        void _error
        body = request.postData()
      }
      mutationBodies.push({ body, method, path: pathname })
      if (options.mutationDelayMs) {
        await new Promise(resolve => setTimeout(resolve, options.mutationDelayMs))
      }
    }

    if (method === 'GET' && pathname === '/api/v1/auth/me') {
      if (authenticated) {
        await route.fulfill(ok(user))
        return
      }
      await route.fulfill(json({
        success: false,
        error: { code: 'A0010003', message: 'missing authentication token' },
      }, 401))
      return
    }
    if (method === 'POST' && pathname === '/api/v1/auth/refresh') {
      await route.fulfill(json({
        success: false,
        error: { code: 'A0010100', message: 'session expired' },
      }, 401))
      return
    }
    if (method === 'POST' && pathname === '/api/v1/auth/logout') {
      await route.fulfill(ok())
      return
    }
    if (method === 'GET' && pathname === '/api/v1/auth/login') {
      await route.fulfill(ok({
        url: 'https://sso.example.test/login?state=mobile-e2e-state',
        state: 'mobile-e2e-state',
      }))
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
      if (options.paginatedCourses) {
        const pageNumber = Number(url.searchParams.get('page') || '1')
        const search = url.searchParams.get('q')?.trim()
        if (search) {
          await route.fulfill(paginatedList([secondCourse], 1))
          return
        }
        await route.fulfill(paginatedList(pageNumber === 1 ? paginatedCoursePage : [secondCourse], 21))
        return
      }
      await route.fulfill(list([course]))
      return
    }
    if (method === 'GET' && /^\/api\/v1\/course\/courses\/\d+$/.test(pathname)) {
      if (options.courseResponseDelayMs) {
        await new Promise(resolve => setTimeout(resolve, options.courseResponseDelayMs))
      }
      if (remainingCourseDetailFailures > 0) {
        remainingCourseDetailFailures -= 1
        await route.fulfill(json({
          success: false,
          error: { code: 'UNI_COURSE_DETAIL_FAILED', message: '课程详情暂时不可用' },
        }))
        return
      }
      const courseID = Number(pathname.split('/').pop())
      await route.fulfill(ok(courseID === secondCourse.id ? secondCourse : course))
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
      if (options.courseSupplementFailure) {
        await route.fulfill(json({
          success: false,
          error: { code: 'UNI_COURSE_STATS_FAILED', message: '课程评分统计暂时不可用' },
        }))
        return
      }
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
    if (
      (method === 'POST' || method === 'DELETE') &&
      /^\/api\/v1\/course\/review\/courses\/\d+\/favorites$/.test(pathname)
    ) {
      await route.fulfill(ok())
      return
    }
    if (method === 'GET' && pathname === '/api/v1/course/review/reviews/latest') {
      const pageNumber = Number(url.searchParams.get('page') || '1')
      const requestedSort = url.searchParams.get('sort') || 'time'
      if (options.reviewSortRace && pageNumber === 1 && requestedSort !== 'time') {
        const isSlowStaleRequest = requestedSort === 'likes'
        await new Promise(resolve => setTimeout(resolve, isSlowStaleRequest ? 250 : 20))
        await route.fulfill(list([{
          ...responseReview,
          id: isSlowStaleRequest ? 'review-sort-stale' : 'review-sort-current',
          title: isSlowStaleRequest ? '过期最热结果' : '当前高分结果',
        }]))
        return
      }
      if (options.paginatedReviews) {
        if (pageNumber === 2 && remainingReviewPage2Failures > 0) {
          remainingReviewPage2Failures -= 1
          await route.fulfill(json({
            success: false,
            error: { code: 'UNI_REVIEW_PAGE_FAILED', message: '评课下一页加载失败' },
          }))
          return
        }
        await route.fulfill(paginatedList(
          pageNumber === 1 ? paginatedReviewPage : [secondPageReview],
          21,
        ))
        return
      }
      await route.fulfill(list([responseReview]))
      return
    }
    if (method === 'POST' && /^\/api\/v1\/course\/review\/reviews\/[^/]+\/votes$/.test(pathname)) {
      await route.fulfill(ok())
      return
    }
    if (method === 'GET' && /^\/api\/v1\/course\/review\/reviews\/[^/]+\/replies$/.test(pathname)) {
      await route.fulfill(list(replyItems))
      return
    }
    if (method === 'POST' && /^\/api\/v1\/course\/review\/reviews\/[^/]+\/replies$/.test(pathname)) {
      const body = mutationBodies[mutationBodies.length - 1]?.body as { content?: unknown } | undefined
      replyItems.push({
        id: 'mobile-reply-created',
        reviewID: review.id,
        content: typeof body?.content === 'string' ? body.content : '',
        authorID: user.id,
        authorName: user.displayName,
        createdAt: now,
        updatedAt: now,
      })
      await route.fulfill(ok({ id: 'mobile-reply-created' }))
      return
    }
    if (method === 'POST' && pathname === '/api/v1/course/review/reviews') {
      await route.fulfill(ok({ id: 'created-mobile-review' }))
      return
    }
    if (method === 'GET' && pathname === '/api/v1/course/review/drafts') {
      await route.fulfill(ok(options.reviewDraft ?? null))
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
      if (options.teacherResponseDelayMs) {
        await new Promise(resolve => setTimeout(resolve, options.teacherResponseDelayMs))
      }
      if (remainingTeacherDetailFailures > 0) {
        remainingTeacherDetailFailures -= 1
        await route.fulfill(json({
          success: false,
          error: { code: 'UNI_TEACHER_DETAIL_FAILED', message: '教师信息暂时不可用' },
        }))
        return
      }
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
      await route.fulfill(ok(userSurface))
      return
    }
    if (method === 'GET' && pathname === '/api/v1/account/phone') {
      await route.fulfill(ok(phoneStatus))
      return
    }
    if (method === 'GET' && pathname === '/api/v1/course/review/user/reviews') {
      if (options.paginatedUserLists) {
        const pageNumber = Number(url.searchParams.get('page') || '1')
        await route.fulfill(paginatedList(
          pageNumber === 1 ? paginatedUserReviewPage : [secondUserReview],
          21,
        ))
        return
      }
      await route.fulfill(list([review]))
      return
    }
    if (method === 'GET' && pathname === '/api/v1/course/review/user/votes') {
      if (options.paginatedUserLists) {
        const pageNumber = Number(url.searchParams.get('page') || '1')
        await route.fulfill(paginatedList(
          pageNumber === 1 ? paginatedUserVotePage : [secondUserVote],
          21,
        ))
        return
      }
      await route.fulfill(list([review]))
      return
    }
    if (method === 'GET' && pathname === '/api/v1/course/review/user/favorites') {
      if (options.paginatedUserLists) {
        const pageNumber = Number(url.searchParams.get('page') || '1')
        await route.fulfill(paginatedList(
          pageNumber === 1 ? paginatedFavoritePage : [secondFavorite],
          21,
        ))
        return
      }
      await route.fulfill(list([favorite]))
      return
    }
    if (method === 'GET' && pathname === '/api/v1/course/review/user/notifications') {
      if (options.paginatedUserLists) {
        const pageNumber = Number(url.searchParams.get('page') || '1')
        await route.fulfill(paginatedList(
          pageNumber === 1 ? paginatedNotificationPage : [secondNotification],
          21,
        ))
        return
      }
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

  return { mutations, mutationBodies, requests }
}

async function gotoUniPage(page: Page, url: string) {
  await page.goto(url)
  await page.waitForLoadState('networkidle')
}

async function expectUniPageTitle(page: Page, title: string) {
  await expect(page).toHaveTitle(title)
}

async function mockWebVerificationTarget(page: Page) {
  await page.route(`${webVerificationOrigin}/**`, async (route) => {
    await route.fulfill({
      contentType: 'text/html',
      body: '<!doctype html><title>Mock Web Verification</title><main>Mock Web Verification</main>',
    })
  })
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

async function setUniFieldValue(page: Page, testID: string, value: string) {
  await page.getByTestId(testID).evaluate((element, nextValue) => {
    const control = element.querySelector('input,textarea') as
      | HTMLInputElement
      | HTMLTextAreaElement
      | null
    if (!control) {
      throw new Error(`Uni form control not found for ${element.getAttribute('data-testid')}`)
    }

    control.value = nextValue
    control.dispatchEvent(new Event('input', { bubbles: true, composed: true }))
    control.dispatchEvent(new Event('change', { bubbles: true, composed: true }))
    element.dispatchEvent(new CustomEvent('input', {
      bubbles: true,
      composed: true,
      detail: { value: nextValue },
    }))
    element.dispatchEvent(new CustomEvent('change', {
      bubbles: true,
      composed: true,
      detail: { value: nextValue },
    }))
  }, value)
}

async function selectUniPickerIndex(page: Page, testID: string, index: number) {
  await page.getByTestId(testID).click()
  const picker = page.locator('.uni-picker-container:visible').last()
  await expect(picker).toBeVisible()

  const selectOption = picker.locator('.uni-picker-select .uni-picker-item').nth(index)
  try {
    await expect(selectOption).toBeVisible({ timeout: 1000 })
    await selectOption.click()
    return
  } catch {
    // Mobile H5 renders the selector as a picker-view overlay instead of the desktop select list.
  }

  const pickerView = picker.locator('uni-picker-view')
  await expect(pickerView).toBeVisible()
  await pickerView.evaluate((element, selectedIndex) => {
    element.dispatchEvent(new CustomEvent('change', {
      bubbles: true,
      composed: true,
      detail: { value: [selectedIndex] },
    }))
  }, index)
  await picker.locator('.uni-picker-action-confirm').click()
}

function requireMutationBody(
  mutationBodies: MockUniApiResult['mutationBodies'],
  method: string,
  path: string,
): Record<string, unknown> {
  const body = mutationBodies.find((item) => item.method === method && item.path === path)?.body
  expect(body, `${method} ${path} request body should be captured`).toBeTruthy()
  expect(typeof body).toBe('object')
  return body as Record<string, unknown>
}

function currentLoginRedirect(page: Page): string | null {
  const hash = new URL(page.url()).hash
  const queryString = hash.includes('?') ? hash.slice(hash.indexOf('?') + 1) : ''
  const encoded = new URLSearchParams(queryString).get('redirect')
  if (!encoded) return null

  let decoded = encoded
  for (let attempt = 0; attempt < 2; attempt += 1) {
    const next = decodeURIComponent(decoded)
    if (next === decoded) break
    decoded = next
  }
  return decoded
}

test.describe('UniAppX H5 surface', () => {
  test('home dashboard renders real feature entrypoints and hot course data', async ({
    page,
  }) => {
    const { requests } = await mockUniApi(page)

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
    expect(requests.some((request) => request.startsWith('GET /api/v1/auth/me'))).toBe(false)
  })

  test('home course shortcut opens the course list tab', async ({ page }) => {
    await mockUniApi(page)

    await gotoUniPage(page, '/')
    await page.getByTestId('uni-home-shortcut-course').click()
    await expect(page).toHaveURL(/#\/pages\/course\/index/)
    await expectUniPageTitle(page, '课程列表')
    await expect(page.getByText(course.name)).toBeVisible()
    await page.waitForLoadState('networkidle')
  })

  test('home review shortcut opens the review square tab', async ({ page }) => {
    await mockUniApi(page)

    await gotoUniPage(page, '/')
    await page.getByTestId('uni-home-shortcut-review').click()
    await expect(page).toHaveURL(/#\/pages\/review\/index/)
    await expectUniPageTitle(page, '评课广场')
    await expect(page.getByText(review.title)).toBeVisible()
    await page.waitForLoadState('networkidle')
  })

  test('home user shortcut opens the guest user center tab', async ({ page }) => {
    await mockUniApi(page)

    await gotoUniPage(page, '/')
    await page.getByTestId('uni-home-shortcut-user').click()
    await expect(page).toHaveURL(/#\/pages\/user\/index/)
    await expectUniPageTitle(page, '个人中心')
    await expect(page.getByTestId('uni-user-login')).toBeVisible()
    await page.waitForLoadState('networkidle')
  })

  test('home hot courses view-all opens the review square tab', async ({ page }) => {
    await mockUniApi(page)

    await gotoUniPage(page, '/')
    await page.getByTestId('uni-home-view-all-reviews').click()
    await expect(page).toHaveURL(/#\/pages\/review\/index/)
    await expectUniPageTitle(page, '评课广场')
    await expect(page.getByText(review.title)).toBeVisible()
    await page.waitForLoadState('networkidle')
  })

  test('home hot course row opens the course detail page', async ({ page }) => {
    await mockUniApi(page)

    await gotoUniPage(page, '/')
    await page.getByTestId(`uni-home-hot-course-${course.id}`).click()
    await expect(page).toHaveURL(new RegExp(`/#/pages/course/detail\\?id=${course.id}`))
    await expectUniPageTitle(page, '课程详情')
    await expect(page.getByText(course.name).first()).toBeVisible()
    await page.waitForLoadState('networkidle')
  })

  test('navigation cards expose button semantics and support keyboard activation', async ({
    page,
  }) => {
    await mockUniApi(page)

    await gotoUniPage(page, '/')
    const courseShortcut = page.getByTestId('uni-home-shortcut-course')
    await expect(courseShortcut).toHaveAttribute('role', 'button')
    await expect(courseShortcut).toHaveAttribute('tabindex', '0')
    await courseShortcut.focus()
    await page.keyboard.press('Enter')
    await expect(page).toHaveURL(/#\/pages\/course\/index/)

    const courseCard = page.getByTestId(`uni-course-card-${course.id}`)
    await expect(courseCard).toHaveAttribute('role', 'button')
    await courseCard.focus()
    await page.keyboard.press('Space')
    await expect(page).toHaveURL(new RegExp(`/#/pages/course/detail\\?id=${course.id}`))

    const teacherCard = page.getByTestId(`uni-course-teacher-${teacher.teacherID}`)
    await teacherCard.focus()
    await page.keyboard.press('Enter')
    await expect(page).toHaveURL(new RegExp(`/#/pages/teacher/profile\\?id=${teacher.teacherID}`))
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

    await page.getByTestId(`uni-course-teacher-${teacher.teacherID}`).click()
    await expect(page).toHaveURL(new RegExp(`/#/pages/teacher/profile\\?id=${teacher.teacherID}`))
    await expectUniPageTitle(page, '教师主页')
    await expect(page.getByText(teacher.teacherName).first()).toBeVisible()
    await expect(page.getByText('授课课程')).toBeVisible()
    await expect(page.getByTestId(`uni-teacher-course-${course.id}`)).toBeVisible()

    await page.getByTestId(`uni-teacher-course-${course.id}`).click()
    await expect(page).toHaveURL(new RegExp(`/#/pages/course/detail\\?id=${course.id}`))
    await expectUniPageTitle(page, '课程详情')
    await expect(page.getByText(course.name).first()).toBeVisible()

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

  test('detail pages coalesce lifecycle loads and retain primary content on supplemental failure', async ({
    page,
  }) => {
    const { requests } = await mockUniApi(page, {
      courseResponseDelayMs: 120,
      courseSupplementFailure: true,
      teacherResponseDelayMs: 120,
    })

    await gotoUniPage(page, `/#/pages/course/detail?id=${course.id}`)
    await expect(page.getByText(course.name).first()).toBeVisible()
    await expect(
      page.locator('[role="alert"]').getByText('部分课程信息暂时不可用，可稍后重试'),
    ).toBeVisible()
    expect(requests.filter(request => (
      request === `GET /api/v1/course/courses/${course.id}`
    ))).toHaveLength(1)

    await gotoUniPage(page, `/#/pages/teacher/profile?id=${teacher.teacherID}`)
    await expect(page.getByText(teacher.teacherName).first()).toBeVisible()
    expect(requests.filter(request => (
      request === `GET /api/v1/course/review/teachers/${teacher.teacherID}/stats`
    ))).toHaveLength(1)
  })

  test('course and teacher fatal load errors expose working retries', async ({ page }) => {
    const { requests } = await mockUniApi(page, {
      courseDetailFailures: 1,
      teacherDetailFailures: 1,
    })

    await gotoUniPage(page, `/#/pages/course/detail?id=${course.id}`)
    await expect(page.getByText('课程详情加载失败')).toBeVisible()
    await page.getByTestId('uni-course-detail-retry').click()
    await expect(page.getByText(course.name).first()).toBeVisible()
    expect(requests.filter(request => (
      request === `GET /api/v1/course/courses/${course.id}`
    ))).toHaveLength(2)

    await gotoUniPage(page, `/#/pages/teacher/profile?id=${teacher.teacherID}`)
    await expect(page.getByText('教师信息加载失败')).toBeVisible()
    await page.getByTestId('uni-teacher-retry').click()
    await expect(page.getByText(teacher.teacherName).first()).toBeVisible()
    expect(requests.filter(request => (
      request === `GET /api/v1/course/review/teachers/${teacher.teacherID}/stats`
    ))).toHaveLength(2)
  })

  test('course list searches, loads more, and opens a course detail page', async ({ page }) => {
    const { requests } = await mockUniApi(page, { paginatedCourses: true })

    await gotoUniPage(page, '/#/pages/course/index')

    await expect(page.getByTestId(`uni-course-card-${course.id}`)).toBeVisible()
    await page.getByTestId('uni-course-load-more').click()
    await expect
      .poll(() => requests.some((request) => (
        request.startsWith('GET /api/v1/course/courses?') &&
        request.includes('page=2') &&
        request.includes('sort=reviewCount')
      )))
      .toBe(true)
    await expect(page.getByTestId(`uni-course-card-${secondCourse.id}`)).toBeVisible()

    await setUniFieldValue(page, 'uni-course-search-input', '算法')
    await page.getByTestId('uni-course-search-submit').click()
    await expect
      .poll(() => requests.some((request) => (
        request.startsWith('GET /api/v1/course/courses?') &&
        request.includes('page=1') &&
        request.includes('q=%E7%AE%97%E6%B3%95') &&
        request.includes('sort=reviewCount')
      )))
      .toBe(true)
    await expect(page.getByTestId(`uni-course-card-${secondCourse.id}`)).toBeVisible()
    await expect(page.getByTestId(`uni-course-card-${course.id}`)).toHaveCount(0)

    await page.getByTestId(`uni-course-card-${secondCourse.id}`).click()
    await expect(page).toHaveURL(new RegExp(`/#/pages/course/detail\\?id=${secondCourse.id}`))
    await expectUniPageTitle(page, '课程详情')
    await expect(page.getByText(secondCourse.name).first()).toBeVisible()
  })

  test('authenticated review square supports sorting, voting, loading more, and course navigation', async ({
    page,
  }) => {
    const { mutationBodies, mutations, requests } = await mockUniApi(page, {
      authenticated: true,
      mutationDelayMs: 120,
      paginatedReviews: true,
    })

    await gotoUniPage(page, '/#/pages/review/index')

    await expect(page.getByTestId(`uni-review-card-${review.id}`)).toBeVisible()
    await page.getByTestId('uni-review-sort-likes').click()
    await expect
      .poll(() => requests.some((request) => (
        request.startsWith('GET /api/v1/course/review/reviews/latest?') &&
        request.includes('sort=likes')
      )))
      .toBe(true)

    const likeButton = page.getByTestId(`uni-review-like-${review.id}`)
    await expect(likeButton).toContainText('7')
    await likeButton.evaluate((element) => {
      const target = element as HTMLElement
      target.click()
      target.click()
    })
    await expect(likeButton).toContainText('8')
    await expect
      .poll(() => mutations.filter(item => (
        item === `POST /api/v1/course/review/reviews/${review.id}/votes`
      )).length)
      .toBe(1)
    const votePayload = requireMutationBody(
      mutationBodies,
      'POST',
      `/api/v1/course/review/reviews/${review.id}/votes`,
    )
    expect(votePayload).toMatchObject({ voteType: 'like' })

    await likeButton.click()
    await expect(likeButton).toContainText('7')
    await expect(likeButton).toHaveAttribute('aria-pressed', 'false')

    const dislikeButton = page.getByTestId(`uni-review-dislike-${review.id}`)
    await expect(dislikeButton).toContainText('1')
    await dislikeButton.click()
    await expect(likeButton).toContainText('7')
    await expect(dislikeButton).toContainText('2')
    await expect
      .poll(() => mutationBodies
        .filter((item) => item.path === `/api/v1/course/review/reviews/${review.id}/votes`)
        .map((item) => item.body))
      .toEqual([{ voteType: 'like' }, { voteType: 'like' }, { voteType: 'dislike' }])

    await page.getByTestId('uni-review-load-more').click()
    await expect
      .poll(() => requests.some((request) => (
        request.startsWith('GET /api/v1/course/review/reviews/latest?') &&
        request.includes('page=2') &&
        request.includes('sort=likes')
      )))
      .toBe(true)
    await expect(page.getByTestId(`uni-review-card-${secondPageReview.id}`)).toBeVisible()

    await page.getByTestId(`uni-review-view-course-${review.id}`).click()
    await expect(page).toHaveURL(new RegExp(`/#/pages/course/detail\\?id=${course.id}`))
    await expectUniPageTitle(page, '课程详情')
    await expect(page.getByText(course.name).first()).toBeVisible()
  })

  test('review square hydrates the server-authoritative vote and toggles it off', async ({
    page,
  }) => {
    const { mutationBodies } = await mockUniApi(page, {
      authenticated: true,
      reviewUserVote: 'like',
    })

    await gotoUniPage(page, '/#/pages/review/index')

    const likeButton = page.getByTestId(`uni-review-like-${review.id}`)
    await expect(likeButton).toHaveAttribute('aria-pressed', 'true')
    await expect(likeButton).toContainText('7')

    await likeButton.click()

    await expect(likeButton).toHaveAttribute('aria-pressed', 'false')
    await expect(likeButton).toContainText('6')
    await expect
      .poll(() => mutationBodies
        .filter((item) => item.path === `/api/v1/course/review/reviews/${review.id}/votes`)
        .map((item) => item.body))
      .toEqual([{ voteType: 'like' }])
  })

  test('review pagination reports a failure and retries the same page without skipping', async ({
    page,
  }) => {
    const { requests } = await mockUniApi(page, {
      authenticated: true,
      paginatedReviews: true,
      reviewPage2Failures: 1,
    })

    await gotoUniPage(page, '/#/pages/review/index')

    await page.getByTestId('uni-review-load-more').click()
    await expect(page.getByText('评课下一页加载失败')).toBeVisible()
    await expect(page.getByTestId(`uni-review-card-${secondPageReview.id}`)).toHaveCount(0)

    await page.getByTestId('uni-review-load-more').click()
    await expect(page.getByTestId(`uni-review-card-${secondPageReview.id}`)).toBeVisible()
    expect(requests.filter((request) => (
      request.startsWith('GET /api/v1/course/review/reviews/latest?')
      && request.includes('page=2')
    ))).toHaveLength(2)
  })

  test('review sorting ignores a slower stale response', async ({ page }) => {
    await mockUniApi(page, { reviewSortRace: true })

    await gotoUniPage(page, '/#/pages/review/index')

    await page.getByTestId('uni-review-sort-likes').click()
    await page.getByTestId('uni-review-sort-rating').click()
    await expect(page.getByText('当前高分结果')).toBeVisible()
    await page.waitForTimeout(300)
    await expect(page.getByText('当前高分结果')).toBeVisible()
    await expect(page.getByText('过期最热结果')).toHaveCount(0)
  })

  test('review course heading is keyboard operable on H5', async ({ page }) => {
    await mockUniApi(page)

    await gotoUniPage(page, '/#/pages/review/index')

    const heading = page.getByTestId(`uni-review-open-${review.id}`)
    await heading.focus()
    await page.keyboard.press('Enter')
    await expect(page).toHaveURL(new RegExp(`/#/pages/course/detail\\?id=${course.id}`))
  })

  test('authenticated course detail toggles favorites and submits a reply', async ({ page }) => {
    const { mutationBodies, mutations } = await mockUniApi(page, {
      authenticated: true,
      mutationDelayMs: 120,
    })

    await gotoUniPage(page, `/#/pages/course/detail?id=${course.id}`)

    const favoriteButton = page.getByTestId('uni-course-favorite')
    await expect(favoriteButton).toContainText('已收藏')
    await favoriteButton.evaluate((element) => {
      const target = element as HTMLElement
      target.click()
      target.click()
    })
    await expect
      .poll(() => mutations.filter(item => (
        item === `DELETE /api/v1/course/review/courses/${course.id}/favorites`
      )).length)
      .toBe(1)
    await expect(favoriteButton).toContainText('收藏')
    await favoriteButton.click()
    await expect
      .poll(() => mutations.includes(`POST /api/v1/course/review/courses/${course.id}/favorites`))
      .toBe(true)
    await expect(favoriteButton).toContainText('已收藏')

    await page.getByTestId(`uni-review-replies-${review.id}`).click()
    await expect(page.getByText(initialReply.content)).toBeVisible()

    await setUniFieldValue(
      page,
      `uni-review-reply-input-${review.id}`,
      '这是一条 UniAppX H5 端到端回复内容。',
    )
    await page.getByTestId(`uni-review-reply-submit-${review.id}`).evaluate((element) => {
      const target = element as HTMLElement
      target.click()
      target.click()
    })

    await expect
      .poll(() => mutations.filter(item => (
        item === `POST /api/v1/course/review/reviews/${review.id}/replies`
      )).length)
      .toBe(1)
    const replyPayload = requireMutationBody(
      mutationBodies,
      'POST',
      `/api/v1/course/review/reviews/${review.id}/replies`,
    )
    expect(replyPayload).toMatchObject({
      content: '这是一条 UniAppX H5 端到端回复内容。',
    })
    await expect(page.getByText('这是一条 UniAppX H5 端到端回复内容。')).toBeVisible()
  })

  test('guest course detail protected actions redirect to login without mutations', async ({
    page,
  }) => {
    const { mutations } = await mockUniApi(page)

    await gotoUniPage(page, `/#/pages/course/detail?id=${course.id}`)

    await page.getByTestId('uni-course-favorite').click()
    await expect(page).toHaveURL(/\/#\/pages\/auth\/login\?redirect=/)
    await expectUniPageTitle(page, '登录')
    expect(currentLoginRedirect(page)).toBe(`/pages/course/detail?id=${course.id}`)
    expect(mutations).not.toContain(
      `POST /api/v1/course/review/courses/${course.id}/favorites`,
    )
    expect(mutations).not.toContain(
      `DELETE /api/v1/course/review/courses/${course.id}/favorites`,
    )

    await gotoUniPage(page, `/#/pages/course/detail?id=${course.id}`)

    await page.getByText('写评课').click()
    await expect(page).toHaveURL(/\/#\/pages\/auth\/login\?redirect=/)
    expect(currentLoginRedirect(page)).toBe(`/pages/course/detail?id=${course.id}`)

    await gotoUniPage(page, `/#/pages/course/detail?id=${course.id}`)

    await page.getByTestId(`uni-review-replies-${review.id}`).click()
    await expect(page.getByText(initialReply.content)).toBeVisible()
    await setUniFieldValue(
      page,
      `uni-review-reply-input-${review.id}`,
      '游客不应提交这条回复。',
    )
    await page.getByTestId(`uni-review-reply-submit-${review.id}`).click()
    await expect(page).toHaveURL(/\/#\/pages\/auth\/login\?redirect=/)
    expect(currentLoginRedirect(page)).toBe(`/pages/course/detail?id=${course.id}`)
    expect(mutations).not.toContain(
      `POST /api/v1/course/review/reviews/${review.id}/replies`,
    )
  })

  test('guest direct review post route redirects to login before draft access', async ({
    page,
  }) => {
    const { mutations, requests } = await mockUniApi(page)

    await gotoUniPage(page, `/#/pages/review/post?courseID=${course.id}`)

    await expect(page).toHaveURL(/\/#\/pages\/auth\/login\?redirect=/)
    await expectUniPageTitle(page, '登录')
    expect(currentLoginRedirect(page)).toBe(`/pages/review/post?courseID=${course.id}`)
    expect(requests).not.toContain('GET /api/v1/course/review/drafts')
    expect(mutations).not.toContain('POST /api/v1/course/review/drafts')
  })

  test('authenticated review post page loads form data and saves a draft', async ({
    page,
  }) => {
    const { mutationBodies, mutations } = await mockUniApi(page, {
      authenticated: true,
      mutationDelayMs: 120,
    })

    await gotoUniPage(page, `/#/pages/review/post?courseID=${course.id}`)

    await expectUniPageTitle(page, '发布评课')
    await expect(page.getByText('发布评课').first()).toBeVisible()
    await expect(page.getByText(course.name)).toBeVisible()
    await expect(page.getByText(term.name).last()).toBeVisible()
    await expect(page.getByText('综合体验')).toBeVisible()

    await selectUniPickerIndex(page, 'uni-review-teacher-picker', 0)
    await expect(page.getByTestId('uni-review-teacher-value')).toContainText(teacher.teacherName)
    await setUniFieldValue(page, 'uni-review-title', '移动端草稿标题')
    await setUniFieldValue(page, 'uni-review-grade', 'A')
    await setUniFieldValue(
      page,
      'uni-review-content',
      '这是一条用于 UniAppX H5 端到端测试的草稿正文。',
    )
    await page.getByTestId('uni-review-rating-overall-5').click()
    await page.getByTestId('uni-review-rating-workload-4').click()
    await page.getByTestId('uni-review-save-draft').evaluate((element) => {
      const target = element as HTMLElement
      target.click()
      target.click()
    })
    await expect
      .poll(() => mutations.filter(item => item === 'POST /api/v1/course/review/drafts').length)
      .toBe(1)
    await expect(page.getByTestId('uni-review-save-draft')).toBeEnabled()
    const draftPayload = requireMutationBody(
      mutationBodies,
      'POST',
      '/api/v1/course/review/drafts',
    )
    expect(draftPayload).toMatchObject({
      content: '这是一条用于 UniAppX H5 端到端测试的草稿正文。',
      courseID: course.id,
      grade: 'A',
      ratings: { overall: 5, workload: 4 },
      teacherID: teacher.teacherID,
      termID: term.id,
      title: '移动端草稿标题',
    })

    await page.getByTestId('uni-review-submit').click()
    await expect
      .poll(() => mutations.includes('DELETE /api/v1/course/review/drafts'))
      .toBe(true)
  })

  test('authenticated review post page submits a complete review', async ({ page }) => {
    const { mutationBodies, mutations } = await mockUniApi(page, {
      authenticated: true,
      mutationDelayMs: 120,
      reviewDraft: {
        id: 'draft-current-course',
        courseID: course.id,
        ratings: {},
        updatedAt: now,
      },
    })

    await gotoUniPage(page, `/#/pages/review/post?courseID=${course.id}`)

    await selectUniPickerIndex(page, 'uni-review-teacher-picker', 0)
    await expect(page.getByTestId('uni-review-teacher-value')).toContainText(teacher.teacherName)
    await setUniFieldValue(page, 'uni-review-title', '移动端提交评课验证')
    await setUniFieldValue(page, 'uni-review-grade', 'A')
    await setUniFieldValue(
      page,
      'uni-review-content',
      '这是一条用于 UniAppX H5 端到端测试的完整评课正文，长度足够通过提交校验。',
    )
    await page.getByTestId('uni-review-rating-overall-5').click()
    await page.getByTestId('uni-review-rating-workload-4').click()

    const submitButton = page.getByTestId('uni-review-submit')
    await expect(submitButton).toBeEnabled()
    await submitButton.evaluate((element) => {
      const target = element as HTMLElement
      target.click()
      target.click()
    })

    await expect
      .poll(() => mutations.filter(item => item === 'POST /api/v1/course/review/reviews').length)
      .toBe(1)
    const payload = requireMutationBody(
      mutationBodies,
      'POST',
      '/api/v1/course/review/reviews',
    )
    expect(payload).toMatchObject({
      courseID: course.id,
      grade: 'A',
      ratings: { overall: 5, workload: 4 },
      teacherID: teacher.teacherID,
      termID: term.id,
      title: '移动端提交评课验证',
    })
    expect(payload.content).toBe('这是一条用于 UniAppX H5 端到端测试的完整评课正文，长度足够通过提交校验。')
    await expect
      .poll(() => mutations.includes('DELETE /api/v1/course/review/drafts'))
      .toBe(true)

    await expect(page).toHaveURL(new RegExp(`/#/pages/course/detail\\?id=${course.id}`))
    await expect(page.getByText(course.name).first()).toBeVisible()
  })

  test('review submission preserves a draft bound to another course', async ({ page }) => {
    const { mutations } = await mockUniApi(page, {
      authenticated: true,
      reviewDraft: {
        id: 'draft-foreign-course',
        courseID: secondCourse.id,
        teacherID: teacher.teacherID,
        termID: term.id,
        title: '另一门课程的草稿标题',
        content: '另一门课程的草稿正文，不应载入当前课程。',
        ratings: { overall: 5, workload: 4 },
        updatedAt: now,
      },
    })

    await gotoUniPage(page, `/#/pages/review/post?courseID=${course.id}`)

    await expect(page.getByTestId('uni-review-title').locator('input')).toHaveValue('')
    await expect(page.getByTestId('uni-review-content').locator('textarea')).toHaveValue('')
    await expect(page.getByTestId('uni-review-teacher-value')).toContainText('请选择教师')

    await setUniFieldValue(page, 'uni-review-title', '当前课程的新评课')
    await setUniFieldValue(
      page,
      'uni-review-content',
      '这是当前课程的新评课正文，提交成功后不能删除另一门课程的草稿。',
    )
    await page.getByTestId('uni-review-rating-overall-5').click()
    await page.getByTestId('uni-review-rating-workload-4').click()
    await page.getByTestId('uni-review-submit').click()

    await expect
      .poll(() => mutations.includes('POST /api/v1/course/review/reviews'))
      .toBe(true)
    await expect(page).toHaveURL(new RegExp(`/#/pages/course/detail\\?id=${course.id}`))
    expect(mutations).not.toContain('DELETE /api/v1/course/review/drafts')
  })

  test('unbound drafts restore content without restoring a teacher', async ({ page }) => {
    const { mutations } = await mockUniApi(page, {
      authenticated: true,
      reviewDraft: {
        id: 'draft-unbound',
        teacherID: teacher.teacherID,
        termID: term.id,
        title: '未绑定课程的草稿',
        content: '这是一份未绑定课程的通用草稿正文，可以恢复到当前课程。',
        ratings: { overall: 5, workload: 4 },
        updatedAt: now,
      },
    })

    await gotoUniPage(page, `/#/pages/review/post?courseID=${course.id}`)

    await expect(page.getByTestId('uni-review-title').locator('input')).toHaveValue(
      '未绑定课程的草稿',
    )
    await expect(page.getByTestId('uni-review-content').locator('textarea')).toHaveValue(
      '这是一份未绑定课程的通用草稿正文，可以恢复到当前课程。',
    )
    await expect(page.getByTestId('uni-review-teacher-value')).toContainText('请选择教师')

    await page.getByTestId('uni-review-submit').click()
    await expect
      .poll(() => mutations.includes('DELETE /api/v1/course/review/drafts'))
      .toBe(true)
  })

  test('authenticated user center pages render profile data and user lists', async ({
    page,
  }) => {
    const { mutations } = await mockUniApi(page, { authenticated: true })

    await gotoUniPage(page, '/#/pages/user/index')
    await expectUniPageTitle(page, '个人中心')
    await expect(page.getByText(user.displayName)).toBeVisible()
    await expect(page.getByText('认证概览')).toBeVisible()
    await expect(page.getByText('学生认证已通过')).toBeVisible()
    await expect(page.getByTestId('uni-user-phone-summary')).toContainText('已绑定')

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

  test('user-center menu is keyboard operable while unavailable verification actions stay inert', async ({
    page,
  }) => {
    await mockUniApi(page, { authenticated: true })

    await gotoUniPage(page, '/#/pages/user/index')

    const approvedStudent = page.getByTestId('uni-user-student-summary')
    await expect(approvedStudent).toHaveAttribute('role', 'button')
    await expect(approvedStudent).toHaveAttribute('aria-disabled', 'true')
    await approvedStudent.focus()
    await page.keyboard.press('Enter')
    await expect(page).toHaveURL(/\/#\/pages\/user\/index$/)

    const approvedPhone = page.getByTestId('uni-user-phone-summary')
    await expect(approvedPhone).toHaveAttribute('role', 'button')
    await expect(approvedPhone).toHaveAttribute('aria-disabled', 'true')

    const reviewsMenu = page.getByRole('button', { name: /我的评课/ })
    await reviewsMenu.focus()
    await page.keyboard.press('Enter')
    await expect(page).toHaveURL(/\/#\/pages\/user\/reviews$/)
    await expect(page.getByText(review.title)).toBeVisible()
  })

  test('authenticated user center opens independent Web student verification when no credential exists', async ({
    page,
  }) => {
    await mockWebVerificationTarget(page)
    await mockUniApi(page, {
      authenticated: true,
      phoneVerified: false,
      userSurface: {
        displayName: user.displayName,
        studentVerificationStatus: 'none',
        phoneBound: false,
        phone: null,
        capabilities: user.capabilities,
      },
    })

    await gotoUniPage(page, '/#/pages/user/index')

    await expectUniPageTitle(page, '个人中心')
    await expect(page.getByTestId('uni-user-student-summary')).toContainText('未完成学生认证')
    await expect(page.getByTestId('uni-user-student-summary')).toContainText('点击去认证')
    await expect(page.getByTestId('uni-user-phone-summary')).toContainText('未绑定')

    await page.getByTestId('uni-user-student-summary').click()
    await page.waitForURL(`${webVerificationOrigin}/user/student-verification`)
    await expect(page.getByText('Mock Web Verification')).toBeVisible()
  })

  test('authenticated user center opens Web phone binding independently of student status', async ({
    page,
  }) => {
    await mockWebVerificationTarget(page)
    await mockUniApi(page, {
      authenticated: true,
      phoneVerified: false,
      userSurface: {
        displayName: user.displayName,
        studentVerificationStatus: 'approved',
        phoneBound: false,
        phone: null,
        capabilities: user.capabilities,
      },
    })

    await gotoUniPage(page, '/#/pages/user/index')

    await expectUniPageTitle(page, '个人中心')
    await expect(page.getByTestId('uni-user-student-summary')).toContainText('学生认证已通过')
    await expect(page.getByTestId('uni-user-phone-summary')).toContainText('未绑定')
    await expect(page.getByTestId('uni-user-phone-summary')).toContainText('点击去认证')

    await page.getByTestId('uni-user-student-summary').focus()
    await page.keyboard.press('Enter')
    await expect(page).toHaveURL(/\/#\/pages\/user\/index$/)

    await page.getByTestId('uni-user-phone-summary').click()
    await page.waitForURL(`${webVerificationOrigin}/user/phone-binding`)
    await expect(page.getByText('Mock Web Verification')).toBeVisible()
  })

  test('authenticated user lists paginate and open linked courses', async ({ page }) => {
    const { requests } = await mockUniApi(page, {
      authenticated: true,
      paginatedUserLists: true,
    })

    await gotoUniPage(page, '/#/pages/user/reviews')
    await page.getByTestId('uni-user-reviews-load-more').click()
    await expect
      .poll(() => requests.some((request) => (
        request.startsWith('GET /api/v1/course/review/user/reviews?') &&
        request.includes('page=2') &&
        request.includes('pageSize=20')
      )))
      .toBe(true)
    await expect(page.getByTestId(`uni-user-review-card-${secondUserReview.id}`)).toBeVisible()
    await page.getByTestId(`uni-user-review-card-${secondUserReview.id}`).click()
    await expect(page).toHaveURL(new RegExp(`/#/pages/course/detail\\?id=${secondCourse.id}`))
    await expect(page.getByText(secondCourse.name).first()).toBeVisible()

    await gotoUniPage(page, '/#/pages/user/votes')
    await page.getByTestId('uni-user-votes-load-more').click()
    await expect
      .poll(() => requests.some((request) => (
        request.startsWith('GET /api/v1/course/review/user/votes?') &&
        request.includes('page=2') &&
        request.includes('pageSize=20') &&
        request.includes('voteType=like')
      )))
      .toBe(true)
    await expect(page.getByTestId(`uni-user-vote-card-${secondUserVote.id}`)).toBeVisible()
    await page.getByTestId(`uni-user-vote-card-${secondUserVote.id}`).click()
    await expect(page).toHaveURL(new RegExp(`/#/pages/course/detail\\?id=${secondCourse.id}`))
    await expect(page.getByText(secondCourse.name).first()).toBeVisible()

    await gotoUniPage(page, '/#/pages/user/favorites')
    await page.getByTestId('uni-user-favorites-load-more').click()
    await expect
      .poll(() => requests.some((request) => (
        request.startsWith('GET /api/v1/course/review/user/favorites?') &&
        request.includes('page=2') &&
        request.includes('pageSize=20')
      )))
      .toBe(true)
    await expect(page.getByTestId(`uni-user-favorite-card-${secondFavorite.id}`)).toBeVisible()
    await page.getByTestId(`uni-user-favorite-card-${secondFavorite.id}`).click()
    await expect(page).toHaveURL(new RegExp(`/#/pages/course/detail\\?id=${secondCourse.id}`))
    await expect(page.getByText(secondCourse.name).first()).toBeVisible()
  })

  test('authenticated notifications support paging and read actions', async ({ page }) => {
    const { mutations, requests } = await mockUniApi(page, {
      authenticated: true,
      mutationDelayMs: 120,
      paginatedUserLists: true,
    })

    await gotoUniPage(page, '/#/pages/user/notifications')

    await page.getByTestId('uni-notification-load-more').click()
    await expect
      .poll(() => requests.some((request) => (
        request.startsWith('GET /api/v1/course/review/user/notifications?') &&
        request.includes('page=2') &&
        request.includes('pageSize=20')
      )))
      .toBe(true)
    await expect(page.getByTestId(`uni-notification-card-${secondNotification.id}`)).toBeVisible()

    await expect(page.getByTestId(`uni-notification-unread-${notification.id}`)).toBeVisible()
    await page.getByTestId(`uni-notification-card-${notification.id}`).evaluate((element) => {
      const target = element as HTMLElement
      target.click()
      target.click()
    })
    await expect
      .poll(() => mutations.filter(item => (
        item === `PUT /api/v1/course/review/user/notifications/${notification.id}/read`
      )).length)
      .toBe(1)
    await expect(page.getByTestId(`uni-notification-unread-${notification.id}`)).toHaveCount(0)

    await page.getByTestId('uni-notification-mark-all').evaluate((element) => {
      const target = element as HTMLElement
      target.click()
      target.click()
    })
    await expect
      .poll(() => mutations.filter(item => (
        item === 'PUT /api/v1/course/review/user/notifications/read-all'
      )).length)
      .toBe(1)
    await expect(page.locator('[data-testid^="uni-notification-unread-"]')).toHaveCount(0)
  })

  test('authenticated user center logs out and returns to guest actions', async ({ page }) => {
    const { mutations } = await mockUniApi(page, { authenticated: true })

    await gotoUniPage(page, '/#/pages/user/index')

    await expect(page.getByTestId('uni-user-logout')).toBeVisible()
    await page.getByTestId('uni-user-logout').click()
    await expect
      .poll(() => mutations.includes('POST /api/v1/auth/logout'))
      .toBe(true)
    await expect(page.getByTestId('uni-user-login')).toBeVisible()
  })

  test('guest user menu sends protected entries to login with redirect', async ({ page }) => {
    const { requests } = await mockUniApi(page)
    await page.route('https://sso.example.test/**', async (route) => {
      await route.fulfill({
        contentType: 'text/html',
        body: '<!doctype html><title>Mock SSO</title><main>Mock SSO</main>',
      })
    })

    await gotoUniPage(page, '/#/pages/user/index')

    await expect(page.getByTestId('uni-user-login')).toBeVisible()
    await page.getByText('我的评课').click()
    await expect(page).toHaveURL(/\/#\/pages\/auth\/login\?redirect=/)
    await expectUniPageTitle(page, '登录')
    await expect(page.getByText('登录 StuHelper')).toBeVisible()

    await Promise.all([
      page.waitForURL('https://sso.example.test/login?state=mobile-e2e-state'),
      page.getByText('使用校园 SSO 登录').click(),
    ])

    const loginRequest = requests.find((request) => request.startsWith('GET /api/v1/auth/login?'))
    expect(loginRequest).toBeDefined()
    const loginURL = new URL(`http://uniappx.test${loginRequest?.slice('GET '.length)}`)
    expect(loginURL.searchParams.get('app')).toBe('uniapp')
    expect(loginURL.searchParams.get('platform')).toBeNull()
    expect(loginURL.searchParams.get('redirect')).toBe('/pages/user/reviews')

    await expect(page.getByText('Mock SSO')).toBeVisible()
  })

  test('guest review vote sends the current page to login redirect', async ({ page }) => {
    const { mutations, requests } = await mockUniApi(page)
    await page.route('https://sso.example.test/**', async (route) => {
      await route.fulfill({
        contentType: 'text/html',
        body: '<!doctype html><title>Mock SSO</title><main>Mock SSO</main>',
      })
    })

    await gotoUniPage(page, '/#/pages/review/index')
    await expect(page.getByTestId(`uni-review-card-${review.id}`)).toBeVisible()

    await page.getByTestId(`uni-review-like-${review.id}`).click()

    await expect(page).toHaveURL(/\/#\/pages\/auth\/login\?redirect=/)
    await expectUniPageTitle(page, '登录')
    expect(mutations).not.toContain(`POST /api/v1/course/review/reviews/${review.id}/votes`)

    await Promise.all([
      page.waitForURL('https://sso.example.test/login?state=mobile-e2e-state'),
      page.getByText('使用校园 SSO 登录').click(),
    ])

    const loginRequest = requests.find((request) => request.startsWith('GET /api/v1/auth/login?'))
    expect(loginRequest).toBeDefined()
    const loginURL = new URL(`http://uniappx.test${loginRequest?.slice('GET '.length)}`)
    expect(loginURL.searchParams.get('app')).toBe('uniapp')
    expect(loginURL.searchParams.get('platform')).toBeNull()
    expect(loginURL.searchParams.get('redirect')).toBe('/pages/review/index')

    await expect(page.getByText('Mock SSO')).toBeVisible()
  })

  test('auth pages render login and callback error states', async ({ page }) => {
    const { mutations, requests } = await mockUniApi(page)

    await gotoUniPage(page, '/#/pages/auth/login')
    await expectUniPageTitle(page, '登录')
    await expect(page.getByText('登录 StuHelper')).toBeVisible()
    await expect(page.getByText('使用校园 SSO 登录')).toBeVisible()
    expect(requests.some((request) => request.startsWith('GET /api/v1/auth/me'))).toBe(false)
    expect(mutations).not.toContain('POST /api/v1/auth/refresh')

    await gotoUniPage(page, '/#/pages/auth/callback')
    await expectUniPageTitle(page, 'SSO 回调')
    await expect(page.getByText('登录失败')).toBeVisible()
    await expect(page.getByText('回调参数缺失，请重新登录')).toBeVisible()
  })

  test('auth login button starts campus SSO and preserves redirect', async ({ page }) => {
    const { requests } = await mockUniApi(page)
    await page.route('https://sso.example.test/**', async (route) => {
      await route.fulfill({
        contentType: 'text/html',
        body: '<!doctype html><title>Mock SSO</title><main>Mock SSO</main>',
      })
    })

    await gotoUniPage(
      page,
      '/#/pages/auth/login?redirect=%2Fpages%2Freview%2Fpost%3FcourseID%3D101',
    )

    await Promise.all([
      page.waitForURL('https://sso.example.test/login?state=mobile-e2e-state'),
      page.getByText('使用校园 SSO 登录').click(),
    ])

    const loginRequest = requests.find((request) => request.startsWith('GET /api/v1/auth/login?'))
    expect(loginRequest).toBeDefined()
    const loginURL = new URL(`http://uniappx.test${loginRequest?.slice('GET '.length)}`)
    expect(loginURL.searchParams.get('app')).toBe('uniapp')
    expect(loginURL.searchParams.get('platform')).toBeNull()
    expect(loginURL.searchParams.get('redirect')).toBe('/pages/review/post?courseID=101')

    await expect(page.getByText('Mock SSO')).toBeVisible()
  })

  test('auth callback rejects mismatched native state before token exchange', async ({
    page,
  }) => {
    const { mutations } = await mockUniApi(page, {
      ssoState: 'mobile-e2e-state',
    })

    await gotoUniPage(page, '/#/pages/auth/callback?code=mobile-e2e-code&state=wrong-state')

    await expectUniPageTitle(page, 'SSO 回调')
    await expect(page.getByText('登录失败')).toBeVisible()
    await expect(page.getByText('安全校验失败，请重新登录')).toBeVisible()
    expect(mutations).not.toContain('POST /api/v1/auth/exchange-native')
    await expect
      .poll(() => page.evaluate(() => localStorage.getItem('stuhelper:sso-state')))
      .toBeNull()

    await page.getByText('重新登录', { exact: true }).click()
    await expect(page).toHaveURL(/\/#\/pages\/auth\/login/)
    await expectUniPageTitle(page, '登录')
  })

  test('auth callback exchanges native code and opens the user center', async ({ page }) => {
    const { mutations } = await mockUniApi(page, {
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
