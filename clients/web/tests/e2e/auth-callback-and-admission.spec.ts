import type { TestInfo } from "@playwright/test";
import { expect, test, type Page } from "./fixtures";

const user = {
    id: "u2",
    name: "bob",
    displayName: "Bob",
    email: "bob@example.com",
    roles: ["user"],
    capabilities: ["review:create"],
    globalCapabilities: ["review:create"],
    capabilityGrants: [],
    isPlatformAdmin: false,
    canAccessAdmin: false,
};

const now = "2026-08-05T08:00:00Z";

const awaitingAccountLinkSession = {
    id: "admission-session-1",
    platform: "qq",
    guildID: "guild-1",
    channelID: "channel-1",
    qqID: "123456",
    status: "awaiting_account_link",
    tokenExpiresAt: "2026-08-05T09:00:00Z",
    tokenConsumedAt: null,
    linkWaitDeadlineAt: "2026-08-05T08:10:00Z",
    submissionWaitDeadlineAt: "2026-08-05T08:30:00Z",
    manualReviewDeadlineAt: null,
    initialMuteUntil: "2026-08-05T08:05:00Z",
    verifiedAt: null,
    cancelledAt: null,
    lastBotError: null,
    eligibilityRevision: null,
    eligibilityEvaluatedAt: null,
    projectionPending: false,
    failureCount: 0,
    remainingRetryCount: 3,
    willBlacklistOnTimeout: true,
};

