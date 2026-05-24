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
    await page.route("**/api/v1/course/categories*", (route) =>
        route.fulfill(ok([{ id: 1, name: "工科" }])),
    );
    await page.route("**/api/v1/course/departments*", (route) =>
        route.fulfill(ok(departments)),
    );
    await page.route("**/api/v1/course/courses?*", (route) =>
        route.fulfill(list(departmentCourses)),
    );
    await page.route("**/api/v1/course/review/reviews/latest*", (route) => {
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
}

async function mockTeacherApi(page: Page) {
    await page.route("**/api/v1/course/review/teachers/hot*", (route) =>
        route.fulfill(list(popularTeachers)),
    );
    await page.route("**/api/v1/course/review/teachers?*", (route) =>
        route.fulfill(list(searchedTeachers)),
    );
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

test.describe("Course community surfaces", () => {
    test.beforeEach(async ({ page }) => {
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
        await expect(page.getByText("最热排序测评")).toBeVisible();
        await expect(page.getByRole("tab", { name: "最热" })).toHaveAttribute(
            "aria-selected",
            "true",
        );

        await page.getByRole("tab", { name: "精选" }).click();
        await expect(page.getByText("高分排序测评")).toBeVisible();

        await page
            .getByRole("button", { name: "计算机科学与技术学院" })
            .click();
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
        await expect(page.getByText("搜索结果")).toBeVisible();
        await expect(page.getByRole("link", { name: /王老师/ })).toBeVisible({
            timeout: 10_000,
        });
    });
});
