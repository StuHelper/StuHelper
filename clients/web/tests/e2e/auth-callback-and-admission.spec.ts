import { expect, test, type Page } from "@playwright/test";

const user = {
    id: "u2",
    name: "bob",
    displayName: "Bob",
    email: "bob@example.com",
    roles: ["verified_student"],
    capabilities: ["review:list:full"],
    globalCapabilities: ["review:list:full"],
    capabilityGrants: [],
    canAccessAdmin: false,
};

const now = "2026-05-24T04:00:00Z";

const joinedSession = {
    id: "admission-session-1",
    platform: "qq",
    guildID: "guild-1",
    channelID: "channel-1",
    qqID: "123456",
    qqNickname: "航小伴",
    status: "joined_muted",
    tokenExpiresAt: "2026-05-24T05:00:00Z",
    linkWaitDeadlineAt: "2026-05-24T04:10:00Z",
    submissionWaitDeadlineAt: "2026-05-24T04:30:00Z",
    manualReviewDeadlineAt: null,
    initialMuteUntil: "2026-05-24T04:05:00Z",
    projectionPending: false,
    maxMaterialBytes: 5_242_880,
};

const linkedSession = {
    ...joinedSession,
    userID: "u2",
    status: "linked",
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

async function mockAuthenticated(page: Page) {
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

test.describe("Auth callback and admission entry", () => {
    test("auth callback consumes a stored OAuth state and redirects to backend callback", async ({
        page,
    }) => {
        let callbackURL: URL | null = null;

        await page.addInitScript(() => {
            sessionStorage.setItem("oauth_state", "oauth-state-1");
        });
        await page.route("**/api/v1/auth/callback?*", (route) => {
            callbackURL = new URL(route.request().url());
            return route.fulfill({
                contentType: "text/html",
                body: "<!doctype html><title>Backend callback</title><main>Backend callback</main>",
            });
        });

        await page.goto("/auth/callback?code=oauth-code-1&state=oauth-state-1");
        await page.waitForURL(/\/api\/v1\/auth\/callback/);

        expect(callbackURL).not.toBeNull();
        expect(callbackURL!.searchParams.get("code")).toBe("oauth-code-1");
        expect(callbackURL!.searchParams.get("state")).toBe("oauth-state-1");
        await expect(page.getByText("Backend callback")).toBeVisible();
    });

    test("logged-in user links an admission session and verifies school email OTP", async ({
        page,
    }) => {
        let linkQQ = "";
        let otpRequestBody: unknown = null;
        let otpVerifyBody: unknown = null;

        await mockAuthenticated(page);
        await page.route(
            "**/api/v1/admission/sessions/ADMIT-1**",
            async (route) => {
                const url = new URL(route.request().url());
                if (url.pathname.endsWith("/link")) {
                    linkQQ = url.searchParams.get("qq") ?? "";
                    await route.fulfill(ok(linkedSession));
                    return;
                }
                await route.fulfill(ok(joinedSession));
            },
        );
        await page.route("**/api/v1/admission/me", (route) =>
            route.fulfill(
                ok({
                    status: "linked",
                    projectionPending: false,
                    credentialKind: "school_email_otp",
                    session: linkedSession,
                }),
            ),
        );
        await page.route("**/api/v1/user/schools", (route) =>
            route.fulfill(
                ok([
                    {
                        schoolID: 1001,
                        schoolName: "测试大学",
                        verificationMethod: "ldap",
                        consentText: null,
                        manualFormFields: null,
                        enabled: true,
                        schoolSsoEnabled: false,
                    },
                ]),
            ),
        );
        await page.route(
            "**/api/v1/admission/school-email/request-otp",
            async (route) => {
                otpRequestBody = route.request().postDataJSON();
                await route.fulfill(ok());
            },
        );
        await page.route(
            "**/api/v1/admission/school-email/verify-otp",
            async (route) => {
                otpVerifyBody = route.request().postDataJSON();
                await route.fulfill(
                    ok({
                        status: "verified",
                        projectionPending: false,
                        credentialKind: "school_email_otp",
                        provisionalExpiresAt: "2026-10-01T00:00:00Z",
                        session: {
                            ...linkedSession,
                            status: "verified",
                            projectionPending: false,
                            manualReviewDeadlineAt: now,
                        },
                    }),
                );
            },
        );

        await page.goto("/admission/a/ADMIT-1?qq=123456");

        await expect(page.getByText("QQ：123456")).toBeVisible();
        await expect(
            page.getByRole("heading", { name: "确认绑定当前 QQ" }),
        ).toBeVisible();
        await page.getByRole("button", { name: "开始认证" }).click();

        await expect(
            page.getByRole("heading", { name: "选择认证方式" }),
        ).toBeVisible();
        await expect(
            page.locator("[data-admission-old-student-flow]"),
        ).toBeVisible();
        expect(linkQQ).toBe("123456");

        await page.locator("[data-school-select]").selectOption("1001");
        await page.getByLabel("学校邮箱").fill("student@test.edu");
        await page.getByRole("button", { name: "发送验证码" }).click();
        await page.getByLabel("验证码").fill("654321");
        await page.getByRole("button", { name: "验证邮箱" }).click();

        await expect(
            page.getByRole("heading", { name: "认证已通过" }),
        ).toBeVisible();
        expect(otpRequestBody).toEqual({
            schoolID: 1001,
            email: "student@test.edu",
        });
        expect(otpVerifyBody).toEqual({
            schoolID: 1001,
            email: "student@test.edu",
            code: "654321",
        });
    });
});
