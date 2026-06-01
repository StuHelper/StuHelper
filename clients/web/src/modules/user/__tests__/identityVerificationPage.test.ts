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

const { default: IdentityVerificationPage } =
    await import("../views/IdentityVerificationPage.vue");

const now = "2026-05-31T00:00:00Z";
let pinia: Pinia;

function notFound() {
    return Promise.reject({ status: 404 });
}

function submittedIdentity(docNumber: string) {
    return {
        userID: 42,
        docType: "MAINLAND_ID",
        realName: "张三",
        verified: false,
        verifyMethod: null,
        reviewedAt: null,
        verifiedAt: null,
        rejectionReason: null,
        createdAt: now,
        updatedAt: now,
        docNumber,
    };
}

function mountPage() {
    return mount(IdentityVerificationPage, {
        global: {
            plugins: [pinia],
        },
    });
}

describe("IdentityVerificationPage", () => {
    beforeEach(() => {
        pinia = createPinia();
        setActivePinia(pinia);
        vi.clearAllMocks();

        mockIdentityApi.getIdentity.mockImplementation(notFound);
        mockIdentityApi.getProfile.mockImplementation(notFound);
        mockIdentityApi.getQQBinding.mockImplementation(notFound);
    });

    it("blocks invalid mainland ID numbers before calling the backend", async () => {
        const wrapper = mountPage();
        await flushPromises();

        await wrapper.find("[data-identity-real-name-input]").setValue("张三");
        await wrapper
            .find("[data-identity-doc-number-input]")
            .setValue("not-an-id-card");
        await flushPromises();

        expect(wrapper.find("[data-identity-doc-number-error]").text()).toBe(
            "user.verification.identity.invalidMainlandID",
        );

        const submitButton = wrapper.find<HTMLButtonElement>(
            "[data-identity-submit]",
        );
        expect(submitButton.element.disabled).toBe(true);

        await submitButton.trigger("click");

        expect(mockIdentityApi.submitIdentity).not.toHaveBeenCalled();
        expect(mockToastError).not.toHaveBeenCalled();
    });

    it("normalizes valid mainland ID numbers before submitting", async () => {
        mockIdentityApi.submitIdentity.mockResolvedValue({
            data: {
                data: submittedIdentity("11010519491231002X"),
            },
        });

        const wrapper = mountPage();
        await flushPromises();

        await wrapper
            .find("[data-identity-real-name-input]")
            .setValue(" 张三 ");
        await wrapper
            .find("[data-identity-doc-number-input]")
            .setValue(" 11010519491231002x ");
        await flushPromises();

        const submitButton = wrapper.find<HTMLButtonElement>(
            "[data-identity-submit]",
        );
        expect(submitButton.element.disabled).toBe(false);

        await submitButton.trigger("click");
        await flushPromises();

        expect(mockIdentityApi.submitIdentity).toHaveBeenCalledWith({
            docType: "MAINLAND_ID",
            realName: "张三",
            docNumber: "11010519491231002X",
        });
    });
});
