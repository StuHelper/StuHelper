/**
 * 认证状态管理
 */
import { defineStore, getActivePinia } from "pinia";
import { ref, computed } from "vue";
import { api } from "@/api";
import { userManager, clearAuth, tokenExpiry } from "@/utils/auth";
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
type LoginURLResponse = components["schemas"]["LoginURLResponse"];
type RefreshResponse = components["schemas"]["RefreshResponse"];
type CapabilityGrant = UserInfo["capabilityGrants"][number];

// 登出结果类型
export type LogoutResult =
    | { ok: true }
    | { ok: false; reason: "network" | "server"; error: unknown };

function getSSOOrigin(): string {
    const ssoURL = import.meta.env.VITE_SSO_URL;
    if (!ssoURL) {
        throw new Error("VITE_SSO_URL is not configured");
    }

    const parsed = new URL(ssoURL);
    if (parsed.protocol !== "https:" && parsed.protocol !== "http:") {
        throw new Error("Invalid SSO URL protocol");
    }

    return parsed.origin;
}

function resolveLoginRedirectTarget(redirect?: string): string | undefined {
    if (typeof window === "undefined") {
        return redirect;
    }

    if (!redirect) {
        return window.location.href;
    }

    if (redirect.startsWith("/") && !redirect.startsWith("//")) {
        return new URL(redirect, window.location.origin).toString();
    }

    try {
        const parsed = new URL(redirect, window.location.origin);
        if (
            parsed.origin === window.location.origin &&
            (parsed.protocol === "https:" || parsed.protocol === "http:")
        ) {
            return parsed.toString();
        }
    } catch {
        // ignore invalid redirect and fall back to current page
    }

    return window.location.href;
}

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

    // OAuth 跳转通用流程
    const startOAuthFlow = async (
        apiCall: () => Promise<{ data?: { data?: LoginURLResponse } }>,
        errorMsg: string,
    ) => {
        clearError();
        loading.value = true;
        try {
            const res = await apiCall();
            const data = readLoginURLPayload(res.data?.data);
            if (!data.url || !data.state) {
                throw new Error("Invalid OAuth response");
            }
            // 校验 OAuth URL 必须指向配置的 SSO Origin
            const oauthURL = new URL(data.url);
            if (
                oauthURL.protocol !== "https:" &&
                oauthURL.protocol !== "http:"
            ) {
                throw new Error("Invalid OAuth URL protocol");
            }
            if (oauthURL.origin !== getSSOOrigin()) {
                throw new Error(
                    "OAuth URL must point to the configured SSO origin",
                );
            }
            sessionStorage.setItem("oauth_state", data.state);
            window.location.href = data.url;
            // 跳转失败时 3 秒后清除 loading 状态
            setTimeout(() => {
                loading.value = false;
            }, 3000);
        } catch (err) {
            loading.value = false;
            const authErr = handleError(err, errorMsg);
            setError(authErr.type, authErr.message);
            throw err;
        }
    };

    // 登录
    const login = (redirect?: string) =>
        startOAuthFlow(
            () => api.auth.login(resolveLoginRedirectTarget(redirect), undefined, "web"),
            i18n.global.t("common.login.loginUrlFailed"),
        );

    const reauthenticate = (redirect?: string) =>
        startOAuthFlow(
            () =>
                api.auth.login(resolveLoginRedirectTarget(redirect), undefined, "web", {
                    prompt: "login",
                    maxAge: 0,
                }),
            i18n.global.t("common.login.loginUrlFailed"),
        );

    // 注册
    const signup = (redirect?: string) =>
        startOAuthFlow(
            () => api.auth.signup(resolveLoginRedirectTarget(redirect)),
            i18n.global.t("common.login.signupUrlFailed"),
        );

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
        reauthenticate,
        signup,
        fetchUser,
        bootstrapSession,
        refreshSession,
        logout,
        clearSession,
        clearError,
    };
});
