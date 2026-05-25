import { expect, test, type Page } from './fixtures';

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

const freshmanAdmissionMe = {
    status: "linked",
    projectionPending: false,
    credentialKind: "freshman_material_manual",
    session: linkedSession,
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
        route.fulfill(
            apiError("A0010100", "login required", 401),
        ),
    );
    await page.route("**/api/v1/auth/refresh", (route) =>
        route.fulfill(
            apiError("A0010100", "login required", 401),
        ),
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
            page.getByRole("button", { name: /Login with SSO|使用 SSO 登录/ }),
        ).toBeVisible();
        expect(backendCallbackRequests).toBe(0);
    });

    test("anonymous admission link starts login with the current admission return URL", async ({
        page,
    }) => {
        let loginURL: URL | null = null;

        await mockUnauthenticated(page);
        await page.route("**/api/v1/admission/sessions/ADMIT-LOGIN**", (route) =>
            route.fulfill(ok(joinedSession)),
        );
        await page.route("**/api/v1/auth/login**", async (route) => {
            loginURL = new URL(route.request().url());
            await route.fulfill(
                ok({
                    state: "admission-login-state",
                    url: "http://localhost:8085/admission-login",
                }),
            );
        });
        await page.route("http://localhost:8085/**", (route) =>
            route.fulfill({
                contentType: "text/html",
                body: "<!doctype html><title>SSO</title><main>SSO login</main>",
            }),
        );

        await page.goto("/admission/a/ADMIT-LOGIN?qq=123456");

        await expect(
            page.getByRole("heading", { name: "登录 StuHelper" }),
        ).toBeVisible();
        await expect(page.getByRole("button", { name: "登录" })).toBeVisible();
        await expect(page.getByRole("button", { name: "注册" })).toBeVisible();

        const admissionURL = page.url();
        await page.getByRole("button", { name: "登录" }).click();
        await page.waitForURL("http://localhost:8085/admission-login");

        expect(loginURL).not.toBeNull();
        expect(loginURL!.searchParams.get("app")).toBe("web");
        expect(loginURL!.searchParams.get("redirect")).toBe(admissionURL);
        await expect(page.getByText("SSO login")).toBeVisible();
    });

    test("admission token mismatch and expired states block submission controls", async ({
        page,
    }) => {
        await mockAuthenticated(page);

        await page.route("**/api/v1/admission/sessions/ADMIT-MISMATCH**", (route) =>
            route.fulfill(apiError("admission.qq_mismatch", "mismatch", 409)),
        );
        await page.goto("/admission/a/ADMIT-MISMATCH?qq=999999");
        await expect(
            page.getByRole("heading", { name: "链接被篡改" }),
        ).toBeVisible();
        await expect(page.getByRole("button", { name: "开始认证" })).toHaveCount(0);
        await expect(page.locator("[data-admission-freshman-flow]")).toHaveCount(0);

        await page.route("**/api/v1/admission/sessions/ADMIT-EXPIRED**", (route) =>
            route.fulfill(apiError("admission.token_expired", "expired", 410)),
        );
        await page.goto("/admission/a/ADMIT-EXPIRED?qq=123456");
        await expect(
            page.getByRole("heading", { name: "链接已失效" }),
        ).toBeVisible();
        await expect(page.getByRole("button", { name: "开始认证" })).toHaveCount(0);
        await expect(page.locator("[data-admission-freshman-flow]")).toHaveCount(0);
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

    test("freshman admission shows the mobile camera prompt without upload controls when camera is unavailable", async ({
        page,
    }) => {
        await page.addInitScript(() => {
            Object.defineProperty(navigator, "mediaDevices", {
                configurable: true,
                value: undefined,
            });
        });
        await mockAuthenticated(page);
        await page.route("**/api/v1/admission/sessions/ADMIT-FRESHMAN**", (route) =>
            route.fulfill(ok(linkedSession)),
        );
        await page.route("**/api/v1/admission/me", (route) =>
            route.fulfill(ok(freshmanAdmissionMe)),
        );
        await page.route("**/api/v1/user/schools", (route) =>
            route.fulfill(
                ok([
                    {
                        schoolID: 1001,
                        schoolName: "测试大学",
                        verificationMethod: "manual",
                        consentText: null,
                        manualFormFields: null,
                        enabled: true,
                        schoolSsoEnabled: false,
                    },
                ]),
            ),
        );

        await page.goto("/admission/a/ADMIT-FRESHMAN?qq=123456");

        await expect(
            page.getByRole("heading", { name: "选择认证方式" }),
        ).toBeVisible();
        await expect(page.locator("[data-admission-freshman-flow]")).toBeVisible();
        await expect(page.locator("[data-camera-unavailable]")).toContainText(
            "请用手机浏览器打开此链接，并允许浏览器访问摄像头。",
        );
        await expect(page.locator('input[type="file"]')).toHaveCount(0);
        await expect(page.getByText(/上传|相册|拖拽|PDF|文件/)).toHaveCount(0);
    });

    test("freshman admission captures camera material and submits it for manual review", async ({
        page,
    }) => {
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
        await page.route("**/api/v1/admission/sessions/ADMIT-CAMERA**", (route) =>
            route.fulfill(
                ok({
                    ...linkedSession,
                    maxMaterialBytes: 1024,
                }),
            ),
        );
        await page.route("**/api/v1/admission/me", (route) =>
            route.fulfill(ok(freshmanAdmissionMe)),
        );
        await page.route("**/api/v1/user/schools", (route) =>
            route.fulfill(ok([])),
        );
        await page.route(
            "**/api/v1/admission/freshman/applications",
            async (route) => {
                applicationBody = route.request().postDataJSON();
                await route.fulfill(
                    ok({
                        id: "freshman-application-1",
                        status: "pending",
                        schoolID: 1001,
                        qqID: "123456",
                        applicantNameMasked: "赵*",
                        materialURL: "",
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
                        status: "pending",
                        schoolID: 1001,
                        qqID: "123456",
                        applicantNameMasked: "赵*",
                        materialURL: "camera://freshman-application-1",
                        failureCount: 0,
                        createdAt: now,
                    }),
                );
            },
        );

        await page.goto("/admission/a/ADMIT-CAMERA?qq=123456");

        await expect(page.locator("[data-admission-freshman-flow]")).toBeVisible();
        await page.getByLabel("学校 ID").fill("1001");
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
            schoolID: 1001,
            applicantName: "赵一",
            departmentOrMajor: "软件工程",
            materialType: "admission_notice",
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
