import {
    expect,
    mockCurrentAccountProjections,
    mockNotificationStream,
    test,
    type Page,
} from "./fixtures";

const now = "2026-08-05T08:00:00Z";
const applicationID = "11111111-1111-4111-8111-111111111111";
const credentialID = "22222222-2222-4222-8222-222222222222";
const phoneOperationID = "33333333-3333-4333-8333-333333333333";

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

const privacyNotice = {
    version: "buaa-verification-v1",
    title: "学生身份信息处理说明",
    summary: "仅用于本次学生身份一致性校验。",
    dataCategories: ["学号", "姓名", "身份证件号"],
    retentionSummary: "本次填写的姓名和身份证件号原文不持久化。",
};

const school = {
    code: "4111010006",
    name: "北京航空航天大学",
    location: "北京",
    methods: [
        {
            method: "real_name_identity_check",
            displayName: "实名信息校验",
            description: "通过服务端完成实名信息一致性校验。",
            availability: "available",
            formFields: [],
            privacyNotice,
        },
        {
            method: "school_sso",
            displayName: "统一身份认证验证",
            description: "使用学校统一身份认证账号完成一次性校验。",
            availability: "available",
            formFields: [],
            privacyNotice: {
                ...privacyNotice,
                version: "buaa-sso-v1",
                dataCategories: ["学号", "统一身份认证密码"],
                retentionSummary: "统一身份认证密码仅在本次请求中使用，不会保存。",
            },
        },
        {
            method: "student_email_outbound_otp",
            displayName: "学校邮箱接收验证码",
            description: "验证码只发送到规范学号邮箱。",
            availability: "available",
            formFields: [],
            privacyNotice: {
                ...privacyNotice,
                version: "buaa-email-v1",
                dataCategories: ["学号", "姓名", "学校邮箱"],
            },
        },
        {
            method: "manual_material_review",
            displayName: "人工材料审核",
            description: "自动方式不可用时拍摄材料提交审核。",
            availability: "available",
            formFields: [
                {
                    key: "department",
                    label: "学院",
                    inputType: "text",
                    required: true,
                    maxLength: 100,
                },
                {
                    key: "studentID",
                    label: "学号或录取编号",
                    inputType: "text",
                    required: true,
                    maxLength: 64,
                },
                {
                    key: "name",
                    label: "姓名",
                    inputType: "text",
                    required: true,
                    maxLength: 100,
                },
                {
                    key: "email",
                    label: "学校邮箱",
                    inputType: "email",
                    required: true,
                    maxLength: 320,
                },
            ],
            privacyNotice: {
                ...privacyNotice,
                version: "manual-v1",
                dataCategories: ["学校信息", "学生材料", "学校邮箱"],
                retentionSummary: "材料按公示的保留期限加密保存，到期删除。",
            },
        },
    ],
};

function json(data: unknown, status = 200) {
    return {
        status,
        contentType: "application/json",
        body: JSON.stringify(data),
    };
}

function ok(data: unknown = null, status = 200) {
    return json({ success: true, data }, status);
}

function application(status: "created" | "in_progress" | "approved", method: string | null = null) {
    return {
        id: applicationID,
        school: { code: school.code, name: school.name },
        status,
        currentMethod: method,
        revision: status === "approved" ? 3 : method ? 2 : 1,
        nextActions: status === "approved" ? ["return_to_consumer"] : method ? ["retry_current_method", "choose_another_method"] : ["choose_method"],
        credential: status === "approved"
            ? {
                id: credentialID,
                schoolCode: school.code,
                schoolName: school.name,
                method,
                status: "active",
                credentialClass: "formal_student",
                subjectDisplay: "2337****",
                verifiedAt: now,
                expiresAt: null,
                reviewRequiredAt: null,
                revision: 1,
            }
            : null,
        createdAt: now,
        updatedAt: now,
        expiresAt: "2026-08-05T09:00:00Z",
    };
}

