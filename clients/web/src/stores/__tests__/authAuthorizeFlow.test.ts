import { beforeEach, describe, expect, it, vi } from "vitest";
import { createPinia, setActivePinia } from "pinia";

const mockLogin = vi.fn();
const mockSignup = vi.fn();
const mockLogout = vi.fn();
const mockGetUser = vi.fn();
const mockSetUser = vi.fn();
const mockClearAuth = vi.fn();
const mockIsTokenExpired = vi.fn(() => false);
const mockRememberAdmissionAuthReturn = vi.fn(() => true);
const mockStoreOAuthState = vi.fn((state: string) => {
    sessionStorage.setItem("oauth_state", state);
    return true;
});
const mockWindowOpen = vi.fn();
const mockSetTimeout = vi.fn();
const mockPopupClose = vi.fn();
const mockCreateElement = vi.fn();
const mockAppendChild = vi.fn();
const mockIframeRemove = vi.fn();
const mockIframeSetAttribute = vi.fn();
const mockFetch = vi.fn();
const mockClearTimeout = vi.fn();
const mockReportFrontendError = vi.fn();

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
    isTokenExpired: mockIsTokenExpired,
    rememberAdmissionAuthReturn: mockRememberAdmissionAuthReturn,
    storeOAuthState: mockStoreOAuthState,
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

