import { expect, test, type Page, type Route } from './fixtures';

function json(data: unknown, status = 200) {
    return {
        status,
        contentType: "application/json",
        body: JSON.stringify(data),
    };
}

function ok(data: unknown) {
    return json({ success: true, data });
}

function list<T>(items: T[]) {
    return ok({ list: items, total: items.length });
}

const webApiRequests: string[] = [];

function recordApiRequest(route: Route) {
    const request = route.request();
    const url = new URL(request.url());
    webApiRequests.push(`${request.method()} ${url.pathname}${url.search}`);
}

function hasWebGetRequest(pathname: string, matches: (url: URL) => boolean) {
    return webApiRequests.some((request) => {
        if (!request.startsWith("GET ")) return false;
        const url = new URL(request.slice("GET ".length), "http://web.e2e");
        return url.pathname === pathname && matches(url);
    });
}

async function mockUnauthenticated(page: Page) {
    await page.route("**/api/v1/auth/me", (route) =>
        route.fulfill(
            json(
                {
                    success: false,
                    error: { code: "A0010100", message: "login required" },
                },
                401,
            ),
        ),
    );
}

const departments = [
    { id: 1, name: "计算机科学与技术学院", code: "CS", category: "工科" },
    { id: 2, name: "数学科学学院", code: "MATH", category: "理科" },
];

const departmentCourses = [
    {
        id: 301,
        name: "编译原理",
        code: "CS301",
        departmentID: 1,
        departmentName: "计算机科学与技术学院",
        credits: 3,
        reviewCount: 9,
    },
];

const latestReview = {
    id: "feed-latest-1",
    courseID: 301,
    courseName: "编译原理",
    teacherID: 21,
    teacherName: "陈老师",
    termID: "2026-spring",
    termName: "2026 春",
    title: "最新聚合测评",
    content: "课堂节奏稳定，代码实验能很好帮助理解编译器前端。",
    ratings: { recommendation: 5, content_quality: 4, workload: 3 },
    likeCount: 8,
    dislikeCount: 0,
    replyCount: 1,
    status: "published",
    createdAt: "2026-05-24T04:00:00Z",
    authorDisplayName: "匿名同学",
};

const hotReview = {
    ...latestReview,
    id: "feed-hot-1",
    title: "最热排序测评",
    likeCount: 32,
};

const ratedReview = {
    ...latestReview,
    id: "feed-rated-1",
    title: "高分排序测评",
    ratings: { recommendation: 5, content_quality: 5, workload: 4 },
};

const teacherProfileReview = {
    ...latestReview,
    id: "teacher-profile-review-1",
    teacherID: 21,
    teacherName: "陈老师",
    title: "教师详情页评价",
    courseName: "编译原理",
};

const popularTeachers = [
    {
        teacherID: 21,
        teacherName: "陈老师",
        departmentName: "计算机科学与技术学院",
        avgRating: 4.7,
        reviewCount: 18,
        courseCount: 3,
    },
    {
        teacherID: 22,
        teacherName: "李老师",
        departmentName: "数学科学学院",
        avgRating: 4.4,
        reviewCount: 11,
        courseCount: 2,
    },
];

const searchedTeachers = [
    {
        teacherID: 23,
        teacherName: "王老师",
        departmentName: "软件学院",
        avgRating: 4.9,
        reviewCount: 7,
        courseCount: 1,
    },
];

async function mockCourseCommunityApi(page: Page) {
    await page.route("**/api/v1/course/categories*", (route) => {
        recordApiRequest(route);
        return route.fulfill(ok([{ id: 1, name: "工科" }]));
    });
    await page.route("**/api/v1/course/departments*", (route) => {
        recordApiRequest(route);
        return route.fulfill(ok(departments));
    });
    await page.route("**/api/v1/course/courses?*", (route) => {
        recordApiRequest(route);
        return route.fulfill(list(departmentCourses));
    });
    await page.route("**/api/v1/course/review/reviews/latest*", (route) => {
        recordApiRequest(route);
        const url = new URL(route.request().url());
        const sort = url.searchParams.get("sort");
        const review =
            sort === "likes"
                ? hotReview
                : sort === "rating"
                  ? ratedReview
                  : latestReview;

        return route.fulfill(list([review]));
    });
    await page.route("**/api/v1/course/review/stats", (route) => {
        recordApiRequest(route);
        return route.fulfill(
            ok({
                courseCount: 120,
                reviewCount: 580,
                departmentCount: 8,
                userCount: 230,
            }),
        );
    });
    await page.route("**/api/v1/course/review/rankings/hot*", (route) => {
        recordApiRequest(route);
        return route.fulfill(ok({ list: [] }));
    });
    await page.route("**/api/v1/course/terms*", (route) => {
        recordApiRequest(route);
        return route.fulfill(
            ok([{ id: "2026-spring", name: "2026 春", isCurrent: true }]),
        );
    });
}

