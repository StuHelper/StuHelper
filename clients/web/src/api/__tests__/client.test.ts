import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const mockClearSession = vi.fn();
const mockTokenExpirySet = vi.fn();
const mockHasStoredSessionHint = vi.fn();

vi.mock("@stuhelper/shared/api", async (importOriginal) => {
    const actual = await importOriginal<typeof import("@stuhelper/shared/api")>();
    return {
        ...actual,
        isRecord: (value: unknown): value is Record<string, unknown> =>
            !!value && typeof value === "object",
        parseApiError: (payload: unknown) => {
            if (
                payload &&
                typeof payload === "object" &&
                "error" in payload &&
                payload.error &&
                typeof payload.error === "object"
            ) {
                const error = payload.error as Record<string, unknown>;
                return {
                    code: typeof error.code === "string" ? error.code : "",
                    message: typeof error.message === "string" ? error.message : "",
                    details:
                        error.details && typeof error.details === "object"
                            ? (error.details as Record<string, unknown>)
                            : undefined,
                };
            }
            if (payload && typeof payload === "object") {
                const record = payload as Record<string, unknown>;
                return {
                    code: typeof record.code === "string" ? record.code : "",
                    message:
                        typeof record.message === "string" ? record.message : "",
                    details:
                        record.details && typeof record.details === "object"
                            ? (record.details as Record<string, unknown>)
                            : undefined,
                };
            }
            return { code: "", message: "", details: undefined };
        },
    };
});

vi.mock("@/stores/auth", () => ({
    useAuthStore: () => ({
        clearSession: mockClearSession,
    }),
}));

vi.mock("@/utils/auth", () => ({
    tokenExpiry: {
        set: mockTokenExpirySet,
    },
}));

vi.mock("@/utils/sessionHint", () => ({
    hasStoredSessionHint: mockHasStoredSessionHint,
    readCookie: (name: string) => {
        const cookies = document.cookie ? document.cookie.split(";") : [];
        const target = `${encodeURIComponent(name)}=`;
        for (const raw of cookies) {
            const cookie = raw.trim();
            if (cookie.startsWith(target)) {
                return decodeURIComponent(cookie.slice(target.length));
            }
        }
        return null;
    },
}));

vi.mock("@/api/errors", () => {
    class MockApiError extends Error {
        code: string;
        status?: number;
        details?: Record<string, unknown>;
        requestID?: string;

        constructor(options: {
            message: string;
            code: string;
            status?: number;
            details?: Record<string, unknown>;
            requestID?: string;
        }) {
            super(options.message);
            this.name = "ApiError";
            this.code = options.code;
            this.status = options.status;
            this.details = options.details;
            this.requestID = options.requestID;
        }

        getUserMessage() {
            return this.message;
        }
    }

    return {
        ApiError: MockApiError,
        httpStatusToDefaultCode: (status: number) => {
            const map: Record<number, string> = {
                401: "A0010100",
                403: "A0010200",
                500: "B0000001",
            };
            return map[status] ?? "B0000001";
        },
        isAuthError: (code: string) => code.startsWith("A001"),
        isCsrfError: (code: string) =>
            code === "A0010202" || code === "A0010203",
        isNetworkError: (code: string) =>
            code === "NETWORK_ERROR" || code === "OFFLINE" || code === "TIMEOUT",
    };
});

