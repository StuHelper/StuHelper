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
        return window.location.href;
    }

    return new URL(sanitized, window.location.origin).toString();
}

export function absoluteURLOnCurrentOrigin(path: string): string {
    if (typeof window === "undefined") return path;
    return new URL(path, window.location.origin).toString();
}
