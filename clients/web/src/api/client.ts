import { createApiClient } from "@stuhelper/shared/api";
import {
    ApiError,
    httpStatusToDefaultCode,
    isAuthError,
    isCsrfError,
} from "./errors";
import { useAuthStore } from "@/stores/auth";
import { tokenExpiry } from "@/utils/auth";

const RAW_API_BASE_URL = import.meta.env.VITE_API_URL?.trim() || "";
const AUTH_REFRESH_PATH = "/api/v1/auth/refresh";
const AUTH_REFRESH_TIMEOUT_MS = 10_000;
const DEFAULT_REQUEST_TIMEOUT_MS = (() => {
    const raw = Number(import.meta.env.VITE_API_TIMEOUT_MS ?? 15_000);
    return Number.isFinite(raw) && raw > 0 ? raw : 15_000;
})();
const CSRF_COOKIE_NAME = "csrf_token";
const CSRF_HEADER_NAME = "X-CSRF-Token";

type RefreshResult =
    | { ok: true }
    | { ok: false; reason: "unauthorized" }
    | { ok: false; reason: "error"; error: ApiError };

let refreshPromise: Promise<RefreshResult> | null = null;

function stripSchemaPrefix(baseUrl: string): string {
    const trimmed = baseUrl.trim().replace(/\/$/, "");
    return trimmed.replace(/\/api(?:\/v1)?$/, "");
}

export function resolveApiBaseUrl(rawBaseUrl: string = RAW_API_BASE_URL): string {
    const normalizedBaseUrl = stripSchemaPrefix(rawBaseUrl);

    if (/^https?:\/\//.test(normalizedBaseUrl)) {
        return normalizedBaseUrl;
    }

    if (typeof window !== "undefined" && window.location?.origin) {
        return window.location.origin;
    }

    return normalizedBaseUrl;
}

function readCookie(name: string): string | null {
    if (typeof document === "undefined") return null;
    const cookies = document.cookie ? document.cookie.split(";") : [];
    const target = `${encodeURIComponent(name)}=`;
    for (const raw of cookies) {
        const cookie = raw.trim();
        if (!cookie.startsWith(target)) continue;
        try {
            return decodeURIComponent(cookie.slice(target.length));
        } catch {
            return null;
        }
    }
    return null;
}

function needsCSRF(method: string): boolean {
    return !["GET", "HEAD", "OPTIONS"].includes(method.toUpperCase());
}

/**
 * 将 API 路径解析为绝对 URL。
 */
export function resolveApiURL(path: string): URL {
    const base = resolveApiBaseUrl();
    return new URL(path, base);
}

function withBrowserSecurity(request: Request): Request {
    const headers = new Headers(request.headers);
    if (needsCSRF(request.method)) {
        const csrfToken = readCookie(CSRF_COOKIE_NAME);
        if (csrfToken) {
            headers.set(CSRF_HEADER_NAME, csrfToken);
        }
    }

    return new Request(request, {
        credentials: "include",
        headers,
    });
}

function isRefreshRequest(request: Request): boolean {
    const url = new URL(request.url, window.location.origin);
    return url.pathname === AUTH_REFRESH_PATH;
}

function getExpiresInFromPayload(payload: unknown): number | null {
    if (!payload || typeof payload !== "object") {
        return null;
    }

    if (!("data" in payload)) {
        return null;
    }
    const data = (payload as Record<string, unknown>).data;
    if (!data || typeof data !== "object") {
        return null;
    }

    if (!("expiresIn" in data)) {
        return null;
    }
    const expiresIn = (data as Record<string, unknown>).expiresIn;
    return typeof expiresIn === "number" ? expiresIn : null;
}

function normalizeClientError(error: unknown): ApiError {
    if (error instanceof ApiError) {
        return error;
    }

    if (typeof navigator !== "undefined" && !navigator.onLine) {
        return new ApiError({ code: "OFFLINE", message: "offline" });
    }

    if (error instanceof DOMException && error.name === "AbortError") {
        return new ApiError({ code: "TIMEOUT", message: "request timeout" });
    }

    return new ApiError({
        code: "NETWORK_ERROR",
        message: error instanceof Error ? error.message : "network error",
    });
}

async function fetchWithTimeout(
    request: Request,
    timeoutMs: number,
): Promise<Response> {
    const controller = new AbortController();
    const upstreamSignal = request.signal;
    const timeoutId = window.setTimeout(() => controller.abort(), timeoutMs);
    const abortFromUpstream = () => controller.abort();

    if (upstreamSignal) {
        if (upstreamSignal.aborted) {
            controller.abort();
        } else {
            upstreamSignal.addEventListener("abort", abortFromUpstream, {
                once: true,
            });
        }
    }

    try {
        return await window.fetch(
            new Request(request, { signal: controller.signal }),
        );
    } finally {
        window.clearTimeout(timeoutId);
        upstreamSignal?.removeEventListener("abort", abortFromUpstream);
    }
}