async function mockAuthenticatedShell(page: Page) {
    await page.addInitScript((value) => {
        localStorage.setItem("stuhelper_user", JSON.stringify(value));
        localStorage.setItem("stuhelper_token_expiry", String(Date.now() + 60 * 60 * 1000));
        sessionStorage.clear();
    }, user);

    await page.route("**/api/v1/auth/me", (route) => route.fulfill(ok(user)));
    await page.route("**/api/v1/auth/refresh", (route) => route.fulfill(ok({ expiresIn: 3600 })));
    await mockCurrentAccountProjections(page, {
        displayName: user.displayName,
        studentVerified: false,
        phoneBound: false,
        capabilities: user.capabilities,
    });
    await page.route("**/api/v1/course/review/user/notifications/unread-count*", (route) =>
        route.fulfill(ok({ count: 0 })),
    );
    await mockNotificationStream(page);
}

async function mockStudentPlatform(page: Page) {
    await page.route("**/api/v1/student-verification/schools", (route) =>
        route.fulfill(ok([school])),
    );
    await page.route("**/api/v1/student-verification/credentials", (route) =>
        route.fulfill(ok([])),
    );
    await page.route("**/api/v1/student-verification/eligibility**", (route) =>
        route.fulfill(ok({
            eligible: false,
            schoolCode: school.code,
            credentialClass: null,
            credentialMethods: [],
            expiresAt: null,
            evaluatedAt: now,
            revision: 1,
        })),
    );
}

async function selectSchoolAndMethod(page: Page, method: string) {
    await page.locator("[data-verification-school-option]").click();
    await page.locator(`[data-verification-method="${method}"]`).click();
    await expect(page.locator("[data-verification-method-form]")).toBeVisible();
}

async function fillOTP(page: Page, groupName: string, code: string) {
    const inputs = page.getByRole("group", { name: groupName }).locator("input");
    for (let index = 0; index < code.length; index += 1) {
        await inputs.nth(index).fill(code[index] ?? "");
    }
}

