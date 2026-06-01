// @vitest-environment jsdom

import { flushPromises, mount } from "@vue/test-utils";
import { createPinia, setActivePinia } from "pinia";
import type { Pinia } from "pinia";
import { beforeEach, describe, expect, it, vi } from "vitest";

const mockRouterPush = vi.fn();
const mockToastSuccess = vi.fn();
const mockToastError = vi.fn();

const mockIdentityApi = vi.hoisted(() => ({
    getIdentity: vi.fn(),
    getProfile: vi.fn(),
    getQQBinding: vi.fn(),
    listSchools: vi.fn(),
    requestStudentEmailOTP: vi.fn(),
    verifyStudentEmailOTP: vi.fn(),
    verifyStudent: vi.fn(),
    submitIdentity: vi.fn(),
    uploadIdentityPhoto: vi.fn(),
    createQQBindingCode: vi.fn(),
    bindPhone: vi.fn(),
    requestBindPhoneOTP: vi.fn(),
}));

vi.mock("@/api", () => ({
    api: {
        identity: mockIdentityApi,
    },
}));

vi.mock("vue-router", () => ({
    useRouter: () => ({
        push: mockRouterPush,
    }),
}));

vi.mock("vue-i18n", () => ({
    createI18n: () => ({
        global: {
            t: (key: string) => key,
            te: () => false,
        },
        install: vi.fn(),
    }),
    useI18n: () => ({
        t: (key: string) => key,
    }),
}));

vi.mock("@/composables/useToast", () => ({
    useToast: () => ({
        success: mockToastSuccess,
        error: mockToastError,
    }),
}));

const { default: StudentVerificationPage } = await import(
    "../views/StudentVerificationPage.vue"
);

const now = "2026-05-31T00:00:00Z";
let pinia: Pinia;

function notFound() {
    return Promise.reject({ status: 404 });
}

function verifiedProfile() {
    return {
        userID: 42,
        schoolID: 10006,
        studentIDs: ["20250001"],
        activeStudentID: "20250001",
        verificationStatus: "verified",
        verificationMethod: "school_email_otp",
        rejectionReason: null,
        reviewedAt: now,
        phone: null,
        phoneVerified: false,
        consentGivenAt: now,
        verifiedAt: now,
        createdAt: now,
        updatedAt: now,
    };
}

describe("StudentVerificationPage", () => {
    beforeEach(() => {
        pinia = createPinia();
        setActivePinia(pinia);
        vi.clearAllMocks();

        mockIdentityApi.getIdentity.mockImplementation(notFound);
        mockIdentityApi.getProfile.mockImplementation(notFound);
        mockIdentityApi.getQQBinding.mockImplementation(notFound);
        mockIdentityApi.listSchools.mockResolvedValue({
            data: {
                data: [
                    {
                        schoolID: 10006,
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
                    },
                ],
            },
        });
    });

    it("uses schoolCode and locks the BUAA student email after academic name match", async () => {
        mockIdentityApi.requestStudentEmailOTP.mockResolvedValue({
            data: {
                data: {
                    email: "20250001@buaa.edu.cn",
                    studentID: "20250001",
                    cooldownSeconds: 60,
                },
            },
        });
        mockIdentityApi.verifyStudentEmailOTP.mockResolvedValue({
            data: {
                data: verifiedProfile(),
            },
        });

        const wrapper = mount(StudentVerificationPage, {
            global: {
                plugins: [pinia],
            },
        });
        await flushPromises();

        expect(
            wrapper.find<HTMLOptionElement>(
                "[data-student-school-select] option",
            ).element.value,
        ).toBe("");

        await wrapper
            .find("[data-student-school-select]")
            .setValue("4111010006");
        await flushPromises();

        const emailInput = wrapper.find<HTMLInputElement>(
            "[data-student-school-email-input]",
        );
        expect(emailInput.element.readOnly).toBe(true);
        expect(emailInput.element.value).toBe("");

        await wrapper.find("[data-student-id-input]").setValue("20250001");
        await wrapper.find("[data-student-name-input]").setValue("张三");
        await wrapper.find("[data-student-email-otp-request]").trigger("click");
        await flushPromises();

        expect(mockIdentityApi.requestStudentEmailOTP).toHaveBeenCalledWith({
            schoolCode: "4111010006",
            studentID: "20250001",
            studentName: "张三",
        });
        expect(emailInput.element.value).toBe("20250001@buaa.edu.cn");

        await wrapper.find("[data-student-email-code-input]").setValue("123456");
        await wrapper.find("[data-student-consent-checkbox]").setValue(true);
        await wrapper.find("[data-student-verification-submit]").trigger("click");
        await flushPromises();

        expect(mockIdentityApi.verifyStudentEmailOTP).toHaveBeenCalledWith({
            schoolCode: "4111010006",
            email: "20250001@buaa.edu.cn",
            code: "123456",
            consent: true,
        });
        expect(mockIdentityApi.verifyStudent).not.toHaveBeenCalled();
    });

    it("clears the derived BUAA email and code when academic identity changes", async () => {
        mockIdentityApi.requestStudentEmailOTP.mockResolvedValue({
            data: {
                data: {
                    email: "20250001@buaa.edu.cn",
                    studentID: "20250001",
                    cooldownSeconds: 60,
                },
            },
        });

        const wrapper = mount(StudentVerificationPage, {
            global: {
                plugins: [pinia],
            },
        });
        await flushPromises();

        await wrapper
            .find("[data-student-school-select]")
            .setValue("4111010006");
        await wrapper.find("[data-student-id-input]").setValue("20250001");
        await wrapper.find("[data-student-name-input]").setValue("张三");
        await wrapper.find("[data-student-email-otp-request]").trigger("click");
        await flushPromises();

        const emailInput = wrapper.find<HTMLInputElement>(
            "[data-student-school-email-input]",
        );
        const codeInput = wrapper.find<HTMLInputElement>(
            "[data-student-email-code-input]",
        );
        const submitButton = wrapper.find<HTMLButtonElement>(
            "[data-student-verification-submit]",
        );
        expect(emailInput.element.value).toBe("20250001@buaa.edu.cn");

        await codeInput.setValue("123456");
        await wrapper.find("[data-student-consent-checkbox]").setValue(true);
        expect(submitButton.element.disabled).toBe(false);

        await wrapper.find("[data-student-id-input]").setValue("20250002");
        await flushPromises();

        expect(emailInput.element.value).toBe("");
        expect(codeInput.element.value).toBe("");
        expect(submitButton.element.disabled).toBe(true);
        expect(mockIdentityApi.verifyStudentEmailOTP).not.toHaveBeenCalled();
    });
});
