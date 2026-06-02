import { beforeEach, describe, expect, it, vi } from "vitest";
import { createPinia, setActivePinia } from "pinia";
import { ApiError } from "@/api/errors";

const mockAuthMe = vi.fn();
const mockGetUser = vi.fn();
const mockSetUser = vi.fn();
const mockClearAuth = vi.fn();

vi.mock("@/api", () => ({
    api: {
        auth: {
            me: mockAuthMe,
        },
    },
}));

vi.mock("@/utils/auth", () => ({
    userManager: {
        getUser: mockGetUser,
        setUser: mockSetUser,
    },
    clearAuth: mockClearAuth,
    tokenExpiry: {
        set: vi.fn(),
    },
}));

vi.mock("@/stores/user", () => ({
    useUserStore: () => ({ reset: vi.fn() }),
}));

vi.mock("@/stores/notification", () => ({
    useNotificationStore: () => ({ stopPolling: vi.fn(), reset: vi.fn() }),
}));

vi.mock("@/stores/courseReview", () => ({
    useCourseStore: () => ({ reset: vi.fn() }),
}));

vi.mock("@/stores/draft", () => ({
    useDraftStore: () => ({ reset: vi.fn() }),
}));

vi.mock("@/stores/verification", () => ({
    useVerificationStore: () => ({ reset: vi.fn() }),
}));

vi.mock("@/i18n", () => ({
    default: {
        global: {
            t: () => "translated",
        },
    },
}));

describe("auth bootstrap", () => {
    beforeEach(() => {
        setActivePinia(createPinia());
        mockAuthMe.mockReset();
        mockGetUser.mockReset();
        mockSetUser.mockReset();
        mockClearAuth.mockReset();
    });

    function muteExpectedBootstrapError() {
        return vi.spyOn(console, "error").mockImplementation(() => undefined);
    }

    it("hydrates user from server when local cache is empty but session is valid", async () => {
        mockGetUser.mockReturnValue(null);
        mockAuthMe.mockResolvedValue({
            data: {
                data: {
                    id: "user_1",
                    name: "alice",
                    displayName: "Alice",
                    email: "alice@example.com",
                    roles: ["admin"],
                    isPlatformAdmin: false,
                    capabilities: ["admin:reviews:manage"],
                    globalCapabilities: ["admin:reviews:manage"],
                    capabilityGrants: [
                        { name: "admin:reviews:manage", global: true },
                    ],
                    canAccessAdmin: true,
                    accountSettingsUrl: "https://account.example.com/account",
                },
            },
        });

        const { useAuthStore } = await import("../auth");
        const store = useAuthStore();

        expect(store.isAuthenticated).toBe(false);

        await expect(store.bootstrapSession()).resolves.toBeTruthy();

        expect(store.isAuthenticated).toBe(true);
        expect(store.user?.id).toBe("user_1");
        expect(store.user?.accountSettingsUrl).toBe(
            "https://account.example.com/account",
        );
        expect(store.globalCapabilities).toEqual(["admin:reviews:manage"]);
        expect(mockSetUser).toHaveBeenCalledTimes(1);
    });

    it("does not clear local session on network-style bootstrap failure", async () => {
        const errorSpy = muteExpectedBootstrapError();
        mockGetUser.mockReturnValue(null);
        mockAuthMe.mockRejectedValue(
            new ApiError({
                code: "NETWORK_ERROR",
                message: "network",
                status: 0,
            }),
        );

        const { useAuthStore } = await import("../auth");
        const store = useAuthStore();

        try {
            await expect(store.bootstrapSession()).resolves.toBeFalsy();
            expect(errorSpy).toHaveBeenCalledWith(
                "[Auth] bootstrapSession failed:",
                expect.any(Error),
            );
        } finally {
            errorSpy.mockRestore();
        }

        expect(store.isAuthenticated).toBe(false);
        expect(mockClearAuth).not.toHaveBeenCalled();
    });

    it("keeps cached users free of stale capability surfaces when bootstrap fails", async () => {
        const errorSpy = muteExpectedBootstrapError();
        mockGetUser.mockReturnValue({
            id: "user_2",
            name: "bob",
            displayName: "Bob",
            globalCapabilities: ["admin:reviews:manage"],
            canAccessAdmin: true,
        });
        mockAuthMe.mockRejectedValue(
            new ApiError({
                code: "NETWORK_ERROR",
                message: "network",
                status: 0,
            }),
        );

        const { useAuthStore } = await import("../auth");
        const store = useAuthStore();

        expect(store.isAuthenticated).toBe(true);
        expect(store.globalCapabilities).toEqual([]);

        try {
            await expect(store.bootstrapSession()).resolves.toBeFalsy();
            expect(errorSpy).toHaveBeenCalledWith(
                "[Auth] bootstrapSession failed:",
                expect.any(Error),
            );
        } finally {
            errorSpy.mockRestore();
        }

        expect(store.isAuthenticated).toBe(true);
        expect(store.globalCapabilities).toEqual([]);
        expect(mockClearAuth).not.toHaveBeenCalled();
    });

    it("fails closed when auth/me returns malformed user capabilities", async () => {
        const errorSpy = muteExpectedBootstrapError();
        mockGetUser.mockReturnValue({
            id: "cached_user",
            name: "cached",
            displayName: "Cached",
        });
        mockAuthMe.mockResolvedValue({
            data: {
                data: {
                    id: "user_3",
                    name: "carol",
                    displayName: "Carol",
                    isPlatformAdmin: false,
                    capabilities: [],
                    globalCapabilities: [],
                    capabilityGrants: [],
                    canAccessAdmin: false,
                },
            },
        });

        const { useAuthStore } = await import("../auth");
        const store = useAuthStore();

        expect(store.isAuthenticated).toBe(true);

        try {
            await expect(
                store.bootstrapSession({ force: true }),
            ).resolves.toBeFalsy();
            expect(errorSpy).toHaveBeenCalledWith(
                "[Auth] bootstrapSession failed:",
                expect.any(Error),
            );
        } finally {
            errorSpy.mockRestore();
        }

        expect(store.isAuthenticated).toBe(true);
        expect(store.bootstrapCompleted).toBe(false);
        expect(mockSetUser).not.toHaveBeenCalled();
        expect(mockClearAuth).not.toHaveBeenCalled();
    });
});