async function mockTeacherApi(page: Page) {
    await page.route("**/api/v1/course/review/teachers/hot*", (route) => {
        recordApiRequest(route);
        return route.fulfill(list(popularTeachers));
    });
    await page.route("**/api/v1/course/review/teachers?*", (route) => {
        recordApiRequest(route);
        return route.fulfill(list(searchedTeachers));
    });
}

async function mockTeacherStatsApi(page: Page) {
    await page.route("**/api/v1/course/review/teachers/21/stats", (route) => {
        recordApiRequest(route);
        return route.fulfill(
            ok({
                teacherID: 21,
                teacherName: "陈老师",
                departmentName: "计算机科学与技术学院",
                avgRating: 4.2,
                courseCount: 2,
                reviewCount: 18,
                courses: [
                    {
                        id: 301,
                        name: "编译原理",
                        avgRating: 4.2,
                        reviewCount: 9,
                    },
                ],
                ratingTrend: [
                    {
                        termID: "2026-spring",
                        termName: "2026 春",
                        avgRating: 4.2,
                    },
                ],
            }),
        );
    });
    await page.route("**/api/v1/course/review/reviews/latest*", (route) => {
        recordApiRequest(route);
        const url = new URL(route.request().url());
        const teacherID = url.searchParams.get("teacherID");
        return route.fulfill(
            list(teacherID === "21" ? [teacherProfileReview] : []),
        );
    });
}

async function fulfillUnexpected(route: Route) {
    await route.fulfill(
        json(
            {
                success: false,
                error: {
                    code: "E2E_UNMOCKED",
                    message: `unmocked web e2e request: ${route.request().method()} ${new URL(route.request().url()).pathname}`,
                },
            },
            500,
        ),
    );
}

async function openCourseListDrawerIfNeeded(page: Page) {
    const courseListDrawerButton = page.getByRole("button", {
        name: "课程列表",
    });
    if (await courseListDrawerButton.isVisible()) {
        await courseListDrawerButton.click();
    }
}

