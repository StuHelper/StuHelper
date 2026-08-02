import {
    expect,
    mockNotificationStream,
    test,
    type Page,
    type Route,
} from './fixtures';

const user = {
    id: "u2",
    name: "bob",
    displayName: "Bob",
    email: "bob@example.com",
    roles: ["verified_student"],
    capabilities: [
        "review:list:full",
        "review:create",
        "review:edit:own",
        "review:delete:own",
    ],
    globalCapabilities: [
        "review:list:full",
        "review:create",
        "review:edit:own",
        "review:delete:own",
    ],
    capabilityGrants: [],
    isPlatformAdmin: false,
    canAccessAdmin: false,
};

function json(data: unknown, status = 200) {
    return {
        status,
        contentType: "application/json",
        body: JSON.stringify(data),
    };
}

function ok(data: unknown = null) {
    return json({ success: true, data });
}

function list<T>(items: T[], total = items.length) {
    return ok({ list: items, total, page: 1, pageSize: 20 });
}

async function fulfillUnexpected(route: Route) {
    const url = new URL(route.request().url());
    await route.fulfill(
        json(
            {
                success: false,
                error: {
                    code: "E2E_UNMOCKED",
                    message: `unmocked web e2e request: ${route.request().method()} ${url.pathname}`,
                },
            },
            500,
        ),
    );
}

async function mockAuth(page: Page) {
    await page.addInitScript((u) => {
        localStorage.setItem("stuhelper_user", JSON.stringify(u));
        localStorage.setItem(
            "stuhelper_token_expiry",
            String(Date.now() + 60 * 60 * 1000),
        );
    }, user);

    await page.route("**/api/v1/auth/me", (route) => route.fulfill(ok(user)));
    await page.route("**/api/v1/auth/refresh", (route) =>
        route.fulfill(ok({ expiresIn: 3600 })),
    );
    await page.route(
        "**/api/v1/course/review/user/notifications/unread-count*",
        (route) => route.fulfill(ok({ count: 0 })),
    );
    await mockNotificationStream(page);
    await page.route("**/api/v1/user/identity", (route) =>
        route.fulfill(ok({ verified: true, status: "verified" })),
    );
    await page.route("**/api/v1/user/profile", (route) =>
        route.fulfill(
            ok({
                verificationStatus: "verified",
                schoolName: "测试大学",
                schoolID: 4111010006,
            }),
        ),
    );
    await page.route("**/api/v1/user/qq-binding", (route) =>
        route.fulfill(
            json(
                {
                    success: false,
                    error: { code: "A0040404", message: "not bound" },
                },
                404,
            ),
        ),
    );
}

const ownReview = {
    id: "my-action-review",
    courseID: 42,
    courseName: "操作系统",
    termID: "2026-spring",
    title: "需要维护的评价",
    content: "原始评价内容，便于确认编辑前状态。",
    ratings: { recommendation: 4, content_quality: 4 },
    likeCount: 2,
    dislikeCount: 0,
    replyCount: 0,
    status: "published",
    createdAt: "2026-05-24T04:00:00Z",
};

const publicReview = {
    id: "public-report-review",
    courseID: 42,
    courseName: "操作系统",
    teacherID: 9,
    teacherName: "赵老师",
    termID: "2026-spring",
    termName: "2026 春",
    title: "可举报的公开评价",
    content: "这条公开评价用于验证举报菜单和提交请求。",
    ratings: { recommendation: 5, workload: 3 },
    likeCount: 6,
    dislikeCount: 1,
    replyCount: 0,
    status: "published",
    createdAt: "2026-05-24T04:00:00Z",
    authorDisplayName: "匿名同学",
};

const existingReply = {
    id: "reply-1",
    reviewID: "public-report-review",
    parentID: null,
    content: "已有回复内容",
    likeCount: 0,
    status: "published",
    isOwner: true,
    createdAt: "2026-05-24T04:05:00Z",
    updatedAt: "2026-05-24T04:05:00Z",
};

