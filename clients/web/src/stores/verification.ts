/**
 * 认证状态管理（实名认证 + 学生认证）
 */
import { defineStore, getActivePinia } from "pinia";
import { computed, ref } from "vue";
import type { components } from "@stuhelper/shared/types";
import { api } from "@/api";
import { getErrorStatus } from "@/api/errors";
import { safeOnScopeDispose } from "@/stores/safeScopeDispose";
import { registerSessionResetHandler } from "@/stores/sessionOrchestrator";

type IdentityInfo = components["schemas"]["UserIdentity"];
type ProfileInfo = components["schemas"]["UserProfile"];
type SchoolConfig = components["schemas"]["SchoolConfig"];
type SubmitIdentityRequest = components["schemas"]["SubmitIdentityRequest"];
type UploadIdentityPhotoRequest =
    components["schemas"]["UploadIdentityPhotoRequest"];
type SubmitStudentVerificationRequest =
    components["schemas"]["SubmitStudentVerificationRequest"];
type StudentEmailAcademicMatchRequest =
    components["schemas"]["StudentEmailAcademicMatchRequest"];
type StudentEmailAcademicMatchResponse =
    components["schemas"]["StudentEmailAcademicMatchResponse"];
type StudentEmailOTPRequest = components["schemas"]["StudentEmailOTPRequest"];
type StudentEmailOTPVerifyRequest =
    components["schemas"]["StudentEmailOTPVerifyRequest"];
type StudentEmailOTPResponse = components["schemas"]["StudentEmailOTPResponse"];
type BindPhoneRequest = components["schemas"]["BindPhoneRequest"];
type QQBindingInfo = components["schemas"]["QQBinding"];
type QQBindingCodeInfo = components["schemas"]["QQBindingCode"];
type IdentityPhotoUploadResult =
    components["schemas"]["IdentityPhotoUploadResult"];
type ManualFieldDescriptor = components["schemas"]["ManualFieldDescriptor"];

const IDENTITY_DOC_TYPE_VALUES = new Set([
    "MAINLAND_ID",
    "HK_MACAU",
    "TW",
    "PASSPORT",
]);
const IDENTITY_VERIFY_METHOD_VALUES = new Set([
    "academic_db_match",
    "tencent_cloud",
    "manual",
]);
const PROFILE_VERIFICATION_STATUS_VALUES = new Set([
    "unverified",
    "pending",
    "verified",
    "rejected",
]);
const PROFILE_VERIFICATION_METHOD_VALUES = new Set(["ldap", "manual", "school_email_otp"]);
const SCHOOL_VERIFICATION_METHOD_VALUES = new Set(["ldap", "manual"]);
const MANUAL_FIELD_TYPE_VALUES = new Set(["text", "textarea", "select", "date"]);