test.describe("Current student verification and phone flows", () => {
    test("real-name information verification uses the independent target API and never exposes roster internals", async ({ page }) => {
        let submitted: Record<string, unknown> | null = null;

        await mockAuthenticatedShell(page);
        await mockStudentPlatform(page);
        await page.route("**/api/v1/student-verification/applications", async (route) => {
            expect(route.request().method()).toBe("POST");
            await route.fulfill(ok(application("created"), 201));
        });
        await page.route(
            `**/api/v1/student-verification/applications/${applicationID}/real-name/verify`,
            async (route) => {
                submitted = route.request().postDataJSON() as Record<string, unknown>;
                await route.fulfill(ok(application("approved", "real_name_identity_check")));
            },
        );

        await page.goto("/user/student-verification");
        await expect(page.getByRole("heading", { name: "学生认证" })).toBeVisible();
        await selectSchoolAndMethod(page, "real_name_identity_check");

        await page.locator("[data-verification-student-id]").fill("20990001");
        await page.locator("[data-verification-name]").fill("测试学生");
        await page.locator("[data-verification-document-number]").fill("110101200501010011");
        await page.locator("[data-verification-consent]").check();
        await page.locator("[data-verification-submit]").click();

        await expect(page.locator("[data-verification-complete]")).toBeVisible();
        expect(submitted).toEqual({
            studentID: "20990001",
            name: "测试学生",
            documentNumber: "110101200501010011",
            privacyNoticeVersion: "buaa-verification-v1",
            sensitiveDataConsent: true,
        });
        await expect(page.locator("[data-student-verification-page]")).not.toContainText("Oracle");
        await expect(page.locator("[data-student-verification-page]")).not.toContainText("SFZJH");
        await expect(page.locator("[data-student-verification-page]")).not.toContainText("腾讯云人脸核身");
    });

    test("school-email verification derives the mailbox server-side and accepts a returned OTP", async ({ page }) => {
        let identityRequest: unknown = null;
        let otpRequest: unknown = null;

        await mockAuthenticatedShell(page);
        await mockStudentPlatform(page);
        await page.route("**/api/v1/student-verification/applications", (route) =>
            route.fulfill(ok(application("created"), 201)),
        );
        await page.route(
            `**/api/v1/student-verification/applications/${applicationID}/email/outbound/otp`,
            async (route) => {
                identityRequest = route.request().postDataJSON();
                await route.fulfill(ok({
                    applicationID,
                    maskedEmail: "2337****@buaa.edu.cn",
                    expiresAt: "2026-08-05T08:10:00Z",
                    resendAvailableAt: "2026-08-05T08:01:00Z",
                    remainingAttempts: 5,
                }));
            },
        );
        await page.route(
            `**/api/v1/student-verification/applications/${applicationID}/email/outbound/verify`,
            async (route) => {
                otpRequest = route.request().postDataJSON();
                await route.fulfill(ok(application("approved", "student_email_outbound_otp")));
            },
        );

        await page.goto("/user/student-verification");
        await selectSchoolAndMethod(page, "student_email_outbound_otp");
        await page.locator("[data-verification-student-id]").fill("20990001");
        await page.locator("[data-verification-name]").fill("测试学生");
        await page.locator("[data-verification-consent]").check();
        await page.locator("[data-verification-submit]").click();

        await expect(page.getByText("2337****@buaa.edu.cn")).toBeVisible();
        await fillOTP(page, "验证码", "654321");
        await page.locator("[data-verification-submit]").click();

        await expect(page.locator("[data-verification-complete]")).toBeVisible();
        expect(identityRequest).toEqual({
            studentID: "20990001",
            name: "测试学生",
            privacyNoticeVersion: "buaa-email-v1",
            sensitiveDataConsent: true,
        });
        expect(otpRequest).toEqual({ code: "654321" });
    });

    test("a user-entered phone can complete through the school-confirmed path without an SMS step", async ({ page }) => {
        let submitted: unknown = null;
        let statusReads = 0;

        await mockAuthenticatedShell(page);
        await page.route("**/api/v1/account/phone", (route) => {
            statusReads += 1;
            return route.fulfill(ok(statusReads === 1
                ? {
                    state: "unbound",
                    maskedPhone: null,
                    method: null,
                    verifiedAt: null,
                    expiresAt: null,
                    publishingRequirementSatisfied: false,
                    revision: 1,
                }
                : {
                    state: "verified",
                    maskedPhone: "138****5678",
                    method: "school_roster_phone_match",
                    verifiedAt: now,
                    expiresAt: null,
                    publishingRequirementSatisfied: true,
                    revision: 2,
                }));
        });
        await page.route("**/api/v1/account/phone/operations", async (route) => {
            submitted = route.request().postDataJSON();
            await route.fulfill(ok({
                id: phoneOperationID,
                operationKind: "bind",
                status: "completed",
                maskedPhone: "138****5678",
                verificationStep: "none",
                smsResendAvailableAt: null,
                expiresAt: "2026-08-05T08:15:00Z",
                revision: 3,
            }, 201));
        });

        await page.goto("/user/phone-binding");
        await page.locator("[data-phone-number]").fill("13812345678");
        await page.getByRole("button", { name: "继续" }).click();

        await expect(page.getByText("138****5678")).toBeVisible();
        await expect(page.getByText("学校账号信息确认")).toBeVisible();
        expect(submitted).toEqual({ phone: "13812345678" });
        await expect(page.locator("[data-phone-sms-step]")).toHaveCount(0);
    });

    test("a phone not confirmed by school data silently falls back to SMS possession verification", async ({ page }) => {
        let smsRequests = 0;
        let verifiedCode: unknown = null;
        let statusReads = 0;

        await mockAuthenticatedShell(page);
        await page.route("**/api/v1/account/phone", (route) => {
            statusReads += 1;
            return route.fulfill(ok(statusReads === 1
                ? {
                    state: "unbound",
                    maskedPhone: null,
                    method: null,
                    verifiedAt: null,
                    expiresAt: null,
                    publishingRequirementSatisfied: false,
                    revision: 1,
                }
                : {
                    state: "verified",
                    maskedPhone: "139****2468",
                    method: "sms_possession",
                    verifiedAt: now,
                    expiresAt: null,
                    publishingRequirementSatisfied: true,
                    revision: 2,
                }));
        });
        await page.route("**/api/v1/account/phone/operations", (route) =>
            route.fulfill(ok({
                id: phoneOperationID,
                operationKind: "bind",
                status: "pending_verification",
                maskedPhone: "139****2468",
                verificationStep: "sms_otp",
                smsResendAvailableAt: null,
                expiresAt: "2026-08-05T08:15:00Z",
                revision: 1,
            }, 201)),
        );
        await page.route(`**/api/v1/account/phone/operations/${phoneOperationID}/sms`, (route) => {
            smsRequests += 1;
            return route.fulfill(ok({
                id: phoneOperationID,
                operationKind: "bind",
                status: "pending_verification",
                maskedPhone: "139****2468",
                verificationStep: "sms_otp",
                smsResendAvailableAt: "2026-08-05T08:01:00Z",
                expiresAt: "2026-08-05T08:15:00Z",
                revision: 2,
            }));
        });
        await page.route(
            `**/api/v1/account/phone/operations/${phoneOperationID}/sms/verify`,
            async (route) => {
                verifiedCode = route.request().postDataJSON();
                await route.fulfill(ok({
                    id: phoneOperationID,
                    operationKind: "bind",
                    status: "completed",
                    maskedPhone: "139****2468",
                    verificationStep: "none",
                    smsResendAvailableAt: null,
                    expiresAt: "2026-08-05T08:15:00Z",
                    revision: 3,
                }));
            },
        );

        await page.goto("/user/phone-binding");
        await page.locator("[data-phone-number]").fill("13912342468");
        await page.getByRole("button", { name: "继续" }).click();

        await expect(page.locator("[data-phone-sms-step]")).toBeVisible();
        await fillOTP(page, "短信验证码", "123456");
        await page.getByRole("button", { name: "确认并绑定" }).click();

        await expect(page.getByText("139****2468")).toBeVisible();
        await expect(page.getByText("短信验证码确认")).toBeVisible();
        expect(smsRequests).toBe(1);
        expect(verifiedCode).toEqual({ code: "123456" });
    });

    test("student verification remains usable at target widths and honors reduced motion", async ({ page }) => {
        await mockAuthenticatedShell(page);
        await mockStudentPlatform(page);
        await page.emulateMedia({ reducedMotion: "reduce" });

        const viewports = [
            { width: 375, height: 812 },
            { width: 667, height: 375 },
            { width: 768, height: 1024 },
            { width: 1024, height: 900 },
            { width: 1440, height: 900 },
        ];

        for (const viewport of viewports) {
            await page.setViewportSize(viewport);
            await page.goto("/user/student-verification");

            const root = page.locator("[data-student-verification-page]");
            const schoolOption = page.locator("[data-verification-school-option]");
            await expect(root).toBeVisible();
            await expect(schoolOption).toBeVisible();

            const layout = await page.evaluate(() => ({
                clientWidth: document.documentElement.clientWidth,
                scrollWidth: document.documentElement.scrollWidth,
            }));
            expect(layout.scrollWidth).toBeLessThanOrEqual(layout.clientWidth + 1);

            const interactionStyle = await schoolOption.evaluate((element) => {
                const style = window.getComputedStyle(element);
                return {
                    height: element.getBoundingClientRect().height,
                    transitionDuration: Number.parseFloat(style.transitionDuration),
                };
            });
            expect(interactionStyle.height).toBeGreaterThanOrEqual(44);
            expect(interactionStyle.transitionDuration).toBeLessThanOrEqual(0.001);
        }
    });
});
