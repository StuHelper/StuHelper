import { onCLS, onFCP, onINP, onLCP, onTTFB, type Metric } from "web-vitals";

const API_BASE_URL = import.meta.env.VITE_API_URL || "/api";
const VITALS_PATH = "/api/v1/metrics/vitals";
const FRONTEND_ERRORS_PATH = "/api/v1/metrics/frontend-errors";

type FrontendErrorKind = "error" | "unhandledrejection";

function resolveApiURL(path: string): string {
    const base = /^https?:\/\//.test(API_BASE_URL)
        ? API_BASE_URL
        : window.location.origin;
    return new URL(path, base).toString();
}

function sendBeaconJSON(path: string, payload: unknown) {
    if (typeof window === "undefined") return;
    const body = JSON.stringify(payload);
    const url = resolveApiURL(path);

    if (
        typeof navigator !== "undefined" &&
        typeof navigator.sendBeacon === "function"
    ) {
        const blob = new Blob([body], { type: "application/json" });
        navigator.sendBeacon(url, blob);
        return;
    }

    void fetch(url, {
        method: "POST",
        body,
        headers: { "Content-Type": "application/json" },
        keepalive: true,
        credentials: "omit",
    }).catch(() => undefined);
}

function reportVital(metric: Metric) {
    sendBeaconJSON(VITALS_PATH, {
        name: metric.name,
        value: metric.value,
        rating: metric.rating,
    });
}

function reportFrontendError(kind: FrontendErrorKind) {
    sendBeaconJSON(FRONTEND_ERRORS_PATH, { kind });
}

export function initObservability() {
    if (typeof window === "undefined") return;

    onCLS(reportVital);
    onINP(reportVital);
    onLCP(reportVital);
    onFCP(reportVital);
    onTTFB(reportVital);

    window.addEventListener("error", () => {
        reportFrontendError("error");
    });

    window.addEventListener("unhandledrejection", () => {
        reportFrontendError("unhandledrejection");
    });
}