const createdReply = {
    id: "reply-created",
    reviewID: "public-report-review",
    parentID: null,
    content: "新增回复内容",
    likeCount: 0,
    status: "published",
    isOwner: true,
    createdAt: "2026-05-24T04:10:00Z",
    updatedAt: "2026-05-24T04:10:00Z",
};

const nextPageReview = {
    ...publicReview,
    id: "public-report-review-page-2",
    title: "第二页公开评价",
    content: "这是加载更多后追加的评价。",
    likeCount: 1,
    dislikeCount: 0,
};

async function mockCourseDetail(page: Page) {
    await page.route("**/api/v1/course/courses/42", (route) =>
        route.fulfill(
            ok({
                id: 42,
                name: "操作系统",
                code: "CS304",
                departmentID: 1,
                departmentName: "计算机科学与技术学院",
                credits: 4,
                reviewCount: 1,
            }),
        ),
    );
    await page.route("**/api/v1/course/review/courses/42/reviews*", (route) =>
        route.fulfill(list([publicReview])),
    );
    await page.route(
        "**/api/v1/course/review/courses/42/rating-stats*",
        (route) =>
            route.fulfill(
                ok({
                    courseID: 42,
                    overall: { termName: "总体", dimensions: [] },
                    byTerm: [],
                    allDimensionKeys: [],
                }),
            ),
    );
    await page.route("**/api/v1/course/review/courses/42/teachers*", (route) =>
        route.fulfill(
            ok([
                {
                    teacherID: 9,
                    teacherName: "赵老师",
                    departmentName: "计算机科学与技术学院",
                    courseCount: 1,
                    reviewCount: 1,
                },
            ]),
        ),
    );
    await page.route(
        "**/api/v1/course/review/courses/42/rating-trend*",
        (route) => route.fulfill(ok({ trend: [] })),
    );
    await page.route(
        "**/api/v1/course/review/courses/42/favorites",
        (route) => route.fulfill(ok({ favorited: false })),
    );
}

async function mockReviewFeed(page: Page) {
    await page.route("**/api/v1/course/categories*", (route) =>
        route.fulfill(ok([])),
    );
    await page.route("**/api/v1/course/departments*", (route) =>
        route.fulfill(ok([])),
    );
    await page.route("**/api/v1/course/review/reviews/latest*", (route) =>
        route.fulfill(list([publicReview])),
    );
}

