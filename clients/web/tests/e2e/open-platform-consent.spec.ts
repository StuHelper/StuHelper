import { expect, test, type Page } from './fixtures';

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

const app = {
    id: 42,
    clientID: "campus-connector",
    displayName: "Campus Connector",
    description: "Connect campus learning services",
    homepageURL: "https://client.example.com",
    privacyPolicyURL: "https://client.example.com/privacy",
};

const scopes = [
    {
        scope: "profile.basic.read",
        displayName: "基础资料",
        sensitivity: "low",
        fields: ["用户名", "头像"],
        reason: "显示登录用户",
    },
    {
        scope: "stu.student.status.read",
        displayName: "学生状态",
        sensitivity: "high",
        fields: ["学校", "认证状态"],
        reason: "确认校内服务访问权限",
    },
];

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
}

test.describe("Open Platform consent flow", () => {
    test.beforeEach(async ({ page }) => {
        await mockAuth(page);
    });

    test("user reviews scopes and accepts an authorization request", async ({
        page,
    }) => {
        let consentToken = "";
        let acceptedBody: unknown = null;

        await page.route("**/api/v1/open-platform/consent?*", async (route) => {
            const url = new URL(route.request().url());
            consentToken = url.searchParams.get("token") ?? "";
            await route.fulfill(
                ok({
                    token: "consent-token",
                    app,
                    scopes,
                    redirectURI: "https://client.example.com/callback",
                    expiresAt: "2026-06-01T10:00:00Z",
                }),
            );
        });
        await page.route(
            "**/api/v1/open-platform/consent/accept",
            async (route) => {
                acceptedBody = route.request().postDataJSON();
                await route.fulfill(
                    ok({
                        redirectURL:
                            "https://client.example.com/callback?code=auth-code&state=xyz",
                    }),
                );
            },
        );
        await page.route("https://client.example.com/**", (route) =>
            route.fulfill({
                contentType: "text/html",
                body: "<!doctype html><title>Client callback</title><main>Client callback</main>",
            }),
        );

        await page.goto("/consent?token=consent-token");

        await expect(
            page.getByRole("heading", { name: /Campus Connector/ }),
        ).toBeVisible();
        await expect(page.getByText("Bob")).toBeVisible();
        await expect(page.getByText("client.example.com")).toBeVisible();
        await expect(page.getByText("profile.basic.read")).toBeVisible();
        await expect(page.getByText("stu.student.status.read")).toBeVisible();
        await expect(page.getByText("显示登录用户")).toBeVisible();
        expect(consentToken).toBe("consent-token");

        await Promise.all([
            page.waitForURL(
                "https://client.example.com/callback?code=auth-code&state=xyz",
            ),
            page.getByRole("button", { name: /允许|Allow/ }).click(),
        ]);

        expect(acceptedBody).toEqual({ token: "consent-token" });
    });

    test("user rejects an authorization request and returns to the client", async ({
        page,
    }) => {
        let deniedBody: unknown = null;

        await page.route("**/api/v1/open-platform/consent?*", async (route) => {
            const url = new URL(route.request().url());
            expect(url.searchParams.get("token")).toBe("deny-consent-token");
            await route.fulfill(
                ok({
                    token: "deny-consent-token",
                    app,
                    scopes,
                    redirectURI: "https://client.example.com/callback",
                    expiresAt: "2026-06-01T10:00:00Z",
                }),
            );
        });
        await page.route(
            "**/api/v1/open-platform/consent/deny",
            async (route) => {
                deniedBody = route.request().postDataJSON();
                await route.fulfill(
                    ok({
                        redirectURL:
                            "https://client.example.com/callback?error=access_denied&state=xyz",
                    }),
                );
            },
        );
        await page.route("https://client.example.com/**", (route) =>
            route.fulfill({
                contentType: "text/html",
                body: "<!doctype html><title>Client denied callback</title><main>Client denied callback</main>",
            }),
        );

        await page.goto("/consent?token=deny-consent-token");

        await expect(
            page.getByRole("heading", { name: /Campus Connector/ }),
        ).toBeVisible();
        await expect(page.getByText("stu.student.status.read")).toBeVisible();

        await Promise.all([
            page.waitForURL(
                "https://client.example.com/callback?error=access_denied&state=xyz",
            ),
            page.getByRole("button", { name: /拒绝|Deny/ }).click(),
        ]);

        expect(deniedBody).toEqual({ token: "deny-consent-token" });
        await expect(page.getByText("Client denied callback")).toBeVisible();
    });

    test("invalid authorization request response fails closed and can retry", async ({
        page,
    }) => {
        let loadCount = 0;

        await page.route("**/api/v1/open-platform/consent?*", async (route) => {
            loadCount += 1;
            await route.fulfill(
                loadCount === 1
                    ? ok({
                          token: "retry-consent-token",
                          app: {
                              id: 42,
                              clientID: "campus-connector",
                              displayName: "Campus Connector",
                          },
                          scopes,
                          redirectURI: "https://client.example.com/callback",
                          expiresAt: "2026-06-01T10:00:00Z",
                      })
                    : ok({
                          token: "retry-consent-token",
                          app,
                          scopes,
                          redirectURI: "https://client.example.com/callback",
                          expiresAt: "2026-06-01T10:00:00Z",
                      }),
            );
        });

        await page.goto("/consent?token=retry-consent-token");

        await expect(
            page.getByRole("heading", {
                name: /授权请求加载失败|Failed to load authorization request/,
            }),
        ).toBeVisible();
        await expect(
            page
                .locator("p")
                .filter({
                    hasText:
                        /授权请求加载失败|Failed to load authorization request/,
                }),
        ).toBeVisible();

        await page.getByRole("button", { name: /重试|Retry/ }).click();
        await expect.poll(() => loadCount).toBe(2);
        await expect(
            page.getByRole("heading", { name: /Campus Connector/ }),
        ).toBeVisible();
        await expect(page.getByText("profile.basic.read")).toBeVisible();
    });

    test("unsafe authorization redirect is rejected without leaving identity page", async ({
        page,
    }) => {
        await page.route("**/api/v1/open-platform/consent?*", async (route) => {
            await route.fulfill(
                ok({
                    token: "unsafe-redirect-token",
                    app,
                    scopes,
                    redirectURI: "https://client.example.com/callback",
                    expiresAt: "2026-06-01T10:00:00Z",
                }),
            );
        });
        await page.route(
            "**/api/v1/open-platform/consent/accept",
            async (route) => {
                await route.fulfill(
                    ok({ redirectURL: "javascript:alert(1)" }),
                );
            },
        );

        await page.goto("/consent?token=unsafe-redirect-token");
        await expect(
            page.getByRole("heading", { name: /Campus Connector/ }),
        ).toBeVisible();

        await page.getByRole("button", { name: /允许|Allow/ }).click();

        await expect(page).toHaveURL(/\/consent\?token=unsafe-redirect-token/);
        await expect(
            page.getByRole("heading", {
                name: /授权操作失败|Authorization failed/,
            }),
        ).toBeVisible();
        await expect(
            page.getByText(/授权操作失败，请重试|Authorization failed. Please retry/),
        ).toBeVisible();
    });

    test("user completes missing profile fields and continues to consent", async ({
        page,
    }) => {
        let profileToken = "";
        let continueBody: unknown = null;
        let nextConsentToken = "";

        await page.route(
            "**/api/v1/open-platform/profile-completion?*",
            async (route) => {
                const url = new URL(route.request().url());
                profileToken = url.searchParams.get("token") ?? "";
                await route.fulfill(
                    ok({
                        token: "profile-token",
                        app,
                        scopes,
                        missingFields: [
                            {
                                key: "profile.phone",
                                displayName: "手机号",
                                actionURL: "/user/phone-binding",
                            },
                            {
                                key: "profile.student",
                                displayName: "学生认证",
                                actionURL: "/user/student-verification",
                            },
                        ],
                        redirectURI: "https://client.example.com/callback",
                        expiresAt: "2026-06-01T10:00:00Z",
                    }),
                );
            },
        );
        await page.route(
            "**/api/v1/open-platform/profile-completion/continue",
            async (route) => {
                continueBody = route.request().postDataJSON();
                await route.fulfill(
                    ok({ consentURL: "/consent?token=next-consent" }),
                );
            },
        );
        await page.route("**/api/v1/open-platform/consent?*", async (route) => {
            const url = new URL(route.request().url());
            nextConsentToken = url.searchParams.get("token") ?? "";
            await route.fulfill(
                ok({
                    token: "next-consent",
                    app,
                    scopes,
                    redirectURI: "https://client.example.com/callback",
                    expiresAt: "2026-06-01T10:00:00Z",
                }),
            );
        });

        await page.goto("/complete-profile?token=profile-token");

        await expect(
            page.getByRole("heading", { name: /Campus Connector/ }),
        ).toBeVisible();
        await expect(page.getByText("手机号")).toBeVisible();
        await expect(page.getByText("学生认证")).toBeVisible();
        await expect(page.getByText("profile.phone")).toBeVisible();
        await expect(page.getByText("stu.student.status.read")).toBeVisible();
        expect(profileToken).toBe("profile-token");

        await Promise.all([
            page.waitForURL(/\/consent\?token=next-consent/),
            page
                .getByRole("button", {
                    name: /我已补全，继续|I have completed this/,
                })
                .click(),
        ]);

        expect(continueBody).toEqual({ token: "profile-token" });
        await expect.poll(() => nextConsentToken).toBe("next-consent");
        await expect(page.getByText("将通过 Connect 披露以下信息")).toBeVisible();
    });

    test("profile completion without a token shows a fail-closed error state", async ({
        page,
    }) => {
        let profileCompletionRequested = false;
        await page.route(
            "**/api/v1/open-platform/profile-completion?*",
            async (route) => {
                profileCompletionRequested = true;
                await route.fulfill(ok({}));
            },
        );

        await page.goto("/complete-profile");

        await expect(
            page.getByRole("heading", {
                name: /资料补全请求加载失败|Failed to load profile completion request/,
            }),
        ).toBeVisible();
        await expect(
            page.getByText(
                /资料补全请求已失效或缺少 token|profile completion request is expired or missing a token/,
            ),
        ).toBeVisible();
        expect(profileCompletionRequested).toBe(false);
    });

    test("invalid profile completion response fails closed and can retry", async ({
        page,
    }) => {
        let loadCount = 0;

        await page.route(
            "**/api/v1/open-platform/profile-completion?*",
            async (route) => {
                loadCount += 1;
                await route.fulfill(
                    loadCount === 1
                        ? ok({
                              token: "retry-profile-token",
                              app,
                              scopes,
                              missingFields: [
                                  {
                                      key: "profile.phone",
                                      displayName: "手机号",
                                  },
                              ],
                              redirectURI: "https://client.example.com/callback",
                              expiresAt: "2026-06-01T10:00:00Z",
                          })
                        : ok({
                              token: "retry-profile-token",
                              app,
                              scopes,
                              missingFields: [
                                  {
                                      key: "profile.phone",
                                      displayName: "手机号",
                                      actionURL: "/user/phone-binding",
                                  },
                              ],
                              redirectURI: "https://client.example.com/callback",
                              expiresAt: "2026-06-01T10:00:00Z",
                          }),
                );
            },
        );

        await page.goto("/complete-profile?token=retry-profile-token");

        await expect(
            page.getByRole("heading", {
                name: /资料补全请求加载失败|Failed to load profile completion request/,
            }),
        ).toBeVisible();
        await expect(
            page
                .locator("p")
                .filter({
                    hasText:
                        /资料补全请求加载失败|Failed to load profile completion request/,
                }),
        ).toBeVisible();

        await page.getByRole("button", { name: /重试|Retry/ }).click();
        await expect.poll(() => loadCount).toBe(2);
        await expect(
            page.getByRole("heading", { name: /Campus Connector/ }),
        ).toBeVisible();
        await expect(page.getByText("profile.phone")).toBeVisible();
    });

    test("unsafe profile completion redirect is rejected without leaving identity page", async ({
        page,
    }) => {
        await page.route(
            "**/api/v1/open-platform/profile-completion?*",
            async (route) => {
                await route.fulfill(
                    ok({
                        token: "unsafe-profile-token",
                        app,
                        scopes,
                        missingFields: [],
                        redirectURI: "https://client.example.com/callback",
                        expiresAt: "2026-06-01T10:00:00Z",
                    }),
                );
            },
        );
        await page.route(
            "**/api/v1/open-platform/profile-completion/continue",
            async (route) => {
                await route.fulfill(
                    ok({ redirectURL: "javascript:alert(1)" }),
                );
            },
        );

        await page.goto("/complete-profile?token=unsafe-profile-token");
        await expect(
            page.getByRole("heading", { name: /Campus Connector/ }),
        ).toBeVisible();
        await expect(
            page.getByText(
                /资料已满足本次授权请求|profile now satisfies this authorization request/,
            ),
        ).toBeVisible();

        await page
            .getByRole("button", {
                name: /我已补全，继续|I have completed this/,
            })
            .click();

        await expect(page).toHaveURL(
            /\/complete-profile\?token=unsafe-profile-token/,
        );
        await expect(
            page.getByRole("heading", {
                name: /继续授权失败|Failed to continue authorization/,
            }),
        ).toBeVisible();
        await expect(
            page.getByText(
                /继续授权失败，请重试|Failed to continue authorization. Please retry/,
            ),
        ).toBeVisible();
    });

    test("profile completion refreshes missing fields and continues to the client redirect", async ({
        page,
    }) => {
        let loadCount = 0;
        let continueBody: unknown = null;

        await page.route(
            "**/api/v1/open-platform/profile-completion?*",
            async (route) => {
                loadCount += 1;
                await route.fulfill(
                    ok({
                        token: "refresh-profile-token",
                        app,
                        scopes,
                        missingFields:
                            loadCount === 1
                                ? [
                                      {
                                          key: "profile.phone",
                                          displayName: "手机号",
                                          actionURL: "/user/phone-binding",
                                      },
                                  ]
                                : [],
                        redirectURI: "https://client.example.com/callback",
                        expiresAt: "2026-06-01T10:00:00Z",
                    }),
                );
            },
        );
        await page.route(
            "**/api/v1/open-platform/profile-completion/continue",
            async (route) => {
                continueBody = route.request().postDataJSON();
                await route.fulfill(
                    ok({
                        redirectURL:
                            "https://client.example.com/callback?code=profile-complete&state=xyz",
                    }),
                );
            },
        );
        await page.route("https://client.example.com/**", (route) =>
            route.fulfill({
                contentType: "text/html",
                body: "<!doctype html><title>Profile complete callback</title><main>Profile complete callback</main>",
            }),
        );

        await page.goto("/complete-profile?token=refresh-profile-token");

        await expect(page.getByText("profile.phone")).toBeVisible();
        await page
            .getByRole("button", { name: /重新检查|Check again/ })
            .click();
        await expect
            .poll(() => loadCount, {
                message: "profile completion refresh should reload the request",
            })
            .toBe(2);
        await expect(
            page.getByText(
                /资料已满足本次授权请求|profile now satisfies this authorization request/,
            ),
        ).toBeVisible();

        await Promise.all([
            page.waitForURL(
                "https://client.example.com/callback?code=profile-complete&state=xyz",
            ),
            page
                .getByRole("button", {
                    name: /我已补全，继续|I have completed this/,
                })
                .click(),
        ]);

        expect(continueBody).toEqual({ token: "refresh-profile-token" });
        await expect(page.getByText("Profile complete callback")).toBeVisible();
    });
});
