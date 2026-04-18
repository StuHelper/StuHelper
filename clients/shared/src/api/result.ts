import { parseApiError } from "./errors";

export interface ApiEnvelope<T> {
    success?: boolean;
    data?: T;
    error?: unknown;
    message?: string;
    code?: string;
}

export interface ApiCallResult<T = unknown> {
    data?: ApiEnvelope<T>;
    error?: unknown;
    response?: {
        status?: number;
    };
}

export function readResultStatus(
    result: ApiCallResult<unknown>,
): number | undefined {
    return result.response?.status;
}

export function extractResultErrorCode(
    result: ApiCallResult<unknown>,
): string | undefined {
    if (result.data) {
        const parsed = parseApiError(result.data);
        if (parsed.code) return parsed.code;
    }

    if (result.error) {
        const parsed = parseApiError(result.error);
        if (parsed.code) return parsed.code;

        if (
            typeof result.error === "object" &&
            result.error !== null &&
            "error" in result.error
        ) {
            const nested = (result.error as Record<string, unknown>).error;
            const nestedParsed = parseApiError({ error: nested });
            if (nestedParsed.code) return nestedParsed.code;
        }
    }

    return undefined;
}

export function extractResultData<T>(result: ApiCallResult<T>): T | undefined {
    const payload = result.data;
    if (
        payload &&
        typeof payload === "object" &&
        "data" in payload &&
        payload.data !== undefined
    ) {
        return payload.data as T;
    }
    return undefined;
}

export function extractOptionalResultData<T>(
    result: ApiCallResult<T>,
): T | null | undefined {
    const payload = result.data;
    if (payload && typeof payload === "object" && "data" in payload) {
        return (payload.data ?? null) as T | null;
    }
    return undefined;
}

export function extractResultList<T>(
    result: ApiCallResult<{ list?: T[]; total?: number }>,
):
    | {
          list: T[];
          total: number;
      }
    | undefined {
    const payload = extractResultData(result);
    if (!payload) {
        return undefined;
    }
    return {
        list: Array.isArray(payload.list) ? payload.list : [],
        total: typeof payload.total === "number" ? payload.total : 0,
    };
}

export function isResultFailure(result: ApiCallResult<unknown>): boolean {
    const status = readResultStatus(result);
    if (status !== undefined && status >= 400) {
        return true;
    }
    if (typeof result.error !== "undefined") {
        return true;
    }
    const payload = result.data;
    return !!(
        payload &&
        typeof payload === "object" &&
        "success" in payload &&
        payload.success === false
    );
}
