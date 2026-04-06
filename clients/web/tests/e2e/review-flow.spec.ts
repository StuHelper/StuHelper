import { expect, test, type Page } from "@playwright/test";

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
    await page.route(
        "**/api/v1/course/review/user/notifications/unread-count*",
        (route) =>
            route.fulfill({
                contentType: "application/json",
                body: JSON.stringify({ success: true, data: { count: 0 } }),
            }),
    );
}

test.beforeEach(async ({ page }) => {
    await mockAuth(page);
});

test("authenticated user can publish a review and vote on a course review", async ({
    page,
}) => {
    let createdReviewPayload: Record<string, unknown> | null = null;
    let votePayload: Record<string, unknown> | null = null;

    await page.route("**/api/v1/course/terms", async (route) => {
        await route.fulfill({
            contentType: "application/json",
            body: JSON.stringify({
                success: true,
                data: [{ id: "2025-fall", name: "2025 秋" }],
            }),
        });
    });

    await page.route("**/api/v1/course/courses/1", async (route) => {
        await route.fulfill({
            contentType: "application/json",
            body: JSON.stringify({
                success: true,
                data: {
                    id: 1,
                    name: "高等数学",
                    code: "MATH101",
                    departmentName: "数学系",
                },
            }),
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
                body: JSON.stringify({ success: true, data: null }),
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
            body: JSON.stringify({ success: true, data: [] }),
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

    await page.route("**/api/v1/course/review/reviews/review-1/votes", async (route) => {
        votePayload = route.request().postDataJSON() as Record<string, unknown>;
        await route.fulfill({
            contentType: "application/json",
            body: JSON.stringify({ success: true, data: { voteType: "like" } }),
        });
    });

    await page.goto("/courses/1/reviews/post");

    await page.getByTestId("review-term").selectOption("2025-fall");
    await page.getByTestId("rating-recommendation-5").click();
    await page.getByTestId("rating-content_quality-4").click();
    await page.getByTestId("rating-workload-3").click();
    await page.getByTestId("rating-grading-4").click();
    await page.getByTestId("review-title").fill("端到端评课验证");
    await page
        .getByTestId("review-content")
        .fill("这是一条用于端到端验证的评课内容，长度足够通过校验。");
    await page.getByTestId("review-submit").click();

    await expect(page).toHaveURL(/\/courses\/1\/reviews$/);
    await expect
        .poll(() => createdReviewPayload?.title as string | undefined)
        .toBe("端到端评课验证");
    expect(createdReviewPayload?.courseID).toBe(1);
    expect(createdReviewPayload?.termID).toBe("2025-fall");
    expect(createdReviewPayload?.ratings).toMatchObject({
        recommendation: 5,
        content_quality: 4,
        workload: 3,
        grading: 4,
    });

    // Verify the review list is rendered after redirect
    await expect(page.getByText("讲解清楚，推荐")).toBeVisible({ timeout: 10_000 });

    await page.getByTestId("review-like-review-1").click();
    await expect.poll(() => votePayload?.voteType as string | undefined).toBe(
        "like",
    );
});
