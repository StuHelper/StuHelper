/**
 * 认证状态管理
 */
import { defineStore, getActivePinia } from "pinia";
import { ref, computed } from "vue";
import { api } from "@/api";
import {
    userManager,
    clearAuth,
    tokenExpiry,
} from "@/utils/auth";
import {
    normalizeConfiguredHTTPOrigin,
    resolvePostLoginRedirectTarget,
} from "@/utils/redirect";
import {
    classifyApiError,
    isApiError,
    isAuthError,
    isCsrfError,
    isNetworkError,
} from "@/api/errors";
import i18n from "@/i18n";
import { runSessionReset } from "@/stores/sessionOrchestrator";
import type { components } from "@stuhelper/shared/types";

// 认证错误类型
export type AuthErrorType =
    | "network"
    | "invalid_state"
    | "auth_failed"
    | "unknown";

export interface AuthError {
    type: AuthErrorType;
    message: string;
}

type UserInfo = components["schemas"]["UserInfo"];
type RefreshResponse = components["schemas"]["RefreshResponse"];
type LoginURLResponse = components["schemas"]["LoginURLResponse"];
type CapabilityGrant = UserInfo["capabilityGrants"][number];

// 登出结果类型
export type LogoutResult =
    | { ok: true }
    | { ok: false; reason: "network" | "server"; error: unknown };

interface LoginAuthorizeOptions {
    prompt?: "login";
    maxAge?: 0;
}

const DEFAULT_SSO_ORIGIN = "https://sso.stuhelper.com";
const SSO_LOGOUT_POPUP_TIMEOUT_MS = 1500;

function isRecord(value: unknown): value is Record<string, unknown> {
    return Boolean(value) && typeof value === "object";
}

function readString(
    record: Record<string, unknown>,
    key: string,
    message: string,
): string {
    const value = record[key];
    if (typeof value !== "string") {
        throw new Error(message);
    }
    return value;
}

function readOptionalString(
    record: Record<string, unknown>,
    key: string,
    message: string,
): string | undefined {
    const value = record[key];
    if (value === undefined) {
        return undefined;
    }
    if (typeof value !== "string") {
        throw new Error(message);
    }
    return value;
}

function readBoolean(
    record: Record<string, unknown>,
    key: string,
    message: string,
): boolean {
    const value = record[key];
    if (typeof value !== "boolean") {
        throw new Error(message);
    }
    return value;
}

function readInteger(
    record: Record<string, unknown>,
    key: string,
    message: string,
): number {
    const value = record[key];
    if (typeof value !== "number" || !Number.isInteger(value)) {
        throw new Error(message);
    }
    return value;
}

function readStringArray(value: unknown, message: string): string[] {
    if (!Array.isArray(value) || value.some((item) => typeof item !== "string")) {
        throw new Error(message);
    }
    return value;
}

function readOptionalStringArray(
    record: Record<string, unknown>,
    key: string,
    message: string,
): string[] | undefined {
    const value = record[key];
    if (value === undefined) {
        return undefined;
    }
    return readStringArray(value, message);
}

function readOptionalAbsoluteURL(
    record: Record<string, unknown>,
    key: string,
    message: string,
): string | undefined {
    const value = readOptionalString(record, key, message);
    if (value === undefined) {
        return undefined;
    }
    try {
        const parsed = new URL(value);
        if (parsed.protocol !== "https:" && parsed.protocol !== "http:") {
            throw new Error(message);
        }
    } catch {
        throw new Error(message);
    }
    return value;
}

function readCapabilityGrant(value: unknown, message: string): CapabilityGrant {
    if (!isRecord(value)) {
        throw new Error(message);
    }

    return {
        name: readString(value, "name", message),
        scopeSchoolIDs: readOptionalStringArray(value, "scopeSchoolIDs", message),
        scopeSectionIDs: readOptionalStringArray(
            value,
            "scopeSectionIDs",
            message,
        ),
        scopeRoles: readOptionalStringArray(value, "scopeRoles", message),
        global: readBoolean(value, "global", message),
    };
}

function readCapabilityGrants(value: unknown, message: string): CapabilityGrant[] {
    if (!Array.isArray(value)) {
        throw new Error(message);
    }
    return value.map((item) => readCapabilityGrant(item, message));
}

