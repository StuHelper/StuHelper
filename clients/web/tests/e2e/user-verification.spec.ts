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
    schoolID: 1001,
    studentIDs: ["20260001"],
    activeStudentID: "20260001",
    verificationStatus: "verified",
    verificationMethod: "ldap",
    verifiedAt: now,
};

const schools = [
    {
        schoolID: 1001,
        schoolName: "测试大学",
        verificationMethod: "ldap",
        consentText: "仅用于校内身份认证",
        manualFormFields: null,
        enabled: true,
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
        state.profile = {
            ...verifiedProfile,
            schoolID: route.request().postDataJSON().schoolID,
        };
        await route.fulfill(ok(state.profile));
    });
    await page.route("**/api/v1/user/profile/bind-phone/otp", (route) =>
        route.fulfill(ok()),
    );
    await page.route("**/api/v1/user/profile/bind-phone", async (route) => {
        const body = route.request().postDataJSON();
        state.profile = {
            ...state.profile,
            phone: "138****5678",
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
                qqID: "123456",
                qqNickname: "航小伴",
                boundAt: now,
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
        await expect(page.getByText("航小伴")).toBeVisible();

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

        await page.locator("#student-school").selectOption("1001");
        await page.getByLabel("学号").fill("20260001");
        await page.getByLabel("统一身份认证密码").fill("secret-pass");
        await page.getByRole("checkbox").check();
        await page.getByRole("button", { name: "验证" }).click();

        await expect(
            page.getByRole("heading", { name: "已认证" }),
        ).toBeVisible();
        await expect(page.getByText("测试大学")).toBeVisible();
        expect(studentBody).toMatchObject({
            schoolID: 1001,
            studentID: "20260001",
            password: "secret-pass",
            consent: true,
        });
    });

    test("user binds phone, creates QQ binding code, and views academic info", async ({
        page,
    }) => {
        const state: UserApiState = {
            identity: { ...verifiedIdentity },
            profile: { ...verifiedProfile, phone: null, phoneVerified: false },
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

        await page.getByLabel("手机号码").fill("13812345678");
        await page.getByRole("button", { name: "发送验证码" }).click();
        await page.getByLabel("验证码").fill("123456");
        await page.getByRole("button", { name: "绑定" }).click();

        await expect(page).toHaveURL(/\/user\/reviews/);
        await page.waitForLoadState("networkidle");
        expect(otpBody).toEqual({ phone: "13812345678" });
        expect(phoneBody).toEqual({
            phone: "13812345678",
            otpCode: "123456",
        });

        await gotoAuthenticatedPage(page, "/user/qq-binding");
        await page.getByRole("button", { name: "生成绑定码" }).click();
        await expect(
            page.getByText("请私聊机器人并发送下面这条命令"),
        ).toBeVisible();
        await expect(page.getByText("绑定 QQ-CODE-1")).toBeVisible();

        await gotoAuthenticatedPage(page, "/user/academic-info");
        await expect(page.getByText("20260001")).toBeVisible();
        await expect(page.getByText("张三")).toBeVisible();
        await expect(page.getByText("计算机学院")).toBeVisible();
        await expect(page.getByText("软件工程")).toBeVisible();
    });
});
