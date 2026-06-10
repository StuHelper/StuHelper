export function normalizeConfiguredHTTPOrigin(
    raw: string | undefined,
    currentOrigin?: string,
): string | null {
    const value = raw?.trim();
    if (!value) return null;

    try {
        const parsed = new URL(value, currentOrigin);
        if (parsed.protocol !== "https:" && parsed.protocol !== "http:") {
            return null;
        }
        if (currentOrigin) {
            const current = new URL(currentOrigin);
            if (current.protocol === "https:" && parsed.protocol === "http:") {
                parsed.protocol = "https:";
            }
        }
        return parsed.origin;
    } catch {
        return null;
    }
}

export function configuredWebOrigin(): string | null {
    if (typeof window === "undefined") return null;
    return normalizeConfiguredHTTPOrigin(import.meta.env.VITE_WEB_URL, window.location.origin) ??
        deriveWebOriginFromCurrentLocation();
}

function deriveWebOriginFromCurrentLocation(): string | null {
    if (typeof window === "undefined") return null;

    try {
        const current = new URL(window.location.href);
        const hostname = current.hostname.toLowerCase();
        if (hostname !== "join.localhost" && hostname !== "join.stuhelper.com") {
            return null;
        }
        current.hostname = hostname.replace(/^join\./, "");
        return current.origin;
    } catch {
        return null;
    }
}

export function isSafeRelativeRedirect(raw: string): boolean {
    return raw.startsWith("/") && !raw.startsWith("//");
}

function isAdmissionRedirectPath(pathname: string): boolean {
    return /^\/verify\/[^/]+\/?$/.test(pathname);
}

function isJoinSelfServiceRedirectPath(pathname: string): boolean {
    return /^\/start\/?$/.test(pathname);
}

function shouldKeepCurrentBusinessOrigin(pathname: string): boolean {
    return isAdmissionRedirectPath(pathname) ||
        isJoinSelfServiceRedirectPath(pathname);
}

function allowedPostLoginRedirectOrigins(): Set<string> {
    const origins = new Set<string>();
    if (typeof window !== "undefined") {
        origins.add(window.location.origin);
    }

    const webOrigin = configuredWebOrigin();
    if (webOrigin) origins.add(webOrigin);

    return origins;
}

export function sanitizePostLoginRedirect(
    redirect: string | null | undefined,
): string | undefined {
    if (!redirect) return undefined;

    if (isSafeRelativeRedirect(redirect)) {
        return redirect;
    }

    if (typeof window === "undefined") return undefined;

    try {
        const parsed = new URL(redirect, window.location.origin);
        if (
            (parsed.protocol === "https:" || parsed.protocol === "http:") &&
            allowedPostLoginRedirectOrigins().has(parsed.origin)
        ) {
            return parsed.toString();
        }
    } catch {
        return undefined;
    }

    return undefined;
}

export function resolvePostLoginRedirectTarget(redirect?: string): string | undefined {
    if (typeof window === "undefined") {
        return redirect;
    }

    const sanitized = sanitizePostLoginRedirect(redirect);
    if (!sanitized) {
        return absoluteURLOnPreferredOrigin(window.location.href, preferredPostLoginRedirectOrigin());
    }

    const parsed = new URL(sanitized, window.location.origin);
    if (
        parsed.origin === window.location.origin &&
        shouldKeepCurrentBusinessOrigin(parsed.pathname)
    ) {
        return parsed.toString();
    }

    return absoluteURLOnPreferredOrigin(sanitized, preferredPostLoginRedirectOrigin());
}

export function absoluteURLOnPreferredOrigin(
    path: string,
    preferredOrigin?: string | null,
): string {
    if (typeof window === "undefined") return path;

    const currentOrigin = window.location.origin;
    const parsed = new URL(path, preferredOrigin || currentOrigin);
    if (
        preferredOrigin &&
        parsed.origin === currentOrigin &&
        parsed.origin !== preferredOrigin
    ) {
        return new URL(
            `${parsed.pathname}${parsed.search}${parsed.hash}`,
            preferredOrigin,
        ).toString();
    }
    return parsed.toString();
}

export function absoluteURLOnCurrentOrigin(path: string): string {
    return absoluteURLOnPreferredOrigin(path);
}

export function accountCenterURL(path: string): string {
    return absoluteURLOnPreferredOrigin(path, configuredWebOrigin());
}

export function accountCenterURLWithRedirect(path: string, redirect: string): string {
    if (!isSafeRelativeRedirect(redirect)) {
        return accountCenterURL(path);
    }

    const separator = path.includes("?") ? "&" : "?";
    const params = new URLSearchParams({ redirect });
    return accountCenterURL(`${path}${separator}${params.toString()}`);
}

export function isAccountCenterPath(pathname: string): boolean {
    return pathname === "/identity" ||
        pathname === "/connect" ||
        pathname === "/developers/apps" ||
        pathname.startsWith("/developers/apps/") ||
        pathname.startsWith("/account/") ||
        pathname === "/consent" ||
        pathname === "/complete-profile" ||
        pathname === "/user/authorized-apps" ||
        pathname === "/user/identity-verification" ||
        pathname === "/user/student-verification" ||
        pathname === "/user/phone-binding" ||
        pathname === "/user/qq-binding" ||
        pathname === "/user/academic-info";
}

export function accountCenterURLForHref(href: string): string | null {
    if (typeof window === "undefined") return null;

    try {
        const parsed = new URL(href, window.location.origin);
        const accountOrigin = configuredWebOrigin();
        if (accountOrigin && parsed.origin === accountOrigin) {
            return parsed.toString();
        }
        if (parsed.origin !== window.location.origin || !isAccountCenterPath(parsed.pathname)) {
            return null;
        }
        return accountCenterURL(`${parsed.pathname}${parsed.search}${parsed.hash}`);
    } catch {
        return null;
    }
}

export function navigateToExternalURL(url: string): void {
    window.location.assign(url);
}

function preferredPostLoginRedirectOrigin(): string | null {
    if (typeof window === "undefined") return null;

    return configuredWebOrigin();
}
