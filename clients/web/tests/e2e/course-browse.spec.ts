import { test, expect, type Page } from './fixtures'

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

async function expectLoginRedirect(page: Page, redirect: string) {
  await expect(page).toHaveURL(/\/login\?/)
  const url = new URL(page.url())
  expect(url.pathname).toBe('/login')
  expect(url.searchParams.get('redirect')).toBe(redirect)
}

test.describe('Course Browse Flow', () => {
  test.beforeEach(async ({ page }) => {
    await mockUnauthenticated(page)

    await page.route('**/api/v1/course/departments*', (route) =>
      route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: [
            { id: 1, name: '计算机科学与技术学院', code: 'CS' },
            { id: 2, name: '数学科学学院', code: 'MATH' },
          ],
        }),
      }),
    )

    await page.route('**/api/v1/course/terms*', (route) =>
      route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: [
            { id: '2025-fall', name: '2025 秋', isCurrent: true },
            { id: '2025-spring', name: '2025 春', isCurrent: false },
          ],
        }),
      }),
    )

    await page.route('**/api/v1/course/categories*', (route) =>
      route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify({ success: true, data: [] }),
      }),
    )

    await page.route('**/api/v1/course/courses?*', (route) =>
      route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: {
            list: [
              {
                id: 1,
                name: '高等数学A',
                code: 'MATH101',
                departmentID: 2,
                departmentName: '数学科学学院',
                credits: 4,
                reviewCount: 15,
              },
              {
                id: 2,
                name: '数据结构',
                code: 'CS201',
                departmentID: 1,
                departmentName: '计算机科学与技术学院',
                credits: 4,
                reviewCount: 23,
              },
            ],
            total: 2,
          },
        }),
      }),
    )

    await page.route('**/api/v1/course/courses/grouped', (route) =>
      route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: {
            groups: [
              {
                departmentID: 2,
                departmentName: '数学科学学院',
                courses: [
                  {
                    id: 1,
                    name: '高等数学A',
                    code: 'MATH101',
                    departmentID: 2,
                    departmentName: '数学科学学院',
                    credits: 4,
                    reviewCount: 15,
                  },
                ],
              },
              {
                departmentID: 1,
                departmentName: '计算机科学与技术学院',
                courses: [
                  {
                    id: 2,
                    name: '数据结构',
                    code: 'CS201',
                    departmentID: 1,
                    departmentName: '计算机科学与技术学院',
                    credits: 4,
                    reviewCount: 23,
                  },
                ],
              },
            ],
          },
        }),
      }),
    )
  })

  test('course list page loads and shows course names', async ({ page }) => {
    await page.goto('/courses/list')
    await page.waitForLoadState('networkidle')

    // Assert actual rendered content from mocked data
    await expect(page.getByText('高等数学A')).toBeVisible({ timeout: 10_000 })
    await expect(page.getByText('数据结构')).toBeVisible()
  })

  test('course hub search finds courses by common Chinese abbreviations', async ({
    page,
  }) => {
    await page.goto('/courses')
    await page.waitForLoadState('networkidle')

    await page
      .getByRole('textbox', {
        name: /搜索课程名称、拼音或首字母/,
      })
      .fill('高数')

    await expect(
      page.getByRole('option', {
        name: /高等数学A.*数学科学学院.*15条测评/,
      }),
    ).toBeVisible()
  })

  test('invalid grouped course response fails closed and can retry', async ({
    page,
  }) => {
    let loadCount = 0

    await page.route('**/api/v1/course/courses/grouped', async (route) => {
      loadCount += 1
      await route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify(
          loadCount === 1
            ? {
                success: true,
                data: {
                  groups: [
                    {
                      departmentID: 2,
                      departmentName: '数学科学学院',
                      courses: [
                        {
                          id: 1,
                          name: '高等数学A',
                          code: 'MATH101',
                          departmentID: 2,
                          departmentName: '数学科学学院',
                          credits: '4',
                          reviewCount: 15,
                        },
                      ],
                    },
                  ],
                },
              }
            : {
                success: true,
                data: {
                  groups: [
                    {
                      departmentID: 2,
                      departmentName: '数学科学学院',
                      courses: [
                        {
                          id: 1,
                          name: '高等数学A',
                          code: 'MATH101',
                          departmentID: 2,
                          departmentName: '数学科学学院',
                          credits: 4,
                          reviewCount: 15,
                        },
                      ],
                    },
                  ],
                },
              },
        ),
      })
    })

    await page.goto('/courses/list')

    await expect(page.getByText('获取课程列表失败，请稍后重试')).toBeVisible({
      timeout: 10_000,
    })
    await expect(page.getByText('没有获取到任何课程数据')).toHaveCount(0)

    await page.getByRole('button', { name: '重试' }).click()

    await expect.poll(() => loadCount).toBe(2)
    await expect(page.getByText('高等数学A')).toBeVisible()
  })

  test('course detail page loads and shows course info', async ({ page }) => {
    await page.route('**/api/v1/course/courses/1', (route) =>
      route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: {
            id: 1,
            name: '高等数学A',
            code: 'MATH101',
            departmentID: 2,
            departmentName: '数学科学学院',
            credits: 4,
            reviewCount: 15,
          },
        }),
      }),
    )

    await page.route('**/api/v1/course/review/courses/1/reviews*', (route) =>
      route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify({ success: true, data: { list: [], total: 0 } }),
      }),
    )

    await page.route(
      '**/api/v1/course/review/courses/1/rating-stats*',
      (route) =>
        route.fulfill({
          contentType: 'application/json',
          body: JSON.stringify({
            success: true,
            data: {
              courseID: 1,
              overall: { termName: '总体', dimensions: [] },
              byTerm: [],
              allDimensionKeys: [],
            },
          }),
        }),
    )

    await page.route(
      '**/api/v1/course/review/courses/1/teachers*',
      (route) =>
        route.fulfill({
          contentType: 'application/json',
          body: JSON.stringify({ success: true, data: [] }),
        }),
    )
    await page.route(
      '**/api/v1/course/review/courses/1/rating-trend*',
      (route) =>
        route.fulfill({
          contentType: 'application/json',
          body: JSON.stringify({ success: true, data: { trend: [] } }),
        }),
    )

    await page.goto('/courses/1')
    await page.waitForLoadState('networkidle')

    // Assert course name and department rendered from mocked data
    await expect(page.getByText('高等数学A')).toBeVisible({ timeout: 10_000 })
    await expect(page.getByText('数学科学学院')).toBeVisible()
    await expect(page).toHaveTitle(/高等数学A - StuHelper/)
  })

  test('course detail teacher filter requests server pages with teacher id', async ({
    page,
  }) => {
    const reviewRequests: URL[] = []
    const wangReview = {
      id: 'teacher-filter-wang',
      courseID: 8,
      courseName: '教师筛选课程',
      teacherID: 1,
      teacherName: '王老师',
      termID: '2026-spring',
      termName: '2026 春',
      title: '王老师评价',
      content: '第一页只有王老师，不能代表所有教师筛选结果。',
      ratings: { recommendation: 4, workload: 3 },
      likeCount: 1,
      dislikeCount: 0,
      replyCount: 0,
      status: 'published',
      createdAt: '2026-05-24T04:00:00Z',
    }
    const liReviewPageOne = {
      ...wangReview,
      id: 'teacher-filter-li-1',
      teacherID: 2,
      teacherName: '李老师',
      title: '李老师第一页评价',
    }
    const liReviewPageTwo = {
      ...liReviewPageOne,
      id: 'teacher-filter-li-2',
      title: '李老师第二页评价',
    }

    await page.route('**/api/v1/course/courses/8', (route) =>
      route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: {
            id: 8,
            name: '教师筛选课程',
            code: 'FILTER101',
            departmentID: 1,
            departmentName: '测试学院',
            credits: 2,
            reviewCount: 3,
          },
        }),
      }),
    )

    await page.route('**/api/v1/course/review/courses/8/reviews*', (route) => {
      const url = new URL(route.request().url())
      reviewRequests.push(url)
      const teacherID = url.searchParams.get('teacherID')
      const pageNo = url.searchParams.get('page')
      const list =
        teacherID === '2'
          ? pageNo === '2'
            ? [liReviewPageTwo]
            : [liReviewPageOne]
          : [wangReview]
      return route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: { list, total: teacherID === '2' ? 2 : 3 },
        }),
      })
    })

    await page.route(
      '**/api/v1/course/review/courses/8/rating-stats*',
      (route) =>
        route.fulfill({
          contentType: 'application/json',
          body: JSON.stringify({
            success: true,
            data: {
              courseID: 8,
              overall: { termName: '总体', dimensions: [] },
              byTerm: [],
              allDimensionKeys: [],
            },
          }),
        }),
    )

    await page.route(
      '**/api/v1/course/review/courses/8/teachers*',
      (route) =>
        route.fulfill({
          contentType: 'application/json',
          body: JSON.stringify({
            success: true,
            data: [
              {
                teacherID: 1,
                teacherName: '王老师',
                departmentName: '测试学院',
                courseCount: 1,
                reviewCount: 1,
              },
              {
                teacherID: 2,
                teacherName: '李老师',
                departmentName: '测试学院',
                courseCount: 1,
                reviewCount: 2,
              },
            ],
          }),
        }),
    )

    await page.route(
      '**/api/v1/course/review/courses/8/rating-trend*',
      (route) =>
        route.fulfill({
          contentType: 'application/json',
          body: JSON.stringify({ success: true, data: { trend: [] } }),
        }),
    )

    await page.goto('/courses/8/reviews')
    await page.waitForLoadState('networkidle')

    await expect(page.getByText('王老师评价')).toBeVisible({
      timeout: 10_000,
    })

    await page.getByRole('button', { name: '李老师' }).click()

    await expect(page.getByText('李老师第一页评价')).toBeVisible()
    await expect(page.getByText('王老师评价')).toHaveCount(0)
    await expect
      .poll(() =>
        reviewRequests.some(
          (url) =>
            url.searchParams.get('teacherID') === '2' &&
            url.searchParams.get('page') === '1',
        ),
      )
      .toBe(true)

    await page.getByRole('button', { name: /加载更多|Load more/ }).click()

    await expect(page.getByText('李老师第二页评价')).toBeVisible()
    await expect
      .poll(() =>
        reviewRequests.some(
          (url) =>
            url.searchParams.get('teacherID') === '2' &&
            url.searchParams.get('page') === '2',
        ),
      )
      .toBe(true)
  })

  test('course detail review list failure is retryable and not shown as empty', async ({
    page,
  }) => {
    let reviewLoads = 0

    await page.route('**/api/v1/course/courses/9', (route) =>
      route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: {
            id: 9,
            name: '评课失败测试课',
            code: 'FAIL101',
            departmentID: 1,
            departmentName: '测试学院',
            credits: 2,
            reviewCount: 1,
          },
        }),
      }),
    )

    await page.route('**/api/v1/course/review/courses/9/reviews*', (route) => {
      reviewLoads += 1
      return route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify(
          reviewLoads === 1
            ? { success: true, data: { list: null, total: 1 } }
            : {
                success: true,
                data: {
                  list: [
                    {
                      id: 'review-retry-success',
                      courseID: 9,
                      courseName: '评课失败测试课',
                      teacherID: 3,
                      teacherName: '周老师',
                      termID: '2026-spring',
                      termName: '2026 春',
                      title: '重试后出现的评价',
                      content: '评课列表失败后应能单独重试恢复。',
                      ratings: { recommendation: 5, workload: 3 },
                      likeCount: 0,
                      dislikeCount: 0,
                      replyCount: 0,
                      status: 'published',
                      createdAt: '2026-05-24T04:00:00Z',
                    },
                  ],
                  total: 1,
                },
              },
        ),
      })
    })

    await page.route(
      '**/api/v1/course/review/courses/9/rating-stats*',
      (route) =>
        route.fulfill({
          contentType: 'application/json',
          body: JSON.stringify({
            success: true,
            data: {
              courseID: 9,
              overall: { termName: '总体', dimensions: [] },
              byTerm: [],
              allDimensionKeys: [],
            },
          }),
        }),
    )

    await page.route(
      '**/api/v1/course/review/courses/9/teachers*',
      (route) =>
        route.fulfill({
          contentType: 'application/json',
          body: JSON.stringify({ success: true, data: [] }),
        }),
    )

    await page.route(
      '**/api/v1/course/review/courses/9/rating-trend*',
      (route) =>
        route.fulfill({
          contentType: 'application/json',
          body: JSON.stringify({ success: true, data: { trend: [] } }),
        }),
    )

    await page.goto('/courses/9/reviews')
    await page.waitForLoadState('networkidle')

    await expect(page.getByText('评课失败测试课')).toBeVisible({
      timeout: 10_000,
    })
    await expect(
      page.getByRole('alert').filter({ hasText: /Load failed|加载失败/i }).first(),
    ).toBeVisible()
    await expect(page.getByText(/^(暂无测评|No reviews yet)$/)).toHaveCount(0)

    await page.getByRole('button', { name: /重试|Retry/ }).click()

    await expect.poll(() => reviewLoads).toBe(2)
    await expect(page.getByText('重试后出现的评价')).toBeVisible()
  })

  test('guest readers can view replies without seeing a reply editor', async ({
    page,
  }) => {
    await page.route('**/api/v1/course/courses/7', (route) =>
      route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: {
            id: 7,
            name: '数据结构',
            code: 'CS201',
            departmentID: 1,
            departmentName: '计算机科学与技术学院',
            credits: 4,
            reviewCount: 1,
          },
        }),
      }),
    )

    await page.route('**/api/v1/course/review/courses/7/reviews*', (route) =>
      route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: {
            list: [
              {
                id: 'guest-reply-review',
                courseID: 7,
                courseName: '数据结构',
                teacherID: 11,
                teacherName: '张老师',
                termID: '2026-spring',
                termName: '2026 春',
                title: '有回复的评价',
                content: '游客可看到预览，但不能直接编辑回复。',
                ratings: { recommendation: 5, workload: 3 },
                likeCount: 3,
                dislikeCount: 0,
                replyCount: 1,
                status: 'published',
                createdAt: '2026-05-24T04:00:00Z',
              },
            ],
            total: 1,
          },
        }),
      }),
    )

    await page.route(
      '**/api/v1/course/review/reviews/guest-reply-review/replies',
      (route) =>
        route.fulfill({
          contentType: 'application/json',
          body: JSON.stringify({
            success: true,
            data: {
              list: [
                {
                  id: 'guest-visible-reply',
                  reviewID: 'guest-reply-review',
                  parentID: null,
                  content: '这条回复游客应该可以直接阅读。',
                  likeCount: 0,
                  status: 'published',
                  isOwner: false,
                  createdAt: '2026-05-24T04:05:00Z',
                  updatedAt: '2026-05-24T04:05:00Z',
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
      '**/api/v1/course/review/courses/7/rating-stats*',
      (route) =>
        route.fulfill({
          contentType: 'application/json',
          body: JSON.stringify({
            success: true,
            data: {
              courseID: 7,
              overall: { termName: '总体', dimensions: [] },
              byTerm: [],
              allDimensionKeys: [],
            },
          }),
        }),
    )

    await page.route(
      '**/api/v1/course/review/courses/7/teachers*',
      (route) =>
        route.fulfill({
          contentType: 'application/json',
          body: JSON.stringify({ success: true, data: [] }),
        }),
    )

    await page.route(
      '**/api/v1/course/review/courses/7/rating-trend*',
      (route) =>
        route.fulfill({
          contentType: 'application/json',
          body: JSON.stringify({ success: true, data: { trend: [] } }),
        }),
    )

    await page.goto('/courses/7/reviews')
    await page.waitForLoadState('networkidle')

    await page.getByRole('button', { name: '查看回复' }).click()

    await expect(page.getByText('这条回复游客应该可以直接阅读。')).toBeVisible()
    await expect(
      page.getByText('登录后参与讨论，回复会保留在当前测评下。'),
    ).toBeVisible()
    await expect(
      page.getByRole('textbox', { name: '回复内容' }),
    ).toHaveCount(0)
  })

  test('empty course detail shows one review prompt without duplicate rating CTA', async ({
    page,
  }) => {
    let favoriteStatusRequests = 0

    await page.route('**/api/v1/course/courses/3', (route) =>
      route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: {
            id: 3,
            name: '空测评课程',
            code: 'EMPTY101',
            departmentID: 1,
            departmentName: '测试学院',
            credits: 2,
            reviewCount: 0,
          },
        }),
      }),
    )

    await page.route('**/api/v1/course/review/courses/3/reviews*', (route) =>
      route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify({ success: true, data: { list: [], total: 0 } }),
      }),
    )

    await page.route(
      '**/api/v1/course/review/courses/3/rating-stats*',
      (route) =>
        route.fulfill({
          contentType: 'application/json',
          body: JSON.stringify({
            success: true,
            data: {
              courseID: 3,
              overall: { termName: '总体', dimensions: [] },
              byTerm: [],
              allDimensionKeys: [],
            },
          }),
        }),
    )

    await page.route(
      '**/api/v1/course/review/courses/3/teachers*',
      (route) =>
        route.fulfill({
          contentType: 'application/json',
          body: JSON.stringify({ success: true, data: [] }),
        }),
    )

    await page.route(
      '**/api/v1/course/review/courses/3/rating-trend*',
      (route) =>
        route.fulfill({
          contentType: 'application/json',
          body: JSON.stringify({ success: true, data: { trend: [] } }),
        }),
    )

    await page.route('**/api/v1/course/review/courses/3/favorites', (route) => {
      favoriteStatusRequests += 1
      return route.fulfill({
        status: 401,
        contentType: 'application/json',
        body: JSON.stringify({
          success: false,
          error: { code: 'A0010100', message: 'login required' },
        }),
      })
    })

    await page.goto('/courses/3/reviews')
    await page.waitForLoadState('networkidle')

    await expect(page.getByText('空测评课程')).toBeVisible({ timeout: 10_000 })
    await expect(
      page.getByRole('button', { name: /^(收藏|Favorite)$/ }),
    ).toBeEnabled()
    await expect(page.getByText(/^(暂无测评|No reviews yet)$/)).toHaveCount(1)
    await expect(
      page.getByText(
        /^(暂无数据，快来添加这门课的第一条测评吧！|No data yet\. Be the first to review this course!)$/,
      ),
    ).toHaveCount(0)
    await expect(
      page.getByRole('button', {
        name: /^(写第一条测评|Write the first review)$/,
      }),
    ).toHaveCount(0)
    expect(favoriteStatusRequests).toBe(0)
  })

  test('malformed course detail response shows full load failure', async ({
    page,
  }) => {
    await page.route('**/api/v1/course/courses/5', (route) =>
      route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: { id: 5, name: '缺失字段的课程' },
        }),
      }),
    )

    await page.route('**/api/v1/course/review/courses/5/reviews*', (route) =>
      route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify({ success: true, data: { list: [], total: 0 } }),
      }),
    )

    await page.route(
      '**/api/v1/course/review/courses/5/rating-stats*',
      (route) =>
        route.fulfill({
          contentType: 'application/json',
          body: JSON.stringify({ success: true, data: null }),
        }),
    )

    await page.route(
      '**/api/v1/course/review/courses/5/teachers*',
      (route) =>
        route.fulfill({
          contentType: 'application/json',
          body: JSON.stringify({ success: true, data: [] }),
        }),
    )

    await page.route(
      '**/api/v1/course/review/courses/5/rating-trend*',
      (route) =>
        route.fulfill({
          contentType: 'application/json',
          body: JSON.stringify({ success: true, data: { trend: [] } }),
        }),
    )

    await page.goto('/courses/5')
    await page.waitForLoadState('networkidle')

    await expect(page.getByText(/Load failed|加载失败/i).first()).toBeVisible({
      timeout: 10_000,
    })
    await expect(
      page.getByRole('button', { name: /Retry|重试/i }),
    ).toBeVisible()
  })

  test('malformed course detail auxiliary responses show partial load failure', async ({
    page,
  }) => {
    await page.route('**/api/v1/course/courses/6', (route) =>
      route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: {
            id: 6,
            name: '局部失败课程',
            code: 'PARTIAL101',
            departmentID: 1,
            departmentName: '测试学院',
            credits: 2,
            reviewCount: 0,
          },
        }),
      }),
    )

    await page.route('**/api/v1/course/review/courses/6/reviews*', (route) =>
      route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify({ success: true, data: { list: [], total: 0 } }),
      }),
    )

    await page.route(
      '**/api/v1/course/review/courses/6/rating-stats*',
      (route) =>
        route.fulfill({
          contentType: 'application/json',
          body: JSON.stringify({
            success: true,
            data: {
              courseID: 6,
              overall: {
                termName: '总体',
                dimensions: [
                  {
                    key: 'overall',
                    name: '总体',
                    avgRating: '5',
                    ratingCount: 1,
                  },
                ],
              },
              byTerm: [],
              allDimensionKeys: ['overall'],
            },
          }),
        }),
    )

    await page.route(
      '**/api/v1/course/review/courses/6/teachers*',
      (route) =>
        route.fulfill({
          contentType: 'application/json',
          body: JSON.stringify({
            success: true,
            data: [
              {
                teacherID: 6,
                teacherName: '局部失败教师',
                departmentName: '测试学院',
                courseCount: -1,
                reviewCount: 1,
              },
            ],
          }),
        }),
    )

    await page.route(
      '**/api/v1/course/review/courses/6/rating-trend*',
      (route) =>
        route.fulfill({
          contentType: 'application/json',
          body: JSON.stringify({
            success: true,
            data: { trend: [{ termName: '2026 春', avgRating: '4' }] },
          }),
        }),
    )

    await page.goto('/courses/6/reviews')
    await page.waitForLoadState('networkidle')

    await expect(page.getByText('局部失败课程')).toBeVisible({
      timeout: 10_000,
    })
    await expect(
      page.getByRole('alert').filter({ hasText: /Load failed|加载失败/i }).first(),
    ).toBeVisible()
  })

  test('guest course detail protected actions redirect to login without mutations', async ({
    page,
  }) => {
    const protectedRoute = '/courses/4/reviews'
    const reviewID = 'web-guest-review'
    const mutations: string[] = []

    await page.route('**/api/v1/course/courses/4', (route) =>
      route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: {
            id: 4,
            name: '游客保护课程',
            code: 'GUEST101',
            departmentID: 1,
            departmentName: '测试学院',
            credits: 3,
            reviewCount: 1,
          },
        }),
      }),
    )

    await page.route('**/api/v1/course/review/courses/4/reviews*', (route) =>
      route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: {
            list: [
              {
                id: reviewID,
                courseID: 4,
                courseName: '游客保护课程',
                termID: '2026-spring',
                title: '游客操作保护验证',
                content: '这条评价用于验证游客不能直接触发受保护动作。',
                teacherName: '陈老师',
                termName: '2026 春',
                grade: 'A',
                ratings: { overall: 5, workload: 4 },
                likeCount: 2,
                dislikeCount: 0,
                replyCount: 0,
                status: 'published',
                createdAt: '2026-05-01T08:00:00Z',
              },
            ],
            total: 1,
          },
        }),
      }),
    )

    await page.route(
      '**/api/v1/course/review/courses/4/rating-stats*',
      (route) =>
        route.fulfill({
          contentType: 'application/json',
          body: JSON.stringify({
            success: true,
            data: {
              courseID: 4,
              overall: { termName: '总体', dimensions: [] },
              byTerm: [],
              allDimensionKeys: [],
            },
          }),
        }),
    )

    await page.route(
      '**/api/v1/course/review/courses/4/teachers*',
      (route) =>
        route.fulfill({
          contentType: 'application/json',
          body: JSON.stringify({
            success: true,
            data: [
              {
                teacherID: 4,
                teacherName: '陈老师',
                departmentName: '测试学院',
                courseCount: 1,
                reviewCount: 1,
              },
            ],
          }),
        }),
    )

    await page.route(
      '**/api/v1/course/review/courses/4/rating-trend*',
      (route) =>
        route.fulfill({
          contentType: 'application/json',
          body: JSON.stringify({ success: true, data: { trend: [] } }),
        }),
    )

    await page.route('**/api/v1/course/review/courses/4/favorites', (route) => {
      if (route.request().method() !== 'GET') {
        mutations.push(
          `${route.request().method()} ${new URL(route.request().url()).pathname}`,
        )
      }
      return route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify({ success: true, data: { favorited: false } }),
      })
    })

    await page.route(
      `**/api/v1/course/review/reviews/${reviewID}/votes`,
      (route) => {
        mutations.push(
          `${route.request().method()} ${new URL(route.request().url()).pathname}`,
        )
        return route.fulfill({
          contentType: 'application/json',
          body: JSON.stringify({ success: true, data: { voteType: 'like' } }),
        })
      },
    )

    await page.route(
      `**/api/v1/course/review/reviews/${reviewID}/replies`,
      (route) => {
        if (route.request().method() !== 'GET') {
          mutations.push(
            `${route.request().method()} ${new URL(route.request().url()).pathname}`,
          )
        }
        return route.fulfill({
          contentType: 'application/json',
          body: JSON.stringify({ success: true, data: { list: [], total: 0 } }),
        })
      },
    )

    await page.goto(protectedRoute)
    await expect(page.getByText('游客保护课程')).toBeVisible({ timeout: 10_000 })
    await page.getByRole('button', { name: /^(收藏|Favorite)$/ }).click()
    await expectLoginRedirect(page, protectedRoute)
    expect(mutations).toEqual([])

    await page.goto(protectedRoute)
    await expect(page.getByText('游客操作保护验证')).toBeVisible()
    await page.getByRole('button', { name: /^(发布测评|Post Review)$/ }).click()
    await expectLoginRedirect(page, '/courses/reviews/post')
    expect(mutations).toEqual([])

    await page.goto(protectedRoute)
    await page.getByTestId(`review-like-${reviewID}`).click()
    await expectLoginRedirect(page, protectedRoute)
    expect(mutations).toEqual([])

    await page.goto(protectedRoute)
    await page.getByRole('button', { name: /^(查看回复|View replies)$/ }).click()
    await expect(page.getByTestId('review-reply-login-prompt')).toBeVisible()
    await expect(
      page.getByRole('textbox', { name: /^(回复内容|Reply content)$/ }),
    ).toHaveCount(0)
    expect(mutations).toEqual([])
  })
})
