/**
 * Current verification projections used by account, review and admission UI.
 *
 * Student eligibility, phone possession and QQ binding are deliberately kept
 * as three separate facts. No legacy identity/profile endpoint is read here.
 */
import { computed, ref } from "vue";
import { defineStore, getActivePinia } from "pinia";
import type { components } from "@stuhelper/shared/types";

import { api } from "@/api";
import { safeOnScopeDispose } from "@/stores/safeScopeDispose";
import { registerSessionResetHandler } from "@/stores/sessionOrchestrator";

type UserSurface = components["schemas"]["UserSurface"];
type PhoneStatus = components["schemas"]["PhoneStatus"];
type QQBinding = components["schemas"]["QQBinding"];
type QQBindingCode = components["schemas"]["QQBindingCode"];

const STUDENT_STATUS_VALUES = new Set(["none", "approved"]);
const PHONE_STATUS_VALUES = new Set([
    "unbound",
    "syncing",
    "verified",
    "review_required",
]);
const PHONE_METHOD_VALUES = new Set([
    "school_roster_phone_match",
    "sms_possession",
]);

export const useVerificationStore = defineStore("verification", () => {
    const pinia = getActivePinia();
    if (!pinia) {
        throw new Error("verification store requires an active Pinia instance");
    }

    const userSurface = ref<UserSurface | null>(null);
    const phoneStatus = ref<PhoneStatus | null>(null);
    const qqBinding = ref<QQBinding | null>(null);
    const qqBindingCode = ref<QQBindingCode | null>(null);
    const loading = ref(false);

    const studentVerified = computed(
        () => userSurface.value?.studentVerificationStatus === "approved",
    );
    const phoneVerified = computed(
        () => phoneStatus.value?.publishingRequirementSatisfied === true,
    );
    const qqBound = computed(() => qqBinding.value !== null);
    const canViewFullReviews = computed(() => studentVerified.value);

    const fetchStatus = async () => {
        loading.value = true;
        try {
            const [nextSurface, nextPhone, nextQQBinding] = await Promise.all([
                fetchUserSurfaceValue(),
                fetchPhoneStatusValue(),
                fetchQQBindingValue(),
            ]);
            userSurface.value = nextSurface;
            phoneStatus.value = nextPhone;
            qqBinding.value = nextQQBinding;
            if (nextQQBinding) {
                qqBindingCode.value = null;
            }
        } finally {
            loading.value = false;
        }
    };

    const fetchUserSurface = async () => {
        const next = await fetchUserSurfaceValue();
        userSurface.value = next;
        return next;
    };

    const fetchPhoneStatus = async () => {
        const next = await fetchPhoneStatusValue();
        phoneStatus.value = next;
        return next;
    };

    const fetchQQBinding = async () => {
        const next = await fetchQQBindingValue();
        qqBinding.value = next;
        if (next) {
            qqBindingCode.value = null;
        }
        return next;
    };

    const createQQBindingCode = async () => {
        const response = await api.identity.createQQBindingCode();
        const next = readQQBindingCode(
            readResponseData(response, "Invalid QQ binding code response"),
            "Invalid QQ binding code response",
        );
        qqBindingCode.value = next;
        return next;
    };

    const reset = () => {
        userSurface.value = null;
        phoneStatus.value = null;
        qqBinding.value = null;
        qqBindingCode.value = null;
        loading.value = false;
    };

    const unregisterSessionReset = registerSessionResetHandler(
        "verification",
        reset,
        pinia,
    );
    safeOnScopeDispose(unregisterSessionReset);

    return {
        userSurface,
        phoneStatus,
        qqBinding,
        qqBindingCode,
        loading,
        studentVerified,
        phoneVerified,
        qqBound,
        canViewFullReviews,
        fetchStatus,
        fetchUserSurface,
        fetchPhoneStatus,
        fetchQQBinding,
        createQQBindingCode,
        reset,
    };
});

async function fetchUserSurfaceValue(): Promise<UserSurface> {
    const response = await api.identity.getUserSurface();
    return readUserSurface(
        readResponseData(response, "Invalid user surface response"),
        "Invalid user surface response",
    );
}