test.describe("Course community surfaces", () => {
    test.beforeEach(async ({ page }) => {
        webApiRequests.length = 0;
        await page.route("**/api/v1/**", fulfillUnexpected);
        await mockUnauthenticated(page);
    });

    test("course about page renders FAQ, contact, and disclaimer content", async ({
        page,
    }) => {
        await page.goto("/courses/about");

        await expect(
            page.getByRole("heading", { name: "关于评课社区@BUAA" }),
        ).toBeVisible();
        await expect(
            page.getByRole("heading", { name: "常见问答 (FAQ)" }),
        ).toBeVisible();
        await expect(page.getByText("联系我们")).toBeVisible();
        await expect(
            page.getByRole("heading", { name: "必要声明" }),
        ).toBeVisible();

        await page
            .getByRole("button", {
                name: "1. 我该测评哪些课程？写一些什么？",
            })
            .click();
        await expect(
            page.getByText("所有的课程，尤其是尚无测评的课程。"),
        ).toBeVisible();

        await page
            .getByRole("button", {
                name: "3. 为什么仍然强调文字内容，而不是只看评分？",
            })
            .click();
        await expect(
            page.getByText("当前社区会展示课程和教师的维度评分、综合评分与趋势"),
        ).toBeVisible();

        await page
            .getByRole("button", {
                name: "5. 我会因为在这里发表信息而被约谈吗？",
            })
            .click();
        await expect(
            page.getByText("游客可以浏览公开摘要，发布测评、互动和查看完整内容需要满足相应权限"),
        ).toBeVisible();
        await expect(page.getByText("本站不设登录验证")).toHaveCount(0);
        await expect(
            page.getByText("为什么本站不收集和展示课程评分？"),
        ).toHaveCount(0);
    });

    test("course review feed renders latest reviews, sort tabs, and department courses", async ({
        page,
    }) => {
        await mockCourseCommunityApi(page);

        await page.goto("/courses/reviews");

        await expect(page.getByText("最新聚合测评")).toBeVisible({
            timeout: 10_000,
        });
        await expect(page.getByRole("tab", { name: "最新" })).toHaveAttribute(
            "aria-selected",
            "true",
        );

        await page.getByRole("tab", { name: "最热" }).click();
        await expect(page).toHaveURL((url) =>
            url.pathname === "/courses/reviews" &&
            url.searchParams.get("sort") === "likes",
        );
        await expect(page.getByText("最热排序测评")).toBeVisible();
        await expect
            .poll(() =>
                hasWebGetRequest(
                    "/api/v1/course/review/reviews/latest",
                    (url) =>
                        url.searchParams.get("sort") === "likes" &&
                        url.searchParams.get("page") === "1" &&
                        url.searchParams.get("pageSize") === "10",
                ),
            )
            .toBe(true);
        await expect(page.getByRole("tab", { name: "最热" })).toHaveAttribute(
            "aria-selected",
            "true",
        );

        await page.getByRole("tab", { name: "精选" }).click();
        await expect(page).toHaveURL((url) =>
            url.pathname === "/courses/reviews" &&
            url.searchParams.get("sort") === "rating",
        );
        await expect(page.getByText("高分排序测评")).toBeVisible();
        await expect
            .poll(() =>
                hasWebGetRequest(
                    "/api/v1/course/review/reviews/latest",
                    (url) =>
                        url.searchParams.get("sort") === "rating" &&
                        url.searchParams.get("page") === "1" &&
                        url.searchParams.get("pageSize") === "10",
                ),
            )
            .toBe(true);

        const requestsBeforeReload = webApiRequests.length;
        await page.reload();

        await expect(page).toHaveURL((url) =>
            url.pathname === "/courses/reviews" &&
            url.searchParams.get("sort") === "rating",
        );
        await expect(page.getByRole("tab", { name: "精选" })).toHaveAttribute(
            "aria-selected",
            "true",
        );
        await expect(page.getByText("高分排序测评")).toBeVisible();
        await expect
            .poll(() =>
                webApiRequests.slice(requestsBeforeReload).some((request) => {
                    if (!request.startsWith("GET ")) return false;
                    const url = new URL(request.slice("GET ".length), "http://web.e2e");
                    return (
                        url.pathname ===
                            "/api/v1/course/review/reviews/latest" &&
                        url.searchParams.get("sort") === "rating" &&
                        url.searchParams.get("page") === "1" &&
                        url.searchParams.get("pageSize") === "10"
                    );
                }),
            )
            .toBe(true);

        await page.getByRole("tab", { name: "最新" }).click();
        await expect(page).toHaveURL((url) =>
            url.pathname === "/courses/reviews" &&
            !url.searchParams.has("sort"),
        );

        await openCourseListDrawerIfNeeded(page);
        await page
            .getByRole("button", { name: "计算机科学与技术学院" })
            .click();
        await expect
            .poll(() =>
                hasWebGetRequest("/api/v1/course/courses", (url) =>
                    (
                        url.searchParams.get("departmentID") === "1" &&
                        url.searchParams.get("page") === "1" &&
                        url.searchParams.get("pageSize") === "100"
                    ),
                ),
            )
            .toBe(true);
        await expect(
            page.getByRole("link", { name: /编译原理/ }),
        ).toBeVisible();
    });

    test("mobile floating module nav opens the review feed on touch tap", async ({
        page,
    }, testInfo) => {
        test.skip(
            testInfo.project.name !== "mobile-chromium",
            "touch navigation regression only applies to mobile contexts",
        );
        await mockCourseCommunityApi(page);

        await page.goto("/courses");
        const activeNav = page.getByTestId("floating-module-nav-active");
        await expect(activeNav).toBeVisible();

        const box = await activeNav.boundingBox();
        const viewport = page.viewportSize();
        if (!box || !viewport) {
            throw new Error("floating module nav must have a visible mobile box");
        }
        expect(box.x).toBeGreaterThan(viewport.width - 72);
        expect(box.y).toBeGreaterThan(viewport.height - 96);
        expect(box.x + box.width).toBeLessThanOrEqual(viewport.width - 8);
        expect(box.y + box.height).toBeLessThanOrEqual(viewport.height - 8);

        await activeNav.tap();

        await expect(page).toHaveURL(/\/courses\/reviews$/);
        await expect(page.getByText("最新聚合测评")).toBeVisible({
            timeout: 10_000,
        });
    });

    test("invalid review feed response fails closed and can retry", async ({
        page,
    }) => {
        let loadCount = 0;
        await mockCourseCommunityApi(page);
        await page.route("**/api/v1/course/review/reviews/latest*", (route) => {
            recordApiRequest(route);
            loadCount += 1;
            return route.fulfill(
                loadCount === 1 ? ok(null) : list([latestReview]),
            );
        });

        await page.goto("/courses/reviews");

        const alert = page.getByRole("alert").filter({
            hasText: "加载测评失败，请稍后重试",
        });
        await expect(alert).toBeVisible({ timeout: 10_000 });
        await expect(page.getByText("暂无测评，来发布第一条吧")).toHaveCount(0);

        await alert.getByRole("button", { name: "重试" }).click();

        await expect.poll(() => loadCount).toBe(2);
        await expect(page.getByText("最新聚合测评")).toBeVisible();
    });

    test("invalid department sidebar response fails closed and can retry", async ({
        page,
    }) => {
        let loadCount = 0;
        let malformed = true;
        await mockCourseCommunityApi(page);
        await page.route("**/api/v1/course/departments*", (route) => {
            recordApiRequest(route);
            loadCount += 1;
            return route.fulfill(
                malformed
                    ? ok([{ id: 1, name: "缺失分类的院系" }])
                    : ok(departments),
            );
        });

        await page.goto("/courses/reviews");
        await expect(page.getByText("最新聚合测评")).toBeVisible({
            timeout: 10_000,
        });
        await openCourseListDrawerIfNeeded(page);

        const alert = page.getByRole("alert").filter({ hasText: "加载失败" });
        await expect(alert).toBeVisible({ timeout: 10_000 });
        await expect(page.getByText("未找到结果")).toHaveCount(0);

        malformed = false;
        await alert.getByRole("button", { name: "重试" }).click();

        await expect.poll(() => loadCount).toBeGreaterThan(1);
        await expect(
            page.getByRole("button", { name: "计算机科学与技术学院" }),
        ).toBeVisible();
    });

    test("invalid course category response fails closed and can retry", async ({
        page,
    }) => {
        let loadCount = 0;
        let malformed = true;
        await mockCourseCommunityApi(page);
        await page.route("**/api/v1/course/categories*", (route) => {
            recordApiRequest(route);
            loadCount += 1;
            return route.fulfill(
                malformed
                    ? ok([{ id: 1, name: 123 }])
                    : ok([{ id: 1, name: "工科" }]),
            );
        });

        await page.goto("/courses/reviews");
        await expect(page.getByText("最新聚合测评")).toBeVisible({
            timeout: 10_000,
        });
        await openCourseListDrawerIfNeeded(page);

        const alert = page.getByRole("alert").filter({ hasText: "加载失败" });
        await expect(alert).toBeVisible({ timeout: 10_000 });
        await expect(page.getByText("未找到结果")).toHaveCount(0);

        malformed = false;
        await alert.getByRole("button", { name: "重试" }).click();

        await expect.poll(() => loadCount).toBeGreaterThan(1);
        await expect(page.getByRole("button", { name: "工科" })).toBeVisible();
        await expect(
            page.getByRole("button", { name: "计算机科学与技术学院" }),
        ).toBeVisible();
    });

    test("invalid department course response fails closed and can retry", async ({
        page,
    }) => {
        let loadCount = 0;
        let malformed = true;
        await mockCourseCommunityApi(page);
        await page.route("**/api/v1/course/courses?*", (route) => {
            recordApiRequest(route);
            loadCount += 1;
            return route.fulfill(
                malformed
                    ? list([{ ...departmentCourses[0], credits: "3" }])
                    : list(departmentCourses),
            );
        });

        await page.goto("/courses/reviews");
        await expect(page.getByText("最新聚合测评")).toBeVisible({
            timeout: 10_000,
        });
        await openCourseListDrawerIfNeeded(page);

        await page
            .getByRole("button", { name: "计算机科学与技术学院" })
            .click();

        const alert = page.getByRole("alert").filter({ hasText: "加载失败" });
        await expect(alert).toBeVisible({ timeout: 10_000 });
        await expect(page.getByText("未找到结果")).toHaveCount(0);

        malformed = false;
        await alert.getByRole("button", { name: "重试" }).click();

        await expect.poll(() => loadCount).toBeGreaterThan(1);
        await expect(
            page.getByRole("link", { name: /编译原理/ }),
        ).toBeVisible();
    });

    test("teacher hub shows popular teachers and searches by name", async ({
        page,
    }) => {
        await mockTeacherApi(page);

        await page.goto("/teachers");

        await expect(
            page.getByRole("heading", { name: "教师主页" }),
        ).toBeVisible();
        await expect(page.getByText("热门教师")).toBeVisible();
        await expect(page.getByRole("link", { name: /陈老师/ })).toBeVisible({
            timeout: 10_000,
        });

        await page.getByLabel("输入教师姓名搜索...").fill("王");
        await expect(page).toHaveURL((url) =>
            url.pathname === "/teachers" && url.searchParams.get("q") === "王",
        );
        await expect(page.getByText("搜索结果")).toBeVisible();
        await expect(page.getByRole("link", { name: /王老师/ })).toBeVisible({
            timeout: 10_000,
        });
        await expect
            .poll(() =>
                hasWebGetRequest("/api/v1/course/review/teachers", (url) =>
                    (
                        url.searchParams.get("q") === "王" &&
                        url.searchParams.get("sort") === "reviews" &&
                        url.searchParams.get("pageSize") === "30"
                    ),
                ),
            )
            .toBe(true);

        const requestsBeforeReload = webApiRequests.length;
        await page.reload();

        await expect(page).toHaveURL((url) =>
            url.pathname === "/teachers" && url.searchParams.get("q") === "王",
        );
        await expect(page.getByLabel("输入教师姓名搜索...")).toHaveValue("王");
        await expect(page.getByText("搜索结果")).toBeVisible();
        await expect(page.getByRole("link", { name: /王老师/ })).toBeVisible({
            timeout: 10_000,
        });
        await expect
            .poll(() =>
                webApiRequests.slice(requestsBeforeReload).some((request) => {
                    if (!request.startsWith("GET ")) return false;
                    const url = new URL(request.slice("GET ".length), "http://web.e2e");
                    return (
                        url.pathname === "/api/v1/course/review/teachers" &&
                        url.searchParams.get("q") === "王" &&
                        url.searchParams.get("sort") === "reviews" &&
                        url.searchParams.get("pageSize") === "30"
                    );
                }),
            )
            .toBe(true);

        await page.getByRole("button", { name: "清除" }).click();
        await expect(page).toHaveURL((url) =>
            url.pathname === "/teachers" && !url.searchParams.has("q"),
        );
        await expect(page.getByLabel("输入教师姓名搜索...")).toHaveValue("");
        await expect(page.getByText("热门教师")).toBeVisible();
        await expect(page.getByRole("link", { name: /陈老师/ })).toBeVisible();
    });

    test("teacher hub ignores stale search failures after the query is cleared", async ({
        page,
    }) => {
        let releaseSearch!: () => void;
        let markSearchRequested!: () => void;
        const searchRequested = new Promise<void>((resolve) => {
            markSearchRequested = resolve;
        });
        const searchRelease = new Promise<void>((resolve) => {
            releaseSearch = resolve;
        });

        await page.route("**/api/v1/course/review/teachers/hot*", (route) => {
            recordApiRequest(route);
            return route.fulfill(list(popularTeachers));
        });
        await page.route("**/api/v1/course/review/teachers?*", async (route) => {
            recordApiRequest(route);
            markSearchRequested();
            await searchRelease;
            return route.fulfill(list([{ ...searchedTeachers[0], courseCount: -1 }]));
        });

        await page.goto("/teachers");
        await expect(page.getByRole("link", { name: /陈老师/ })).toBeVisible({
            timeout: 10_000,
        });

        const searchInput = page.getByLabel("输入教师姓名搜索...");
        await searchInput.fill("王");
        await searchRequested;
        await searchInput.fill("");
        releaseSearch();

        await expect(page.getByText("热门教师")).toBeVisible();
        await expect(page.getByRole("link", { name: /陈老师/ })).toBeVisible();
        await expect(page.getByText("加载失败")).toHaveCount(0);
        await expect(page.getByText("未找到匹配的教师，换个名字试试。")).toHaveCount(0);
    });

    test("teacher hub loads additional search pages when more teachers match", async ({
        page,
    }) => {
        const makeTeacher = (id: number) => ({
            ...searchedTeachers[0],
            teacherID: id,
            teacherName: `王老师 ${id}`,
            reviewCount: id,
        });

        await page.route("**/api/v1/course/review/teachers/hot*", (route) => {
            recordApiRequest(route);
            return route.fulfill(list(popularTeachers));
        });
        await page.route("**/api/v1/course/review/teachers?*", (route) => {
            recordApiRequest(route);
            const url = new URL(route.request().url());
            const requestedPage = url.searchParams.get("page") ?? "1";
            const teachers =
                requestedPage === "2"
                    ? [makeTeacher(31)]
                    : Array.from({ length: 30 }, (_, index) =>
                          makeTeacher(index + 1),
                      );
            return route.fulfill(ok({ list: teachers, total: 31 }));
        });

        await page.goto("/teachers");
        await expect(page.getByRole("link", { name: /陈老师/ })).toBeVisible({
            timeout: 10_000,
        });

        await page.getByLabel("输入教师姓名搜索...").fill("王");
        await expect(page.getByRole("link", { name: /王老师 30\b/ })).toBeVisible({
            timeout: 10_000,
        });
        await expect(page.getByRole("link", { name: /王老师 31\b/ })).toHaveCount(0);

        await page.getByRole("button", { name: "加载更多" }).click();

        await expect(page.getByRole("link", { name: /王老师 31\b/ })).toBeVisible();
        await expect
            .poll(() =>
                hasWebGetRequest("/api/v1/course/review/teachers", (url) =>
                    (
                        url.searchParams.get("q") === "王" &&
                        url.searchParams.get("sort") === "reviews" &&
                        url.searchParams.get("page") === "2" &&
                        url.searchParams.get("pageSize") === "30"
                    ),
                ),
            )
            .toBe(true);
        await expect(page.getByRole("button", { name: "加载更多" })).toHaveCount(0);
    });

    test("teacher profile shows readable overall rating and course context", async ({
        page,
    }) => {
        await mockTeacherStatsApi(page);

        await page.goto("/teachers/21");

        await expect(
            page.getByRole("heading", { name: "陈老师" }),
        ).toBeVisible({ timeout: 10_000 });
        await expect(page.getByText("计算机科学与技术学院")).toBeVisible();
        await expect(page.getByText("4.2 / 5")).toBeVisible();
        await expect(page.getByText("综合评分")).toBeVisible();
        await expect(page.getByText("2026 春")).toBeVisible();
        await expect(
            page.getByRole("link", { name: /编译原理/ }),
        ).toBeVisible();
        await expect(page.getByText("最新评价")).toBeVisible();
        await expect(page.getByText("教师详情页评价")).toBeVisible();
        await expect
            .poll(() =>
                hasWebGetRequest(
                    "/api/v1/course/review/reviews/latest",
                    (url) =>
                        url.searchParams.get("teacherID") === "21" &&
                        url.searchParams.get("sort") === "time" &&
                        url.searchParams.get("page") === "1" &&
                        url.searchParams.get("pageSize") === "6",
                ),
            )
            .toBe(true);
        await expect(page).toHaveTitle(/陈老师 - StuHelper/);
    });

    test("teacher profile keeps the newest teacher when an older stats request finishes late", async ({
        page,
    }) => {
        let releaseSlowStats!: () => void;
        let markSlowStatsRequested!: () => void;
        let slowStatsFulfilled = false;
        const slowStatsRequested = new Promise<void>((resolve) => {
            markSlowStatsRequested = resolve;
        });
        const slowStatsRelease = new Promise<void>((resolve) => {
            releaseSlowStats = resolve;
        });
        const nextTeacherReview = {
            ...teacherProfileReview,
            id: "teacher-profile-review-2",
            teacherID: 22,
            teacherName: "李老师",
            title: "李老师详情页评价",
            courseName: "高等数学A",
        };

        await page.route(
            "**/api/v1/course/review/teachers/21/stats",
            async (route) => {
                recordApiRequest(route);
                markSlowStatsRequested();
                await slowStatsRelease;
                await route.fulfill(
                    ok({
                        teacherID: 21,
                        teacherName: "陈老师",
                        departmentName: "计算机科学与技术学院",
                        avgRating: 4.2,
                        courseCount: 2,
                        reviewCount: 18,
                        courses: [
                            {
                                id: 301,
                                name: "编译原理",
                                avgRating: 4.2,
                                reviewCount: 9,
                            },
                        ],
                        ratingTrend: [
                            {
                                termID: "2026-spring",
                                termName: "2026 春",
                                avgRating: 4.2,
                            },
                        ],
                    }),
                );
                slowStatsFulfilled = true;
            },
        );
        await page.route("**/api/v1/course/review/teachers/22/stats", (route) => {
            recordApiRequest(route);
            return route.fulfill(
                ok({
                    teacherID: 22,
                    teacherName: "李老师",
                    departmentName: "数学科学学院",
                    avgRating: 4.6,
                    courseCount: 1,
                    reviewCount: 11,
                    courses: [
                        {
                            id: 302,
                            name: "高等数学A",
                            avgRating: 4.6,
                            reviewCount: 11,
                        },
                    ],
                    ratingTrend: [
                        {
                            termID: "2026-spring",
                            termName: "2026 春",
                            avgRating: 4.6,
                        },
                    ],
                }),
            );
        });
        await page.route("**/api/v1/course/review/reviews/latest*", (route) => {
            recordApiRequest(route);
            const url = new URL(route.request().url());
            return route.fulfill(
                list(
                    url.searchParams.get("teacherID") === "22"
                        ? [nextTeacherReview]
                        : [teacherProfileReview],
                ),
            );
        });

        await page.goto("/teachers/21");
        await slowStatsRequested;
        await page.evaluate(() => {
            window.history.pushState(null, "", "/teachers/22");
            window.dispatchEvent(new PopStateEvent("popstate"));
        });

        await expect(
            page.getByRole("heading", { name: "李老师" }),
        ).toBeVisible({ timeout: 10_000 });
        releaseSlowStats();
        await expect.poll(() => slowStatsFulfilled).toBe(true);
        await page.waitForLoadState("networkidle");

        await expect(
            page.getByRole("heading", { name: "李老师" }),
        ).toBeVisible();
        await expect(
            page.getByRole("heading", { name: "陈老师" }),
        ).toHaveCount(0);
        await expect(page.getByText("高等数学A")).toBeVisible();
        await expect(page.getByText("李老师详情页评价")).toBeVisible();
        await expect(page.getByText("教师详情页评价")).toHaveCount(0);
    });

    test("invalid popular teachers response fails closed and can retry", async ({
        page,
    }) => {
        let loadCount = 0;

        await page.route("**/api/v1/course/review/teachers/hot*", (route) => {
            recordApiRequest(route);
            loadCount += 1;
            return route.fulfill(
                loadCount === 1
                    ? ok({
                          list: [
                              {
                                  ...popularTeachers[0],
                                  reviewCount: "18",
                              },
                          ],
                      })
                    : list(popularTeachers),
            );
        });

        await page.goto("/teachers");

        await expect(page.getByText("加载失败")).toBeVisible({
            timeout: 10_000,
        });
        await expect(page.getByText("暂无教师数据。")).toHaveCount(0);

        await page.getByRole("button", { name: "重试" }).click();

        await expect.poll(() => loadCount).toBe(2);
        await expect(page.getByRole("link", { name: /陈老师/ })).toBeVisible();
    });

    test("invalid teacher search response fails closed and can retry", async ({
        page,
    }) => {
        let searchCount = 0;

        await page.route("**/api/v1/course/review/teachers/hot*", (route) => {
            recordApiRequest(route);
            return route.fulfill(list(popularTeachers));
        });
        await page.route("**/api/v1/course/review/teachers?*", (route) => {
            recordApiRequest(route);
            searchCount += 1;
            return route.fulfill(
                searchCount === 1
                    ? list([{ ...searchedTeachers[0], courseCount: -1 }])
                    : list(searchedTeachers),
            );
        });

        await page.goto("/teachers");
        await expect(page.getByRole("link", { name: /陈老师/ })).toBeVisible({
            timeout: 10_000,
        });

        await page.getByLabel("输入教师姓名搜索...").fill("王");

        await expect(page.getByText("加载失败")).toBeVisible({
            timeout: 10_000,
        });
        await expect(page.getByText("未找到匹配的教师，换个名字试试。")).toHaveCount(0);

        await page.getByRole("button", { name: "重试" }).click();

        await expect.poll(() => searchCount).toBe(2);
        await expect(page.getByRole("link", { name: /王老师/ })).toBeVisible();
    });
});