async function performBrowserFetch(request: Request): Promise<Response> {
    const timeoutMs = isRefreshRequest(request)
        ? AUTH_REFRESH_TIMEOUT_MS
        : DEFAULT_REQUEST_TIMEOUT_MS;
    return fetchWithTimeout(request, timeoutMs);
}

function createRefreshRequest(): Request {
    const headers = new Headers();
    const csrfToken = readCookie(CSRF_COOKIE_NAME);
    if (csrfToken) {
        headers.set(CSRF_HEADER_NAME, csrfToken);
    }

    return new Request(resolveApiURL(AUTH_REFRESH_PATH), {
        method: "POST",
        credentials: "include",
        headers,
    });
}

async function refreshAccessToken(): Promise<RefreshResult> {
    if (typeof window === "undefined") {
        return {
            ok: false,
            reason: "error",
            error: new ApiError({
                code: "NETWORK_ERROR",
                message: "window is not available",
            }),
        };
    }
    if (refreshPromise) return refreshPromise;

    refreshPromise = (async () => {
        try {
            const response = await performBrowserFetch(createRefreshRequest());

            if (!response.ok) {
                const payload = await response
                    .clone()
                    .json()
                    .catch(() => null);
                const error = buildApiError(response, payload);

                if (
                    response.status === 401 ||
                    (!isCsrfError(error.code) && isAuthError(error.code))
                ) {
                    return { ok: false, reason: "unauthorized" } as const;
                }

                return { ok: false, reason: "error", error } as const;
            }

            const payload = await response.json().catch(() => null);
            const expiresIn = getExpiresInFromPayload(payload);
            if (typeof expiresIn === "number") {
                tokenExpiry.set(expiresIn);
            }

            return { ok: true } as const;
        } catch (error) {
            return {
                ok: false,
                reason: "error",
                error: normalizeClientError(error),
            } as const;
        } finally {
            refreshPromise = null;
        }
    })();

    return refreshPromise;
}

async function authenticatedFetch(input: Request): Promise<Response> {
    const initialSeed = input.clone();
    const initialRequest = withBrowserSecurity(initialSeed);
    const response = await performBrowserFetch(initialRequest);

    if (response.status !== 401 || isRefreshRequest(initialRequest)) {
        return response;
    }

    const refreshed = await refreshAccessToken();
    if (!refreshed.ok) {
        if (refreshed.reason === "unauthorized") {
            useAuthStore().clearSession();
            return response;
        }

        throw refreshed.error;
    }

    const retryRequest = withBrowserSecurity(input.clone());
    const retryResponse = await performBrowserFetch(retryRequest);

    if (retryResponse.status === 401) {
        useAuthStore().clearSession();
        return retryResponse;
    }

    return retryResponse;
}

function buildApiError(response: Response, payload: unknown): ApiError {
    let errorData: {
        code?: string;
        message?: string;
        details?: Record<string, unknown>;
    } | undefined;

    if (typeof payload === "object" && payload !== null && "error" in payload) {
        const rawError = (payload as Record<string, unknown>).error;
        if (typeof rawError === "object" && rawError !== null) {
            errorData = rawError as {
                code?: string;
                message?: string;
                details?: Record<string, unknown>;
            };
        }
    }

    return new ApiError({
        message:
            errorData?.message || response.statusText || "API request failed",
        code: errorData?.code || httpStatusToDefaultCode(response.status),
        status: response.status,
        details: errorData?.details,
        requestID: response.headers.get("X-Request-Id") || undefined,
    });
}

export const apiClientOptions = {
    baseUrl: resolveApiBaseUrl(),
    credentials: "include" as const,
    fetch: ((input: RequestInfo | URL, init?: RequestInit) => {
        const request = new Request(input, init);
        return authenticatedFetch(request);
    }) as typeof fetch,
};

export const apiClient = createApiClient(apiClientOptions);

apiClient.use({
    async onResponse({ response }) {
        if (response.ok) return response;

        const payload = await response
            .clone()
            .json()
            .catch(() => null);
        throw buildApiError(response, payload);
    },
    onError({ error }) {
        throw normalizeClientError(error);
    },
});

export const __testing__ = {
    DEFAULT_REQUEST_TIMEOUT_MS,
    AUTH_REFRESH_TIMEOUT_MS,
    readCookie,
    withBrowserSecurity,
    fetchWithTimeout,
    performBrowserFetch,
    authenticatedFetch,
    buildApiError,
    resolveApiBaseUrl,
    normalizeClientError,
    resolveApiURL,
};
