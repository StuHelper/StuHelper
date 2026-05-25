import { onCLS, onFCP, onINP, onLCP, onTTFB, type Metric } from "web-vitals";

const API_BASE_URL = import.meta.env.VITE_API_URL || "/api";
const VITALS_PATH = "/api/v1/metrics/vitals";
const FRONTEND_ERRORS_PATH = "/api/v1/metrics/frontend-errors";
const CSRF_COOKIE_NAME = "csrf_token";
const CSRF_HEADER_NAME = "X-CSRF-Token";

type FrontendErrorKind = "error" | "unhandledrejection";

function resolveApiURL(path: string): string {
    const base = /^https?:\/\//.test(API_BASE_URL)
        ? API_BASE_URL
        : window.location.origin;
    return new URL(path, base).toString();
}

function readCookie(name: string): string | null {
    const cookies = document.cookie ? document.cookie.split(";") : [];
    const target = `${encodeURIComponent(name)}=`;
    for (const raw of cookies) {
        const cookie = raw.trim();
        if (!cookie.startsWith(target)) continue;
        try {
            return decodeURIComponent(cookie.slice(target.length));
        } catch (_error) { void _error;
            return null;
        }
    }
    return null;
}

function sendBeaconJSON(path: string, payload: unknown) {
    if (typeof window === "undefined") return;
    const body = JSON.stringify(payload);
    const url = resolveApiURL(path);
    if (typeof navigator !== "undefined" && typeof navigator.sendBeacon === "function") {
        const accepted = navigator.sendBeacon(
            url,
            new Blob([body], { type: "application/json" }),
        );
        if (accepted) return;
    }

    const csrfToken = readCookie(CSRF_COOKIE_NAME);
    const headers: Record<string, string> = {
        "Content-Type": "application/json",
    };
    if (csrfToken) {
        headers[CSRF_HEADER_NAME] = csrfToken;
    }

    void fetch(url, {
        method: "POST",
        body,
        headers,
        keepalive: true,
        credentials: "include",
    }).catch(() => undefined);
}

function reportVital(metric: Metric) {
    sendBeaconJSON(VITALS_PATH, {
        name: metric.name,
        value: metric.value,
        rating: metric.rating,
    });
}

function reportFrontendError(kind: FrontendErrorKind, message?: string, source?: string, lineno?: number) {
    sendBeaconJSON(FRONTEND_ERRORS_PATH, { kind, message, source, lineno });
}

export function initObservability() {
    if (typeof window === "undefined") return;

    onCLS(reportVital);
    onINP(reportVital);
    onLCP(reportVital);
    onFCP(reportVital);
    onTTFB(reportVital);

    window.addEventListener("error", (event) => {
        reportFrontendError("error", event.message, event.filename, event.lineno);
    });

    window.addEventListener("unhandledrejection", (event) => {
        const reason = event.reason instanceof Error ? event.reason.message : String(event.reason);
        reportFrontendError("unhandledrejection", reason);
    });
}

export const observabilityTestInternals =
    import.meta.env.MODE === "test"
        ? {
            sendBeaconJSON,
        }
        : undefined;
