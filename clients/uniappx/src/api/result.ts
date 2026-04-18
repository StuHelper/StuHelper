import { useAuthStore } from "@/stores/auth";
import { translate } from "@/i18n";
import {
    extractOptionalResultData,
    extractResultData,
    extractResultErrorCode,
    extractResultList,
    isResultFailure,
    parseApiError,
    readResultStatus,
    type ApiCallResult,
    type ApiEnvelope,
} from "@stuhelper/shared/api";

export type { ApiCallResult, ApiEnvelope };

export interface StructuredApiError {
    code?: string;
    details?: unknown;
    message?: string;
}

export function extractErrorMessage(result: ApiCallResult<unknown>): string {
    const status = readResultStatus(result);
    const code = extractResultErrorCode(result);
    if (status === 401 || code?.startsWith("A00101"))
        return translate("common.sessionExpired");
    for (const payload of [result.data, result.error]) {
        const parsed = parseApiError(payload);
        if (parsed.message) {
            return parsed.message;
        }
    }
    return translate("common.retryLater");
}

function handleUnauthorized(status?: number) {
    if (status === 401) {
        useAuthStore().clearSession();
    }
}

export function unwrapData<T>(result: ApiCallResult<T>): T {
    const status = readResultStatus(result);
    const payload = extractResultData(result);

    if (typeof payload !== "undefined") {
        return payload;
    }

    handleUnauthorized(status);
    throw new Error(extractErrorMessage(result));
}

export function unwrapOptionalData<T>(result: ApiCallResult<T>): T | null {
    const status = readResultStatus(result);
    const payload = extractOptionalResultData(result);

    if (typeof payload !== "undefined") {
        return payload;
    }

    if (status === 401 || status === 404) {
        handleUnauthorized(status);
        return null;
    }

    throw new Error(extractErrorMessage(result));
}

export function assertMutationSuccess(result: ApiCallResult<unknown>): void {
    const status = readResultStatus(result);
    if (status === 401) {
        handleUnauthorized(status);
    }

    if (isResultFailure(result)) {
        throw new Error(extractErrorMessage(result));
    }
}

export function unwrapListData<T>(
    result: ApiCallResult<{ list?: T[]; total?: number }>,
): {
    list: T[];
    total: number;
} {
    const payload = extractResultList(result);
    if (payload) {
        return payload;
    }
    throw new Error(extractErrorMessage(result));
}
