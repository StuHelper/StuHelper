import { beforeEach, describe, expect, it, vi } from "vitest";
import { createPinia, setActivePinia } from "pinia";

const mockLogout = vi.fn();
const mockRefresh = vi.fn();
const mockGetUser = vi.fn();
const mockSetUser = vi.fn();
const mockClearAuth = vi.fn();
const mockTokenExpirySet = vi.fn();

vi.mock("@/api", () => ({
    api: {
        auth: {
            me: vi.fn(),
            login: vi.fn(),
            signup: vi.fn(),
            refresh: mockRefresh,
            logout: mockLogout,
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
        set: mockTokenExpirySet,
    },
}));

vi.mock("@/i18n", () => ({
    default: {
        global: {
            t: () => "translated",
        },
    },
}));

describe("auth session reset", () => {
    beforeEach(() => {
        setActivePinia(createPinia());
        mockLogout.mockReset();
        mockRefresh.mockReset();
        mockGetUser.mockReset();
        mockSetUser.mockReset();
        mockClearAuth.mockReset();
        mockTokenExpirySet.mockReset();
        mockGetUser.mockReturnValue(null);
    });

    it("runs registered session reset handlers after successful logout", async () => {
        mockLogout.mockResolvedValue({});
        const resetSpy = vi.fn();
        const { registerSessionResetHandler } =
            await import("@/stores/sessionOrchestrator");
        registerSessionResetHandler("user", resetSpy);

        const { useAuthStore } = await import("../auth");
        const store = useAuthStore();
        await expect(store.logout()).resolves.toEqual({ ok: true });

        expect(mockClearAuth).toHaveBeenCalledTimes(1);
        expect(resetSpy).toHaveBeenCalledTimes(1);
    });

    it("runs registered session reset handlers when clearSession is called locally", async () => {
        const resetSpy = vi.fn();
        const { registerSessionResetHandler } =
            await import("@/stores/sessionOrchestrator");
        registerSessionResetHandler("user", resetSpy);

        const { useAuthStore } = await import("../auth");
        const store = useAuthStore();
        store.clearSession();

        expect(mockClearAuth).toHaveBeenCalledTimes(1);
        expect(resetSpy).toHaveBeenCalledTimes(1);
    });

    it("rejects malformed refresh responses without updating local expiry", async () => {
        mockRefresh.mockResolvedValue({ data: { data: { expiresIn: "3600" } } });

        const { useAuthStore } = await import("../auth");
        const store = useAuthStore();

        await expect(store.refreshSession()).rejects.toThrow(
            "Invalid refresh response",
        );

        expect(mockTokenExpirySet).not.toHaveBeenCalled();
        expect(mockClearAuth).not.toHaveBeenCalled();
    });
});