function readLoginURLPayload(
    payload: unknown,
    message = "Invalid OAuth response",
): LoginURLResponse {
    if (!isRecord(payload)) {
        throw new Error(message);
    }
    return {
        url: readString(payload, "url", message),
        state: readString(payload, "state", message),
    };
}

function configuredSSOOrigin(): string {
    if (typeof window === "undefined") return DEFAULT_SSO_ORIGIN;
    return normalizeConfiguredHTTPOrigin(
        import.meta.env.VITE_SSO_URL,
        window.location.origin,
    ) ?? DEFAULT_SSO_ORIGIN;
}

function upstreamSSOAccountSwitchLogoutURL(): string {
    const url = new URL("/api/sso-logout", configuredSSOOrigin());
    url.searchParams.set("logoutAll", "true");
    url.searchParams.set("_", Date.now().toString());
    return url.toString();
}

function readUserInfoPayload(
    payload: unknown,
    message = "Invalid user response",
): UserInfo {
    if (!isRecord(payload)) {
        throw new Error(message);
    }

    return {
        id: readString(payload, "id", message),
        name: readString(payload, "name", message),
        displayName: readString(payload, "displayName", message),
        avatar: readOptionalString(payload, "avatar", message),
        email: readOptionalString(payload, "email", message),
        roles: readStringArray(payload.roles, message),
        isPlatformAdmin: readBoolean(payload, "isPlatformAdmin", message),
        capabilities: readStringArray(payload.capabilities, message),
        globalCapabilities: readStringArray(payload.globalCapabilities, message),
        capabilityGrants: readCapabilityGrants(payload.capabilityGrants, message),
        canAccessAdmin: readBoolean(payload, "canAccessAdmin", message),
        accountSettingsUrl: readOptionalAbsoluteURL(
            payload,
            "accountSettingsUrl",
            message,
        ),
    };
}

function readRefreshPayload(
    payload: unknown,
    message = "Invalid refresh response",
): RefreshResponse {
    if (!isRecord(payload)) {
        throw new Error(message);
    }
    return {
        expiresIn: readInteger(payload, "expiresIn", message),
    };
}

function normalizeStoredUser(
    data: ReturnType<typeof userManager.getUser>,
): UserInfo | null {
    if (!data) {
        return null;
    }

    return {
        id: data.id,
        name: data.name,
        displayName: data.displayName,
        avatar: data.avatar,
        roles: [],
        isPlatformAdmin: false,
        capabilities: [],
        globalCapabilities: [],
        capabilityGrants: [],
        canAccessAdmin: false,
    };
}

