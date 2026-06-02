import { expect, mockNotificationStream, test, type Page } from './fixtures';

const storedUser = {
    id: "user_1",
    name: "alice",
    displayName: "Alice",
    email: "alice@example.com",
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

async function mockAuth(page: Page) {
    await page.addInitScript((u) => {
        localStorage.setItem("stuhelper_user", JSON.stringify(u));
        localStorage.setItem(
            "stuhelper_token_expiry",
            String(Date.now() + 60 * 60 * 1000),
        );
    }, storedUser);

    await page.route("**/api/v1/auth/me", (route) =>
        route.fulfill({
            contentType: "application/json",
            body: JSON.stringify({ success: true, data: storedUser }),
        }),
    );
    await page.route("**/api/v1/auth/refresh", (route) =>
        route.fulfill({
            contentType: "application/json",
            body: JSON.stringify({ success: true, data: { expiresIn: 3600 } }),
        }),
    );
    await page.route("**/api/v1/user/me", (route) =>
        route.fulfill({
            contentType: "application/json",
            body: JSON.stringify({
                success: true,
                data: {
                    displayName: storedUser.displayName,
                    identityStatus: "approved",
                    verificationStatus: "approved",
                    phoneBound: true,
                    capabilities: storedUser.capabilities,
                },
            }),
        }),
    );
    await page.route("**/api/v1/user/identity", (route) =>
        route.fulfill({
            contentType: "application/json",
            body: JSON.stringify({
                success: true,
                data: { verified: true, status: "verified" },
            }),
        }),
    );
    await page.route("**/api/v1/user/profile", (route) =>
        route.fulfill({
            contentType: "application/json",
            body: JSON.stringify({
                success: true,
                data: { verificationStatus: "verified", schoolID: 4111010006 },
            }),
        }),
    );
    await page.route("**/api/v1/user/qq-binding", (route) =>
        route.fulfill({
            status: 404,
            contentType: "application/json",
            body: JSON.stringify({
                success: false,
                error: { code: "A0040404", message: "not bound" },
            }),
        }),
    );
    await page.route(
        "**/api/v1/course/review/user/notifications/unread-count*",
        (route) =>
            route.fulfill({
                contentType: "application/json",
                body: JSON.stringify({ success: true, data: { count: 0 } }),
            }),
    );
    await mockNotificationStream(page);
}

function requireRecord(
    value: Record<string, unknown> | null,
    message: string,
): Record<string, unknown> {
    expect(value, message).not.toBeNull();
    return value as Record<string, unknown>;
}

function ratingDimension(key: string, name: string, sortOrder: number) {
    return {
        id: `dim-${key}`,
        schoolID: 4111010006,
        key,
        name,
        description: "",
        sortOrder,
        isActive: true,
        createdAt: "2026-04-03T10:00:00Z",
        updatedAt: "2026-04-03T10:00:00Z",
    };
}

function term(id: string, name: string, isCurrent = true) {
    return { id, name, isCurrent };
}

function course(overrides: Record<string, unknown> = {}) {
    return {
        id: 1,
        schoolID: 4111010006,
        departmentID: 1,
        departmentName: "数学系",
        code: "MATH101",
        name: "高等数学",
        credits: 4,
        category: "公共基础",
        reviewCount: 1,
        ...overrides,
    };
}

function teacher(overrides: Record<string, unknown> = {}) {
    return {
        teacherID: 1,
        teacherName: "王老师",
        departmentName: "数学系",
        avgRating: 4.5,
        courseCount: 1,
        reviewCount: 1,
        ...overrides,
    };
}

async function mockPostReviewBootstrap(page: Page, termsData: unknown) {
    await page.route("**/api/v1/course/terms", async (route) => {
        await route.fulfill({
            contentType: "application/json",
            body: JSON.stringify({
                success: true,
                data: termsData,
            }),
        });
    });

    await page.route("**/api/v1/course/review/rating-dimensions", async (route) => {
        await route.fulfill({
            contentType: "application/json",
            body: JSON.stringify({
                success: true,
                data: [ratingDimension("difficulty", "Difficulty", 1)],
            }),
        });
    });

    await page.route("**/api/v1/course/review/drafts", async (route) => {
        await route.fulfill({
            status: 404,
            contentType: "application/json",
            body: JSON.stringify({
                success: false,
                error: { code: "R0040404", message: "draft not found" },
            }),
        });
    });
}

test.beforeEach(async ({ page }) => {
    await mockAuth(page);
});

test("post review page fails closed when terms response is malformed", async ({
    page,
}) => {
    await mockPostReviewBootstrap(page, [{ id: "2025-fall", name: "2025 秋" }]);

    await page.goto("/courses/reviews/post");

    await expect(
        page.getByRole("alert").filter({ hasText: /Load failed|加载失败/i }),
    ).toBeVisible({ timeout: 10_000 });
    await expect(page.getByTestId("review-term")).not.toContainText("2025 秋");
});

test("post review page fails closed when rating dimensions response is malformed", async ({
    page,
}) => {
    await mockPostReviewBootstrap(page, [term("2025-fall", "2025 秋")]);
    await page.route("**/api/v1/course/review/rating-dimensions", async (route) => {
        await route.fulfill({
            contentType: "application/json",
            body: JSON.stringify({ success: true, data: [{ key: "difficulty" }] }),
        });
    });

    await page.goto("/courses/reviews/post");

    await expect(
        page.getByText(/评分维度加载失败|Failed to load rating dimensions/i),
    ).toBeVisible({ timeout: 10_000 });
    await expect(
        page.getByText(/暂无可用评分维度|No rating dimensions/i),
    ).toHaveCount(0);
});

test("post review course autocomplete fails closed when search response is malformed", async ({
    page,
}) => {
    await mockPostReviewBootstrap(page, [term("2025-fall", "2025 秋")]);
    await page.route("**/api/v1/course/courses/search*", async (route) => {
        await route.fulfill({
            contentType: "application/json",
            body: JSON.stringify({
                success: true,
                data: {
                    list: [course({ credits: "4" })],
                    total: 1,
                },
            }),
        });
    });

    await page.goto("/courses/reviews/post");

    await page
        .getByPlaceholder(/高等数学|gaodengshuxue|gdsx|Search by course/i)
        .fill("高等数学");

    await expect(
        page.getByRole("alert").filter({ hasText: /Load failed|加载失败/i }),
    ).toBeVisible({ timeout: 10_000 });
    await expect(page.getByText(/not listed yet|暂未被收录/i)).toHaveCount(0);
});

test("post review teacher selector fails closed when teachers response is malformed", async ({
    page,
}) => {
    await mockPostReviewBootstrap(page, [term("2025-fall", "2025 秋")]);
    await page.route("**/api/v1/course/courses/search*", async (route) => {
        await route.fulfill({
            contentType: "application/json",
            body: JSON.stringify({
                success: true,
                data: {
                    list: [course()],
                    total: 1,
                },
            }),
        });
    });
    await page.route("**/api/v1/course/review/courses/1/teachers", async (route) => {
        await route.fulfill({
            contentType: "application/json",
            body: JSON.stringify({
                success: true,
                data: [teacher({ courseCount: -1 })],
            }),
        });
    });

    await page.goto("/courses/reviews/post");

    await page
        .getByPlaceholder(/高等数学|gaodengshuxue|gdsx|Search by course/i)
        .fill("高等数学");
    await page.getByText("高等数学").click();

    await expect(
        page.getByRole("alert").filter({ hasText: /Load failed|加载失败/i }),
    ).toBeVisible({ timeout: 10_000 });
    await expect(page.getByTestId("review-course-selected")).toContainText(
        "高等数学",
    );
});

test("post review preselected course fails closed when course response is malformed", async ({
    page,
}) => {
    await mockPostReviewBootstrap(page, [term("2025-fall", "2025 秋")]);
    await page.addInitScript(() => {
        window.sessionStorage.setItem("review_post_preselect_course_id", "1");
    });
    await page.route("**/api/v1/course/courses/1", async (route) => {
        await route.fulfill({
            contentType: "application/json",
            body: JSON.stringify({
                success: true,
                data: course({ reviewCount: "1" }),
            }),
        });
    });

    await page.goto("/courses/reviews/post");

    await expect(
        page.getByRole("alert").filter({ hasText: /Load failed|加载失败/i }),
    ).toBeVisible({ timeout: 10_000 });
    await expect(page.getByTestId("review-course-selected")).toHaveCount(0);
});

test("post review content check fails closed when response is malformed", async ({
    page,
}) => {
    let createReviewCalled = false;

    await mockPostReviewBootstrap(page, [term("2025-fall", "2025 秋")]);
    await page.route("**/api/v1/course/courses/search*", async (route) => {
        await route.fulfill({
            contentType: "application/json",
            body: JSON.stringify({
                success: true,
                data: {
                    list: [course()],
                    total: 1,
                },
            }),
        });
    });
    await page.route("**/api/v1/course/review/courses/1/teachers", async (route) => {
        await route.fulfill({
            contentType: "application/json",
            body: JSON.stringify({ success: true, data: [] }),
        });
    });
    await page.route("**/api/v1/course/review/content/check", async (route) => {
        await route.fulfill({
            contentType: "application/json",
            body: JSON.stringify({
                success: true,
                data: { isValid: true, matchCount: -1 },
            }),
        });
    });
    await page.route("**/api/v1/course/review/reviews", async (route) => {
        if (route.request().method() === "POST") {
            createReviewCalled = true;
        }
        await route.fulfill({
            status: 201,
            contentType: "application/json",
            body: JSON.stringify({ success: true, data: { id: "review-new" } }),
        });
    });
    await page.route("**/api/v1/course/review/drafts", async (route) => {
        if (route.request().method() === "POST") {
            await route.fulfill({
                contentType: "application/json",
                body: JSON.stringify({
                    success: true,
                    data: {
                        id: "draft-1",
                        updatedAt: "2026-04-03T10:00:00Z",
                        ...route.request().postDataJSON(),
                    },
                }),
            });
            return;
        }

        await route.fulfill({
            status: 404,
            contentType: "application/json",
            body: JSON.stringify({
                success: false,
                error: { code: "R0040404", message: "draft not found" },
            }),
        });
    });

    await page.goto("/courses/reviews/post");
    await page
        .getByPlaceholder(/高等数学|gaodengshuxue|gdsx|Search by course/i)
        .fill("高等数学");
    await page.getByText("高等数学").click();
    await page.getByTestId("review-term").selectOption("2025-fall");
    await page.getByTestId("rating-difficulty-5").click();
    await page.getByTestId("review-title").fill("内容审核异常验证");
    await page
        .getByTestId("review-content")
        .fill("这是一条用于验证内容审核异常时不会提交的评课正文，长度足够通过前端校验。");

    await page.getByTestId("review-submit").click();

    await expect(
        page.locator('p[role="alert"]').filter({ hasText: /Load failed|加载失败/i }),
    ).toBeVisible({ timeout: 10_000 });
    await expect(page).toHaveURL(/\/courses\/reviews\/post$/);
    expect(createReviewCalled).toBe(false);
});

test("authenticated user can publish a review and vote on a course review", async ({
    page,
}) => {
    let createdReviewPayload: Record<string, unknown> | null = null;
    let savedDraftPayload: Record<string, unknown> | null = null;
    let votePayload: Record<string, unknown> | null = null;

    await page.route("**/api/v1/course/terms", async (route) => {
        await route.fulfill({
            contentType: "application/json",
            body: JSON.stringify({
                success: true,
                data: [term("2025-fall", "2025 秋")],
            }),
        });
    });

    await page.route("**/api/v1/course/review/rating-dimensions", async (route) => {
        await route.fulfill({
            contentType: "application/json",
            body: JSON.stringify({
                success: true,
                data: [
                    ratingDimension("difficulty", "Difficulty", 1),
                    ratingDimension("workload", "Workload", 2),
                    ratingDimension("usefulness", "Usefulness", 3),
                    ratingDimension("teaching", "Teaching", 4),
                    ratingDimension("grading", "Grading", 5),
                ],
            }),
        });
    });

    await page.route("**/api/v1/course/courses/1", async (route) => {
        await route.fulfill({
            contentType: "application/json",
            body: JSON.stringify({
                success: true,
                data: course(),
            }),
        });
    });

    await page.route("**/api/v1/course/courses/search*", async (route) => {
        await route.fulfill({
            contentType: "application/json",
            body: JSON.stringify({
                success: true,
                data: {
                    list: [course()],
                    total: 1,
                },
            }),
        });
    });

    await page.route("**/api/v1/course/review/courses/1/favorites", async (route) => {
        await route.fulfill({
            contentType: "application/json",
            body: JSON.stringify({ success: true, data: { favorited: false } }),
        });
    });

    await page.route("**/api/v1/course/review/reviews", async (route) => {
        if (route.request().method() !== "POST") {
            await route.fallback();
            return;
        }

        createdReviewPayload = route.request().postDataJSON() as Record<
            string,
            unknown
        >;
        await route.fulfill({
            status: 201,
            contentType: "application/json",
            body: JSON.stringify({
                success: true,
                data: { id: "review-new" },
            }),
        });
    });

    const reviewListResponse = {
        success: true,
        data: {
            list: [
                {
                    id: "review-1",
                    courseID: 1,
                    courseName: "高等数学",
                    termID: "2025-fall",
                    termName: "2025 秋",
                    title: "讲解清楚，推荐",
                    content: "这是一条用于端到端验证的评课内容，长度足够通过校验。",
                    ratings: {
                        recommendation: 5,
                        content_quality: 4,
                        workload: 3,
                        grading: 4,
                    },
                    likeCount: 0,
                    dislikeCount: 0,
                    replyCount: 0,
                    status: "published",
                    createdAt: "2026-04-03T10:00:00Z",
                },
            ],
            total: 1,
            page: 1,
            pageSize: 20,
        },
    };

    await page.route("**/api/v1/course/review/courses/1/reviews**", async (route) => {
        await route.fulfill({
            contentType: "application/json",
            body: JSON.stringify(reviewListResponse),
        });
    });

    await page.route(
        "**/api/v1/course/review/courses/1/rating-stats",
        async (route) => {
            await route.fulfill({
                contentType: "application/json",
                body: JSON.stringify({
                    success: true,
                    data: {
                        courseID: 1,
                        overall: { termName: "总体", dimensions: [] },
                        byTerm: [],
                        allDimensionKeys: [],
                    },
                }),
            });
        },
    );

    await page.route("**/api/v1/course/review/courses/1/teachers", async (route) => {
        await route.fulfill({
            contentType: "application/json",
            body: JSON.stringify({ success: true, data: [] }),
        });
    });

    await page.route("**/api/v1/course/review/courses/1/rating-trend*", async (route) => {
        await route.fulfill({
            contentType: "application/json",
            body: JSON.stringify({ success: true, data: { trend: [] } }),
        });
    });

    await page.route("**/api/v1/course/review/content/check", async (route) => {
        await route.fulfill({
            contentType: "application/json",
            body: JSON.stringify({
                success: true,
                data: { isValid: true },
            }),
        });
    });

    await page.route("**/api/v1/course/review/drafts", async (route) => {
        if (route.request().method() === "POST") {
            savedDraftPayload = route.request().postDataJSON() as Record<
                string,
                unknown
            >;
            await route.fulfill({
                contentType: "application/json",
                body: JSON.stringify({
                    success: true,
                    data: {
                        id: "draft-1",
                        ...savedDraftPayload,
                        updatedAt: "2026-04-03T10:00:00Z",
                    },
                }),
            });
            return;
        }

        if (route.request().method() === "DELETE") {
            await route.fulfill({
                contentType: "application/json",
                body: JSON.stringify({ success: true, data: null }),
            });
            return;
        }

        await route.fulfill({
            status: 404,
            contentType: "application/json",
            body: JSON.stringify({
                success: false,
                error: { code: "R0040404", message: "draft not found" },
            }),
        });
    });

    await page.route("**/api/v1/course/review/reviews/review-1/votes", async (route) => {
        votePayload = route.request().postDataJSON() as Record<string, unknown>;
        await route.fulfill({
            contentType: "application/json",
            body: JSON.stringify({ success: true, data: { voteType: "like" } }),
        });
    });

    await page.goto("/courses/reviews/post");

    await expect(page).toHaveURL(/\/courses\/reviews\/post$/);
    await page
        .getByPlaceholder(/高等数学|gaodengshuxue|gdsx|Search by course/i)
        .fill("高等数学");
    await page.getByText("高等数学").click();
    await expect(page.getByTestId("review-course-selected")).toContainText(
        "高等数学",
    );

    await page.getByTestId("review-term").selectOption("2025-fall");
    await page.getByTestId("rating-difficulty-5").click();
    await page.getByTestId("rating-workload-3").click();
    await page.getByTestId("rating-usefulness-4").click();
    await page.getByTestId("rating-teaching-4").click();
    await page.getByTestId("rating-grading-4").click();
    await page.getByTestId("review-title").fill("端到端评课验证");
    await page
        .getByTestId("review-content")
        .fill("这是一条用于端到端验证的评课内容，长度足够通过校验。");
    await page.getByTestId("review-grade").selectOption("A+");
    await expect.poll(() => savedDraftPayload?.grade as string | undefined).toBe(
        "A+",
    );
    await page.getByTestId("review-submit").click();

    await expect(page).toHaveURL(/\/courses\/1\/reviews$/);
    await expect
        .poll(() => createdReviewPayload?.title as string | undefined)
        .toBe("端到端评课验证");
    const createdPayload = requireRecord(
        createdReviewPayload,
        "created review payload should be captured",
    );
    expect(createdPayload.courseID).toBe(1);
    expect(createdPayload.termID).toBe("2025-fall");
    expect(createdPayload.grade).toBe("A+");
    expect(createdPayload.ratings).toMatchObject({
        difficulty: 5,
        workload: 3,
        usefulness: 4,
        teaching: 4,
        grading: 4,
    });

    // Verify the review list is rendered after redirect
    await expect(page.getByText("讲解清楚，推荐")).toBeVisible({ timeout: 10_000 });

    await page.getByTestId("review-like-review-1").click();
    await expect.poll(() => votePayload?.voteType as string | undefined).toBe(
        "like",
    );
});