test.describe("Review actions", () => {
    test.beforeEach(async ({ page }) => {
        await page.route("**/api/v1/**", fulfillUnexpected);
        await mockAuth(page);
    });

    test("user edits and deletes their own review from user center", async ({
        page,
    }) => {
        const updatedContent = "编辑后的课程评价内容符合最低长度";
        let updateBody: unknown = null;
        let deleteCalled = false;

        await page.route("**/api/v1/course/review/user/reviews*", (route) =>
            route.fulfill(list([ownReview], 1)),
        );
        await page.route("**/api/v1/course/review/user/votes*", (route) =>
            route.fulfill(list([], 0)),
        );
        await page.route("**/api/v1/course/review/user/favorites*", (route) =>
            route.fulfill(list([], 0)),
        );
        await page.route(
            "**/api/v1/course/review/reviews/my-action-review",
            async (route) => {
                if (route.request().method() === "PUT") {
                    updateBody = route.request().postDataJSON();
                    await route.fulfill(
                        ok({ ...ownReview, content: updatedContent }),
                    );
                    return;
                }
                if (route.request().method() === "DELETE") {
                    deleteCalled = true;
                    await route.fulfill(ok());
                    return;
                }
                await fulfillUnexpected(route);
            },
        );

        await page.goto("/user/reviews");

        await expect(page.getByText("需要维护的评价")).toBeVisible({
            timeout: 10_000,
        });
        await page.getByRole("button", { name: "编辑我的评价" }).click();
        await page.locator("textarea").fill(updatedContent);
        await page.getByRole("button", { name: "保存" }).click();

        await expect
            .poll(() => updateBody)
            .toMatchObject({
                content: updatedContent,
                ratings: { recommendation: 4, content_quality: 4 },
            });
        await expect(page.getByText(updatedContent)).toBeVisible();
        await expect(page.getByText("评价已更新")).toBeVisible();

        await page.getByRole("button", { name: "删除我的评价" }).click();
        await expect(
            page.getByRole("group", { name: "确定要删除这条评价吗？" }),
        ).toBeVisible();
        expect(deleteCalled).toBe(false);

        await page
            .getByTestId(`review-delete-confirm-action-${ownReview.id}`)
            .click();
        await expect.poll(() => deleteCalled).toBe(true);
        await expect(page.getByText("需要维护的评价")).toHaveCount(0);
        await expect(page.getByText("评价已删除")).toBeVisible();
    });

    test("user toggles course favorite from course detail", async ({
        page,
    }) => {
        const favoriteMethods: string[] = [];

        await mockCourseDetail(page);
        await page.route(
            "**/api/v1/course/review/courses/42/favorites",
            async (route) => {
                const method = route.request().method();
                favoriteMethods.push(method);
                if (method === "GET") {
                    await route.fulfill(ok({ favorited: false }));
                    return;
                }
                await route.fulfill(ok());
            },
        );

        await page.goto("/courses/42/reviews");

        const favoriteButton = page.getByRole("button", { name: "收藏" });
        await expect(favoriteButton).toBeVisible({ timeout: 10_000 });
        await favoriteButton.click();
        await expect(
            page.getByRole("button", { name: "已收藏" }),
        ).toHaveAttribute("aria-pressed", "true");

        await page.getByRole("button", { name: "已收藏" }).click();
        await expect(
            page.getByRole("button", { name: "收藏" }),
        ).toHaveAttribute("aria-pressed", "false");
        await expect
            .poll(() => favoriteMethods)
            .toEqual(["GET", "POST", "DELETE"]);
    });

    test("user votes, replies, and deletes an owned reply from course detail", async ({
        page,
    }) => {
        const voteBodies: unknown[] = [];
        let createReplyBody: unknown = null;
        let deletedReplyID = "";

        await mockCourseDetail(page);
        await page.route(
            "**/api/v1/course/review/courses/42/reviews*",
            (route) =>
                route.fulfill(list([{ ...publicReview, replyCount: 1 }])),
        );
        await page.route(
            "**/api/v1/course/review/reviews/public-report-review/votes",
            async (route) => {
                voteBodies.push(route.request().postDataJSON());
                await route.fulfill(ok());
            },
        );
        await page.route(
            "**/api/v1/course/review/reviews/public-report-review/replies",
            async (route) => {
                if (route.request().method() === "GET") {
                    await route.fulfill(list([existingReply], 1));
                    return;
                }
                createReplyBody = route.request().postDataJSON();
                await route.fulfill(ok(createdReply));
            },
        );
        await page.route(
            "**/api/v1/course/review/replies/reply-1",
            async (route) => {
                deletedReplyID = "reply-1";
                await route.fulfill(ok());
            },
        );
        await page.goto("/courses/42/reviews");
        await expect(page.getByText("可举报的公开评价")).toBeVisible({
            timeout: 10_000,
        });

        await page.getByTestId("review-like-public-report-review").click();
        await expect
            .poll(() => voteBodies)
            .toContainEqual({ voteType: "like" });
        await expect(
            page.getByTestId("review-like-public-report-review"),
        ).toHaveAttribute("aria-pressed", "true");

        await page.getByRole("button", { name: "查看回复" }).click();
        await expect(page.getByText("已有回复内容")).toBeVisible();

        await page.getByLabel("回复内容").fill("新增回复内容");
        await page.getByRole("button", { name: "发送" }).click();

        await expect
            .poll(() => createReplyBody)
            .toEqual({ content: "新增回复内容" });
        await expect(page.getByText("新增回复内容")).toBeVisible();

        const existingReplyCard = page.getByTestId("reply-card-reply-1");
        await existingReplyCard.getByRole("button", { name: "删除" }).click();
        await expect(
            existingReplyCard.getByTestId("reply-delete-confirm-reply-1"),
        ).toContainText("确定要删除这条回复吗？");
        await existingReplyCard.getByRole("button", { name: "确认" }).click();
        await expect.poll(() => deletedReplyID).toBe("reply-1");
        await expect(page.getByText("已有回复内容")).toHaveCount(0);
    });

    test("invalid reply list response fails closed and can retry", async ({
        page,
    }) => {
        let loadCount = 0;

        await mockCourseDetail(page);
        await page.route(
            "**/api/v1/course/review/courses/42/reviews*",
            (route) =>
                route.fulfill(list([{ ...publicReview, replyCount: 1 }])),
        );
        await page.route(
            "**/api/v1/course/review/reviews/public-report-review/replies",
            async (route) => {
                if (route.request().method() !== "GET") {
                    await route.fallback();
                    return;
                }

                loadCount += 1;
                await route.fulfill(
                    loadCount === 1
                        ? list([{ ...existingReply, likeCount: -1 }], 1)
                        : list([existingReply], 1),
                );
            },
        );

        await page.goto("/courses/42/reviews");
        await expect(page.getByText("可举报的公开评价")).toBeVisible({
            timeout: 10_000,
        });

        await page.getByRole("button", { name: "查看回复" }).click();

        await expect(page.getByText("加载回复失败")).toBeVisible();
        await expect(page.getByText("暂无回复")).toHaveCount(0);

        await page.getByRole("button", { name: "重试" }).click();

        await expect.poll(() => loadCount).toBe(2);
        await expect(page.getByText("已有回复内容")).toBeVisible();
    });

    test("user loads additional reviews from course detail", async ({
        page,
    }) => {
        const requestedReviewQueries: string[] = [];

        await mockCourseDetail(page);
        await page.route(
            "**/api/v1/course/review/courses/42/reviews*",
            (route) => {
                const url = new URL(route.request().url());
                const requestedPage = url.searchParams.get("page") ?? "1";
                requestedReviewQueries.push(url.search);
                return route.fulfill(
                    list(
                        requestedPage === "2"
                            ? [nextPageReview]
                            : [publicReview],
                        2,
                    ),
                );
            },
        );

        await page.goto("/courses/42/reviews");

        await expect(page.getByText("可举报的公开评价")).toBeVisible({
            timeout: 10_000,
        });
        await page.getByRole("button", { name: "加载更多" }).click();

        await expect(page.getByText("第二页公开评价")).toBeVisible();
        await expect
            .poll(() =>
                requestedReviewQueries.map((query) => {
                    const params = new URLSearchParams(query);
                    return {
                        page: params.get("page"),
                        pageSize: params.get("pageSize"),
                        sort: params.get("sort"),
                    };
                }),
            )
            .toEqual([
                { page: "1", pageSize: "20", sort: "time" },
                { page: "2", pageSize: "20", sort: "time" },
            ]);
    });

    test("user reports a public review from the review feed", async ({
        page,
    }) => {
        let reportBody: unknown = null;

        await mockReviewFeed(page);
        await page.route(
            "**/api/v1/course/review/reviews/public-report-review/reports",
            async (route) => {
                reportBody = route.request().postDataJSON();
                await route.fulfill(ok());
            },
        );

        await page.goto("/courses/reviews");

        await expect(page.getByText("可举报的公开评价")).toBeVisible({
            timeout: 10_000,
        });
        await page.getByRole("button", { name: "举报" }).click();
        await expect(page.getByText("举报原因")).toBeVisible();
        await page.getByRole("button", { name: "垃圾信息" }).click();

        await expect.poll(() => reportBody).toEqual({ reason: "spam" });
        await expect(page.getByRole("alert")).toContainText("举报已提交");
    });
});