export const useAuthStore = defineStore("auth", () => {
    const pinia = getActivePinia();
    if (!pinia) {
        throw new Error("auth store requires an active Pinia instance");
    }

    // localStorage 数据可能损坏，读取失败时降级为空
    let initialUser: UserInfo | null = null;
    try {
        initialUser = normalizeStoredUser(userManager.getUser());
    } catch {
        initialUser = null;
    }

    // 状态
    const user = ref<UserInfo | null>(initialUser);
    const loading = ref(false);
    const error = ref<AuthError | null>(null);
    const bootstrapPending = ref(false);
    const bootstrapCompleted = ref(false);
    let bootstrapPromise: Promise<boolean> | null = null;

    // 计算属性
    const isAuthenticated = computed(() => !!user.value);
    const globalCapabilities = computed(
        () => user.value?.globalCapabilities ?? [],
    );

    // 清除错误
    const clearError = () => {
        error.value = null;
    };

    // 设置错误
    const setError = (type: AuthErrorType, message: string) => {
        error.value = { type, message };
    };

    // 处理 API 错误
    const handleError = (err: unknown, defaultMsg: string): AuthError => {
        return classifyApiError<AuthErrorType>(err, {
            networkType: "network",
            apiType: "auth_failed",
            unknownType: "unknown",
            fallbackMessage: defaultMsg,
        });
    };

    const startLoginFlow = async (
        redirect?: string,
        options?: LoginAuthorizeOptions,
    ) => {
        clearError();
        loading.value = true;
        try {
            const redirectTarget = redirect
                ? resolvePostLoginRedirectTarget(redirect)
                : redirect;
            const res = await api.auth.login(redirectTarget, undefined, "web", {
                prompt: options?.prompt,
                maxAge: options?.maxAge,
            });
            const data = readLoginURLPayload(res.data?.data);
            sessionStorage.setItem("oauth_state", data.state);
            window.location.href = data.url;
            setTimeout(() => {
                loading.value = false;
            }, 3000);
        } catch (err) {
            loading.value = false;
            const authErr = handleError(
                err,
                i18n.global.t("common.login.loginUrlFailed"),
            );
            setError(authErr.type, authErr.message);
            throw err;
        }
    };

    const startSignupFlow = async (redirect?: string) => {
        clearError();
        loading.value = true;
        try {
            const redirectTarget = redirect
                ? resolvePostLoginRedirectTarget(redirect)
                : redirect;
            const res = await api.auth.signup(redirectTarget, "web", "web");
            const data = readLoginURLPayload(res.data?.data);
            sessionStorage.setItem("oauth_state", data.state);
            window.location.href = data.url;
            setTimeout(() => {
                loading.value = false;
            }, 3000);
        } catch (err) {
            loading.value = false;
            const authErr = handleError(
                err,
                i18n.global.t("common.login.signupUrlFailed"),
            );
            setError(authErr.type, authErr.message);
            throw err;
        }
    };

    // 登录
    const login = (redirect?: string) =>
        startLoginFlow(redirect);

    const upstreamLogin = (redirect?: string) =>
        startLoginFlow(redirect);

    const reauthenticate = (redirect?: string) =>
        startLoginFlow(redirect, {
            prompt: "login",
            maxAge: 0,
        });

    const upstreamReauthenticate = (redirect?: string) =>
        startLoginFlow(redirect, {
            prompt: "login",
            maxAge: 0,
        });

    const openUpstreamSSOLogoutWindow = (): Window | null => {
        if (typeof window === "undefined" || typeof window.open !== "function") {
            return null;
        }
        return window.open(
            "about:blank",
            "stuhelper-sso-account-switch",
            "popup,width=480,height=640",
        );
    };

    const closeUpstreamSSOLogoutWindow = (popup: Window | null) => {
        if (!popup) return;
        try {
            popup.close();
        } catch {
            // Cross-origin popup close can fail in hardened browsers; navigation
            // has already been attempted, so the account switch flow can proceed.
        }
    };

    const logoutUpstreamSSOForAccountSwitch = async (popup: Window | null) => {
        if (!popup) {
            throw new Error("upstream SSO logout window was blocked");
        }
        try {
            popup.location.href = upstreamSSOAccountSwitchLogoutURL();
        } catch (error) {
            closeUpstreamSSOLogoutWindow(popup);
            throw error;
        }
        await new Promise<void>((resolve) => {
            window.setTimeout(resolve, SSO_LOGOUT_POPUP_TIMEOUT_MS);
        });
        closeUpstreamSSOLogoutWindow(popup);
    };

    const switchAccount = async (redirect?: string) => {
        clearError();
        loading.value = true;
        let upstreamLogoutWindow = openUpstreamSSOLogoutWindow();
        try {
            if (!upstreamLogoutWindow) {
                throw new Error("upstream SSO logout window was blocked");
            }
            const localLogout = await logout();
            if (!localLogout.ok) {
                closeUpstreamSSOLogoutWindow(upstreamLogoutWindow);
                upstreamLogoutWindow = null;
                const fallback = new Error("local logout unavailable");
                const error = localLogout.error instanceof Error
                    ? localLogout.error
                    : fallback;
                setError(
                    localLogout.reason === "server" ? "auth_failed" : "network",
                    i18n.global.t("common.login.logoutFailed"),
                );
                throw error;
            }
            await logoutUpstreamSSOForAccountSwitch(upstreamLogoutWindow);
            upstreamLogoutWindow = null;
            await startLoginFlow(redirect, {
                prompt: "login",
                maxAge: 0,
            });
        } catch (err) {
            closeUpstreamSSOLogoutWindow(upstreamLogoutWindow);
            loading.value = false;
            const authErr = handleError(
                err,
                i18n.global.t("common.login.loginUrlFailed"),
            );
            setError(authErr.type, authErr.message);
            throw err;
        }
    };

    // 注册
    const signup = (redirect?: string) =>
        startSignupFlow(redirect);

    const upstreamSignup = (redirect?: string) =>
        startSignupFlow(redirect);

    // 获取当前用户
    const fetchUser = async () => {
        clearError();
        loading.value = true;
        try {
            const res = await api.auth.me();
            const normalizedUser = readUserInfoPayload(res.data?.data);
            user.value = normalizedUser;
            userManager.setUser(normalizedUser);
            bootstrapCompleted.value = true;
            return normalizedUser;
        } catch (err) {
            // 区分网络错误和认证错误
            if (isApiError(err) && !isNetworkError(err.code)) {
                clearAuth();
                user.value = null;
            }
            const authErr = handleError(
                err,
                i18n.global.t("common.login.fetchUserFailed"),
            );
            setError(authErr.type, authErr.message);
            throw err;
        } finally {
            loading.value = false;
        }
    };

    const bootstrapSession = async (options?: { force?: boolean }) => {
        if (!options?.force) {
            if (bootstrapCompleted.value && !bootstrapPending.value) {
                return isAuthenticated.value;
            }
            if (bootstrapPromise) {
                return bootstrapPromise;
            }
        }

        clearError();
        bootstrapPending.value = true;

        bootstrapPromise = (async () => {
            try {
                const res = await api.auth.me();
                const normalizedUser = readUserInfoPayload(res.data?.data);
                user.value = normalizedUser;
                userManager.setUser(normalizedUser);
                bootstrapCompleted.value = true;
                return true;
            } catch (err) {
                if (
                    isApiError(err) &&
                    !isNetworkError(err.code) &&
                    !isCsrfError(err.code) &&
                    (err.status === 401 ||
                        err.status === 403 ||
                        isAuthError(err.code))
                ) {
                    clearAuth();
                    user.value = null;
                    bootstrapCompleted.value = true;
                    return false;
                }

                // 网络错误 / 超时 / 5xx：不设置 bootstrapCompleted，允许后续导航重试
                const authErr = handleError(
                    err,
                    i18n.global.t("common.login.fetchUserFailed"),
                );
                setError(authErr.type, authErr.message);
                if (import.meta.env.DEV) {
                    console.error("[Auth] bootstrapSession failed:", err);
                }
                return false;
            } finally {
                bootstrapPending.value = false;
                bootstrapPromise = null;
            }
        })();

        return bootstrapPromise;
    };

    // 刷新会话（依赖 HttpOnly refresh token Cookie）
    const refreshSession = async () => {
        try {
            const res = await api.auth.refresh();
            const data = readRefreshPayload(res.data?.data);
            tokenExpiry.set(data.expiresIn);
            return data;
        } catch (err) {
            if (
                isApiError(err) &&
                !isCsrfError(err.code) &&
                (err.status === 401 || isAuthError(err.code))
            ) {
                clearSession();
            }
            throw err;
        }
    };

    // 登出（返回结构化结果，调用方根据结果决定跳转或提示重试）
    const logout = async (): Promise<LogoutResult> => {
        clearError();
        loading.value = true;
        try {
            await api.auth.logout();
            resetAllStores();
            return { ok: true };
        } catch (err) {
            // 区分认证失效 vs CSRF 拦截 vs 服务端错误 vs 网络错误
            if (isApiError(err)) {
                // CSRF 403: 请求被中间件拦截，但服务端会话仍有效，应提示重试
                if (isCsrfError(err.code)) {
                    return { ok: false, reason: "network", error: err };
                }
                // 401 或非 CSRF 的认证错误: 服务端会话已失效，本地直接清理即可
                if (err.status === 401 || isAuthError(err.code)) {
                    resetAllStores();
                    return { ok: true };
                }
                // 5xx 服务端错误：服务端会话可能仍有效，提示用户稍后重试
                if (err.status && err.status >= 500) {
                    return { ok: false, reason: "server", error: err };
                }
            }
            // 网络/超时/离线：无法确认服务端状态，提示检查网络
            return { ok: false, reason: "network", error: err };
        } finally {
            loading.value = false;
        }
    };

    // 提取 store 重置逻辑，logout 和 clearSession 共用
    function resetAllStores() {
        clearAuth();
        user.value = null;
        bootstrapPending.value = false;
        bootstrapCompleted.value = true;
        runSessionReset(pinia);
    }

    // 清除本地会话（不调用 API，用于 token 过期等场景）
    const clearSession = () => {
        resetAllStores();
    };

    return {
        user,
        loading,
        error,
        bootstrapPending,
        bootstrapCompleted,
        isAuthenticated,
        globalCapabilities,
        login,
        upstreamLogin,
        reauthenticate,
        upstreamReauthenticate,
        switchAccount,
        signup,
        upstreamSignup,
        fetchUser,
        bootstrapSession,
        refreshSession,
        logout,
        clearSession,
        clearError,
    };
});
