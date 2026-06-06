import { expect, mockNotificationStream, test, type Page } from "./fixtures";

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

const now = "2026-05-24T04:00:00Z";

const verifiedIdentity = {
    userID: 12,
    docType: "MAINLAND_ID",
    realName: "张三",
    verified: true,
    verifyMethod: "manual",
    reviewedAt: now,
    verifiedAt: now,
    rejectionReason: null,
    createdAt: now,
    updatedAt: now,
};

const rejectedIdentity = {
    ...verifiedIdentity,
    verified: false,
    reviewedAt: now,
    verifiedAt: null,
    rejectionReason: "证件号码与姓名不匹配",
};

const unverifiedProfile = {
    userID: 12,
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

const verifiedProfile = {
    ...unverifiedProfile,
    schoolID: 4111010001,
    studentIDs: ["20260001"],
    activeStudentID: "20260001",
    verificationStatus: "verified",
    verificationMethod: "ldap",
    verifiedAt: now,
};

const rejectedProfile = {
    ...unverifiedProfile,
    schoolID: 4111010001,
    studentIDs: [],
    activeStudentID: null,
    verificationStatus: "rejected",
    verificationMethod: "ldap",
    rejectionReason: "统一身份认证失败",
    reviewedAt: now,
};

const schools = [
    {
        schoolID: 4111010001,
        schoolCode: "4111010001",
        schoolName: "测试大学",
        verificationMethod: "ldap",
        consentText: "仅用于校内身份认证",
        manualFormFields: null,
        enabled: true,
        schoolSsoEnabled: false,
        schoolEmailOtpEnabled: false,
    },
    {
        schoolID: 4111010002,
        schoolCode: "4111010002",
        schoolName: "人工审核大学",
        verificationMethod: "manual",
        consentText: "请确认人工审核材料真实有效",
        manualFormFields: [
            {
                key: "studentId",
                label: "学号",
                type: "text",
                required: true,
                placeholder: "请输入学号",
            },
            {
                key: "college",
                label: "学院",
                type: "select",
                required: true,
                options: ["计算机学院", "材料学院"],
                placeholder: "请选择学院",
            },
            {
                key: "note",
                label: "补充说明",
                type: "textarea",
                required: false,
                placeholder: "补充说明",
            },
            {
                key: "enrolledAt",
                label: "入学日期",
                type: "date",
                required: false,
            },
        ],
        enabled: true,
        schoolSsoEnabled: false,
        schoolEmailOtpEnabled: false,
    },
    {
        schoolID: 4111010006,
        schoolCode: "4111010006",
        schoolName: "北京航空航天大学",
        verificationMethod: "manual",
        consentText: "仅用于北航学生身份认证",
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
];

type UserApiState = {
    identity: null | Record<string, unknown>;
    profile: Record<string, unknown>;
    qqBinding?: null | Record<string, unknown>;
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

function notFound(message = "not found") {
    return json(
        {
            success: false,
            error: { code: "A0040404", message },
        },
        404,
    );
}

async function gotoAuthenticatedPage(page: Page, path: string) {
    const unreadCountLoaded = page.waitForResponse((response) => {
        const url = new URL(response.url());
        return (
            response.request().method() === "GET" &&
            url.pathname ===
                "/api/v1/course/review/user/notifications/unread-count" &&
            response.status() === 200
        );
    });

    await page.goto(path);
    await unreadCountLoaded;
}

async function mockUserApi(page: Page, state: UserApiState) {
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
    await page.route(
        "**/api/v1/course/review/user/notifications/unread-count*",
        (route) => route.fulfill(ok({ count: 0 })),
    );
    await mockNotificationStream(page);
    await page.route("**/api/v1/course/review/user/reviews*", (route) =>
        route.fulfill(ok({ list: [], total: 0, page: 1, pageSize: 10 })),
    );
    await page.route("**/api/v1/course/review/user/votes*", (route) =>
        route.fulfill(ok({ list: [], total: 0, page: 1, pageSize: 10 })),
    );
    await page.route("**/api/v1/course/review/user/favorites*", (route) =>
        route.fulfill(ok({ list: [], total: 0, page: 1, pageSize: 10 })),
    );
    await page.route("**/api/v1/user/identity", async (route) => {
        if (route.request().method() === "POST") {
            const body = route.request().postDataJSON();
            state.identity = {
                userID: 12,
                docType: body.docType,
                realName: body.realName,
                verified: false,
                verifyMethod: "manual",
                reviewedAt: null,
                verifiedAt: null,
                rejectionReason: null,
                createdAt: now,
                updatedAt: now,
            };
            await route.fulfill(ok(state.identity));
            return;
        }
        await route.fulfill(
            state.identity === null
                ? notFound("identity not found")
                : ok(state.identity),
        );
    });
    await page.route("**/api/v1/user/profile", (route) =>
        route.fulfill(ok(state.profile)),
    );
    await page.route("**/api/v1/user/profile/verify", async (route) => {
        const body = route.request().postDataJSON();
        const school = schools.find(
            (item) => item.schoolCode === body.schoolCode,
        );
        state.profile = {
            ...verifiedProfile,
            schoolID: school?.schoolID ?? null,
        };
        await route.fulfill(ok(state.profile));
    });
    await page.route("**/api/v1/user/profile/bind-phone/otp", (route) =>
        route.fulfill(ok()),
    );
    await page.route("**/api/v1/user/profile/bind-phone", async (route) => {
        const body = route.request().postDataJSON();
        const phone = String(body.phone ?? "");
        state.profile = {
            ...state.profile,
            phone:
                phone.length === 11
                    ? `${phone.slice(0, 3)}****${phone.slice(7)}`
                    : phone,
            phoneVerified: true,
            updatedAt: now,
        };
        await route.fulfill(ok({ phone: body.phone }));
    });
    await page.route("**/api/v1/user/qq-binding", (route) =>
        route.fulfill(
            state.qqBinding
                ? ok(state.qqBinding)
                : notFound("qq binding not found"),
        ),
    );
    await page.route("**/api/v1/user/qq-binding/code", (route) =>
        route.fulfill(
            ok({
                code: "QQ-CODE-1",
                expiresAt: "2026-05-24T05:00:00Z",
            }),
        ),
    );
    await page.route("**/api/v1/user/schools", (route) =>
        route.fulfill(ok(schools)),
    );
    await page.route("**/api/v1/user/profile/academic-info", (route) =>
        route.fulfill(
            ok({
                xh: "20260001",
                xm: "张三",
                yxdm: "计算机学院",
                zydm: "软件工程",
                bjdm: "软件 2601",
                xznj: "2026",
                rxnj: "2026",
                pyccdm: "本科",
                sjh: "138****5678",
                dzxx: "zhangsan@example.com",
            }),
        ),
    );
}

test.describe("User verification flows", () => {
    test("rejected identity and student verification pages allow resubmission", async ({
        page,
    }) => {
        const state: UserApiState = {
            identity: { ...rejectedIdentity },
            profile: { ...rejectedProfile },
            qqBinding: null,
        };

        await mockUserApi(page, state);

        await gotoAuthenticatedPage(page, "/user/identity-verification");
        await expect(
            page.getByRole("heading", { name: "已拒绝" }),
        ).toBeVisible();
        await expect(page.getByText("证件号码与姓名不匹配")).toBeVisible();

        await page.getByRole("button", { name: "重新提交" }).click();
        await expect(page.getByLabel("真实姓名")).toBeVisible();
        await expect(page.getByLabel("证件号码")).toBeVisible();
        await expect(
            page.getByRole("button", { name: "提交认证" }),
        ).toBeDisabled();

        state.identity = { ...verifiedIdentity };
        await gotoAuthenticatedPage(page, "/user/student-verification");
        await expect(
            page.getByRole("heading", { name: "已拒绝" }),
        ).toBeVisible();
        await expect(
            page.getByText("验证失败，请检查学号和密码"),
        ).toBeVisible();

        await page.getByRole("button", { name: "重新提交" }).click();
        await expect(page.locator("#student-school")).toBeVisible();
        await page.locator("#student-school").selectOption("4111010001");
        await expect(page.getByRole("button", { name: "验证" })).toBeDisabled();
    });

    test("verified and bound account detail pages render persisted status", async ({
        page,
    }) => {
        const state: UserApiState = {
            identity: { ...verifiedIdentity },
            profile: {
                ...verifiedProfile,
                phone: "138****5678",
                phoneVerified: true,
            },
            qqBinding: {
                userID: 12,
                qqID: "123456",
                boundAt: now,
                createdAt: now,
                updatedAt: now,
            },
        };

        await mockUserApi(page, state);

        await gotoAuthenticatedPage(page, "/user/identity-verification");
        await expect(
            page.getByRole("heading", { name: "实名认证" }),
        ).toBeVisible();
        await expect(
            page.getByRole("heading", { name: "已认证" }),
        ).toBeVisible();
        await expect(page.getByText("大陆居民身份证")).toBeVisible();
        await expect(page.getByText("张三")).toBeVisible();

        await gotoAuthenticatedPage(page, "/user/student-verification");
        await expect(
            page.getByRole("heading", { name: "学生认证" }),
        ).toBeVisible();
        await expect(
            page.getByRole("heading", { name: "已认证" }),
        ).toBeVisible();
        await expect(page.getByText("测试大学")).toBeVisible();

        await gotoAuthenticatedPage(page, "/user/phone-binding");
        await expect(
            page.getByRole("heading", { name: "绑定手机" }),
        ).toBeVisible();
        await expect(
            page.getByRole("heading", { name: "已绑定" }),
        ).toBeVisible();
        await expect(page.getByText("138****5678")).toBeVisible();

        await gotoAuthenticatedPage(page, "/user/qq-binding");
        await expect(
            page.getByRole("heading", { name: "绑定 QQ" }),
        ).toBeVisible();
        await expect(
            page.getByRole("heading", { name: "已绑定" }),
        ).toBeVisible();
        await expect(page.getByText("123456")).toBeVisible();

        await gotoAuthenticatedPage(page, "/user/academic-info");
        await expect(page.getByText("20260001")).toBeVisible();
        await expect(page.getByText("张三")).toBeVisible();
        await expect(page.getByText("计算机学院")).toBeVisible();
        await expect(page.getByText("软件工程")).toBeVisible();
    });

    test("user submits identity verification and completes LDAP student verification", async ({
        page,
    }) => {
        const state: UserApiState = {
            identity: null,
            profile: { ...unverifiedProfile },
            qqBinding: null,
        };
        let identityBody: unknown = null;
        let studentBody: unknown = null;

        await mockUserApi(page, state);
        await page.route("**/api/v1/user/identity", async (route) => {
            if (route.request().method() === "POST") {
                identityBody = route.request().postDataJSON();
            }
            await route.fallback();
        });
        await page.route("**/api/v1/user/profile/verify", async (route) => {
            studentBody = route.request().postDataJSON();
            await route.fallback();
        });

        await gotoAuthenticatedPage(page, "/user/identity-verification");

        await page.getByLabel("真实姓名").fill("张三");
        await page.getByLabel("证件号码").fill("110101200001010010");
        await page.getByRole("button", { name: "提交认证" }).click();

        await expect(
            page.getByRole("heading", { name: "审核中" }),
        ).toBeVisible();
        await expect(page.getByText("张三")).toBeVisible();
        expect(identityBody).toMatchObject({
            docType: "MAINLAND_ID",
            realName: "张三",
            docNumber: "110101200001010010",
        });

        state.identity = { ...verifiedIdentity };
        await gotoAuthenticatedPage(page, "/user/student-verification");

        await page.locator("#student-school").selectOption("4111010001");
        await page.getByLabel("学号").fill("20260001");
        await page.getByLabel("统一身份认证密码").fill("secret-pass");
        await page.getByRole("checkbox").check();
        await page.getByRole("button", { name: "验证" }).click();

        await expect(
            page.getByRole("heading", { name: "已认证" }),
        ).toBeVisible();
        await expect(page.getByText("测试大学")).toBeVisible();
        expect(studentBody).toMatchObject({
            schoolCode: "4111010001",
            studentID: "20260001",
            password: "secret-pass",
            consent: true,
        });
    });

    test("BUAA student verification derives and locks the student email by school code", async ({
        page,
    }) => {
        const state: UserApiState = {
            identity: { ...verifiedIdentity },
            profile: { ...unverifiedProfile },
            qqBinding: null,
        };
        let academicMatchBody: unknown = null;
        let otpRequestBody: unknown = null;
        let otpVerifyBody: unknown = null;

        await mockUserApi(page, state);
        await page.route(
            "**/api/v1/user/profile/school-email/academic-match",
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
            "**/api/v1/user/profile/school-email/request-otp",
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
            "**/api/v1/user/profile/school-email/verify-otp",
            async (route) => {
                otpVerifyBody = route.request().postDataJSON();
                state.profile = {
                    ...verifiedProfile,
                    schoolID: 4111010006,
                    studentIDs: ["20250001"],
                    activeStudentID: "20250001",
                    verificationMethod: "school_email_otp",
                };
                await route.fulfill(ok(state.profile));
            },
        );

        await gotoAuthenticatedPage(page, "/user/student-verification");

        await page.locator("#student-school").selectOption("4111010006");
        const emailInput = page.locator("[data-student-school-email-input]");
        await expect(emailInput).toHaveJSProperty("readOnly", true);
        await expect(emailInput).toHaveValue("");
        await page.locator("[data-student-id-input]").fill("20250001");
        await page.locator("[data-student-name-input]").fill("张三");
        await page.locator("[data-student-email-otp-request]").click();
        await expect(emailInput).toHaveValue("20250001@buaa.edu.cn");
        await page.locator("[data-student-email-code-input]").fill("123456");
        await page.locator("[data-student-consent-checkbox]").check();
        await page.locator("[data-student-verification-submit]").click();

        await expect(
            page.getByRole("heading", { name: "已认证" }),
        ).toBeVisible();
        await expect(
            page.getByText("北京航空航天大学（4111010006）"),
        ).toBeVisible();
        expect(academicMatchBody).toEqual({
            schoolCode: "4111010006",
            studentID: "20250001",
            studentName: "张三",
        });
        expect(otpRequestBody).toEqual({
            schoolCode: "4111010006",
            studentID: "20250001",
            studentName: "张三",
        });
        expect(otpVerifyBody).toEqual({
            schoolCode: "4111010006",
            email: "20250001@buaa.edu.cn",
            code: "123456",
            consent: true,
        });
    });

    test("user submits passport identity with uploaded document photos", async ({
        page,
    }) => {
        const state: UserApiState = {
            identity: null,
            profile: { ...unverifiedProfile },
            qqBinding: null,
        };
        const uploads: unknown[] = [];
        let identityBody: unknown = null;

        await mockUserApi(page, state);
        await page.route("**/api/v1/user/identity/uploads", async (route) => {
            const body = route.request().postDataJSON();
            uploads.push(body);
            await route.fulfill(
                ok({ key: `identity/${body.slot}-${body.filename}` }),
            );
        });
        await page.route("**/api/v1/user/identity", async (route) => {
            if (route.request().method() === "POST") {
                identityBody = route.request().postDataJSON();
            }
            await route.fallback();
        });

        await gotoAuthenticatedPage(page, "/user/identity-verification");

        await page.getByText("护照").click();
        await page.getByLabel("真实姓名").fill("Alice Passport");
        await page.getByLabel("证件号码").fill("P12345678");

        const png = {
            name: "passport.png",
            mimeType: "image/png",
            buffer: Buffer.from(
                "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMB/az+0XcAAAAASUVORK5CYII=",
                "base64",
            ),
        };

        const imagePreviews = page.locator("img");
        await page.locator('input[type="file"]').nth(0).setInputFiles(png);
        await expect(imagePreviews).toHaveCount(1);
        await page
            .locator('input[type="file"]')
            .nth(0)
            .setInputFiles({
                ...png,
                name: "passport-back.png",
            });
        await expect(imagePreviews).toHaveCount(2);
        await page
            .locator('input[type="file"]')
            .nth(0)
            .setInputFiles({
                ...png,
                name: "passport-selfie.png",
            });
        await expect(imagePreviews).toHaveCount(3);

        await page.getByRole("button", { name: "提交认证" }).click();

        await expect(
            page.getByRole("heading", { name: "审核中" }),
        ).toBeVisible();
        expect(uploads).toEqual([
            expect.objectContaining({
                slot: "front",
                filename: "passport.png",
                contentType: "image/png",
            }),
            expect.objectContaining({
                slot: "back",
                filename: "passport-back.png",
                contentType: "image/png",
            }),
            expect.objectContaining({
                slot: "selfie",
                filename: "passport-selfie.png",
                contentType: "image/png",
            }),
        ]);
        expect(identityBody).toMatchObject({
            docType: "PASSPORT",
            realName: "Alice Passport",
            docNumber: "P12345678",
            docPhotoFront: "identity/front-passport.png",
            docPhotoBack: "identity/back-passport-back.png",
            docPhotoSelfie: "identity/selfie-passport-selfie.png",
        });
    });

    test("user submits manual student verification dynamic fields", async ({
        page,
    }) => {
        const state: UserApiState = {
            identity: null,
            profile: { ...unverifiedProfile },
            qqBinding: null,
        };
        let manualBody: unknown = null;

        await mockUserApi(page, state);
        await page.route("**/api/v1/user/profile/verify", async (route) => {
            const body = route.request().postDataJSON();
            manualBody = body;
            state.profile = {
                ...unverifiedProfile,
                schoolID: 4111010001,
                verificationStatus: "pending",
                verificationMethod: "manual",
                consentGivenAt: now,
                manualFormData: body.manualFormData,
                updatedAt: now,
            };
            await route.fulfill(ok(state.profile));
        });

        await gotoAuthenticatedPage(page, "/user/student-verification");

        await page.locator("#student-school").selectOption("4111010002");
        await expect(page.locator("#manual-studentId")).toBeVisible();
        await expect(page.getByRole("button", { name: "验证" })).toBeDisabled();

        await page.locator("#manual-studentId").fill("M20260002");
        await page.locator("#manual-college").selectOption("计算机学院");
        await page.locator("#manual-note").fill("人工审核材料已上传到学校系统");
        await page.locator("#manual-enrolledAt").fill("2026-09-01");
        await page.getByRole("checkbox").check();
        await page.getByRole("button", { name: "验证" }).click();

        await expect(
            page.getByRole("heading", { name: "审核中" }),
        ).toBeVisible();
        expect(manualBody).toMatchObject({
            schoolCode: "4111010002",
            manualFormData: {
                studentId: "M20260002",
                college: "计算机学院",
                note: "人工审核材料已上传到学校系统",
                enrolledAt: "2026-09-01",
            },
            consent: true,
        });
    });

    test("user updates SSO-synced phone, creates QQ binding code, and views academic info", async ({
        page,
    }) => {
        const state: UserApiState = {
            identity: { ...verifiedIdentity },
            profile: {
                ...verifiedProfile,
                phone: "138****5678",
                phoneVerified: true,
            },
            qqBinding: null,
        };
        let otpBody: unknown = null;
        let phoneBody: unknown = null;

        await mockUserApi(page, state);
        await page.route(
            "**/api/v1/user/profile/bind-phone/otp",
            async (route) => {
                otpBody = route.request().postDataJSON();
                await route.fallback();
            },
        );
        await page.route("**/api/v1/user/profile/bind-phone", async (route) => {
            phoneBody = route.request().postDataJSON();
            await route.fallback();
        });

        await gotoAuthenticatedPage(page, "/user/phone-binding");

        await expect(
            page.getByRole("heading", { name: "已绑定" }),
        ).toBeVisible();
        await expect(page.getByText("138****5678")).toBeVisible();
        await expect(page.getByText("同步写入统一身份认证账号")).toBeVisible();
        await page.getByLabel("手机号码").fill("13912345678");
        await page.getByRole("button", { name: "发送验证码" }).click();
        await page.getByLabel("验证码").fill("123456");
        await page.getByRole("button", { name: "更新手机号" }).click();

        await expect(page).toHaveURL(/\/user\/phone-binding/);
        await expect(page.getByText("139****5678")).toBeVisible();
        expect(otpBody).toEqual({ phone: "13912345678" });
        expect(phoneBody).toEqual({
            phone: "13912345678",
            otpCode: "123456",
        });

        await page.goto("/user/qq-binding");
        await expect(
            page.getByRole("heading", { name: "绑定 QQ" }),
        ).toBeVisible();
        await page.getByRole("button", { name: "生成绑定码" }).click();
        await expect(
            page.getByText("请私聊机器人并发送下面这条命令"),
        ).toBeVisible();
        await expect(page.getByText("绑定 QQ-CODE-1")).toBeVisible();

        await page.goto("/user/academic-info");
        await expect(page.getByText("20260001")).toBeVisible();
        await expect(page.getByText("张三")).toBeVisible();
    });

    test("invalid QQ binding code response fails closed", async ({ page }) => {
        const state: UserApiState = {
            identity: { ...verifiedIdentity },
            profile: { ...verifiedProfile },
            qqBinding: null,
        };

        await mockUserApi(page, state);
        await page.unroute("**/api/v1/user/qq-binding/code");
        await page.route("**/api/v1/user/qq-binding/code", (route) =>
            route.fulfill(ok(null)),
        );

        await gotoAuthenticatedPage(page, "/user/qq-binding");
        await page.getByRole("button", { name: "生成绑定码" }).click();

        await expect(
            page.getByRole("alert").filter({ hasText: "操作失败，请重试" }),
        ).toBeVisible();
        await expect(
            page.getByText("请私聊机器人并发送下面这条命令"),
        ).toHaveCount(0);
        await expect(page.getByText("绑定 QQ-CODE-1")).toHaveCount(0);
    });

    test("academic info page renders verification-required and empty states", async ({
        page,
    }) => {
        const state: UserApiState = {
            identity: { ...verifiedIdentity },
            profile: { ...unverifiedProfile },
            qqBinding: null,
        };

        await mockUserApi(page, state);
        await page.unroute("**/api/v1/user/profile/academic-info");
        await page.route("**/api/v1/user/profile/academic-info", (route) =>
            route.fulfill(
                json(
                    {
                        success: false,
                        error: {
                            code: "A0040300",
                            message: "student verification required",
                        },
                    },
                    403,
                ),
            ),
        );

        await gotoAuthenticatedPage(page, "/user/academic-info");
        await expect(
            page.getByText("请先完成学生认证后查看学业信息"),
        ).toBeVisible();
        await expect(
            page.getByRole("link", { name: "去认证" }),
        ).toHaveAttribute("href", "/user/student-verification");

        await page.unroute("**/api/v1/user/profile/academic-info");
        await page.route("**/api/v1/user/profile/academic-info", (route) =>
            route.fulfill(
                json(
                    {
                        success: false,
                        error: {
                            code: "A0040404",
                            message: "academic info not found",
                        },
                    },
                    404,
                ),
            ),
        );

        await gotoAuthenticatedPage(page, "/user/academic-info");
        await expect(page.getByText("暂无学籍数据")).toBeVisible();

        await page.unroute("**/api/v1/user/profile/academic-info");
        let malformedAcademicInfo = true;
        await page.route("**/api/v1/user/profile/academic-info", (route) =>
            route.fulfill(
                ok(
                    malformedAcademicInfo
                        ? {
                              xh: "20260001",
                              xm: "张三",
                              sjh: 13812345678,
                          }
                        : {
                              xh: "20260001",
                              xm: "张三",
                              yxdm: "计算机学院",
                          },
                ),
            ),
        );

        await gotoAuthenticatedPage(page, "/user/academic-info");
        await expect(page.getByText("加载失败")).toBeVisible();
        await expect(page.getByRole("button", { name: "重试" })).toBeVisible();
        await expect(page.getByText("暂无学籍数据")).toHaveCount(0);

        malformedAcademicInfo = false;
        await page.getByRole("button", { name: "重试" }).click();
        await expect(page.getByText("20260001")).toBeVisible();
        await expect(page.getByText("计算机学院")).toBeVisible();
    });
});
