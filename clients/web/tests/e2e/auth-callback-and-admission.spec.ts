import type { TestInfo } from "@playwright/test";
import { allowExpectedConsoleError, expect, test, type Page } from "./fixtures";

const user = {
    id: "u2",
    name: "bob",
    displayName: "Bob",
    email: "bob@example.com",
    roles: ["verified_student"],
    capabilities: ["review:list:full"],
    globalCapabilities: ["review:list:full"],
    capabilityGrants: [],
    isPlatformAdmin: false,
    canAccessAdmin: false,
};

const now = "2026-05-24T04:00:00Z";

const joinedSession = {
    id: "admission-session-1",
    platform: "qq",
    guildID: "guild-1",
    channelID: "channel-1",
    qqID: "123456",
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

const freshmanAdmissionMe = {
    status: "linked",
    projectionPending: false,
    credentialKind: "freshman_material_manual",
    session: linkedSession,
};

const unverifiedStudentProfile = {
    userID: 2,
    schoolID: null,
    studentIDs: [],
    activeStudentID: null,
    verificationStatus: "unverified",
    verificationMethod: null,
    rejectionReason: null,
    reviewedAt: null,
    phone: null,
    phoneVerified: false,
    consentGivenAt: null,
    verifiedAt: null,
    createdAt: now,
    updatedAt: now,
};

const verifiedStudentProfile = {
    ...unverifiedStudentProfile,
    schoolID: 4111010006,
    studentIDs: ["20260001"],
    activeStudentID: "20260001",
    verificationStatus: "verified",
    verificationMethod: "school_email_otp",
    consentGivenAt: now,
    verifiedAt: now,
};

const joinStartSchool = {
    schoolID: 4111010006,
    schoolCode: "4111010006",
    schoolName: "北京航空航天大学",
    verificationMethod: "manual",
    approvalPolicy: "auto",
    consentText: "同意使用学校认证信息完成学生身份认证。",
    manualFormFields: null,
    enabled: true,
    schoolSsoEnabled: false,
    schoolEmailOtpEnabled: true,
    schoolEmailIdentityPolicy: {
        type: "academic_student_email",
        studentIDEmailDomain: "buaa.edu.cn",
        requireStudentName: true,
    },
};

interface JoinStartReadinessState {
    profile: typeof unverifiedStudentProfile | typeof verifiedStudentProfile;
    qqBinding: {
        userID: number;
        qqID: string;
        boundAt: string;
        createdAt: string;
        updatedAt: string;
    } | null;
}

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
    return json(
        {
            success: false,
            error: { code, message },
        },
        status,
    );
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

async function mockJoinStartReadiness(
    page: Page,
    state: JoinStartReadinessState,
) {
    await page.route("**/api/v1/user/identity", (route) =>
        route.fulfill(apiError("A0040404", "identity not found", 404)),
    );
    await page.route("**/api/v1/user/profile", (route) =>
        route.fulfill(ok(state.profile)),
    );
    await page.route("**/api/v1/user/qq-binding", (route) =>
        route.fulfill(
            state.qqBinding
                ? ok(state.qqBinding)
                : apiError("A0040404", "qq binding not found", 404),
        ),
    );
    await page.route("**/api/v1/user/schools", (route) =>
        route.fulfill(ok([joinStartSchool])),
    );
}

async function confirmAdmissionQQBinding(page: Page, qq = "123456") {
    await expect(
        page.getByRole("heading", { name: "确认绑定 QQ" }),
    ).toBeVisible();
    await page.locator("[data-admission-bind-confirmation-input]").fill(qq);
    await page.locator("[data-admission-bind-confirmation-submit]").click();
}

function joinAdmissionURL(testInfo: TestInfo, path: string): string {
    const url = new URL(path, String(testInfo.project.use.baseURL));
    url.hostname = "join.localhost";
    return url.toString();
}

function createDeferred<T = void>() {
    let resolve!: (value: T) => void;
    let reject!: (reason?: unknown) => void;
    const promise = new Promise<T>((promiseResolve, promiseReject) => {
        resolve = promiseResolve;
        reject = promiseReject;
    });
    return { promise, reject, resolve };
}

test.describe("Auth callback and admission entry", () => {
    test("auth callback consumes an upstream OAuth state and redirects to backend callback", async ({
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

    test("auth callback consumes an identity OAuth state and returns to saved app redirect", async ({
        page,
    }) => {
        await page.addInitScript(() => {
            sessionStorage.setItem("identity_oauth_state", "identity-state-1");
            sessionStorage.setItem(
                "identity_code_verifier",
                "identity-verifier-1",
            );
            sessionStorage.setItem("post_login_redirect", "/identity");
        });

        await page.goto(
            "/auth/callback?code=identity-code-1&state=identity-state-1",
        );
        await page.waitForURL(/\/identity/);
    });

    test("auth callback rejects oversized parameters before reaching backend callback", async ({
        page,
    }) => {
        let backendCallbackRequests = 0;
        await page.route("**/api/v1/auth/callback?*", async (route) => {
            backendCallbackRequests += 1;
            await route.fulfill({
                contentType: "text/html",
                body: "<!doctype html><title>Unexpected backend callback</title>",
            });
        });

        await page.goto(
            `/auth/callback?code=oauth-code-1&state=${"s".repeat(4097)}`,
        );

        await expect(page).toHaveURL(/\/login\?error=invalid_callback/);
        await expect(
            page.getByRole("button", {
                name: /Continue with unified sign-in|使用统一身份认证登录/,
            }),
        ).toBeVisible();
        expect(backendCallbackRequests).toBe(0);
    });

    test("anonymous admission link starts SSO with the current admission return URL", async ({
        page,
    }, testInfo) => {
        let loginRequestURL: URL | null = null;

        await mockUnauthenticated(page);
        await page.route(
            "**/api/v1/admission/sessions/ADMIT-LOGIN**",
            (route) => route.fulfill(ok(joinedSession)),
        );
        await page.route("**/api/v1/auth/login**", async (route) => {
            loginRequestURL = new URL(route.request().url());
            await route.fulfill(
                ok({
                    url: "https://sso.stuhelper.com/login/oauth/authorize?client_id=stuhelper-web&state=admission-sso-state",
                    state: "admission-sso-state",
                }),
            );
        });
        await page.route("https://sso.stuhelper.com/**", (route) =>
            route.fulfill({
                contentType: "text/html",
                body: "<!doctype html><title>SSO</title><main>SSO authorize</main>",
            }),
        );

        await page.goto(joinAdmissionURL(testInfo, "/verify/ADMIT-LOGIN"));

        await expect(
            page.getByRole("heading", { name: "登录 StuHelper" }),
        ).toBeVisible();
        await expect(page.getByRole("button", { name: "登录" })).toBeVisible();
        await expect(page.getByRole("button", { name: "注册" })).toBeVisible();

        const admissionURL = page.url();
        await page.getByRole("button", { name: "登录" }).click();
        await page.waitForURL(
            (url) =>
                url.hostname === "sso.stuhelper.com" &&
                url.pathname === "/login/oauth/authorize",
        );

        expect(loginRequestURL).not.toBeNull();
        expect(loginRequestURL!.searchParams.get("app")).toBe("web");
        expect(loginRequestURL!.searchParams.get("redirect")).toBe(
            admissionURL,
        );
        const ssoURL = new URL(page.url());
        expect(ssoURL.searchParams.get("client_id")).toBe("stuhelper-web");
        expect(ssoURL.searchParams.get("state")).toBe("admission-sso-state");
        await expect(page.getByText("SSO authorize")).toBeVisible();
    });

    test("main host admission links render the regular not found page", async ({
        page,
    }) => {
        await mockUnauthenticated(page);

        await page.goto("/verify/ADMIT-MAIN-HOST");

        await expect(
            page.getByRole("heading", { name: /Page Not Found|页面不存在/i }),
        ).toBeVisible({ timeout: 10_000 });
        await expect(
            page.getByRole("heading", { name: "入群身份认证" }),
        ).toHaveCount(0);
    });

    test("main host join self-service start renders the regular not found page", async ({
        page,
    }) => {
        await mockUnauthenticated(page);

        await page.goto("/start");

        await expect(
            page.getByRole("heading", { name: /Page Not Found|页面不存在/i }),
        ).toBeVisible({ timeout: 10_000 });
        await expect(
            page.getByRole("heading", { name: "学生认证与 QQ 绑定" }),
        ).toHaveCount(0);
    });

    test("anonymous join self-service start starts SSO with the join start return URL", async ({
        page,
    }, testInfo) => {
        let loginRequestURL: URL | null = null;

        await mockUnauthenticated(page);
        await page.route("**/api/v1/auth/login**", async (route) => {
            loginRequestURL = new URL(route.request().url());
            await route.fulfill(
                ok({
                    url: "https://sso.stuhelper.com/login/oauth/authorize?client_id=stuhelper-web&state=join-start-state",
                    state: "join-start-state",
                }),
            );
        });
        await page.route("https://sso.stuhelper.com/**", (route) =>
            route.fulfill({
                contentType: "text/html",
                body: "<!doctype html><title>SSO</title><main>SSO authorize</main>",
            }),
        );

        const joinStartURL = joinAdmissionURL(testInfo, "/start");
        await page.goto(joinStartURL);

        await expect(
            page.getByRole("heading", { name: "登录 StuHelper" }),
        ).toBeVisible();
        await page.getByRole("button", { name: "登录" }).click();
        await page.waitForURL(
            (url) =>
                url.hostname === "sso.stuhelper.com" &&
                url.pathname === "/login/oauth/authorize",
        );

        expect(loginRequestURL).not.toBeNull();
        expect(loginRequestURL!.searchParams.get("redirect")).toBe(
            joinStartURL,
        );
    });

    test("authenticated join self-service start shows account readiness steps", async ({
        page,
    }, testInfo) => {
        let admissionSessionRequests = 0;
        const readiness: JoinStartReadinessState = {
            profile: unverifiedStudentProfile,
            qqBinding: null,
        };

        await mockAuthenticated(page);
        await mockJoinStartReadiness(page, readiness);
        await page.route("**/api/v1/admission/sessions/**", (route) => {
            admissionSessionRequests += 1;
            return route.fulfill(
                apiError("admission.session_not_expected", "unexpected", 500),
            );
        });

        await page.goto(joinAdmissionURL(testInfo, "/start"));
        await expect(
            page.getByRole("heading", { name: "完成学生认证" }),
        ).toBeVisible();
        await expect(
            page.locator("[data-student-verification-panel]"),
        ).toBeVisible();
        expect(admissionSessionRequests).toBe(0);

        readiness.profile = verifiedStudentProfile;
        await page.reload();
        await expect(
            page.getByRole("heading", { name: "绑定 QQ" }),
        ).toBeVisible();
        await expect(page.locator("[data-qq-binding-panel]")).toBeVisible();

        readiness.qqBinding = {
            userID: 2,
            qqID: "123456",
            boundAt: now,
            createdAt: now,
            updatedAt: now,
        };
        await page.reload();
        await expect(
            page.getByRole("heading", { name: "账号已准备好" }),
        ).toBeVisible();
    });

    test("join admission link with a trailing slash opens and keeps the join return URL", async ({
        page,
    }, testInfo) => {
        let previewRequests = 0;
        let loginRequestURL: URL | null = null;

        await mockUnauthenticated(page);
        await page.route(
            "**/api/v1/admission/sessions/ADMIT-TRAILING",
            (route) => {
                previewRequests += 1;
                return route.fulfill(ok(joinedSession));
            },
        );
        await page.route("**/api/v1/auth/login**", async (route) => {
            loginRequestURL = new URL(route.request().url());
            await route.fulfill(
                ok({
                    url: "https://sso.stuhelper.com/login/oauth/authorize?client_id=stuhelper-web&state=admission-trailing-state",
                    state: "admission-trailing-state",
                }),
            );
        });
        await page.route("https://sso.stuhelper.com/**", (route) =>
            route.fulfill({
                contentType: "text/html",
                body: "<!doctype html><title>SSO</title><main>SSO authorize</main>",
            }),
        );

        const joinURL = new URL(
            "/verify/ADMIT-TRAILING/",
            String(testInfo.project.use.baseURL),
        );
        joinURL.hostname = "join.localhost";

        await page.goto(joinURL.toString());

        await expect(
            page.getByRole("heading", { name: "登录 StuHelper" }),
        ).toBeVisible();
        await expect(page.getByText("QQ：123456")).toBeVisible();
        expect(previewRequests).toBe(1);

        await page.getByRole("button", { name: "登录" }).click();
        await page.waitForURL(
            (url) =>
                url.hostname === "sso.stuhelper.com" &&
                url.pathname === "/login/oauth/authorize",
        );

        expect(loginRequestURL).not.toBeNull();
        expect(loginRequestURL!.searchParams.get("redirect")).toBe(
            joinURL.toString(),
        );
    });

    test("join admission link with source query opens and uses normalized return URL", async ({
        page,
    }, testInfo) => {
        let loginRequestURL: URL | null = null;

        await mockUnauthenticated(page);
        await page.route(
            "**/api/v1/admission/sessions/ADMIT-SOURCE**",
            (route) => route.fulfill(ok(joinedSession)),
        );
        await page.route("**/api/v1/auth/login**", async (route) => {
            loginRequestURL = new URL(route.request().url());
            await route.fulfill(
                ok({
                    url: "https://sso.stuhelper.com/login/oauth/authorize?client_id=stuhelper-web&state=admission-source-state",
                    state: "admission-source-state",
                }),
            );
        });
        await page.route("https://sso.stuhelper.com/**", (route) =>
            route.fulfill({
                contentType: "text/html",
                body: "<!doctype html><title>SSO</title><main>SSO authorize</main>",
            }),
        );

        const joinURL = new URL(
            "/verify/ADMIT-SOURCE?from=qq",
            String(testInfo.project.use.baseURL),
        );
        joinURL.hostname = "join.localhost";
        const normalizedJoinURL = new URL(
            "/verify/ADMIT-SOURCE",
            String(testInfo.project.use.baseURL),
        );
        normalizedJoinURL.hostname = "join.localhost";

        await page.goto(joinURL.toString());

        await expect(
            page.getByRole("heading", { name: "登录 StuHelper" }),
        ).toBeVisible();
        await expect(page.getByText("QQ：123456")).toBeVisible();

        await page.getByRole("button", { name: "登录" }).click();
        await page.waitForURL(
            (url) =>
                url.hostname === "sso.stuhelper.com" &&
                url.pathname === "/login/oauth/authorize",
        );

        expect(loginRequestURL).not.toBeNull();
        expect(loginRequestURL!.searchParams.get("redirect")).toBe(
            normalizedJoinURL.toString(),
        );
    });

    test("anonymous admission signup starts SSO signup with the current admission return URL", async ({
        page,
    }, testInfo) => {
        let loginRequests = 0;
        let signupRequestURL: URL | null = null;

        await mockUnauthenticated(page);
        await page.route(
            "**/api/v1/admission/sessions/ADMIT-SIGNUP**",
            (route) => route.fulfill(ok(joinedSession)),
        );
        await page.route("**/api/v1/auth/login**", (route) => {
            loginRequests += 1;
            return route.fulfill(
                apiError("unexpected.login", "unexpected login", 500),
            );
        });
        await page.route("**/api/v1/auth/signup**", async (route) => {
            signupRequestURL = new URL(route.request().url());
            await route.fulfill(
                ok({
                    url: "https://sso.stuhelper.com/signup/oauth/authorize?client_id=stuhelper-web&state=admission-sso-signup-state",
                    state: "admission-sso-signup-state",
                }),
            );
        });
        await page.route("https://sso.stuhelper.com/**", (route) =>
            route.fulfill({
                contentType: "text/html",
                body: "<!doctype html><title>SSO</title><main>SSO signup authorize</main>",
            }),
        );

        await page.goto(joinAdmissionURL(testInfo, "/verify/ADMIT-SIGNUP"));

        await expect(
            page.getByRole("heading", { name: "登录 StuHelper" }),
        ).toBeVisible();
        const admissionURL = page.url();
        await page.getByRole("button", { name: "注册" }).click();
        await page.waitForURL(
            (url) =>
                url.hostname === "sso.stuhelper.com" &&
                url.pathname === "/signup/oauth/authorize",
        );

        expect(loginRequests).toBe(0);
        expect(signupRequestURL).not.toBeNull();
        expect(signupRequestURL!.searchParams.get("app")).toBe("web");
        expect(signupRequestURL!.searchParams.get("redirect")).toBe(
            admissionURL,
        );
        const ssoURL = new URL(page.url());
        expect(ssoURL.searchParams.get("client_id")).toBe("stuhelper-web");
        expect(ssoURL.searchParams.get("state")).toBe(
            "admission-sso-signup-state",
        );
        await expect(page.getByText("SSO signup authorize")).toBeVisible();
    });

    test("admission return URL resumes the verify page without a manual refresh", async ({
        page,
    }, testInfo) => {
        let authenticated = false;
        let loginRequestURL: URL | null = null;
        let linkQQ = "";

        allowExpectedConsoleError(
            page,
            /Failed to load resource: net::ERR_NETWORK_CHANGED/,
        );
        allowExpectedConsoleError(
            page,
            /\[App\] bootstrap failed: TypeError: Failed to fetch dynamically imported module: .*AdmissionPage\.vue/,
        );
        await page.route("**/api/v1/auth/me", (route) =>
            route.fulfill(
                authenticated
                    ? ok(user)
                    : apiError("A0010100", "login required", 401),
            ),
        );
        await page.route("**/api/v1/auth/refresh", (route) =>
            route.fulfill(
                authenticated
                    ? ok({ expiresIn: 3600 })
                    : apiError("A0010100", "login required", 401),
            ),
        );
        await page.route("**/api/v1/auth/login**", async (route) => {
            loginRequestURL = new URL(route.request().url());
            await route.fulfill(
                ok({
                    url: "https://sso.stuhelper.com/login/oauth/authorize?client_id=stuhelper-web&state=admission-sso-state",
                    state: "admission-sso-state",
                }),
            );
        });
        await page.route("**/api/v1/user/qq-binding", (route) =>
            route.fulfill(ok(null)),
        );
        await page.route("https://sso.stuhelper.com/**", (route) =>
            route.fulfill({
                contentType: "text/html",
                body: "<!doctype html><title>SSO</title><main>SSO authorize</main>",
            }),
        );
        await page.route(
            "**/api/v1/admission/sessions/ADMIT-RETURN**",
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
        await page.route("**/api/v1/admission/me**", (route) =>
            route.fulfill(ok(freshmanAdmissionMe)),
        );
        await page.route("**/api/v1/user/schools", (route) =>
            route.fulfill(
                ok([
                    {
                        schoolID: 4111010006,
                        schoolCode: "4111010006",
                        schoolName: "北京航空航天大学",
                        verificationMethod: "manual",
                        consentText: null,
                        manualFormFields: null,
                        enabled: true,
                        schoolSsoEnabled: false,
                        schoolEmailOtpEnabled: false,
                    },
                ]),
            ),
        );

        await page.goto(joinAdmissionURL(testInfo, "/verify/ADMIT-RETURN"));
        await expect(
            page.getByRole("heading", { name: "登录 StuHelper" }),
        ).toBeVisible();
        const admissionURL = page.url();
        await page.getByRole("button", { name: "登录" }).click();
        await page.waitForURL(
            (url) =>
                url.hostname === "sso.stuhelper.com" &&
                url.pathname === "/login/oauth/authorize",
        );
        await expect(page.getByText("SSO authorize")).toBeVisible();

        authenticated = true;
        await page.goto(joinAdmissionURL(testInfo, "/verify/ADMIT-RETURN"));
        await page.waitForURL(
            (url) =>
                url.hostname === "join.localhost" &&
                url.pathname === "/verify/ADMIT-RETURN",
        );

        await expect(
            page.getByRole("heading", { name: "确认绑定当前 QQ" }),
        ).toBeVisible();
        await page.getByRole("button", { name: "开始认证" }).click();
        await confirmAdmissionQQBinding(page);
        await expect(
            page.getByRole("heading", { name: "选择认证方式" }),
        ).toBeVisible();

        expect(loginRequestURL).not.toBeNull();
        expect(loginRequestURL!.searchParams.get("redirect")).toBe(
            admissionURL,
        );
        expect(linkQQ).toBe("");
    });

    test("reopening a consumed admission link resumes for the originally logged-in account", async ({
        page,
    }, testInfo) => {
        let previewRequests = 0;
        let linkRequests = 0;
        let linkQQ = "";

        allowExpectedConsoleError(
            page,
            /Failed to load resource: net::ERR_NETWORK_CHANGED/,
        );
        await mockAuthenticated(page);
        await page.route(
            "**/api/v1/admission/sessions/ADMIT-CONSUMED**",
            async (route) => {
                const url = new URL(route.request().url());
                if (url.pathname.endsWith("/link")) {
                    linkRequests += 1;
                    linkQQ = url.searchParams.get("qq") ?? "";
                    await route.fulfill(ok(linkedSession));
                    return;
                }
                previewRequests += 1;
                await route.fulfill(
                    apiError("admission.token_consumed", "consumed", 409),
                );
            },
        );
        await page.route("**/api/v1/admission/me**", (route) =>
            route.fulfill(ok(freshmanAdmissionMe)),
        );
        await page.route("**/api/v1/user/schools", (route) =>
            route.fulfill(
                ok([
                    {
                        schoolID: 4111010006,
                        schoolCode: "4111010006",
                        schoolName: "北京航空航天大学",
                        verificationMethod: "manual",
                        consentText: null,
                        manualFormFields: null,
                        enabled: true,
                        schoolSsoEnabled: false,
                        schoolEmailOtpEnabled: false,
                    },
                ]),
            ),
        );

        await page.goto(joinAdmissionURL(testInfo, "/verify/ADMIT-CONSUMED"));

        const flowHeading = page.getByRole("heading", { name: "选择认证方式" });
        await expect(flowHeading).toBeVisible({ timeout: 5_000 });
        await expect(
            page.getByRole("heading", { name: "链接已失效" }),
        ).toHaveCount(0);
        await expect(
            page.getByRole("heading", { name: "账号不匹配" }),
        ).toHaveCount(0);
        expect(previewRequests).toBeGreaterThanOrEqual(1);
        expect(linkRequests).toBeGreaterThanOrEqual(1);
        expect(linkQQ).toBe("");
    });

    test("admission token mismatch and expired states block submission controls", async ({
        page,
    }, testInfo) => {
        await mockAuthenticated(page);

        await page.route(
            "**/api/v1/admission/sessions/ADMIT-MISMATCH**",
            (route) =>
                route.fulfill(
                    apiError("admission.qq_mismatch", "mismatch", 400),
                ),
        );
        await page.goto(joinAdmissionURL(testInfo, "/verify/ADMIT-MISMATCH"));
        await expect(
            page.getByRole("heading", { name: "QQ 账号不匹配" }),
        ).toBeVisible();
        await expect(
            page.getByText("当前登录的 StuHelper 账号已绑定其他 QQ"),
        ).toBeVisible();
        await expect(
            page.getByRole("button", { name: "开始认证" }),
        ).toHaveCount(0);
        await expect(
            page.locator("[data-admission-freshman-flow]"),
        ).toHaveCount(0);

        await page.route(
            "**/api/v1/admission/sessions/ADMIT-MISSING**",
            (route) =>
                route.fulfill(
                    apiError("admission.token_not_found", "missing", 404),
                ),
        );
        await page.goto(joinAdmissionURL(testInfo, "/verify/ADMIT-MISSING"));
        await expect(
            page.getByRole("heading", { name: "认证链接无效" }),
        ).toBeVisible();
        await expect(
            page.getByText("请回到 QQ 群使用最新链接"),
        ).toBeVisible();
        await expect(
            page.getByRole("heading", { name: "链接已失效" }),
        ).toHaveCount(0);
        await expect(
            page.getByRole("button", { name: "开始认证" }),
        ).toHaveCount(0);
        await page
            .locator("[data-admission-copy-reissue-command]")
            .click();
        await expect(
            page.getByText(/重新生成指令已复制|复制失败，请手动复制/),
        ).toBeVisible();

        await page.route(
            "**/api/v1/admission/sessions/ADMIT-EXPIRED**",
            (route) =>
                route.fulfill(
                    apiError("admission.token_expired", "expired", 410),
                ),
        );
        await page.goto(joinAdmissionURL(testInfo, "/verify/ADMIT-EXPIRED"));
        await expect(
            page.getByRole("heading", { name: "链接已失效" }),
        ).toBeVisible();
        await expect(
            page.getByRole("button", { name: "开始认证" }),
        ).toHaveCount(0);
        await expect(
            page.locator("[data-admission-freshman-flow]"),
        ).toHaveCount(0);
    });

    test("logged-in user links an admission session and verifies school email OTP", async ({
        page,
    }, testInfo) => {
        let linkQQ = "";
        let academicMatchBody: unknown = null;
        let otpRequestBody: unknown = null;
        let otpVerifyBody: unknown = null;

        await mockAuthenticated(page);
        await page.route("**/api/v1/user/qq-binding", (route) =>
            route.fulfill(ok(null)),
        );
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
        await page.route("**/api/v1/admission/me**", (route) =>
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
                        schoolID: 4111010006,
                        schoolCode: "4111010006",
                        schoolName: "北京航空航天大学",
                        verificationMethod: "manual",
                        consentText: null,
                        manualFormFields: null,
                        enabled: true,
                        schoolSsoEnabled: false,
                        schoolEmailOtpEnabled: true,
                        schoolEmailIdentityPolicy: {
                            type: "academic_student_email",
                            studentIDEmailDomain: "buaa.edu.cn",
                            requireStudentName: true,
                        },
                    },
                ]),
            ),
        );
        await page.route(
            "**/api/v1/admission/school-email/academic-match",
            async (route) => {
                academicMatchBody = route.request().postDataJSON();
                await route.fulfill(
                    ok({
                        matched: true,
                        email: "20250001@buaa.edu.cn",
                        studentID: "20250001",
                        message: "学号和姓名已匹配。",
                    }),
                );
            },
        );
        await page.route(
            "**/api/v1/admission/school-email/request-otp",
            async (route) => {
                otpRequestBody = route.request().postDataJSON();
                await route.fulfill(
                    ok({
                        email: "20250001@buaa.edu.cn",
                        studentID: "20250001",
                        cooldownSeconds: 60,
                    }),
                );
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

        await page.goto(joinAdmissionURL(testInfo, "/verify/ADMIT-1"));

        await expect(page.getByText("QQ：123456")).toBeVisible();
        await expect(
            page.getByRole("heading", { name: "确认绑定当前 QQ" }),
        ).toBeVisible();
        await page.getByRole("button", { name: "开始认证" }).click();
        await confirmAdmissionQQBinding(page, " 123456 ");

        await expect(
            page.getByRole("heading", { name: "选择认证方式" }),
        ).toBeVisible();
        await expect(
            page.locator("[data-admission-old-student-flow]"),
        ).toBeVisible();
        expect(linkQQ).toBe("");

        await page.locator("[data-school-select]").selectOption("4111010006");
        const academicEmailInput = page.locator("[data-academic-email-input]");
        await expect(academicEmailInput).toHaveJSProperty("readOnly", true);
        await expect(academicEmailInput).toHaveValue("");
        await page.locator("[data-academic-student-id-input]").fill("20250001");
        await page.locator("[data-academic-student-name-input]").fill("张三");
        await page.getByRole("button", { name: "校验并发送验证码" }).click();
        await expect(academicEmailInput).toHaveValue("20250001@buaa.edu.cn");
        await page.getByLabel("验证码").fill("654321");
        await page.getByRole("button", { name: "验证邮箱" }).click();

        await expect(
            page.getByRole("heading", { name: "认证已通过" }),
        ).toBeVisible();
        expect(academicMatchBody).toEqual({
            schoolCode: "4111010006",
            admissionSessionID: "admission-session-1",
            studentID: "20250001",
            studentName: "张三",
        });
        expect(otpRequestBody).toEqual({
            schoolCode: "4111010006",
            admissionSessionID: "admission-session-1",
            studentID: "20250001",
            studentName: "张三",
        });
        expect(otpVerifyBody).toEqual({
            schoolCode: "4111010006",
            admissionSessionID: "admission-session-1",
            email: "20250001@buaa.edu.cn",
            code: "654321",
        });
    });

    test("freshman admission offers mobile handoff when desktop camera is unavailable", async ({
        page,
    }, testInfo) => {
        allowExpectedConsoleError(
            page,
            /Failed to load resource: net::ERR_NETWORK_CHANGED/,
        );
        allowExpectedConsoleError(
            page,
            /\[App\] bootstrap failed: TypeError: Failed to fetch dynamically imported module: .*AdmissionPage\.vue/,
        );
        await page.addInitScript(() => {
            Object.defineProperty(navigator, "mediaDevices", {
                configurable: true,
                value: undefined,
            });
        });
        await mockAuthenticated(page);
        await page.route(
            "**/api/v1/admission/sessions/ADMIT-FRESHMAN**",
            (route) => route.fulfill(ok(linkedSession)),
        );
        await page.route("**/api/v1/admission/me**", (route) =>
            route.fulfill(ok(freshmanAdmissionMe)),
        );
        await page.route("**/api/v1/user/schools", (route) =>
            route.fulfill(
                ok([
                    {
                        schoolID: 4111010006,
                        schoolCode: "4111010006",
                        schoolName: "北京航空航天大学",
                        verificationMethod: "manual",
                        consentText: null,
                        manualFormFields: null,
                        enabled: true,
                        schoolSsoEnabled: false,
                        schoolEmailOtpEnabled: false,
                    },
                ]),
            ),
        );

        await page.goto(joinAdmissionURL(testInfo, "/verify/ADMIT-FRESHMAN"));

        await expect(
            page.getByRole("heading", { name: "选择认证方式" }),
        ).toBeVisible();
        await expect(
            page.locator("[data-admission-freshman-flow]"),
        ).toBeVisible();
        await expect(
            page.getByRole("tab", { name: "新生认证" }),
        ).toHaveAttribute("aria-selected", "true");
        await expect(
            page.getByRole("tab", { name: "老生认证" }),
        ).toHaveCount(0);
        await expect(
            page.getByRole("button", { name: "当前浏览器不支持摄像头" }),
        ).toBeDisabled();
        await expect(
            page.getByRole("button", { name: "手机扫码拍照" }),
        ).toBeVisible();
        await expect(
            page.getByText(
                "请用手机浏览器打开此链接，并允许浏览器访问摄像头。",
            ),
        ).toHaveCount(0);
        await expect(page.getByText("Permission denied")).toHaveCount(0);
        await expect(page.locator("[data-freshman-school-select]")).toHaveValue(
            "4111010006",
        );
        await expect(page.locator('input[type="file"]')).toHaveCount(0);
        await expect(page.getByText(/相册|拖拽|PDF|文件/)).toHaveCount(0);
    });

    test("freshman admission creates a mobile camera handoff and reacts to SSE continuation", async ({
        page,
    }, testInfo) => {
        let applicationBody: unknown = null;
        let handoffCreated = false;
        let eventSourceRequested = false;
        let pollingRequests = 0;

        await mockAuthenticated(page);
        await page.route(
            "**/api/v1/admission/sessions/ADMIT-HANDOFF**",
            (route) => route.fulfill(ok(linkedSession)),
        );
        await page.route("**/api/v1/admission/me**", (route) =>
            route.fulfill(ok(freshmanAdmissionMe)),
        );
        await page.route("**/api/v1/user/schools", (route) =>
            route.fulfill(
                ok([
                    {
                        schoolID: 4111010006,
                        schoolCode: "4111010006",
                        schoolName: "北京航空航天大学",
                        verificationMethod: "manual",
                        consentText: null,
                        manualFormFields: null,
                        enabled: true,
                        schoolSsoEnabled: false,
                        schoolEmailOtpEnabled: false,
                    },
                ]),
            ),
        );
        await page.route(
            "**/api/v1/admission/freshman/applications",
            async (route) => {
                applicationBody = route.request().postDataJSON();
                await route.fulfill(
                    ok({
                        id: "freshman-application-1",
                        userID: user.id,
                        status: "pending",
                        schoolID: 4111010006,
                        qqID: "123456",
                        applicantNameMasked: "赵*",
                        materialType: "admission_notice",
                        failureCount: 0,
                        createdAt: now,
                    }),
                );
            },
        );
        await page.route(
            "**/api/v1/admission/freshman/applications/freshman-application-1/camera-handoffs",
            async (route) => {
                handoffCreated = true;
                await route.fulfill(
                    ok({
                        id: "handoff-1",
                        applicationID: "freshman-application-1",
                        userID: user.id,
                        status: "pending",
                        maxMaterialBytes: 5_242_880,
                        mobileURL:
                            "https://join.stuhelper.com/admission/freshman/camera/mobile-token",
                        expiresAt: "2026-05-24T04:30:00Z",
                        createdAt: now,
                    }),
                );
            },
        );
        await page.route(
            "**/api/v1/admission/freshman/camera-handoffs/handoff-1/events",
            async (route) => {
                eventSourceRequested = true;
                const uploaded = {
                    id: "handoff-1",
                    applicationID: "freshman-application-1",
                    userID: user.id,
                    status: "uploaded",
                    maxMaterialBytes: 5_242_880,
                    mobileURL:
                        "https://join.stuhelper.com/admission/freshman/camera/mobile-token",
                    expiresAt: "2026-05-24T04:30:00Z",
                    uploadedAt: now,
                    createdAt: now,
                };
                const locked = {
                    ...uploaded,
                    status: "locked",
                    continueOn: "desktop",
                    chosenAt: now,
                };
                await route.fulfill({
                    status: 200,
                    contentType: "text/event-stream",
                    body:
                        [uploaded, locked]
                            .map(
                                (handoff) =>
                                    `event: handoff\ndata: ${JSON.stringify(handoff)}\n`,
                            )
                            .join("\n") + "\n",
                });
            },
        );
        await page.route(
            "**/api/v1/admission/freshman/camera-handoffs/handoff-1",
            async (route) => {
                pollingRequests += 1;
                await route.fulfill(
                    ok({
                        id: "handoff-1",
                        applicationID: "freshman-application-1",
                        userID: user.id,
                        status: "uploaded",
                        maxMaterialBytes: 5_242_880,
                        mobileURL:
                            "https://join.stuhelper.com/admission/freshman/camera/mobile-token",
                        expiresAt: "2026-05-24T04:30:00Z",
                        uploadedAt: now,
                        createdAt: now,
                    }),
                );
            },
        );

        await page.goto(joinAdmissionURL(testInfo, "/verify/ADMIT-HANDOFF"));

        await expect(
            page.locator("[data-admission-freshman-flow]"),
        ).toBeVisible();
        await page
            .locator("[data-freshman-school-select]")
            .selectOption("4111010006");
        await page.locator("[data-freshman-applicant-name-input]").fill("赵一");
        await page.getByRole("button", { name: "手机扫码拍照" }).click();

        await expect(
            page.getByRole("heading", { name: "等待管理员审核" }),
        ).toBeVisible();
        expect(applicationBody).toEqual({
            schoolCode: "4111010006",
            applicantName: "赵一",
            materialType: "admission_notice",
            admissionSessionID: "admission-session-1",
        });
        expect(handoffCreated).toBe(true);
        expect(eventSourceRequested).toBe(true);
        expect(pollingRequests).toBe(0);
    });

    test("freshman admission allows desktop material submission while a mobile handoff is still pending", async ({
        page,
    }, testInfo) => {
        let cameraCaptureBody: unknown = null;
        let handoffCreated = false;

        await page.addInitScript(() => {
            Object.defineProperty(navigator, "mediaDevices", {
                configurable: true,
                value: {
                    getUserMedia: async () => new MediaStream(),
                },
            });
            Object.defineProperty(HTMLVideoElement.prototype, "videoWidth", {
                configurable: true,
                get: () => 2,
            });
            Object.defineProperty(HTMLVideoElement.prototype, "videoHeight", {
                configurable: true,
                get: () => 2,
            });
            HTMLVideoElement.prototype.play = async () => undefined;
            HTMLCanvasElement.prototype.getContext = ((contextId: string) => {
                if (contextId !== "2d") {
                    return null;
                }
                return {
                    drawImage: () => undefined,
                } as unknown as CanvasRenderingContext2D;
            }) as HTMLCanvasElement["getContext"];
            HTMLCanvasElement.prototype.toDataURL = () =>
                "data:image/jpeg;base64,QUJDRA==";
        });
        await mockAuthenticated(page);
        await page.route(
            "**/api/v1/admission/sessions/ADMIT-HANDOFF-DESKTOP**",
            (route) => route.fulfill(ok(linkedSession)),
        );
        await page.route("**/api/v1/admission/me**", (route) =>
            route.fulfill(ok(freshmanAdmissionMe)),
        );
        await page.route("**/api/v1/user/schools", (route) =>
            route.fulfill(
                ok([
                    {
                        schoolID: 4111010006,
                        schoolCode: "4111010006",
                        schoolName: "北京航空航天大学",
                        verificationMethod: "manual",
                        consentText: null,
                        manualFormFields: null,
                        enabled: true,
                        schoolSsoEnabled: false,
                        schoolEmailOtpEnabled: false,
                    },
                ]),
            ),
        );
        await page.route("**/api/v1/admission/freshman/applications", (route) =>
            route.fulfill(
                ok({
                    id: "freshman-application-1",
                    userID: user.id,
                    status: "pending",
                    schoolID: 4111010006,
                    qqID: "123456",
                    applicantNameMasked: "赵*",
                    materialType: "admission_notice",
                    failureCount: 0,
                    createdAt: now,
                }),
            ),
        );
        await page.route(
            "**/api/v1/admission/freshman/applications/freshman-application-1/camera-handoffs",
            async (route) => {
                handoffCreated = true;
                await route.fulfill(
                    ok({
                        id: "handoff-pending",
                        applicationID: "freshman-application-1",
                        userID: user.id,
                        status: "pending",
                        maxMaterialBytes: 5_242_880,
                        mobileURL:
                            "https://join.stuhelper.com/admission/freshman/camera/mobile-token",
                        expiresAt: "2026-05-24T04:30:00Z",
                        createdAt: now,
                    }),
                );
            },
        );
        await page.route(
            "**/api/v1/admission/freshman/camera-handoffs/handoff-pending**",
            (route) =>
                route.fulfill(
                    ok({
                        id: "handoff-pending",
                        applicationID: "freshman-application-1",
                        userID: user.id,
                        status: "pending",
                        maxMaterialBytes: 5_242_880,
                        mobileURL:
                            "https://join.stuhelper.com/admission/freshman/camera/mobile-token",
                        expiresAt: "2026-05-24T04:30:00Z",
                        createdAt: now,
                    }),
                ),
        );
        await page.route(
            "**/api/v1/admission/freshman/applications/freshman-application-1/camera-captures",
            async (route) => {
                cameraCaptureBody = route.request().postDataJSON();
                await route.fulfill(
                    ok({
                        id: "freshman-application-1",
                        userID: user.id,
                        status: "pending",
                        schoolID: 4111010006,
                        qqID: "123456",
                        applicantNameMasked: "赵*",
                        materialType: "admission_notice",
                        materialURL:
                            "https://stuhelper.com/materials/freshman-application-1.jpg",
                        failureCount: 0,
                        createdAt: now,
                    }),
                );
            },
        );

        await page.goto(
            joinAdmissionURL(testInfo, "/verify/ADMIT-HANDOFF-DESKTOP"),
        );

        await expect(
            page.locator("[data-admission-freshman-flow]"),
        ).toBeVisible();
        await page
            .locator("[data-freshman-school-select]")
            .selectOption("4111010006");
        await page.getByLabel("姓名").fill("赵一");
        await page.getByRole("button", { name: "手机扫码拍照" }).click();
        await expect(
            page.locator("[data-freshman-mobile-handoff]"),
        ).toBeVisible();
        await page.getByRole("button", { name: "打开摄像头" }).click();
        await page.getByRole("button", { name: "拍摄" }).click();
        await expect(page.getByAltText("录取材料预览")).toBeVisible();
        await page.getByRole("button", { name: "提交材料" }).click();

        await expect(
            page.getByRole("heading", { name: "等待管理员审核" }),
        ).toBeVisible();
        expect(handoffCreated).toBe(true);
        expect(cameraCaptureBody).toMatchObject({
            contentType: "image/jpeg",
            imageBase64: "QUJDRA==",
        });
    });

    test("freshman mobile camera token uploads without requiring login", async ({
        page,
    }, testInfo) => {
        let uploadBody: unknown = null;
        let continuationBody: unknown = null;
        let authRequestCount = 0;

        await page.addInitScript(() => {
            Object.defineProperty(navigator, "mediaDevices", {
                configurable: true,
                value: {
                    getUserMedia: async () => new MediaStream(),
                },
            });
            Object.defineProperty(HTMLVideoElement.prototype, "videoWidth", {
                configurable: true,
                get: () => 2,
            });
            Object.defineProperty(HTMLVideoElement.prototype, "videoHeight", {
                configurable: true,
                get: () => 2,
            });
            HTMLVideoElement.prototype.play = async () => undefined;
            HTMLCanvasElement.prototype.getContext = ((contextId: string) => {
                if (contextId !== "2d") {
                    return null;
                }
                return {
                    drawImage: () => undefined,
                } as unknown as CanvasRenderingContext2D;
            }) as HTMLCanvasElement["getContext"];
            HTMLCanvasElement.prototype.toDataURL = () =>
                "data:image/jpeg;base64,QUJDRA==";
        });
        await page.route("**/api/v1/auth/**", (route) => {
            authRequestCount += 1;
            return route.fulfill(
                apiError("unexpected_auth", "unexpected auth call", 500),
            );
        });
        await page.route(
            "**/api/v1/admission/freshman/mobile-camera-handoffs/mobile-token",
            (route) =>
                route.fulfill(
                    ok({
                        id: "handoff-1",
                        applicationID: "freshman-application-1",
                        userID: user.id,
                        status: "pending",
                        maxMaterialBytes: 5_242_880,
                        mobileURL:
                            "https://join.stuhelper.com/admission/freshman/camera/mobile-token",
                        expiresAt: "2026-05-24T04:30:00Z",
                        createdAt: now,
                    }),
                ),
        );
        await page.route(
            "**/api/v1/admission/freshman/mobile-camera-handoffs/mobile-token/camera-capture",
            async (route) => {
                uploadBody = route.request().postDataJSON();
                await route.fulfill(
                    ok({
                        id: "handoff-1",
                        applicationID: "freshman-application-1",
                        userID: user.id,
                        status: "uploaded",
                        maxMaterialBytes: 5_242_880,
                        mobileURL:
                            "https://join.stuhelper.com/admission/freshman/camera/mobile-token",
                        expiresAt: "2026-05-24T04:30:00Z",
                        uploadedAt: now,
                        createdAt: now,
                    }),
                );
            },
        );
        await page.route(
            "**/api/v1/admission/freshman/mobile-camera-handoffs/mobile-token/continue",
            async (route) => {
                continuationBody = route.request().postDataJSON();
                await route.fulfill(
                    ok({
                        id: "handoff-1",
                        applicationID: "freshman-application-1",
                        userID: user.id,
                        status: "locked",
                        continueOn: "mobile",
                        maxMaterialBytes: 5_242_880,
                        mobileURL:
                            "https://join.stuhelper.com/admission/freshman/camera/mobile-token",
                        expiresAt: "2026-05-24T04:30:00Z",
                        uploadedAt: now,
                        chosenAt: now,
                        createdAt: now,
                    }),
                );
            },
        );

        await page.goto(
            joinAdmissionURL(testInfo, "/admission/freshman/camera/mobile-token"),
        );

        await expect(page.locator('[data-state="ready"]')).toBeVisible();
        await page.getByRole("button", { name: "打开摄像头" }).click();
        await page.getByRole("button", { name: "拍摄" }).click();
        await page.getByRole("button", { name: "上传材料" }).click();
        await expect(page.locator('[data-state="uploaded"]')).toBeVisible();
        await page.getByRole("button", { name: "在手机端继续" }).click();
        await expect(page.locator('[data-state="mobile"]')).toBeVisible();

        expect(uploadBody).toMatchObject({
            contentType: "image/jpeg",
            imageBase64: "QUJDRA==",
        });
        expect(typeof (uploadBody as { capturedAt?: unknown }).capturedAt).toBe(
            "string",
        );
        expect(continuationBody).toEqual({ continueOn: "mobile" });
        expect(authRequestCount).toBe(0);
    });

    test("freshman mobile camera ignores stale previews after token changes", async ({
        page,
    }, testInfo) => {
        const stalePreview = createDeferred();
        let authRequestCount = 0;
        let stalePreviewRequested = false;

        await page.route("**/api/v1/auth/**", (route) => {
            authRequestCount += 1;
            return route.fulfill(
                apiError("unexpected_auth", "unexpected auth call", 500),
            );
        });
        await page.route(
            "**/api/v1/admission/freshman/mobile-camera-handoffs/stale-token",
            async (route) => {
                stalePreviewRequested = true;
                await stalePreview.promise;
                await route.fulfill(
                    ok({
                        id: "handoff-stale",
                        applicationID: "freshman-application-stale",
                        userID: user.id,
                        status: "pending",
                        maxMaterialBytes: 5_242_880,
                        mobileURL:
                            "https://join.stuhelper.com/admission/freshman/camera/stale-token",
                        expiresAt: "2026-05-24T04:30:00Z",
                        createdAt: now,
                    }),
                );
            },
        );
        await page.route(
            "**/api/v1/admission/freshman/mobile-camera-handoffs/current-token",
            (route) =>
                route.fulfill(
                    ok({
                        id: "handoff-current",
                        applicationID: "freshman-application-current",
                        userID: user.id,
                        status: "uploaded",
                        maxMaterialBytes: 5_242_880,
                        mobileURL:
                            "https://join.stuhelper.com/admission/freshman/camera/current-token",
                        expiresAt: "2026-05-24T04:30:00Z",
                        uploadedAt: now,
                        createdAt: now,
                    }),
                ),
        );

        await page.goto(
            joinAdmissionURL(
                testInfo,
                "/admission/freshman/camera/stale-token",
            ),
        );
        await expect.poll(() => stalePreviewRequested).toBe(true);

        await page.evaluate(() => {
            window.history.pushState(
                {},
                "",
                "/admission/freshman/camera/current-token",
            );
            window.dispatchEvent(new PopStateEvent("popstate"));
        });

        await expect(page.locator('[data-state="uploaded"]')).toBeVisible();
        stalePreview.resolve();
        await expect(page.locator('[data-state="uploaded"]')).toBeVisible();
        await expect(page.locator('[data-state="ready"]')).toHaveCount(0);
        expect(authRequestCount).toBe(0);
    });

    test("freshman mobile camera enforces the handoff material size limit before upload", async ({
        page,
    }, testInfo) => {
        let uploadRequests = 0;
        let authRequestCount = 0;

        await page.addInitScript(() => {
            Object.defineProperty(navigator, "mediaDevices", {
                configurable: true,
                value: {
                    getUserMedia: async () => new MediaStream(),
                },
            });
            Object.defineProperty(HTMLVideoElement.prototype, "videoWidth", {
                configurable: true,
                get: () => 2,
            });
            Object.defineProperty(HTMLVideoElement.prototype, "videoHeight", {
                configurable: true,
                get: () => 2,
            });
            HTMLVideoElement.prototype.play = async () => undefined;
            HTMLCanvasElement.prototype.getContext = ((contextId: string) => {
                if (contextId !== "2d") {
                    return null;
                }
                return {
                    drawImage: () => undefined,
                } as unknown as CanvasRenderingContext2D;
            }) as HTMLCanvasElement["getContext"];
            HTMLCanvasElement.prototype.toDataURL = () =>
                "data:image/jpeg;base64,QUJDRA==";
        });
        await page.route("**/api/v1/auth/**", (route) => {
            authRequestCount += 1;
            return route.fulfill(
                apiError("unexpected_auth", "unexpected auth call", 500),
            );
        });
        await page.route(
            "**/api/v1/admission/freshman/mobile-camera-handoffs/mobile-token",
            (route) =>
                route.fulfill(
                    ok({
                        id: "handoff-1",
                        applicationID: "freshman-application-1",
                        userID: user.id,
                        status: "pending",
                        maxMaterialBytes: 1,
                        mobileURL:
                            "https://join.stuhelper.com/admission/freshman/camera/mobile-token",
                        expiresAt: "2026-05-24T04:30:00Z",
                        createdAt: now,
                    }),
                ),
        );
        await page.route(
            "**/api/v1/admission/freshman/mobile-camera-handoffs/mobile-token/camera-capture",
            (route) => {
                uploadRequests += 1;
                return route.fulfill(
                    apiError("unexpected_upload", "unexpected upload", 500),
                );
            },
        );

        await page.goto(
            joinAdmissionURL(testInfo, "/admission/freshman/camera/mobile-token"),
        );

        await expect(page.locator('[data-state="ready"]')).toBeVisible();
        await page.getByRole("button", { name: "打开摄像头" }).click();
        await page.getByRole("button", { name: "拍摄" }).click();

        await expect(page.getByText("拍摄图片超过材料大小限制")).toBeVisible();
        await expect(
            page.getByRole("button", { name: "上传材料" }),
        ).toBeDisabled();
        expect(uploadRequests).toBe(0);
        expect(authRequestCount).toBe(0);
    });

    test("freshman mobile camera closes the stream when preview playback fails", async ({
        page,
    }, testInfo) => {
        let authRequestCount = 0;

        await page.addInitScript(() => {
            const state = globalThis as unknown as { __cameraStopCount?: number };
            state.__cameraStopCount = 0;

            Object.defineProperty(navigator, "mediaDevices", {
                configurable: true,
                value: {
                    getUserMedia: async () => {
                        const stream = new MediaStream();
                        Object.defineProperty(stream, "getTracks", {
                            configurable: true,
                            value: () => [
                                {
                                    stop: () => {
                                        state.__cameraStopCount =
                                            (state.__cameraStopCount ?? 0) + 1;
                                    },
                                },
                            ],
                        });
                        return stream;
                    },
                },
            });
            HTMLVideoElement.prototype.play = async () => {
                throw new Error("Preview playback failed");
            };
        });
        await page.route("**/api/v1/auth/**", (route) => {
            authRequestCount += 1;
            return route.fulfill(
                apiError("unexpected_auth", "unexpected auth call", 500),
            );
        });
        await page.route(
            "**/api/v1/admission/freshman/mobile-camera-handoffs/mobile-token",
            (route) =>
                route.fulfill(
                    ok({
                        id: "handoff-1",
                        applicationID: "freshman-application-1",
                        userID: user.id,
                        status: "pending",
                        maxMaterialBytes: 5_242_880,
                        mobileURL:
                            "https://join.stuhelper.com/admission/freshman/camera/mobile-token",
                        expiresAt: "2026-05-24T04:30:00Z",
                        createdAt: now,
                    }),
                ),
        );

        await page.goto(
            joinAdmissionURL(testInfo, "/admission/freshman/camera/mobile-token"),
        );

        await expect(page.locator('[data-state="ready"]')).toBeVisible();
        await page.getByRole("button", { name: "打开摄像头" }).click();

        await expect(page.getByText("Preview playback failed")).toBeVisible();
        await expect(
            page.getByRole("button", { name: "打开摄像头" }),
        ).toBeVisible();
        await expect(page.getByRole("button", { name: "拍摄" })).toHaveCount(0);
        await expect
            .poll(() =>
                page.evaluate(() => {
                    const state = globalThis as unknown as {
                        __cameraStopCount?: number;
                    };
                    return state.__cameraStopCount ?? 0;
                }),
            )
            .toBe(1);
        expect(authRequestCount).toBe(0);
    });

    test("freshman admission captures camera material and submits it for manual review", async ({
        page,
    }, testInfo) => {
        let applicationBody: unknown = null;
        let cameraCaptureBody: unknown = null;

        await page.addInitScript(() => {
            Object.defineProperty(navigator, "mediaDevices", {
                configurable: true,
                value: {
                    getUserMedia: async () => new MediaStream(),
                },
            });
            Object.defineProperty(HTMLVideoElement.prototype, "videoWidth", {
                configurable: true,
                get: () => 2,
            });
            Object.defineProperty(HTMLVideoElement.prototype, "videoHeight", {
                configurable: true,
                get: () => 2,
            });
            HTMLVideoElement.prototype.play = async () => undefined;
            HTMLCanvasElement.prototype.getContext = ((contextId: string) => {
                if (contextId !== "2d") {
                    return null;
                }
                return {
                    drawImage: () => undefined,
                } as unknown as CanvasRenderingContext2D;
            }) as HTMLCanvasElement["getContext"];
            HTMLCanvasElement.prototype.toDataURL = () =>
                "data:image/jpeg;base64,QUJDRA==";
        });
        await mockAuthenticated(page);
        await page.route(
            "**/api/v1/admission/sessions/ADMIT-CAMERA**",
            (route) =>
                route.fulfill(
                    ok({
                        ...linkedSession,
                        maxMaterialBytes: 1024,
                    }),
                ),
        );
        await page.route("**/api/v1/admission/me**", (route) =>
            route.fulfill(ok(freshmanAdmissionMe)),
        );
        await page.route("**/api/v1/user/schools", (route) =>
            route.fulfill(
                ok([
                    {
                        schoolID: 4111010006,
                        schoolCode: "4111010006",
                        schoolName: "北京航空航天大学",
                        verificationMethod: "manual",
                        consentText: null,
                        manualFormFields: null,
                        enabled: true,
                        schoolSsoEnabled: false,
                        schoolEmailOtpEnabled: false,
                    },
                ]),
            ),
        );
        await page.route(
            "**/api/v1/admission/freshman/applications",
            async (route) => {
                applicationBody = route.request().postDataJSON();
                await route.fulfill(
                    ok({
                        id: "freshman-application-1",
                        userID: user.id,
                        status: "pending",
                        schoolID: 4111010006,
                        qqID: "123456",
                        applicantNameMasked: "赵*",
                        materialType: "admission_notice",
                        failureCount: 0,
                        createdAt: now,
                    }),
                );
            },
        );
        await page.route(
            "**/api/v1/admission/freshman/applications/freshman-application-1/camera-captures",
            async (route) => {
                cameraCaptureBody = route.request().postDataJSON();
                await route.fulfill(
                    ok({
                        id: "freshman-application-1",
                        userID: user.id,
                        status: "pending",
                        schoolID: 4111010006,
                        qqID: "123456",
                        applicantNameMasked: "赵*",
                        materialType: "admission_notice",
                        materialURL:
                            "https://stuhelper.com/materials/freshman-application-1.jpg",
                        failureCount: 0,
                        createdAt: now,
                    }),
                );
            },
        );

        await page.goto(joinAdmissionURL(testInfo, "/verify/ADMIT-CAMERA"));

        await expect(
            page.locator("[data-admission-freshman-flow]"),
        ).toBeVisible();
        await page
            .locator("[data-freshman-school-select]")
            .selectOption("4111010006");
        await page.getByLabel("姓名").fill("赵一");
        await page.getByLabel("院系或专业").fill("软件工程");
        await page.getByRole("button", { name: "打开摄像头" }).click();
        await page.getByRole("button", { name: "拍摄" }).click();
        await expect(page.getByAltText("录取材料预览")).toBeVisible();
        await page.getByRole("button", { name: "提交材料" }).click();

        await expect(
            page.getByRole("heading", { name: "等待管理员审核" }),
        ).toBeVisible();
        expect(applicationBody).toEqual({
            schoolCode: "4111010006",
            applicantName: "赵一",
            departmentOrMajor: "软件工程",
            materialType: "admission_notice",
            admissionSessionID: "admission-session-1",
        });
        expect(cameraCaptureBody).toMatchObject({
            contentType: "image/jpeg",
            imageBase64: "QUJDRA==",
        });
        expect(
            typeof (cameraCaptureBody as { capturedAt?: unknown }).capturedAt,
        ).toBe("string");
    });
});
