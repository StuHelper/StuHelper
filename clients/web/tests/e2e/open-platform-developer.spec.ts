import { expect, mockNotificationStream, test, type Page } from './fixtures';

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

type QueryRecord = Record<string, string>;

type MockRedirectURIRequest = {
    id: number;
    redirectURIs: string[];
    reason: string;
    status: string;
    reviewerUserID: number | null;
    reviewedAt: string | null;
    decisionNote: string | null;
    createdAt: string;
    updatedAt: string;
};

type MockAppListItem = {
    app: Record<string, unknown>;
    scopes: Array<Record<string, unknown>>;
    redirectURIRequests: MockRedirectURIRequest[];
};

function captureQuery(urlString: string): QueryRecord {
    return Object.fromEntries(new URL(urlString).searchParams.entries());
}

function ok(data: unknown = null) {
    return {
        contentType: "application/json",
        body: JSON.stringify({ success: true, data }),
    };
}

function makeDeveloperApp(input: {
    id: number;
    displayName: string;
    status: string;
}): MockAppListItem {
    const suffix = String(input.id).padStart(2, "0");

    return {
        app: {
            id: input.id,
            clientID: `op_app_${suffix}`,
            displayName: input.displayName,
            description: `${input.displayName} integration`,
            homepageURL: `https://app-${suffix}.example.com`,
            privacyPolicyURL: `https://app-${suffix}.example.com/privacy`,
            redirectURIs: [`https://app-${suffix}.example.com/callback`],
            status: input.status,
            createdAt: "2026-05-01T10:00:00Z",
            updatedAt: "2026-05-01T10:00:00Z",
        },
        scopes: [
            {
                id: input.id,
                scope: "profile.basic.read",
                displayName: "用户基本信息",
                sensitivity: "low",
                fields: ["用户名"],
                reason: "显示用户",
                status: input.status === "pending" ? "pending" : "approved",
                reviewerUserID: null,
                reviewedAt: null,
                decisionNote: null,
                createdAt: "2026-05-01T10:00:00Z",
                updatedAt: "2026-05-01T10:00:00Z",
            },
        ],
        redirectURIRequests: [],
    };
}