export const useVerificationStore = defineStore("verification", () => {
    const pinia = getActivePinia();
    if (!pinia) {
        throw new Error("verification store requires an active Pinia instance");
    }

    // 状态
    const identity = ref<IdentityInfo | null>(null);
    const profile = ref<ProfileInfo | null>(null);
    const qqBinding = ref<QQBindingInfo | null>(null);
    const qqBindingCode = ref<QQBindingCodeInfo | null>(null);
    const schools = ref<SchoolConfig[]>([]);
    const loading = ref(false);

    // 计算属性
    const identityVerified = computed(() => identity.value?.verified === true);
    const studentVerified = computed(
        () => profile.value?.verificationStatus === "verified",
    );
    const qqBound = computed(() => qqBinding.value !== null);
    const canViewFullReviews = computed(() => studentVerified.value);

    // 并行获取实名认证和学生认证状态
    const fetchStatus = async () => {
        loading.value = true;
        try {
            const [identityData, profileData, qqBindingData] = await Promise.all([
                loadNullableResource(
                    () => api.identity.getIdentity(),
                    readIdentityPayload,
                    "Invalid identity response",
                ),
                loadNullableResource(
                    () => api.identity.getProfile(),
                    readProfilePayload,
                    "Invalid profile response",
                ),
                loadNullableResource(
                    () => api.identity.getQQBinding(),
                    readQQBindingPayload,
                    "Invalid QQ binding response",
                ),
            ]);

            identity.value = identityData;
            profile.value = profileData;
            qqBinding.value = qqBindingData;
            if (qqBindingData) {
                qqBindingCode.value = null;
            }
        } finally {
            loading.value = false;
        }
    };

    // 获取学校列表
    const fetchSchools = async () => {
        const res = await api.identity.listSchools();
        schools.value = readSchoolListPayload(
            res.data?.data,
            "Invalid school list response",
        );
    };

    // 提交实名认证
    const submitIdentity = async (data: SubmitIdentityRequest) => {
        const res = await api.identity.submitIdentity(data);
        const nextIdentity = readIdentityPayload(
            res.data?.data,
            "Invalid identity response",
        );
        identity.value = nextIdentity;
        return nextIdentity;
    };

    const uploadIdentityPhoto = async (data: UploadIdentityPhotoRequest) => {
        const res = await api.identity.uploadIdentityPhoto(data);
        const uploaded = readIdentityPhotoUploadPayload(
            res.data?.data,
            "Invalid identity photo upload response",
        );
        return uploaded.key;
    };

    // 学生认证
    const verifyStudent = async (data: SubmitStudentVerificationRequest) => {
        const res = await api.identity.verifyStudent(data);
        const nextProfile = readProfilePayload(
            res.data?.data,
            "Invalid student verification response",
        );
        profile.value = nextProfile;
        return nextProfile;
    };

    const requestStudentEmailOTP = async (
        data: StudentEmailOTPRequest,
    ): Promise<StudentEmailOTPResponse> => {
        const res = await api.identity.requestStudentEmailOTP(data);
        return readStudentEmailOTPPayload(
            res.data?.data,
            "Invalid student email OTP response",
        );
    };

    const matchStudentEmailAcademicStudent = async (
        data: StudentEmailAcademicMatchRequest,
    ): Promise<StudentEmailAcademicMatchResponse> => {
        const res = await api.identity.matchStudentEmailAcademicStudent(data);
        return readStudentEmailAcademicMatchPayload(
            res.data?.data,
            "Invalid student email academic match response",
        );
    };

    const verifyStudentEmailOTP = async (
        data: StudentEmailOTPVerifyRequest,
    ) => {
        const res = await api.identity.verifyStudentEmailOTP(data);
        const nextProfile = readProfilePayload(
            res.data?.data,
            "Invalid student verification response",
        );
        profile.value = nextProfile;
        return nextProfile;
    };

    // 请求绑定手机验证码
    const requestBindPhoneOTP = async (phone: string) => {
        await api.identity.requestBindPhoneOTP(phone);
    };

    // 绑定手机
    const bindPhone = async (data: BindPhoneRequest) => {
        await api.identity.bindPhone(data);
        // 绑定成功后必须刷新 profile；刷新失败应显式抛出，避免 UI 误判为已绑定。
        const profileRes = await api.identity.getProfile();
        profile.value = readProfilePayload(
            profileRes.data?.data,
            "Invalid profile response after phone binding",
        );
    };

    const fetchQQBinding = async () => {
        qqBinding.value = await loadNullableResource(
            () => api.identity.getQQBinding(),
            readQQBindingPayload,
            "Invalid QQ binding response",
        );
        if (qqBinding.value) {
            qqBindingCode.value = null;
        }
        return qqBinding.value;
    };

    const createQQBindingCode = async () => {
        const res = await api.identity.createQQBindingCode();
        qqBindingCode.value = readQQBindingCodePayload(
            res.data?.data,
            "Invalid QQ binding code response",
        );
        return qqBindingCode.value;
    };

    // 重置状态（setup store 不支持 $reset）
    const reset = () => {
        identity.value = null;
        profile.value = null;
        qqBinding.value = null;
        qqBindingCode.value = null;
        schools.value = [];
        loading.value = false;
    };

    const unregisterSessionReset = registerSessionResetHandler(
        "verification",
        reset,
        pinia,
    );
    safeOnScopeDispose(unregisterSessionReset);

    return {
        identity,
        profile,
        qqBinding,
        qqBindingCode,
        schools,
        loading,
        identityVerified,
        studentVerified,
        qqBound,
        canViewFullReviews,
        fetchStatus,
        fetchQQBinding,
        fetchSchools,
        createQQBindingCode,
        submitIdentity,
        uploadIdentityPhoto,
        verifyStudent,
        matchStudentEmailAcademicStudent,
        requestStudentEmailOTP,
        verifyStudentEmailOTP,
        bindPhone,
        requestBindPhoneOTP,
        reset,
    };
});

async function loadNullableResource<T>(
    loader: () => Promise<{ data?: { data?: T | null } }>,
    reader: (value: unknown, message: string) => T,
    message: string,
): Promise<T | null> {
    try {
        const response = await loader();
        return reader(response.data?.data, message);
    } catch (error) {
        if (getErrorStatus(error) === 404) {
            return null;
        }
        throw error;
    }
}

function isRecord(value: unknown): value is Record<string, unknown> {
    return Boolean(value) && typeof value === "object";
}

function readString(
    record: Record<string, unknown>,
    key: string,
    message: string,
): string {
    const value = record[key];
    if (typeof value !== "string") {
        throw new Error(message);
    }
    return value;
}

function readOptionalString(
    record: Record<string, unknown>,
    key: string,
    message: string,
): string | undefined {
    const value = record[key];
    if (value === undefined) {
        return undefined;
    }
    if (typeof value !== "string") {
        throw new Error(message);
    }
    return value;
}

