import { expect, test, type Page } from "@playwright/test";

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
        expect(nextConsentToken).toBe("next-consent");
        await expect(page.getByText("将获取以下权限")).toBeVisible();
    });
});