describe("browser API client", () => {
    const fetchMock = vi.fn<typeof fetch>();

    beforeEach(() => {
        vi.resetModules();
        vi.useRealTimers();
        fetchMock.mockReset();
        mockClearSession.mockReset();
        mockHasStoredSessionHint.mockReset();
        mockTokenExpirySet.mockReset();
        mockHasStoredSessionHint.mockReturnValue(true);
        vi.stubEnv("VITE_API_URL", "/api");

        Object.defineProperty(globalThis, "window", {
            configurable: true,
            value: {
                fetch: fetchMock,
                location: { origin: "http://127.0.0.1:3000" },
                setTimeout,
                clearTimeout,
            },
        });
        Object.defineProperty(globalThis, "document", {
            configurable: true,
            value: {
                cookie: "csrf_token=test-csrf",
            },
        });
        Object.defineProperty(globalThis, "navigator", {
            configurable: true,
            value: { onLine: true },
        });
    });

    afterEach(() => {
        vi.useRealTimers();
        vi.unstubAllGlobals();
    });

    it("resolves the browser API base to the current origin when configured with a relative path", async () => {
        const { __testing__, apiClientOptions } = await import("../client");
        const warnSpy = vi.spyOn(console, "warn").mockImplementation(() => undefined);

        try {
            expect(__testing__.resolveApiBaseUrl("/api")).toBe(
                "http://127.0.0.1:3000",
            );
            expect(__testing__.resolveApiBaseUrl("/api/v1")).toBe(
                "http://127.0.0.1:3000",
            );
            expect(__testing__.resolveApiURL("/api/v1/auth/callback").toString()).toBe(
                "http://127.0.0.1:3000/api/v1/auth/callback",
            );
            expect(apiClientOptions.baseUrl).toBe("http://127.0.0.1:3000");
            expect(warnSpy).not.toHaveBeenCalled();
        } finally {
            warnSpy.mockRestore();
        }
    });

    it("warns once in development when VITE_API_URL is missing", async () => {
        vi.stubEnv("VITE_API_URL", "");
        const warnSpy = vi.spyOn(console, "warn").mockImplementation(() => undefined);

        try {
            const { __testing__ } = await import("../client");

            expect(__testing__.resolveApiBaseUrl("")).toBe("http://127.0.0.1:3000");
            expect(__testing__.resolveApiBaseUrl("")).toBe("http://127.0.0.1:3000");
            expect(warnSpy).toHaveBeenCalledTimes(1);
            expect(warnSpy).toHaveBeenCalledWith(
                "[api] VITE_API_URL not set, falling back to window.location.origin",
            );
        } finally {
            warnSpy.mockRestore();
        }
    });

    it("strips duplicated API prefixes from absolute base URLs", async () => {
        const { __testing__ } = await import("../client");

        expect(__testing__.resolveApiBaseUrl("https://api.example.com/api")).toBe(
            "https://api.example.com",
        );
        expect(
            __testing__.resolveApiBaseUrl("https://api.example.com/api/v1"),
        ).toBe("https://api.example.com");
    });

    it("injects CSRF header for unsafe requests", async () => {
        const { __testing__ } = await import("../client");

        const request = __testing__.withBrowserSecurity(
            new Request("http://127.0.0.1:3000/api/v1/course/review/reviews", {
                method: "POST",
            }),
        );

        expect(request.credentials).toBe("include");
        expect(request.headers.get("X-CSRF-Token")).toBe("test-csrf");
    });

    it("preserves top-level request headers in browser transport", async () => {
        fetchMock.mockResolvedValueOnce(
            new Response(JSON.stringify({ data: { ok: true } }), {
                status: 200,
                headers: { "Content-Type": "application/json" },
            }),
        );

        const { apiClient } = await import("../client");
        await apiClient.POST("/api/v1/open-platform/resources/access/check" as never, {
            body: {
                action: "read",
                resourceID: "resource-42",
                resourceType: "resource_item",
            },
            headers: {
                Authorization: "Bearer resource-access-token",
            },
        } as never);

        const request = fetchMock.mock.calls[0][0] as Request;
        expect(request.headers.get("Authorization")).toBe(
            "Bearer resource-access-token",
        );
        expect(request.headers.get("X-CSRF-Token")).toBe("test-csrf");
    });

    it("uses JSON content type only for JSON request bodies", async () => {
        fetchMock.mockResolvedValue(
            new Response(JSON.stringify({ data: { ok: true } }), {
                status: 200,
                headers: { "Content-Type": "application/json" },
            }),
        );

        const { apiClient } = await import("../client");
        await apiClient.POST("/api/v1/open-platform/resources/access/check" as never, {
            body: {
                action: "read",
            },
        } as never);

        const jsonRequest = fetchMock.mock.calls[0][0] as Request;
        expect(jsonRequest.headers.get("Content-Type")).toBe("application/json");

        const formData = new FormData();
        formData.set("file", new Blob(["hello"]), "hello.txt");
        await apiClient.POST("/api/v1/admission/materials" as never, {
            body: formData,
        } as never);

        const formRequest = fetchMock.mock.calls[1][0] as Request;
        expect(formRequest.headers.get("Content-Type")).toMatch(
            /^multipart\/form-data; boundary=/,
        );
        expect(formRequest.headers.get("Content-Type")).not.toBe("application/json");
        expect(formRequest.headers.get("X-CSRF-Token")).toBe("test-csrf");
    });

    it("refreshes once and retries the original request after a 401", async () => {
        fetchMock
            .mockResolvedValueOnce(new Response(null, { status: 401 }))
            .mockResolvedValueOnce(
                new Response(JSON.stringify({ data: { expiresIn: 120 } }), {
                    status: 200,
                    headers: { "Content-Type": "application/json" },
                }),
            )
            .mockResolvedValueOnce(
                new Response(JSON.stringify({ success: true }), {
                    status: 200,
                    headers: { "Content-Type": "application/json" },
                }),
            );

        const { apiClientOptions } = await import("../client");
        const response = await apiClientOptions.fetch(
            new Request("http://127.0.0.1:3000/api/v1/course/review/reviews", {
                method: "POST",
            }),
        );

        expect(response.status).toBe(200);
        expect(fetchMock).toHaveBeenCalledTimes(3);
        const refreshRequest = fetchMock.mock.calls[1][0] as Request;
        expect(refreshRequest.url).toBe(
            "http://127.0.0.1:3000/api/v1/auth/refresh",
        );
        expect(refreshRequest.method).toBe("POST");
        expect(refreshRequest.headers.get("X-CSRF-Token")).toBe("test-csrf");
        expect(mockTokenExpirySet).toHaveBeenCalledWith(120);
        expect(mockClearSession).not.toHaveBeenCalled();
    });

    it("keeps refresh CSRF failures recoverable without clearing the local session", async () => {
        fetchMock
            .mockResolvedValueOnce(
                new Response(
                    JSON.stringify({
                        error: { code: "A0010100", message: "login required" },
                    }),
                    {
                        status: 401,
                        headers: { "Content-Type": "application/json" },
                    },
                ),
            )
            .mockResolvedValueOnce(
                new Response(
                    JSON.stringify({
                        error: { code: "A0010202", message: "csrf invalid" },
                    }),
                    {
                        status: 403,
                        headers: { "Content-Type": "application/json" },
                    },
                ),
            );

        const { apiClient } = await import("../client");

        await expect(apiClient.GET("/api/v1/auth/me" as never)).rejects.toMatchObject({
            code: "A0010202",
            message: "csrf invalid",
            status: 403,
        });
        expect(fetchMock).toHaveBeenCalledTimes(2);
        const refreshRequest = fetchMock.mock.calls[1][0] as Request;
        expect(refreshRequest.url).toBe(
            "http://127.0.0.1:3000/api/v1/auth/refresh",
        );
        expect(refreshRequest.headers.get("X-CSRF-Token")).toBe("test-csrf");
        expect(mockClearSession).not.toHaveBeenCalled();
    });

    it("returns a real fetch Response when automatic refresh fails", async () => {
        fetchMock
            .mockResolvedValueOnce(
                new Response(
                    JSON.stringify({
                        error: { code: "A0010100", message: "login required" },
                    }),
                    {
                        status: 401,
                        headers: { "Content-Type": "application/json" },
                    },
                ),
            )
            .mockResolvedValueOnce(
                new Response(
                    JSON.stringify({
                        error: { code: "A0010202", message: "csrf invalid" },
                    }),
                    {
                        status: 403,
                        headers: { "Content-Type": "application/json" },
                    },
                ),
            );

        const { apiClientOptions } = await import("../client");
        const response = await apiClientOptions.fetch(
            new Request("http://127.0.0.1:3000/api/v1/user/profile", {
                method: "GET",
            }),
        );

        expect(response).toBeInstanceOf(Response);
        expect(response.status).toBe(403);
        await expect(response.json()).resolves.toEqual({
            error: {
                code: "A0010202",
                message: "csrf invalid",
            },
        });
        expect(mockClearSession).not.toHaveBeenCalled();
    });

    it("does not refresh a 401 response without a stored browser session hint", async () => {
        mockHasStoredSessionHint.mockReturnValue(false);
        fetchMock.mockResolvedValueOnce(
            new Response(
                JSON.stringify({
                    error: { code: "A0010100", message: "login required" },
                }),
                {
                    status: 401,
                    headers: { "Content-Type": "application/json" },
                },
            ),
        );

        const { apiClientOptions } = await import("../client");
        const response = await apiClientOptions.fetch(
            new Request("http://127.0.0.1:3000/api/v1/course/review/reviews", {
                method: "GET",
            }),
        );

        expect(response.status).toBe(401);
        expect(fetchMock).toHaveBeenCalledTimes(1);
    });

    it("clears the local session when refresh is rejected as unauthorized", async () => {
        fetchMock
            .mockResolvedValueOnce(new Response(null, { status: 401 }))
            .mockResolvedValueOnce(
                new Response(
                    JSON.stringify({
                        error: { code: "UNAUTHORIZED", message: "unauthorized" },
                    }),
                    {
                        status: 401,
                        headers: { "Content-Type": "application/json" },
                    },
                ),
            );

        const { apiClientOptions } = await import("../client");
        const response = await apiClientOptions.fetch(
            new Request("http://127.0.0.1:3000/api/v1/user/profile", {
                method: "GET",
            }),
        );

        expect(response.status).toBe(401);
        expect(fetchMock).toHaveBeenCalledTimes(2);
        expect(mockClearSession).toHaveBeenCalledTimes(1);
    });

    it("maps unauthorized refresh without response headers to an ApiError", async () => {
        fetchMock
            .mockResolvedValueOnce(
                new Response(
                    JSON.stringify({
                        error: { code: "A0010100", message: "login required" },
                    }),
                    {
                        status: 401,
                        headers: { "Content-Type": "application/json" },
                    },
                ),
            )
            .mockResolvedValueOnce(
                new Response(
                    JSON.stringify({
                        error: { code: "A0010100", message: "refresh required" },
                    }),
                    {
                        status: 401,
                        headers: { "Content-Type": "application/json" },
                    },
                ),
            );

        const { apiClient } = await import("../client");

        await expect(apiClient.GET("/api/v1/auth/me")).rejects.toMatchObject({
            code: "A0010100",
            message: "login required",
            status: 401,
        });
        expect(mockClearSession).toHaveBeenCalledTimes(1);
    });

    it("aborts ordinary requests after the default timeout window", async () => {
        vi.useFakeTimers();
        Object.assign(window, {
            setTimeout,
            clearTimeout,
        });
        fetchMock.mockImplementation(
            (request: RequestInfo | URL) =>
                new Promise<Response>((_, reject) => {
                    const req = request as Request;
                    req.signal.addEventListener("abort", () => {
                        reject(new DOMException("request timeout", "AbortError"));
                    });
                }),
        );

        const { __testing__ } = await import("../client");
        const promise = __testing__.performBrowserFetch(
            new Request("http://127.0.0.1:3000/api/v1/course/courses", {
                method: "GET",
            }),
        );
        const assertion = expect(promise).rejects.toMatchObject({
            name: "AbortError",
        });

        await vi.advanceTimersByTimeAsync(__testing__.DEFAULT_REQUEST_TIMEOUT_MS);
        await assertion;
    });

    it("deduplicates concurrent GET requests outside refresh", async () => {
        let resolveFetch: ((value: Response) => void) | null = null;
        fetchMock.mockImplementation(
            () =>
                new Promise<Response>((resolve) => {
                    resolveFetch = resolve;
                }),
        );

        const { apiClient } = await import("../client");
        const pendingA = apiClient.GET("/api/v1/course/courses");
        const pendingB = apiClient.GET("/api/v1/course/courses");

        expect(fetchMock).toHaveBeenCalledTimes(1);

        resolveFetch?.(
            new Response(JSON.stringify({ data: [{ id: 1 }] }), {
                status: 200,
                headers: { "Content-Type": "application/json" },
            }),
        );

        const [resultA, resultB] = await Promise.all([pendingA, pendingB]);
        expect(resultA.data).toEqual(resultB.data);
        expect(fetchMock).toHaveBeenCalledTimes(1);
    });

    it("does not deduplicate GET requests that carry an AbortSignal", async () => {
        fetchMock.mockResolvedValue(
            new Response(JSON.stringify({ data: [] }), {
                status: 200,
                headers: { "Content-Type": "application/json" },
            }),
        );

        const { apiClient } = await import("../client");
        const controllerA = new AbortController();
        const controllerB = new AbortController();

        await Promise.all([
            apiClient.GET("/api/v1/course/courses", { signal: controllerA.signal }),
            apiClient.GET("/api/v1/course/courses", { signal: controllerB.signal }),
        ]);

        expect(fetchMock).toHaveBeenCalledTimes(2);
    });
});