function readNullableString(
    record: Record<string, unknown>,
    key: string,
    message: string,
): string | null | undefined {
    const value = record[key];
    if (value === undefined || value === null) {
        return value;
    }
    if (typeof value !== "string") {
        throw new Error(message);
    }
    return value;
}

function readBoolean(
    record: Record<string, unknown>,
    key: string,
    message: string,
): boolean {
    const value = record[key];
    if (typeof value !== "boolean") {
        throw new Error(message);
    }
    return value;
}

function readOptionalBoolean(
    record: Record<string, unknown>,
    key: string,
    message: string,
): boolean | undefined {
    const value = record[key];
    if (value === undefined) {
        return undefined;
    }
    if (typeof value !== "boolean") {
        throw new Error(message);
    }
    return value;
}

function readInteger(
    record: Record<string, unknown>,
    key: string,
    message: string,
): number {
    const value = record[key];
    if (typeof value !== "number" || !Number.isInteger(value)) {
        throw new Error(message);
    }
    return value;
}

function readNullableInteger(
    record: Record<string, unknown>,
    key: string,
    message: string,
): number | null | undefined {
    const value = record[key];
    if (value === undefined || value === null) {
        return value;
    }
    if (typeof value !== "number" || !Number.isInteger(value)) {
        throw new Error(message);
    }
    return value;
}

function readStringArrayOrNull(
    record: Record<string, unknown>,
    key: string,
    message: string,
): string[] | null | undefined {
    const value = record[key];
    if (value === undefined || value === null) {
        return value;
    }
    if (!Array.isArray(value) || value.some((item) => typeof item !== "string")) {
        throw new Error(message);
    }
    return value;
}

function readEnum<T extends string>(
    record: Record<string, unknown>,
    key: string,
    values: Set<string>,
    message: string,
): T {
    const value = readString(record, key, message);
    if (!values.has(value)) {
        throw new Error(message);
    }
    return value as T;
}

function readNullableEnum<T extends string>(
    record: Record<string, unknown>,
    key: string,
    values: Set<string>,
    message: string,
): T | null | undefined {
    const value = record[key];
    if (value === undefined || value === null) {
        return value;
    }
    if (typeof value !== "string" || !values.has(value)) {
        throw new Error(message);
    }
    return value as T;
}

function readIdentityPayload(value: unknown, message: string): IdentityInfo {
    if (!isRecord(value)) {
        throw new Error(message);
    }

    return {
        userID: readInteger(value, "userID", message),
        docType: readEnum<IdentityInfo["docType"]>(
            value,
            "docType",
            IDENTITY_DOC_TYPE_VALUES,
            message,
        ),
        realName: readString(value, "realName", message),
        verified: readBoolean(value, "verified", message),
        verifyMethod: readNullableEnum<NonNullable<IdentityInfo["verifyMethod"]>>(
            value,
            "verifyMethod",
            IDENTITY_VERIFY_METHOD_VALUES,
            message,
        ),
        reviewedAt: readNullableString(value, "reviewedAt", message),
        verifiedAt: readNullableString(value, "verifiedAt", message),
        rejectionReason: readNullableString(value, "rejectionReason", message),
        createdAt: readString(value, "createdAt", message),
        updatedAt: readString(value, "updatedAt", message),
    };
}

function readProfilePayload(value: unknown, message: string): ProfileInfo {
    if (!isRecord(value)) {
        throw new Error(message);
    }

    return {
        userID: readInteger(value, "userID", message),
        schoolID: readNullableInteger(value, "schoolID", message),
        studentIDs: readStringArrayOrNull(value, "studentIDs", message),
        activeStudentID: readNullableString(value, "activeStudentID", message),
        verificationStatus: readEnum<ProfileInfo["verificationStatus"]>(
            value,
            "verificationStatus",
            PROFILE_VERIFICATION_STATUS_VALUES,
            message,
        ),
        verificationMethod: readNullableEnum<
            NonNullable<ProfileInfo["verificationMethod"]>
        >(value, "verificationMethod", PROFILE_VERIFICATION_METHOD_VALUES, message),
        rejectionReason: readNullableString(value, "rejectionReason", message),
        reviewedAt: readNullableString(value, "reviewedAt", message),
        phone: readNullableString(value, "phone", message),
        phoneVerified: readOptionalBoolean(value, "phoneVerified", message),
        consentGivenAt: readNullableString(value, "consentGivenAt", message),
        verifiedAt: readNullableString(value, "verifiedAt", message),
        createdAt: readString(value, "createdAt", message),
        updatedAt: readString(value, "updatedAt", message),
    };
}

