// @vitest-environment jsdom

import { flushPromises, mount } from "@vue/test-utils";
import { createPinia, setActivePinia } from "pinia";
import type { Pinia } from "pinia";
import { beforeEach, describe, expect, it, vi } from "vitest";

const mockRouterPush = vi.fn();
const mockToastSuccess = vi.fn();
const mockToastError = vi.fn();
const mockRoute = vi.hoisted(() => ({
    query: {} as Record<string, string>,
}));

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
    useRoute: () => mockRoute,
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

vi.mock("@/api/errors", async (importOriginal) => {
    const actual = await importOriginal<typeof import("@/api/errors")>();
    return {
        ...actual,
        isApiError: (error: unknown) =>
            error instanceof Error &&
            typeof (error as Error & { code?: unknown }).code === "string",
    };
});

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

function verifiedIdentity(docNumber: string) {
    return {
        ...submittedIdentity(docNumber),
        verified: true,
        verifyMethod: "academic_db_match",
        verifiedAt: now,
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
        mockRoute.query = {};

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
        expect(submitButton.attributes("type")).toBe("submit");

        await wrapper
            .find("[data-identity-verification-form]")
            .trigger("submit");

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

        await wrapper
            .find("[data-identity-verification-form]")
            .trigger("submit");
        await flushPromises();

        expect(mockIdentityApi.submitIdentity).toHaveBeenCalledWith({
            docType: "MAINLAND_ID",
            realName: "张三",
            docNumber: "11010519491231002X",
        });
    });

    it("requests manual evidence when mainland academic matching is unavailable", async () => {
        const evidenceRequiredError = Object.assign(
            new Error("manual evidence required"),
            { code: "A0030005" },
        );
        mockIdentityApi.submitIdentity.mockRejectedValue(
            evidenceRequiredError,
        );

        const wrapper = mountPage();
        await flushPromises();

        await wrapper.find("[data-identity-real-name-input]").setValue("张三");
        await wrapper
            .find("[data-identity-doc-number-input]")
            .setValue("11010519491231002X");
        await wrapper
            .find("[data-identity-verification-form]")
            .trigger("submit");
        await flushPromises();

        const evidencePrompt = wrapper.find(
            "[data-identity-manual-evidence-required]",
        );
        expect(evidencePrompt.exists()).toBe(true);
        expect(evidencePrompt.attributes("role")).toBe("alert");
        expect(wrapper.findAll('input[type="file"]')).toHaveLength(3);
        expect(
            wrapper.find<HTMLButtonElement>("[data-identity-submit]").element
                .disabled,
        ).toBe(true);
        expect(mockToastError).toHaveBeenCalledWith(
            "user.verification.identity.manualEvidencePrompt",
        );
    });

    it("returns to the intended page after successful identity verification", async () => {
        mockRoute.query = { redirect: "/courses/reviews/post" };
        mockIdentityApi.getIdentity
            .mockImplementationOnce(notFound)
            .mockResolvedValue({
                data: {
                    data: verifiedIdentity("11010519491231002X"),
                },
            });
        mockIdentityApi.submitIdentity.mockResolvedValue({
            data: {
                data: verifiedIdentity("11010519491231002X"),
            },
        });

        const wrapper = mountPage();
        await flushPromises();

        await wrapper.find("[data-identity-real-name-input]").setValue("张三");
        await wrapper
            .find("[data-identity-doc-number-input]")
            .setValue("11010519491231002X");
        await flushPromises();

        await wrapper
            .find("[data-identity-verification-form]")
            .trigger("submit");
        await flushPromises();

        expect(mockRouterPush).toHaveBeenCalledWith("/courses/reviews/post");
        expect(mockRouterPush).toHaveBeenCalledTimes(1);
    });
});