vi.mock("@/utils/observability", () => ({
    reportFrontendError: mockReportFrontendError,
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
        mockIsTokenExpired.mockReset();
        mockIsTokenExpired.mockReturnValue(false);
        mockRememberAdmissionAuthReturn.mockReset();
        mockRememberAdmissionAuthReturn.mockReturnValue(true);
        mockStoreOAuthState.mockClear();
        mockWindowOpen.mockReset();
        mockSetTimeout.mockReset();
        mockPopupClose.mockReset();
        mockCreateElement.mockReset();
        mockAppendChild.mockReset();
        mockIframeRemove.mockReset();
        mockIframeSetAttribute.mockReset();
        mockFetch.mockReset();
        mockClearTimeout.mockReset();
        mockReportFrontendError.mockReset();
        mockGetUser.mockReturnValue(null);
        mockFetch.mockResolvedValue({
            ok: true,
            status: 200,
            headers: {
                get: (key: string) => key === "content-type" ? "application/json" : null,
            },
            json: async () => ({ status: "ok" }),
        });
        vi.stubEnv("VITE_SSO_URL", "https://sso.stuhelper.com");
        vi.stubGlobal("sessionStorage", createMemoryStorage());
        const popup = {
            close: mockPopupClose,
            location: {
                href: "",
            },
        };
        mockWindowOpen.mockReturnValue(popup);
        let iframeLoad: (() => void) | undefined;
        const iframe = {
            title: "",
            style: {} as Record<string, string>,
            remove: mockIframeRemove,
            setAttribute: mockIframeSetAttribute,
            addEventListener: vi.fn((event: string, handler: () => void) => {
                if (event === "load") iframeLoad = handler;
            }),
            src: "",
        };
        mockCreateElement.mockReturnValue(iframe);
        mockAppendChild.mockImplementation((element: typeof iframe) => {
            queueMicrotask(() => iframeLoad?.());
            return element;
        });
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
            clearTimeout: mockClearTimeout,
            fetch: mockFetch,
        });
        vi.stubGlobal("document", {
            createElement: mockCreateElement,
            body: {
                appendChild: mockAppendChild,
            },
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

    it("reuses one pending authorization request for repeated login calls", async () => {
        const loginResponse = createDeferred<{
            data: { data: { state: string; url: string } };
        }>();
        mockLogin.mockReturnValueOnce(loginResponse.promise);

        const { useAuthStore } = await import("../auth");
        const store = useAuthStore();

        const first = store.login("https://join.stuhelper.com/verify/token");
        const second = store.login("https://join.stuhelper.com/verify/token");

        expect(mockLogin).toHaveBeenCalledTimes(1);
        loginResponse.resolve({
            data: { data: { state: "single-state", url: "#single" } },
        });
        await Promise.all([first, second]);

        expect(sessionStorage.getItem("oauth_state")).toBe("single-state");
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

    it("keeps join self-service start as the login return target", async () => {
        mockLogin.mockResolvedValue({
            data: { data: { state: "start-state", url: "#start" } },
        });
        vi.stubGlobal("window", {
            open: mockWindowOpen,
            location: {
                href: "https://join.stuhelper.com/start",
                hash: "",
                origin: "https://join.stuhelper.com",
            },
            setTimeout: mockSetTimeout,
            clearTimeout: mockClearTimeout,
            fetch: mockFetch,
        });

        const { useAuthStore } = await import("../auth");
        const store = useAuthStore();

        await store.login("https://join.stuhelper.com/start");

        expect(mockLogin).toHaveBeenCalledWith(
            "https://join.stuhelper.com/start",
            undefined,
            "web",
            { prompt: undefined, maxAge: undefined },
        );
        expect(sessionStorage.getItem("oauth_state")).toBe("start-state");
        expect(mockRememberAdmissionAuthReturn).toHaveBeenCalledWith(
            "https://join.stuhelper.com/start",
        );
    });

    it("continues login when the sessionStorage state copy cannot be stored", async () => {
        mockLogin.mockResolvedValue({
            data: { data: { state: "login-state", url: "#login" } },
        });
        mockStoreOAuthState.mockReturnValueOnce(false);

        const { useAuthStore } = await import("../auth");
        const store = useAuthStore();

        await store.login("https://join.stuhelper.com/verify/token");

        expect(mockLogin).toHaveBeenCalledTimes(1);
        expect(sessionStorage.getItem("oauth_state")).toBeNull();
        expect(window.location.href).toBe("#login");
        expect(mockReportFrontendError).toHaveBeenCalledWith(
            "error",
            "auth.oauth_state_storage_unavailable",
        );
    });

    it("continues signup when the sessionStorage state copy cannot be stored", async () => {
        mockSignup.mockResolvedValue({
            data: { data: { state: "signup-state", url: "#signup" } },
        });
        mockStoreOAuthState.mockReturnValueOnce(false);

        const { useAuthStore } = await import("../auth");
        const store = useAuthStore();

        await store.signup("https://join.stuhelper.com/verify/token");

        expect(mockSignup).toHaveBeenCalledTimes(1);
        expect(sessionStorage.getItem("oauth_state")).toBeNull();
        expect(window.location.href).toBe("#signup");
        expect(mockReportFrontendError).toHaveBeenCalledWith(
            "error",
            "auth.oauth_state_storage_unavailable",
        );
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
        expect(mockClearAuth).toHaveBeenCalledTimes(2);
        expect(mockWindowOpen).not.toHaveBeenCalled();
        expect(mockFetch).toHaveBeenCalledTimes(1);
        const [logoutURL, logoutOptions] = mockFetch.mock.calls[0];
        expect(String(logoutURL)).toMatch(
            /^https:\/\/sso\.stuhelper\.com\/api\/sso-logout\?logoutAll=false&_/,
        );
        expect(logoutOptions).toMatchObject({
            method: "POST",
            credentials: "include",
            cache: "no-store",
        });
        expect(mockCreateElement).not.toHaveBeenCalled();
        expect(mockLogin).toHaveBeenCalledWith(
            "https://join.stuhelper.com/verify/token",
            undefined,
            "web",
            { prompt: "login", maxAge: 0 },
        );
        expect(window.location.href).toBe("#switch");
        expect(window.location.href).not.toContain("/api/sso-logout");
        expect(sessionStorage.getItem("oauth_state")).toBe("switch-state");
    });

    it("does not require a popup before switching accounts", async () => {
        mockLogout.mockResolvedValue({});
        mockLogin.mockResolvedValue({
            data: { data: { state: "switch-state", url: "#switch" } },
        });

        const { useAuthStore } = await import("../auth");
        const store = useAuthStore();

        await store.switchAccount("https://join.stuhelper.com/verify/token");

        expect(mockWindowOpen).not.toHaveBeenCalled();
        expect(mockLogout).toHaveBeenCalledTimes(1);
        expect(mockClearAuth).toHaveBeenCalledTimes(2);
        expect(mockFetch).toHaveBeenCalledTimes(1);
        expect(mockLogin).toHaveBeenCalledTimes(1);
    });

    it("falls back after an upstream SSO logout fetch failure and still starts forced login", async () => {
        mockLogout.mockResolvedValue({});
        mockFetch.mockRejectedValue(new Error("cors blocked"));
        mockLogin.mockResolvedValue({
            data: { data: { state: "switch-state", url: "#switch" } },
        });

        const { useAuthStore } = await import("../auth");
        const store = useAuthStore();

        await store.switchAccount("https://join.stuhelper.com/verify/token");

        expect(mockLogout).toHaveBeenCalledTimes(1);
        expect(mockClearAuth).toHaveBeenCalledTimes(2);
        expect(mockFetch).toHaveBeenCalledTimes(1);
        expect(mockCreateElement).toHaveBeenCalledWith("iframe");
        expect(mockIframeSetAttribute).toHaveBeenCalledWith("aria-hidden", "true");
        expect(mockAppendChild).toHaveBeenCalledTimes(1);
        const iframe = mockCreateElement.mock.results[0]?.value as { src: string };
        expect(iframe.src).toMatch(
            /^https:\/\/sso\.stuhelper\.com\/api\/sso-logout\?logoutAll=false&_/,
        );
        expect(mockIframeRemove).toHaveBeenCalledTimes(1);
        expect(mockLogin).toHaveBeenCalledWith(
            "https://join.stuhelper.com/verify/token",
            undefined,
            "web",
            { prompt: "login", maxAge: 0 },
        );
        expect(window.location.href).toBe("#switch");
        expect(window.location.href).not.toContain("/api/sso-logout");
    });

    it("clears local state and starts forced login when local logout fails during account switch", async () => {
        mockLogout.mockRejectedValue(new Error("local logout unavailable"));
        mockLogin.mockResolvedValue({
            data: { data: { state: "switch-state", url: "#switch" } },
        });

        const { useAuthStore } = await import("../auth");
        const store = useAuthStore();

        await store.switchAccount("https://join.stuhelper.com/verify/token");

        expect(mockClearAuth).toHaveBeenCalledTimes(2);
        expect(mockWindowOpen).not.toHaveBeenCalled();
        expect(mockFetch).toHaveBeenCalledTimes(1);
        expect(mockCreateElement).not.toHaveBeenCalled();
        expect(mockLogin).toHaveBeenCalledWith(
            "https://join.stuhelper.com/verify/token",
            undefined,
            "web",
            { prompt: "login", maxAge: 0 },
        );
        expect(window.location.href).toBe("#switch");
        expect(window.location.href).not.toContain("/api/sso-logout");
    });
});

function createDeferred<T>() {
    let resolve!: (value: T) => void;
    let reject!: (reason?: unknown) => void;
    const promise = new Promise<T>((promiseResolve, promiseReject) => {
        resolve = promiseResolve;
        reject = promiseReject;
    });
    return { promise, reject, resolve };
}
