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
type BindPhoneRequest = components["schemas"]["BindPhoneRequest"];
type QQBindingInfo = components["schemas"]["QQBinding"];
type QQBindingCodeInfo = components["schemas"]["QQBindingCode"];

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
                loadNullableResource(() => api.identity.getIdentity()),
                loadNullableResource(() => api.identity.getProfile()),
                loadNullableResource(() => api.identity.getQQBinding()),
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
        schools.value = res.data?.data ?? [];
    };

    // 提交实名认证
    const submitIdentity = async (data: SubmitIdentityRequest) => {
        const res = await api.identity.submitIdentity(data);
        identity.value = res.data?.data ?? null;
        return res.data?.data ?? null;
    };

    const uploadIdentityPhoto = async (data: UploadIdentityPhotoRequest) => {
        const res = await api.identity.uploadIdentityPhoto(data);
        return res.data?.data?.key ?? "";
    };

    // 学生认证
    const verifyStudent = async (data: SubmitStudentVerificationRequest) => {
        const res = await api.identity.verifyStudent(data);
        profile.value = res.data?.data ?? null;
        return res.data?.data ?? null;
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
        profile.value = profileRes.data?.data ?? null;
    };

    const fetchQQBinding = async () => {
        qqBinding.value = await loadNullableResource(() => api.identity.getQQBinding());
        if (qqBinding.value) {
            qqBindingCode.value = null;
        }
        return qqBinding.value;
    };

    const createQQBindingCode = async () => {
        const res = await api.identity.createQQBindingCode();
        qqBindingCode.value = res.data?.data ?? null;
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
        bindPhone,
        requestBindPhoneOTP,
        reset,
    };
});

async function loadNullableResource<T>(
    loader: () => Promise<{ data?: { data?: T } }>,
): Promise<T | null> {
    try {
        const response = await loader();
        return response.data?.data ?? null;
    } catch (error) {
        if (getErrorStatus(error) === 404) {
            return null;
        }
        throw error;
    }
}
