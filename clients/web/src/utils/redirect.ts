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

export function configuredIdentityOrigin(): string | null {
    if (typeof window === "undefined") return null;
    return normalizeConfiguredHTTPOrigin(import.meta.env.VITE_IDENTITY_URL, window.location.origin);
}

export function configuredWebOrigin(): string | null {
    if (typeof window === "undefined") return null;
    return normalizeConfiguredHTTPOrigin(import.meta.env.VITE_WEB_URL, window.location.origin);
}

export function isSafeRelativeRedirect(raw: string): boolean {
    return raw.startsWith("/") && !raw.startsWith("//");
}

function allowedPostLoginRedirectOrigins(): Set<string> {
    const origins = new Set<string>();
    if (typeof window !== "undefined") {
        origins.add(window.location.origin);
    }

    const webOrigin = configuredWebOrigin();
    if (webOrigin) origins.add(webOrigin);

    const identityOrigin = configuredIdentityOrigin();
    if (identityOrigin) origins.add(identityOrigin);

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

    const identityRedirect = identityPortalURLForHref(sanitized);
    if (identityRedirect) {
        return identityRedirect;
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

export function identityPortalURL(path: string): string {
    return absoluteURLOnPreferredOrigin(path, configuredIdentityOrigin());
}

export function isIdentityPortalPath(pathname: string): boolean {
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

export function identityPortalURLForHref(href: string): string | null {
    if (typeof window === "undefined") return null;

    try {
        const parsed = new URL(href, window.location.origin);
        const identityOrigin = configuredIdentityOrigin();
        if (identityOrigin && parsed.origin === identityOrigin) {
            return parsed.toString();
        }
        if (parsed.origin !== window.location.origin || !isIdentityPortalPath(parsed.pathname)) {
            return null;
        }
        return identityPortalURL(`${parsed.pathname}${parsed.search}${parsed.hash}`);
    } catch {
        return null;
    }
}

export function navigateToExternalURL(url: string): void {
    window.location.assign(url);
}

function preferredPostLoginRedirectOrigin(): string | null {
    if (typeof window === "undefined") return null;

    const identityOrigin = configuredIdentityOrigin();
    if (identityOrigin && window.location.origin === identityOrigin) {
        return identityOrigin;
    }

    return configuredWebOrigin();
}