async function mockAuth(page: Page) {
    await page.addInitScript((u) => {
        localStorage.setItem("stuhelper_user", JSON.stringify(u));
        localStorage.setItem(
            "stuhelper_token_expiry",
            String(Date.now() + 60 * 60 * 1000),
        );
    }, user);

    await page.route("**/api/v1/auth/me", (route) =>
        route.fulfill({
            contentType: "application/json",
            body: JSON.stringify({ success: true, data: user }),
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
    await mockNotificationStream(page);
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
                data: {
                    verificationStatus: "verified",
                    schoolName: "测试大学",
                    schoolID: 4111010006,
                },
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
}

test.describe("Open Platform developer portal", () => {
    test.beforeEach(async ({ page }) => {
        await mockAuth(page);
    });

    test("invalid developer app list response fails closed and can retry", async ({
        page,
    }) => {
        let loadCount = 0;

        await page.route("**/api/v1/open-platform/apps**", async (route) => {
            const request = route.request();
            const url = new URL(request.url());

            if (
                request.method() !== "GET" ||
                url.pathname !== "/api/v1/open-platform/apps"
            ) {
                await route.fulfill({
                    status: 500,
                    contentType: "application/json",
                    body: JSON.stringify({
                        success: false,
                        error: {
                            code: "E2E_UNMOCKED",
                            message: `unmocked ${request.method()} ${url.pathname}`,
                        },
                    }),
                });
                return;
            }

            loadCount += 1;
            const malformedApp = makeDeveloperApp({
                id: 1,
                displayName: "Portal App 01",
                status: "approved",
            });
            await route.fulfill(
                loadCount === 1
                    ? ok({
                          list: [
                              {
                                  ...malformedApp,
                                  app: {
                                      ...malformedApp.app,
                                      status: "enabled",
                                  },
                              },
                          ],
                          total: 1,
                      })
                    : ok({
                          list: [
                              makeDeveloperApp({
                                  id: 1,
                                  displayName: "Portal App 01",
                                  status: "approved",
                              }),
                          ],
                          total: 1,
                      }),
            );
        });

        await page.goto("/developers/apps");

        const status = page.getByRole("status").filter({
            hasText: /加载失败|Load failed/,
        });
        await expect(status).toBeVisible({ timeout: 10_000 });
        await expect(page.getByText(/应用列表加载失败|Failed to load applications/)).toBeVisible();
        await expect(page.getByText(/暂无应用|No applications/)).toHaveCount(0);

        await status.getByRole("button", { name: /重试|Retry/ }).click();

        await expect.poll(() => loadCount).toBe(2);
        await expect(
            page.locator("article").filter({ hasText: "Portal App 01" }),
        ).toBeVisible();
    });

    test("invalid developer app audit response fails closed and can retry", async ({
        page,
    }) => {
        let auditLoadCount = 0;

        await page.route("**/api/v1/open-platform/apps**", async (route) => {
            const request = route.request();
            const url = new URL(request.url());

            if (
                request.method() === "GET" &&
                url.pathname === "/api/v1/open-platform/apps"
            ) {
                await route.fulfill(
                    ok({
                        list: [
                            makeDeveloperApp({
                                id: 7,
                                displayName: "Campus Connector",
                                status: "approved",
                            }),
                        ],
                        total: 1,
                    }),
                );
                return;
            }

            if (
                request.method() === "GET" &&
                url.pathname === "/api/v1/open-platform/apps/7/audit-events"
            ) {
                auditLoadCount += 1;
                await route.fulfill(
                    auditLoadCount === 1
                        ? ok({
                              list: [
                                  {
                                      id: 91,
                                      eventType:
                                          "open_platform.app.profile_updated",
                                      requestID: "req-profile-update",
                                      scopes: ["future.scope"],
                                      endpoint: null,
                                      result: "success",
                                      details: { reason: "更新品牌资料" },
                                      createdAt: "2026-05-03T10:00:00Z",
                                  },
                              ],
                              total: 1,
                          })
                        : ok({
                              list: [
                                  {
                                      id: 91,
                                      appID: 7,
                                      eventType:
                                          "open_platform.app.profile_updated",
                                      requestID: "req-profile-update",
                                      scopes: [],
                                      endpoint: null,
                                      result: "success",
                                      details: { reason: "更新品牌资料" },
                                      createdAt: "2026-05-03T10:00:00Z",
                                  },
                              ],
                              total: 1,
                          }),
                );
                return;
            }

            await route.fulfill({
                status: 500,
                contentType: "application/json",
                body: JSON.stringify({
                    success: false,
                    error: {
                        code: "E2E_UNMOCKED",
                        message: `unmocked ${request.method()} ${url.pathname}`,
                    },
                }),
            });
        });

        await page.goto("/developers/apps");

        const appArticle = page
            .locator("article")
            .filter({ hasText: "Campus Connector" })
            .first();
        await expect(appArticle).toBeVisible({ timeout: 10_000 });

        await appArticle.getByRole("button", { name: /活动记录|Activity/ }).click();
        await expect(
            page.getByText(/应用活动加载失败，请重试|Failed to load application activity. Please retry/),
        ).toBeVisible();
        await expect(page.getByText(/暂无应用活动记录|No application activity yet/)).toHaveCount(0);

        const auditPanel = appArticle.locator("section").filter({
            hasText: /应用活动|Application Activity/,
        });
        await auditPanel.getByRole("button", { name: /刷新|Refresh/ }).click();

        await expect.poll(() => auditLoadCount).toBe(2);
        await expect(page.getByText(/应用资料已更新|Application profile updated/)).toBeVisible();
        await expect(page.getByText("原因：更新品牌资料")).toBeVisible();
    });

    test("invalid rotated secret response keeps the rotation dialog open", async ({
        page,
    }) => {
        let rotateCalled = false;

        await page.route("**/api/v1/open-platform/apps**", async (route) => {
            const request = route.request();
            const url = new URL(request.url());

            if (
                request.method() === "GET" &&
                url.pathname === "/api/v1/open-platform/apps"
            ) {
                await route.fulfill(
                    ok({
                        list: [
                            makeDeveloperApp({
                                id: 7,
                                displayName: "Campus Connector",
                                status: "approved",
                            }),
                        ],
                        total: 1,
                    }),
                );
                return;
            }

            if (
                request.method() === "POST" &&
                url.pathname === "/api/v1/open-platform/apps/7/secret/rotate"
            ) {
                rotateCalled = true;
                await route.fulfill(
                    ok({
                        app: {
                            displayName: "Campus Connector",
                            clientID: "op_app_07",
                        },
                        clientSecret: "ids_rotated_secret",
                    }),
                );
                return;
            }

            await route.fulfill({
                status: 500,
                contentType: "application/json",
                body: JSON.stringify({
                    success: false,
                    error: {
                        code: "E2E_UNMOCKED",
                        message: `unmocked ${request.method()} ${url.pathname}`,
                    },
                }),
            });
        });

        await page.goto("/developers/apps");

        const appArticle = page
            .locator("article")
            .filter({ hasText: "Campus Connector" })
            .first();
        await expect(appArticle).toBeVisible({ timeout: 10_000 });

        await appArticle.getByRole("button", { name: /轮换密钥|Rotate Secret/ }).click();
        const dialog = page.getByRole("dialog", {
            name: /轮换 client secret|Rotate client secret/,
        });
        await expect(dialog).toBeVisible();
        await dialog.getByLabel(/操作原因|Reason/).fill("例行轮换");
        await dialog.getByRole("button", { name: /确认轮换|Confirm rotation/ }).click();

        await expect.poll(() => rotateCalled).toBe(true);
        await expect(dialog).toBeVisible();
        await expect(
            dialog.getByText(/轮换 client secret 失败，请重试|Failed to rotate client secret. Please retry/),
        ).toBeVisible();
        await expect(page.getByText(/的新 client secret|New client secret for/)).toHaveCount(0);
    });

    test("developer lists and submits apps", async ({ page }) => {
        let appSubmitted = false;
        let secretRotated = false;
        let redirectChangeSubmitted = false;
        let submittedBody: unknown = null;
        let redirectChangeBody: unknown = null;
        const listQueries: QueryRecord[] = [];

        const appList: MockAppListItem[] = [
            {
                app: {
                    id: 7,
                    clientID: "op_existing",
                    displayName: "Campus Connector",
                    description: "Connect campus services",
                    homepageURL: "https://connector.example.com",
                    privacyPolicyURL: "https://connector.example.com/privacy",
                    redirectURIs: ["https://connector.example.com/callback"],
                    status: "approved",
                    createdAt: "2026-04-01T10:00:00Z",
                    updatedAt: "2026-04-02T10:00:00Z",
                },
                scopes: [
                    {
                        id: 1,
                        scope: "profile.basic.read",
                        displayName: "用户基本信息",
                        sensitivity: "low",
                        fields: ["用户名", "用户昵称", "头像地址"],
                        reason: "显示登录用户",
                        status: "approved",
                        reviewerUserID: null,
                        reviewedAt: null,
                        decisionNote: null,
                        createdAt: "2026-04-01T10:00:00Z",
                        updatedAt: "2026-04-02T10:00:00Z",
                    },
                ],
                redirectURIRequests: [],
            },
        ];

        await page.route(
            "**/api/v1/open-platform/apps/*/secret/rotate",
            async (route) => {
                secretRotated = true;
                expect(route.request().method()).toBe("POST");
                expect(route.request().postDataJSON()).toEqual({
                    reason: "例行轮换",
                });
                await route.fulfill({
                    contentType: "application/json",
                    body: JSON.stringify({
                        success: true,
                        data: {
                            app: appList[0].app,
                            clientSecret: "ids_rotated_secret",
                        },
                    }),
                });
            },
        );

        await page.route(
            "**/api/v1/open-platform/apps/*/redirect-uris",
            async (route) => {
                redirectChangeSubmitted = true;
                expect(route.request().method()).toBe("POST");
                redirectChangeBody = route.request().postDataJSON();
                appList[0].redirectURIRequests = [
                    {
                        id: 9,
                        redirectURIs: [
                            "https://connector.example.com/oauth/callback",
                        ],
                        reason: "迁移 OAuth 回调",
                        status: "pending",
                        reviewerUserID: null,
                        reviewedAt: null,
                        decisionNote: null,
                        createdAt: "2026-05-02T10:00:00Z",
                        updatedAt: "2026-05-02T10:00:00Z",
                    },
                ];
                await route.fulfill({
                    status: 201,
                    contentType: "application/json",
                    body: JSON.stringify({
                        success: true,
                        data: appList[0].redirectURIRequests[0],
                    }),
                });
            },
        );

        await page.route("**/api/v1/open-platform/apps*", async (route) => {
            const request = route.request();

            if (request.method() === "POST") {
                appSubmitted = true;
                submittedBody = request.postDataJSON();
                appList.unshift({
                    app: {
                        id: 8,
                        clientID: "op_new",
                        displayName: "Library Sync",
                        description: "Sync library notices",
                        homepageURL: "https://library.example.com",
                        privacyPolicyURL: "https://library.example.com/privacy",
                        redirectURIs: ["https://library.example.com/callback"],
                        status: "pending",
                        createdAt: "2026-05-01T10:00:00Z",
                        updatedAt: "2026-05-01T10:00:00Z",
                    },
                    scopes: [
                        {
                            id: 2,
                            scope: "profile.basic.read",
                            displayName: "用户基本信息",
                            sensitivity: "low",
                            fields: ["用户名", "用户昵称", "头像地址"],
                            reason: "用于显示登录用户",
                            status: "pending",
                            reviewerUserID: null,
                            reviewedAt: null,
                            decisionNote: null,
                            createdAt: "2026-05-01T10:00:00Z",
                            updatedAt: "2026-05-01T10:00:00Z",
                        },
                        {
                            id: 3,
                            scope: "email.read",
                            displayName: "邮箱",
                            sensitivity: "medium",
                            fields: ["邮箱地址"],
                            reason: "用于发送借阅通知",
                            status: "pending",
                            reviewerUserID: null,
                            reviewedAt: null,
                            decisionNote: null,
                            createdAt: "2026-05-01T10:00:00Z",
                            updatedAt: "2026-05-01T10:00:00Z",
                        },
                    ],
                    redirectURIRequests: [],
                });
                await route.fulfill({
                    status: 201,
                    contentType: "application/json",
                    body: JSON.stringify({
                        success: true,
                        data: { app: appList[0].app },
                    }),
                });
                return;
            }

            listQueries.push(captureQuery(request.url()));
            await route.fulfill({
                contentType: "application/json",
                body: JSON.stringify({
                    success: true,
                    data: {
                        list: appList,
                        total: appList.length,
                    },
                }),
            });
        });

        await page.goto("/developers/apps");
        await page.waitForLoadState("networkidle");
        await expect
            .poll(() => listQueries)
            .toContainEqual({ page: "1", pageSize: "10", status: "all" });

        const existingApp = page
            .locator("article")
            .filter({ hasText: "Campus Connector" })
            .first();
        await expect(existingApp).toBeVisible({ timeout: 10_000 });
        await expect(existingApp).toContainText("op_existing");

        await existingApp.getByRole("button", { name: "轮换密钥" }).click();
        await expect(
            page.getByRole("dialog", { name: "轮换 client secret" }),
        ).toBeVisible();
        await page.getByLabel("操作原因").fill("例行轮换");
        await page.getByRole("button", { name: "确认轮换" }).click();
        await expect.poll(() => secretRotated).toBe(true);
        await expect(page.getByText("ids_rotated_secret")).toBeVisible();

        await existingApp.getByRole("button", { name: "变更回调地址" }).click();
        await page
            .getByLabel("新回调地址 1")
            .fill("https://connector.example.com/oauth/callback");
        await page.getByLabel("回调地址变更原因").fill("迁移 OAuth 回调");
        await page.getByRole("button", { name: "提交变更" }).click();

        await expect.poll(() => redirectChangeSubmitted).toBe(true);
        expect(redirectChangeBody).toEqual({
            redirectURIs: ["https://connector.example.com/oauth/callback"],
            reason: "迁移 OAuth 回调",
        });
        await expect(
            page.getByText("https://connector.example.com/oauth/callback"),
        ).toBeVisible();

        await page.getByLabel("应用名称").fill("Library Sync");
        await page.getByLabel("应用说明").fill("Sync library notices");
        await page.getByLabel("主页 URL").fill("https://library.example.com");
        await page
            .getByLabel("隐私政策 URL")
            .fill("https://library.example.com/privacy");
        await page
            .getByLabel("回调地址 1")
            .fill("https://library.example.com/callback");
        await page
            .getByLabel("profile.basic.read 用途说明")
            .fill("用于显示登录用户");
        await page.getByRole("checkbox", { name: /邮箱/ }).check();
        await page.getByLabel("email.read 用途说明").fill("用于发送借阅通知");
        await page.getByRole("button", { name: "提交审核" }).click();

        await expect.poll(() => appSubmitted).toBe(true);
        expect(submittedBody).toMatchObject({
            displayName: "Library Sync",
            homepageURL: "https://library.example.com",
            privacyPolicyURL: "https://library.example.com/privacy",
            redirectURIs: ["https://library.example.com/callback"],
            scopes: [
                { scope: "profile.basic.read", reason: "用于显示登录用户" },
                { scope: "email.read", reason: "用于发送借阅通知" },
            ],
        });
        await expect(page.getByText("Library Sync")).toBeVisible();
    });

    test("developer filters app status and paginates app list", async ({
        page,
    }) => {
        const listQueries: QueryRecord[] = [];
        const appList = Array.from({ length: 12 }, (_, index) =>
            makeDeveloperApp({
                id: index + 1,
                displayName: `Portal App ${String(index + 1).padStart(2, "0")}`,
                status: index % 2 === 0 ? "pending" : "approved",
            }),
        );

        await page.route("**/api/v1/open-platform/apps*", async (route) => {
            const request = route.request();
            const url = new URL(request.url());

            if (
                request.method() !== "GET" ||
                url.pathname !== "/api/v1/open-platform/apps"
            ) {
                await route.fulfill({
                    status: 500,
                    contentType: "application/json",
                    body: JSON.stringify({
                        success: false,
                        error: {
                            code: "E2E_UNMOCKED",
                            message: `unmocked ${request.method()} ${url.pathname}`,
                        },
                    }),
                });
                return;
            }

            const query = captureQuery(request.url());
            listQueries.push(query);

            const requestedPage = Number(query.page ?? "1");
            const pageSize = Number(query.pageSize ?? "10");
            const status = query.status ?? "all";
            const filteredApps =
                status === "all"
                    ? appList
                    : appList.filter((item) => item.app.status === status);
            const start = (requestedPage - 1) * pageSize;

            await route.fulfill({
                contentType: "application/json",
                body: JSON.stringify({
                    success: true,
                    data: {
                        list: filteredApps.slice(start, start + pageSize),
                        total: filteredApps.length,
                    },
                }),
            });
        });

        await page.goto("/developers/apps");
        await page.waitForLoadState("networkidle");

        await expect(
            page.getByRole("heading", { name: "Portal App 01" }),
        ).toBeVisible();
        await expect
            .poll(() => listQueries)
            .toContainEqual({ page: "1", pageSize: "10", status: "all" });

        await page.getByRole("button", { name: "下一页" }).click();
        await expect(
            page.getByRole("heading", { name: "Portal App 11" }),
        ).toBeVisible();
        await expect
            .poll(() => listQueries)
            .toContainEqual({ page: "2", pageSize: "10", status: "all" });

        await page.getByLabel("状态", { exact: true }).selectOption("pending");
        await expect(
            page.getByRole("heading", { name: "Portal App 01" }),
        ).toBeVisible();
        await expect(
            page.getByRole("heading", { name: "Portal App 02" }),
        ).toBeHidden();
        await expect(page.getByRole("button", { name: "下一页" })).toBeHidden();
        await expect
            .poll(() => listQueries)
            .toContainEqual({ page: "1", pageSize: "10", status: "pending" });
    });

    test("developer updates, withdraws, and audits existing apps", async ({
        page,
    }) => {
        const capturedRequests: Array<{
            body: unknown;
            method: string;
            path: string;
        }> = [];
        const auditQueries: QueryRecord[] = [];
        const listQueries: QueryRecord[] = [];

        const approvedApp: MockAppListItem = {
            app: {
                id: 7,
                clientID: "op_existing",
                displayName: "Campus Connector",
                description: "Connect campus services",
                homepageURL: "https://connector.example.com",
                privacyPolicyURL: "https://connector.example.com/privacy",
                redirectURIs: ["https://connector.example.com/callback"],
                status: "approved",
                createdAt: "2026-04-01T10:00:00Z",
                updatedAt: "2026-04-02T10:00:00Z",
            },
            scopes: [
                {
                    id: 1,
                    scope: "profile.basic.read",
                    displayName: "用户基本信息",
                    sensitivity: "low",
                    fields: ["用户名", "用户昵称", "头像地址"],
                    reason: "显示登录用户",
                    status: "approved",
                    reviewerUserID: null,
                    reviewedAt: null,
                    decisionNote: null,
                    createdAt: "2026-04-01T10:00:00Z",
                    updatedAt: "2026-04-02T10:00:00Z",
                },
                {
                    id: 2,
                    scope: "email.read",
                    displayName: "邮箱",
                    sensitivity: "medium",
                    fields: ["邮箱地址"],
                    reason: "旧邮件用途",
                    status: "rejected",
                    reviewerUserID: 99,
                    reviewedAt: "2026-04-03T10:00:00Z",
                    decisionNote: "说明不充分",
                    createdAt: "2026-04-01T10:00:00Z",
                    updatedAt: "2026-04-03T10:00:00Z",
                },
                {
                    id: 3,
                    scope: "phone.read",
                    displayName: "手机号",
                    sensitivity: "high",
                    fields: ["已绑定手机号"],
                    reason: "短信通知",
                    status: "pending",
                    reviewerUserID: null,
                    reviewedAt: null,
                    decisionNote: null,
                    createdAt: "2026-04-04T10:00:00Z",
                    updatedAt: "2026-04-04T10:00:00Z",
                },
            ],
            redirectURIRequests: [
                {
                    id: 9,
                    redirectURIs: [
                        "https://connector.example.com/oauth/callback",
                    ],
                    reason: "迁移 OAuth 回调",
                    status: "pending",
                    reviewerUserID: null,
                    reviewedAt: null,
                    decisionNote: null,
                    createdAt: "2026-05-02T10:00:00Z",
                    updatedAt: "2026-05-02T10:00:00Z",
                },
            ],
        };

        const pendingApp: MockAppListItem = {
            app: {
                id: 8,
                clientID: "op_pending",
                displayName: "Pending Tool",
                description: "A pending integration",
                homepageURL: "https://pending.example.com",
                privacyPolicyURL: "https://pending.example.com/privacy",
                redirectURIs: ["https://pending.example.com/callback"],
                status: "pending",
                createdAt: "2026-05-01T10:00:00Z",
                updatedAt: "2026-05-01T10:00:00Z",
            },
            scopes: [
                {
                    id: 4,
                    scope: "profile.basic.read",
                    displayName: "用户基本信息",
                    sensitivity: "low",
                    fields: ["用户名"],
                    reason: "显示用户",
                    status: "pending",
                    reviewerUserID: null,
                    reviewedAt: null,
                    decisionNote: null,
                    createdAt: "2026-05-01T10:00:00Z",
                    updatedAt: "2026-05-01T10:00:00Z",
                },
            ],
            redirectURIRequests: [],
        };

        const appList = [approvedApp, pendingApp];

        await page.route("**/api/v1/open-platform/apps**", async (route) => {
            const request = route.request();
            const url = new URL(request.url());
            const path = url.pathname;
            const method = request.method();
            const body = request.postData() ? request.postDataJSON() : null;

            if (method === "GET" && path === "/api/v1/open-platform/apps") {
                listQueries.push(captureQuery(request.url()));
                await route.fulfill({
                    contentType: "application/json",
                    body: JSON.stringify({
                        success: true,
                        data: {
                            list: appList,
                            total: appList.length,
                        },
                    }),
                });
                return;
            }

            if (
                method === "GET" &&
                path === "/api/v1/open-platform/apps/7/audit-events"
            ) {
                auditQueries.push(captureQuery(request.url()));
                await route.fulfill({
                    contentType: "application/json",
                    body: JSON.stringify({
                        success: true,
                        data: {
                            list: [
                                {
                                    id: 91,
                                    appID: 7,
                                    eventType:
                                        "open_platform.app.profile_updated",
                                    requestID: "req-profile-update",
                                    scopes: [],
                                    endpoint: null,
                                    result: "success",
                                    details: { reason: "更新品牌资料" },
                                    createdAt: "2026-05-03T10:00:00Z",
                                },
                            ],
                            total: 1,
                        },
                    }),
                });
                return;
            }

            capturedRequests.push({ path, method, body });

            if (path === "/api/v1/open-platform/apps/7") {
                approvedApp.app = {
                    ...approvedApp.app,
                    ...body,
                    updatedAt: "2026-05-03T10:00:00Z",
                } as Record<string, unknown>;
                await route.fulfill({
                    contentType: "application/json",
                    body: JSON.stringify({
                        success: true,
                        data: { app: approvedApp.app },
                    }),
                });
                return;
            }

            if (path === "/api/v1/open-platform/apps/7/scopes") {
                approvedApp.scopes.push({
                    id: 5,
                    scope: "email.read",
                    displayName: "邮箱",
                    sensitivity: "medium",
                    fields: ["邮箱地址"],
                    reason: "发送通知邮件",
                    status: "pending",
                    reviewerUserID: null,
                    reviewedAt: null,
                    decisionNote: null,
                    createdAt: "2026-05-03T10:00:00Z",
                    updatedAt: "2026-05-03T10:00:00Z",
                });
                await route.fulfill({
                    status: 201,
                    contentType: "application/json",
                    body: JSON.stringify({
                        success: true,
                        data: { scopes: approvedApp.scopes },
                    }),
                });
                return;
            }

            if (
                path ===
                "/api/v1/open-platform/apps/7/scopes/phone.read/withdraw"
            ) {
                approvedApp.scopes = approvedApp.scopes.map((scope) =>
                    scope.scope === "phone.read"
                        ? {
                              ...scope,
                              status: "withdrawn",
                              updatedAt: "2026-05-03T10:00:00Z",
                          }
                        : scope,
                );
                await route.fulfill({
                    contentType: "application/json",
                    body: JSON.stringify({
                        success: true,
                        data: approvedApp.scopes[2],
                    }),
                });
                return;
            }

            if (
                path ===
                "/api/v1/open-platform/apps/7/redirect-uri-requests/9/withdraw"
            ) {
                approvedApp.redirectURIRequests =
                    approvedApp.redirectURIRequests.map((requestItem) =>
                        requestItem.id === 9
                            ? {
                                  ...requestItem,
                                  status: "withdrawn",
                                  updatedAt: "2026-05-03T10:00:00Z",
                              }
                            : requestItem,
                    );
                await route.fulfill({
                    contentType: "application/json",
                    body: JSON.stringify({
                        success: true,
                        data: approvedApp.redirectURIRequests[0],
                    }),
                });
                return;
            }

            if (path === "/api/v1/open-platform/apps/8/withdraw") {
                pendingApp.app = {
                    ...pendingApp.app,
                    status: "revoked",
                    updatedAt: "2026-05-03T10:00:00Z",
                };
                await route.fulfill({
                    contentType: "application/json",
                    body: JSON.stringify({
                        success: true,
                        data: { app: pendingApp.app },
                    }),
                });
                return;
            }

            await route.fulfill({
                status: 500,
                contentType: "application/json",
                body: JSON.stringify({
                    success: false,
                    error: {
                        code: "E2E_UNMOCKED",
                        message: `unmocked ${method} ${path}`,
                    },
                }),
            });
        });

        await page.goto("/developers/apps");
        await page.waitForLoadState("networkidle");
        await expect
            .poll(() => listQueries)
            .toContainEqual({ page: "1", pageSize: "10", status: "all" });

        const approvedArticle = page
            .locator("article")
            .filter({ hasText: "Campus Connector" })
            .first();
        await expect(approvedArticle).toBeVisible();

        await approvedArticle.getByRole("button", { name: "活动记录" }).click();
        await expect(page.getByText("应用资料已更新")).toBeVisible();
        await expect(page.getByText("原因：更新品牌资料")).toBeVisible();
        await expect
            .poll(() => auditQueries)
            .toContainEqual({ pageSize: "10" });

        await approvedArticle.getByRole("button", { name: "编辑资料" }).click();
        await approvedArticle
            .getByLabel("应用名称")
            .fill("Campus Connector Pro");
        await approvedArticle
            .getByLabel("应用说明")
            .fill("Connect campus services with profile updates");
        await approvedArticle
            .getByLabel("主页 URL")
            .fill("https://connector.example.com/pro");
        await approvedArticle
            .getByLabel("隐私政策 URL")
            .fill("https://connector.example.com/privacy-v2");
        await approvedArticle
            .getByLabel("应用资料变更原因")
            .fill("更新品牌资料");
        await page.getByRole("button", { name: "保存资料" }).click();

        const updatedArticle = page
            .locator("article")
            .filter({ hasText: "Campus Connector Pro" })
            .first();
        await expect(updatedArticle).toBeVisible();

        await updatedArticle.getByRole("button", { name: "申请权限" }).click();
        await updatedArticle.getByRole("checkbox", { name: /邮箱/ }).check();
        await updatedArticle
            .getByLabel("email.read 新用途说明")
            .fill("发送通知邮件");
        await updatedArticle
            .getByRole("button", { name: "提交权限申请" })
            .click();

        await page
            .locator("li")
            .filter({ hasText: "phone.read" })
            .getByRole("button", { name: "撤回" })
            .click();
        await page.getByLabel("操作原因").fill("不再需要手机号权限");
        await page.getByRole("button", { name: "确认撤回申请" }).click();

        await updatedArticle
            .getByRole("button", { name: "撤回" })
            .first()
            .click();
        await page.getByLabel("操作原因").fill("回调迁移延期");
        await page.getByRole("button", { name: "确认撤回申请" }).click();

        const pendingArticle = page
            .locator("article")
            .filter({ hasText: "Pending Tool" })
            .first();
        await pendingArticle.getByRole("button", { name: "撤回应用" }).click();
        await page.getByLabel("操作原因").fill("暂缓上线");
        await page.getByRole("button", { name: "确认撤回应用" }).click();

        await expect
            .poll(() => capturedRequests)
            .toContainEqual({
                path: "/api/v1/open-platform/apps/7",
                method: "PATCH",
                body: {
                    displayName: "Campus Connector Pro",
                    description: "Connect campus services with profile updates",
                    homepageURL: "https://connector.example.com/pro",
                    privacyPolicyURL:
                        "https://connector.example.com/privacy-v2",
                    reason: "更新品牌资料",
                },
            });
        await expect
            .poll(() => capturedRequests)
            .toContainEqual({
                path: "/api/v1/open-platform/apps/7/scopes",
                method: "POST",
                body: {
                    scopes: [{ scope: "email.read", reason: "发送通知邮件" }],
                },
            });
        await expect
            .poll(() => capturedRequests)
            .toContainEqual({
                path: "/api/v1/open-platform/apps/7/scopes/phone.read/withdraw",
                method: "POST",
                body: { reason: "不再需要手机号权限" },
            });
        await expect
            .poll(() => capturedRequests)
            .toContainEqual({
                path: "/api/v1/open-platform/apps/7/redirect-uri-requests/9/withdraw",
                method: "POST",
                body: { reason: "回调迁移延期" },
            });
        await expect
            .poll(() => capturedRequests)
            .toContainEqual({
                path: "/api/v1/open-platform/apps/8/withdraw",
                method: "POST",
                body: { reason: "暂缓上线" },
            });
    });
});
