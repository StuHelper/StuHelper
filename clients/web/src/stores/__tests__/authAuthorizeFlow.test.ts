import { beforeEach, describe, expect, it, vi } from "vitest";
import { createPinia, setActivePinia } from "pinia";

const mockLogin = vi.fn();
const mockSignup = vi.fn();
const mockLogout = vi.fn();
const mockGetUser = vi.fn();
const mockSetUser = vi.fn();
const mockClearAuth = vi.fn();
const mockWindowOpen = vi.fn();
const mockSetTimeout = vi.fn();
const mockPopupClose = vi.fn();

vi.mock("@/api", () => ({
    api: {
        auth: {
            me: vi.fn(),
            login: mockLogin,
            signup: mockSignup,
            refresh: vi.fn(),
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

function createMemoryStorage(): Storage {
    const values = new Map<string, string>();
    return {
        get length() {
            return values.size;
        },
        clear: () => values.clear(),
        getItem: (key: string) => values.get(key) ?? null,
        key: (index: number) => Array.from(values.keys())[index] ?? null,
        removeItem: (key: string) => values.delete(key),
        setItem: (key: string, value: string) => values.set(key, value),
    } as Storage;
}

describe("auth authorize flow", () => {
    beforeEach(() => {
        setActivePinia(createPinia());
        mockLogin.mockReset();
        mockSignup.mockReset();
        mockLogout.mockReset();
        mockGetUser.mockReset();
        mockSetUser.mockReset();
        mockClearAuth.mockReset();
        mockWindowOpen.mockReset();
        mockSetTimeout.mockReset();
        mockPopupClose.mockReset();
        mockGetUser.mockReturnValue(null);
        vi.stubEnv("VITE_SSO_URL", "https://sso.stuhelper.com");
        vi.stubGlobal("sessionStorage", createMemoryStorage());
        const popup = {
            close: mockPopupClose,
            location: {
                href: "",
            },
        };
        mockWindowOpen.mockReturnValue(popup);
        mockSetTimeout.mockImplementation((handler: () => void) => {
            queueMicrotask(handler);
            return 1;
        });
        vi.stubGlobal("window", {
            open: mockWindowOpen,
            location: {
                href: "https://join.stuhelper.com/verify/token",
                hash: "",
                origin: "https://join.stuhelper.com",
            },
            setTimeout: mockSetTimeout,
        });
    });

    it("uses the login authorization endpoint for login", async () => {
        mockLogin.mockResolvedValue({
            data: { data: { state: "login-state", url: "#login" } },
        });

        const { useAuthStore } = await import("../auth");
        const store = useAuthStore();

        await store.login("https://join.stuhelper.com/verify/token");

        expect(mockLogin).toHaveBeenCalledWith(
            "https://join.stuhelper.com/verify/token",
            undefined,
            "web",
            { prompt: undefined, maxAge: undefined },
        );
        expect(mockSignup).not.toHaveBeenCalled();
        expect(sessionStorage.getItem("oauth_state")).toBe("login-state");
    });

    it("uses the signup authorization endpoint for signup", async () => {
        mockSignup.mockResolvedValue({
            data: { data: { state: "signup-state", url: "#signup" } },
        });

        const { useAuthStore } = await import("../auth");
        const store = useAuthStore();

        await store.signup("https://join.stuhelper.com/verify/token");

        expect(mockSignup).toHaveBeenCalledWith(
            "https://join.stuhelper.com/verify/token",
            "web",
            "web",
        );
        expect(mockLogin).not.toHaveBeenCalled();
        expect(sessionStorage.getItem("oauth_state")).toBe("signup-state");
    });

    it("logs out local and upstream SSO account sessions before switching accounts", async () => {
        mockLogout.mockResolvedValue({});
        mockLogin.mockResolvedValue({
            data: { data: { state: "switch-state", url: "#switch" } },
        });

        const { useAuthStore } = await import("../auth");
        const store = useAuthStore();

        await store.switchAccount("https://join.stuhelper.com/verify/token");

        expect(mockLogout).toHaveBeenCalledTimes(1);
        expect(mockClearAuth).toHaveBeenCalledTimes(1);
        expect(mockWindowOpen).toHaveBeenCalledWith(
            "about:blank",
            "stuhelper-sso-account-switch",
            "popup,width=480,height=640",
        );
        const popup = mockWindowOpen.mock.results[0]?.value as { location: { href: string } };
        expect(popup.location.href).toMatch(
            /^https:\/\/sso\.stuhelper\.com\/api\/sso-logout\?logoutAll=true&_/,
        );
        expect(mockPopupClose).toHaveBeenCalledTimes(1);
        expect(mockLogin).toHaveBeenCalledWith(
            "https://join.stuhelper.com/verify/token",
            undefined,
            "web",
            { prompt: "login", maxAge: 0 },
        );
        expect(sessionStorage.getItem("oauth_state")).toBe("switch-state");
    });

    it("does not clear local state when upstream SSO logout popup is blocked", async () => {
        mockWindowOpen.mockReturnValueOnce(null);
        mockLogout.mockResolvedValue({});
        mockLogin.mockResolvedValue({
            data: { data: { state: "switch-state", url: "#switch" } },
        });

        const { useAuthStore } = await import("../auth");
        const store = useAuthStore();

        await expect(
            store.switchAccount("https://join.stuhelper.com/verify/token"),
        ).rejects.toThrow("upstream SSO logout window was blocked");

        expect(mockLogout).not.toHaveBeenCalled();
        expect(mockClearAuth).not.toHaveBeenCalled();
        expect(mockLogin).not.toHaveBeenCalled();
    });

    it("does not start login when local logout fails", async () => {
        mockLogout.mockRejectedValue(new Error("local logout unavailable"));
        mockLogin.mockResolvedValue({
            data: { data: { state: "switch-state", url: "#switch" } },
        });

        const { useAuthStore } = await import("../auth");
        const store = useAuthStore();

        await expect(
            store.switchAccount("https://join.stuhelper.com/verify/token"),
        ).rejects.toThrow("local logout unavailable");

        expect(mockClearAuth).not.toHaveBeenCalled();
        expect(mockWindowOpen).toHaveBeenCalledTimes(1);
        expect(mockPopupClose).toHaveBeenCalledTimes(1);
        expect(mockLogin).not.toHaveBeenCalled();
    });
});