const awaitingRequirementsSession = {
    ...awaitingAccountLinkSession,
    userID: "2",
    status: "awaiting_requirements",
    tokenConsumedAt: now,
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

function apiError(code: string, message: string, status = 400) {
    return json({ success: false, error: { code, message } }, status);
}

async function mockUnauthenticated(page: Page) {
    await page.route("**/api/v1/auth/me", (route) =>
        route.fulfill(apiError("A0010100", "login required", 401)),
    );
    await page.route("**/api/v1/auth/refresh", (route) =>
        route.fulfill(apiError("A0010100", "login required", 401)),
    );
}

async function mockAuthenticated(page: Page) {
    await page.addInitScript((value) => {
        localStorage.setItem("stuhelper_user", JSON.stringify(value));
        localStorage.setItem("stuhelper_token_expiry", String(Date.now() + 60 * 60 * 1000));
    }, user);
    await page.route("**/api/v1/auth/me", (route) => route.fulfill(ok(user)));
    await page.route("**/api/v1/auth/refresh", (route) =>
        route.fulfill(ok({ expiresIn: 3600 })),
    );
}

async function mockReadiness(
    page: Page,
    state: { studentVerified: boolean; qqBinding: null | Record<string, unknown> },
) {
    await page.route("**/api/v1/user/me", (route) =>
        route.fulfill(ok({
            displayName: user.displayName,
            phone: null,
            studentVerificationStatus: state.studentVerified ? "approved" : "none",
            phoneBound: false,
            capabilities: user.capabilities,
        })),
    );
    await page.route("**/api/v1/account/phone", (route) =>
        route.fulfill(ok({
            state: "unbound",
            maskedPhone: null,
            method: null,
            verifiedAt: null,
            expiresAt: null,
            publishingRequirementSatisfied: false,
            revision: 1,
        })),
    );
    await page.route("**/api/v1/user/qq-binding", (route) =>
        route.fulfill(ok(state.qqBinding)),
    );
}

async function confirmAdmissionQQBinding(page: Page, qq = "123456") {
    await expect(page.getByRole("heading", { name: "确认绑定 QQ" })).toBeVisible();
    await page.locator("[data-admission-bind-confirmation-input]").fill(qq);
    await page.locator("[data-admission-bind-confirmation-submit]").click();
}

function joinAdmissionURL(testInfo: TestInfo, path: string): string {
    const url = new URL(path, String(testInfo.project.use.baseURL));
    url.hostname = "join.localhost";
    return url.toString();
}

test.describe("Auth callback and decoupled admission entry", () => {
    test("auth callback consumes an upstream OAuth state and redirects to the backend callback", async ({ page }) => {
        let callbackURL: URL | null = null;
        await page.addInitScript(() => sessionStorage.setItem("oauth_state", "oauth-state-1"));
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

    test("auth callback rejects oversized parameters before reaching the backend", async ({ page }) => {
        let backendRequests = 0;
        await page.route("**/api/v1/auth/callback?*", (route) => {
            backendRequests += 1;
            return route.fulfill({ contentType: "text/html", body: "unexpected" });
        });

        await page.goto(`/auth/callback?code=oauth-code-1&state=${"s".repeat(4097)}`);

        await expect(page).toHaveURL(/\/login\?error=invalid_callback/);
        expect(backendRequests).toBe(0);
    });

    test("anonymous admission links use the exact normalized join URL as the SSO return", async ({ page }, testInfo) => {
        let loginRequestURL: URL | null = null;
        await mockUnauthenticated(page);
        await page.route("**/api/v1/admission/sessions/ADMIT-LOGIN", (route) =>
            route.fulfill(ok(awaitingAccountLinkSession)),
        );
        await page.route("**/api/v1/auth/login**", (route) => {
            loginRequestURL = new URL(route.request().url());
            return route.fulfill(ok({
                url: "https://sso.stuhelper.com/login/oauth/authorize?client_id=stuhelper-web&state=admission-state",
                state: "admission-state",
            }));
        });
        await page.route("https://sso.stuhelper.com/**", (route) =>
            route.fulfill({
                contentType: "text/html",
                body: "<!doctype html><main>SSO authorize</main>",
            }),
        );

        const joinURL = joinAdmissionURL(testInfo, "/verify/ADMIT-LOGIN?from=qq");
        const normalizedURL = joinAdmissionURL(testInfo, "/verify/ADMIT-LOGIN");
        await page.goto(joinURL);
        await page.getByRole("button", { name: "登录" }).click();
        await page.waitForURL((url) => url.hostname === "sso.stuhelper.com");

        expect(loginRequestURL).not.toBeNull();
        expect(loginRequestURL!.searchParams.get("redirect")).toBe(normalizedURL);
    });

    test("main-site admission and start paths remain isolated from the join surface", async ({ page }) => {
        await mockUnauthenticated(page);

        await page.goto("/verify/ADMIT-MAIN");
        await expect(page.getByRole("heading", { name: /Page Not Found|页面不存在/i })).toBeVisible();
        await expect(page.locator("[data-admission-page-root]")).toHaveCount(0);

        await page.goto("/start");
        await expect(page.getByRole("heading", { name: /Page Not Found|页面不存在/i })).toBeVisible();
        await expect(page.locator("[data-join-start]")).toHaveCount(0);
    });

    test("join self-service readiness consumes current student, phone, and QQ projections only", async ({ page }, testInfo) => {
        const state: { studentVerified: boolean; qqBinding: null | Record<string, unknown> } = {
            studentVerified: false,
            qqBinding: null,
        };
        let admissionRequests = 0;

        await mockAuthenticated(page);
        await mockReadiness(page, state);
        await page.route("**/api/v1/admission/sessions/**", (route) => {
            admissionRequests += 1;
            return route.fulfill(apiError("unexpected", "unexpected", 500));
        });

        await page.goto(joinAdmissionURL(testInfo, "/start"));
        await expect(page.getByRole("heading", { name: "完成学生认证" })).toBeVisible();
        const studentHref = new URL(
            await page.locator("[data-open-student-verification]").getAttribute("href") ?? "",
        );
        expect(studentHref.hostname).toBe("localhost");
        expect(studentHref.pathname).toBe("/user/student-verification");
        expect(new URL(studentHref.searchParams.get("redirect") ?? "").hostname).toBe("join.localhost");

        state.studentVerified = true;
        await page.reload();
        await expect(page.getByRole("heading", { name: "绑定 QQ" })).toBeVisible();

        state.qqBinding = {
            userID: 2,
            qqID: "123456",
            boundAt: now,
            createdAt: now,
            updatedAt: now,
        };
        await page.reload();
        await expect(page.getByRole("heading", { name: "账号已准备好" })).toBeVisible();
        expect(admissionRequests).toBe(0);
    });

    test("linking a QQ session opens the independent student verification center instead of embedding a school flow", async ({ page }, testInfo) => {
        let linked = false;
        await mockAuthenticated(page);
        await page.route("**/api/v1/user/qq-binding", (route) => route.fulfill(ok(null)));
        await page.route("**/api/v1/admission/sessions/ADMIT-1", (route) =>
            route.fulfill(ok(awaitingAccountLinkSession)),
        );
        await page.route("**/api/v1/admission/sessions/ADMIT-1/link", (route) => {
            linked = true;
            return route.fulfill(ok(awaitingRequirementsSession));
        });
        await page.route("**/api/v1/admission/me**", (route) =>
            route.fulfill(ok({
                status: "awaiting_requirements",
                projectionPending: false,
                session: awaitingRequirementsSession,
            })),
        );

        await page.goto(joinAdmissionURL(testInfo, "/verify/ADMIT-1"));
        await page.getByRole("button", { name: "开始认证" }).click();
        await confirmAdmissionQQBinding(page);

        await expect(page.getByRole("heading", { name: "完成账号级学生认证" })).toBeVisible();
        const verificationLink = page.locator("[data-admission-open-student-verification]");
        const href = new URL(await verificationLink.getAttribute("href") ?? "");
        expect(href.hostname).toBe("localhost");
        expect(href.pathname).toBe("/user/student-verification");
        const returnURL = new URL(href.searchParams.get("redirect") ?? "");
        expect(returnURL.hostname).toBe("join.localhost");
        expect(returnURL.pathname).toBe("/verify/ADMIT-1");
        await expect(page.locator("[data-admission-old-student-flow]")).toHaveCount(0);
        await expect(page.locator("[data-admission-freshman-flow]")).toHaveCount(0);
        await expect(page.locator("[data-school-email-otp-form]")).toHaveCount(0);
        expect(linked).toBe(true);
    });

    test("a consumed link resumes only through its remembered account session", async ({ page }, testInfo) => {
        let linkRequests = 0;
        await mockAuthenticated(page);
        await page.route("**/api/v1/user/qq-binding", (route) => route.fulfill(ok(null)));
        await page.route("**/api/v1/admission/sessions/ADMIT-CONSUMED", (route) =>
            route.fulfill(apiError("admission.token_consumed", "consumed", 409)),
        );
        await page.route("**/api/v1/admission/sessions/ADMIT-CONSUMED/link", (route) => {
            linkRequests += 1;
            return route.fulfill(ok(awaitingRequirementsSession));
        });
        await page.route("**/api/v1/admission/me**", (route) =>
            route.fulfill(ok({
                status: "awaiting_requirements",
                projectionPending: false,
                session: awaitingRequirementsSession,
            })),
        );

        await page.goto(joinAdmissionURL(testInfo, "/verify/ADMIT-CONSUMED"));

        await expect(page.getByRole("heading", { name: "完成账号级学生认证" })).toBeVisible();
        expect(linkRequests).toBeGreaterThanOrEqual(1);
        await expect(page.getByRole("heading", { name: "链接已失效" })).toHaveCount(0);
    });

    test("QQ mismatch and expired links are terminal and expose no verification controls", async ({ page }, testInfo) => {
        await mockAuthenticated(page);
        await page.route("**/api/v1/user/qq-binding", (route) => route.fulfill(ok(null)));
        await page.route("**/api/v1/admission/sessions/ADMIT-MISMATCH", (route) =>
            route.fulfill(apiError("admission.qq_mismatch", "mismatch", 400)),
        );

        await page.goto(joinAdmissionURL(testInfo, "/verify/ADMIT-MISMATCH"));
        await expect(page.getByRole("heading", { name: "QQ 账号不匹配" })).toBeVisible();
        await expect(page.locator("[data-admission-open-student-verification]")).toHaveCount(0);

        await page.route("**/api/v1/admission/sessions/ADMIT-EXPIRED", (route) =>
            route.fulfill(apiError("admission.token_expired", "expired", 410)),
        );
        await page.goto(joinAdmissionURL(testInfo, "/verify/ADMIT-EXPIRED"));
        await expect(page.getByRole("heading", { name: "链接已失效" })).toBeVisible();
        await expect(page.locator("[data-admission-open-student-verification]")).toHaveCount(0);
    });
});