function readManualFieldDescriptor(
    value: unknown,
    message: string,
): ManualFieldDescriptor {
    if (!isRecord(value)) {
        throw new Error(message);
    }

    return {
        key: readString(value, "key", message),
        label: readString(value, "label", message),
        type: readEnum<ManualFieldDescriptor["type"]>(
            value,
            "type",
            MANUAL_FIELD_TYPE_VALUES,
            message,
        ),
        required: readBoolean(value, "required", message),
        options: readStringArrayOrNull(value, "options", message),
        placeholder: readNullableString(value, "placeholder", message),
    };
}

function readManualFieldDescriptors(
    record: Record<string, unknown>,
    key: string,
    message: string,
): ManualFieldDescriptor[] | null | undefined {
    const value = record[key];
    if (value === undefined || value === null) {
        return value;
    }
    if (!Array.isArray(value)) {
        throw new Error(message);
    }
    return value.map((item) => readManualFieldDescriptor(item, message));
}

function readSchoolPayload(value: unknown, message: string): SchoolConfig {
    if (!isRecord(value)) {
        throw new Error(message);
    }

    return {
        schoolID: readInteger(value, "schoolID", message),
        schoolCode: readString(value, "schoolCode", message),
        schoolName: readString(value, "schoolName", message),
        verificationMethod: readEnum<SchoolConfig["verificationMethod"]>(
            value,
            "verificationMethod",
            SCHOOL_VERIFICATION_METHOD_VALUES,
            message,
        ),
        consentText: readNullableString(value, "consentText", message),
        manualFormFields: readManualFieldDescriptors(
            value,
            "manualFormFields",
            message,
        ),
        enabled: readBoolean(value, "enabled", message),
        schoolSsoEnabled: readBoolean(value, "schoolSsoEnabled", message),
        schoolEmailOtpEnabled: readBoolean(
            value,
            "schoolEmailOtpEnabled",
            message,
        ),
        schoolEmailIdentityPolicy: readNullableSchoolEmailIdentityPolicy(
            value,
            "schoolEmailIdentityPolicy",
            message,
        ),
    };
}

function readNullableSchoolEmailIdentityPolicy(
    record: Record<string, unknown>,
    key: string,
    message: string,
): SchoolConfig["schoolEmailIdentityPolicy"] {
    const value = record[key];
    if (value === undefined || value === null) {
        return undefined;
    }
    if (!isRecord(value)) {
        throw new Error(message);
    }
    const type = readString(value, "type", message);
    if (type !== "academic_student_email") {
        throw new Error(message);
    }
    return {
        type,
        studentIDEmailDomain: readOptionalString(
            value,
            "studentIDEmailDomain",
            message,
        ),
        requireStudentName: readBoolean(value, "requireStudentName", message),
    };
}

function readSchoolListPayload(value: unknown, message: string): SchoolConfig[] {
    if (!Array.isArray(value)) {
        throw new Error(message);
    }
    return value.map((item) => readSchoolPayload(item, message));
}

function readStudentEmailOTPPayload(
    value: unknown,
    message: string,
): StudentEmailOTPResponse {
    if (!isRecord(value)) {
        throw new Error(message);
    }
    return {
        email: readString(value, "email", message),
        studentID: readOptionalString(value, "studentID", message),
        cooldownSeconds: readInteger(value, "cooldownSeconds", message),
    };
}

function readStudentEmailAcademicMatchPayload(
    value: unknown,
    message: string,
): StudentEmailAcademicMatchResponse {
    if (!isRecord(value)) {
        throw new Error(message);
    }
    return {
        matched: readBoolean(value, "matched", message),
        email: readOptionalString(value, "email", message),
        studentID: readOptionalString(value, "studentID", message),
        message: readOptionalString(value, "message", message),
    };
}

function readQQBindingPayload(value: unknown, message: string): QQBindingInfo {
    if (!isRecord(value)) {
        throw new Error(message);
    }

    return {
        userID: readInteger(value, "userID", message),
        qqID: readString(value, "qqID", message),
        boundAt: readString(value, "boundAt", message),
        createdAt: readString(value, "createdAt", message),
        updatedAt: readString(value, "updatedAt", message),
    };
}

function readQQBindingCodePayload(value: unknown, message: string): QQBindingCodeInfo {
    if (!isRecord(value)) {
        throw new Error(message);
    }

    return {
        code: readString(value, "code", message),
        expiresAt: readString(value, "expiresAt", message),
    };
}

function readIdentityPhotoUploadPayload(
    value: unknown,
    message: string,
): IdentityPhotoUploadResult {
    if (!isRecord(value)) {
        throw new Error(message);
    }

    return {
        key: readString(value, "key", message),
        rejectionReason: readNullableString(value, "rejectionReason", message),
        createdAt: readOptionalString(value, "createdAt", message),
        updatedAt: readOptionalString(value, "updatedAt", message),
    };
}
