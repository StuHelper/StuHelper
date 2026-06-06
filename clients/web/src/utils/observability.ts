import { onCLS, onFCP, onINP, onLCP, onTTFB, type Metric } from "web-vitals";

import { buildSecurityHeaders, CSRF_COOKIE_NAME } from "@stuhelper/shared/api";
import { resolveApiURL } from "@/api/client";
import { readCookie } from "@/utils/sessionHint";

const VITALS_PATH = "/api/v1/metrics/vitals";
const FRONTEND_ERRORS_PATH = "/api/v1/metrics/frontend-errors";

type FrontendErrorKind = "error" | "unhandledrejection";

function sendBeaconJSON(path: string, payload: unknown) {
    if (typeof window === "undefined") return;
    const body = JSON.stringify(payload);
    const url = resolveApiURL(path).toString();
    if (typeof navigator !== "undefined" && typeof navigator.sendBeacon === "function") {
        // 页面卸载期指标上报依赖 sendBeacon，shared API client 无法保证发送完成。
        const accepted = navigator.sendBeacon(
            url,
            new Blob([body], { type: "application/json" }),
        );
        if (accepted) return;
    }

    const headers = buildSecurityHeaders("POST", {
        "Content-Type": "application/json",
    }, {
        csrfToken: readCookie(CSRF_COOKIE_NAME),
    });

    // sendBeacon 不可用时使用 keepalive fetch 兜底，仍是卸载期上报例外。
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
