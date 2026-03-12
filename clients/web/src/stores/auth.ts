/**
 * 认证状态管理
 */
import { defineStore } from "pinia";
import { ref, computed } from "vue";
import { api } from "@/api";
import { userManager, clearAuth, tokenExpiry } from "@/utils/auth";
import { isApiError, isAuthError, isCsrfError, isNetworkError } from "@/api/errors";
import { useUserStore } from "@/stores/user";
import { useNotificationStore } from "@/stores/notification";
import { useCourseStore } from "@/stores/courseReview";
import { useDraftStore } from "@/stores/draft";
import i18n from "@/i18n";
import type { components, operations } from "@stuhelper/shared";

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
type CurrentUserResponse =
    operations["getCurrentUser"]["responses"][200]["content"]["application/json"]["data"];

// 登出结果类型
export type LogoutResult =
    | { ok: true; ssoLogoutURL?: string }
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

function normalizeCurrentUser(
    data: CurrentUserResponse,
    _currentUser: UserInfo | null,
): UserInfo {
    return {
        id: data.id,
        name: data.name,
        displayName: data.displayName ?? data.name,
        email: data.email,
        avatar: _currentUser?.avatar,
        // I-4: isAdmin 应由服务端在 /auth/me 或 /auth/callback 响应中提供
        // 不从 localStorage 继承 isAdmin，防止服务端撤销权限后客户端仍显示管理员 UI
        isAdmin: data.isAdmin === true,
    };
}

export const useAuthStore = defineStore("auth", () => {
    // M-91: 用 try-catch 包裹 localStorage 读取，防止数据损坏导致 store 初始化失败
    let initialUser: UserInfo | null = null;
    try {
        initialUser = userManager.getUser();
    } catch {
        initialUser = null;
    }

    // 状态
    const user = ref<UserInfo | null>(initialUser);
    const loading = ref(false);
    const error = ref<AuthError | null>(null);

    // 计算属性
    const isAuthenticated = computed(() => !!user.value);

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
        const t = i18n.global.t;
        if (isApiError(err)) {
            if (isNetworkError(err.code)) {
                return { type: "network", message: t("errors.NETWORK_ERROR") };
            }
            return { type: "auth_failed", message: err.getUserMessage() };
        }
        if (err instanceof Error) {
            return { type: "unknown", message: err.message };
        }
        return { type: "unknown", message: defaultMsg };
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
            const data = res.data?.data;
            if (!data?.url || !data?.state) {
                throw new Error("Invalid OAuth response");
            }
            // H-03: 仅允许跳转到配置的 SSO Origin，避免错误配置或恶意 URL
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
            // S-4: 如果浏览器未能成功跳转（弹窗拦截、扩展阻断等），
            // 延迟清除 loading 状态，避免永远转圈
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
    const login = () =>
        startOAuthFlow(
            () => api.auth.login(),
            i18n.global.t("common.login.loginUrlFailed"),
        );

    // 注册
    const signup = () =>
        startOAuthFlow(
            () => api.auth.signup(),
            i18n.global.t("common.login.signupUrlFailed"),
        );

    // 处理 OAuth 回调
    const handleCallback = async (code: string, state: string) => {
        clearError();
        loading.value = true;
        try {
            const savedState = sessionStorage.getItem("oauth_state");
            if (savedState !== state) {
                setError(
                    "invalid_state",
                    i18n.global.t("common.login.invalidState"),
                );
                throw new Error("Invalid state parameter");
            }

            const res = await api.auth.callback(code, state);
            const data = res.data?.data;
            if (!data?.user) {
                throw new Error("Invalid callback response");
            }
            userManager.setUser(data.user);
            user.value = data.user;
            // 存储 token 过期时间，供客户端预检使用
            if (data.expiresIn) {
                tokenExpiry.set(data.expiresIn);
            }
            sessionStorage.removeItem("oauth_state");
            return data;
        } catch (err) {
            if (!error.value) {
                const authErr = handleError(
                    err,
                    i18n.global.t("common.login.callbackFailed"),
                );
                setError(authErr.type, authErr.message);
            }
            throw err;
        } finally {
            loading.value = false;
        }
    };

    // 获取当前用户
    const fetchUser = async () => {
        clearError();
        loading.value = true;
        try {
            const res = await api.auth.me();
            const data = res.data?.data;
            if (!data) {
                throw new Error("Invalid user response");
            }
            const normalizedUser = normalizeCurrentUser(data, user.value);
            user.value = normalizedUser;
            userManager.setUser(normalizedUser);
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

    const bootstrapSession = async () => {
        if (user.value) return true;

        try {
            const res = await api.auth.me();
            const data = res.data?.data;
            if (!data) {
                return false;
            }
            const normalizedUser = normalizeCurrentUser(data, null);
            user.value = normalizedUser;
            userManager.setUser(normalizedUser);
            return true;
        } catch (err) {
            if (isApiError(err) && !isNetworkError(err.code) && !isCsrfError(err.code)) {
                clearAuth();
                user.value = null;
            }
            return false;
        }
    };

    // 刷新会话（依赖 HttpOnly refresh token Cookie）
    const refreshSession = async () => {
        try {
            const res = await api.auth.refresh();
            const data = res.data?.data;
            if (typeof data?.expiresIn === "number") {
                tokenExpiry.set(data.expiresIn);
            }
            return data;
        } catch (err) {
            if (
                isApiError(err) &&
                !isCsrfError(err.code) &&
                (err.status === 401 || isAuthError(err.code))
            ) {
                clearAuth();
                user.value = null;
            }
            throw err;
        }
    };

    // 登出（返回结构化结果，调用方根据结果决定跳转或提示重试）
    const logout = async (): Promise<LogoutResult> => {
        clearError();
        loading.value = true;
        try {
            const res = await api.auth.logout();
            const data = res.data?.data;
            // API 成功 → 清理本地状态 + 返回 SSO 登出 URL
            resetAllStores();
            return { ok: true, ssoLogoutURL: data?.ssoLogoutURL };
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
        useUserStore().reset();
        useCourseStore().reset();
        useDraftStore().reset();
        const notificationStore = useNotificationStore();
        notificationStore.stopPolling();
        notificationStore.reset();
    }

    // 清除本地会话（不调用 API，用于 token 过期等场景）
    const clearSession = () => {
        resetAllStores();
    };

    return {
        user,
        loading,
        error,
        isAuthenticated,
        login,
        signup,
        handleCallback,
        fetchUser,
        bootstrapSession,
        refreshSession,
        logout,
        clearSession,
        clearError,
    };
});