async function fetchPhoneStatusValue(): Promise<PhoneStatus> {
    const response = await api.studentVerification.getPhoneStatus();
    return readPhoneStatus(
        readResponseData(response, "Invalid phone status response"),
        "Invalid phone status response",
    );
}

async function fetchQQBindingValue(): Promise<QQBinding | null> {
    const response = await api.identity.getQQBinding();
    const value = readResponseData(response, "Invalid QQ binding response");
    return value === null
        ? null
        : readQQBinding(value, "Invalid QQ binding response");
}

function readResponseData(response: unknown, message: string): unknown {
    if (!isRecord(response) || !isRecord(response.data) || !("data" in response.data)) {
        throw new Error(message);
    }
    return response.data.data;
}

function readUserSurface(value: unknown, message: string): UserSurface {
    if (!isRecord(value)) throw new Error(message);
    const status = readString(value, "studentVerificationStatus", message);
    if (!STUDENT_STATUS_VALUES.has(status)) throw new Error(message);
    const capabilities = value.capabilities;
    if (!Array.isArray(capabilities) || capabilities.some((item) => typeof item !== "string")) {
        throw new Error(message);
    }
    return {
        displayName: readString(value, "displayName", message),
        avatarURL: readOptionalString(value, "avatarURL", message),
        phone: readNullableString(value, "phone", message),
        studentVerificationStatus: status as UserSurface["studentVerificationStatus"],
        phoneBound: readBoolean(value, "phoneBound", message),
        capabilities,
    };
}

function readPhoneStatus(value: unknown, message: string): PhoneStatus {
    if (!isRecord(value)) throw new Error(message);
    const state = readString(value, "state", message);
    if (!PHONE_STATUS_VALUES.has(state)) throw new Error(message);
    const method = readNullableString(value, "method", message);
    if (method != null && !PHONE_METHOD_VALUES.has(method)) throw new Error(message);
    return {
        state: state as PhoneStatus["state"],
        maskedPhone: readNullableString(value, "maskedPhone", message),
        method: method as PhoneStatus["method"],
        verifiedAt: readNullableString(value, "verifiedAt", message),
        expiresAt: readNullableString(value, "expiresAt", message),
        publishingRequirementSatisfied: readBoolean(
            value,
            "publishingRequirementSatisfied",
            message,
        ),
        revision: readNumber(value, "revision", message),
    };
}

function readQQBinding(value: unknown, message: string): QQBinding {
    if (!isRecord(value)) throw new Error(message);
    return {
        userID: readNumber(value, "userID", message),
        qqID: readString(value, "qqID", message),
        boundAt: readString(value, "boundAt", message),
        createdAt: readString(value, "createdAt", message),
        updatedAt: readString(value, "updatedAt", message),
    };
}

function readQQBindingCode(value: unknown, message: string): QQBindingCode {
    if (!isRecord(value)) throw new Error(message);
    return {
        code: readString(value, "code", message),
        expiresAt: readString(value, "expiresAt", message),
    };
}

function isRecord(value: unknown): value is Record<string, unknown> {
    return value !== null && typeof value === "object";
}

function readString(
    value: Record<string, unknown>,
    key: string,
    message: string,
): string {
    const field = value[key];
    if (typeof field !== "string") throw new Error(message);
    return field;
}

function readOptionalString(
    value: Record<string, unknown>,
    key: string,
    message: string,
): string | undefined {
    const field = value[key];
    if (field === undefined) return undefined;
    if (typeof field !== "string") throw new Error(message);
    return field;
}

function readNullableString(
    value: Record<string, unknown>,
    key: string,
    message: string,
): string | null | undefined {
    const field = value[key];
    if (field === undefined || field === null) return field;
    if (typeof field !== "string") throw new Error(message);
    return field;
}

function readBoolean(
    value: Record<string, unknown>,
    key: string,
    message: string,
): boolean {
    const field = value[key];
    if (typeof field !== "boolean") throw new Error(message);
    return field;
}

function readNumber(
    value: Record<string, unknown>,
    key: string,
    message: string,
): number {
    const field = value[key];
    if (typeof field !== "number" || !Number.isFinite(field)) throw new Error(message);
    return field;
}
