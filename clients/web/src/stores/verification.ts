/**
 * 认证状态管理（实名认证 + 学生认证）
 */
import { defineStore } from "pinia";
import { ref, computed } from "vue";
import type { components } from "@stuhelper/shared";
import { api } from "@/api";

type IdentityInfo = components["schemas"]["UserIdentity"];
type ProfileInfo = components["schemas"]["UserProfile"];
type SchoolConfig = components["schemas"]["SchoolConfig"];
type SubmitIdentityRequest = components["schemas"]["SubmitIdentityRequest"];
type SubmitStudentVerificationRequest =
    components["schemas"]["SubmitStudentVerificationRequest"];
type BindPhoneRequest = components["schemas"]["BindPhoneRequest"];

export const useVerificationStore = defineStore("verification", () => {
    // 状态
    const identity = ref<IdentityInfo | null>(null);
    const profile = ref<ProfileInfo | null>(null);
    const schools = ref<SchoolConfig[]>([]);
    const loading = ref(false);

    // 计算属性
    const identityVerified = computed(() => identity.value?.verified === true);
    const studentVerified = computed(
        () => profile.value?.verificationStatus === "verified",
    );
    const canViewFullReviews = computed(() => studentVerified.value);

    // 并行获取实名认证和学生认证状态
    const fetchStatus = async () => {
        loading.value = true;
        try {
            const [identityRes, profileRes] = await Promise.allSettled([
                api.identity.getIdentity(),
                api.identity.getProfile(),
            ]);

            // 404 表示用户尚未提交，视为 null
            if (identityRes.status === "fulfilled") {
                identity.value = identityRes.value.data?.data ?? null;
            } else {
                const err = identityRes.reason as Error & { status?: number };
                if (err.status === 404) {
                    identity.value = null;
                } else {
                    throw err;
                }
            }

            if (profileRes.status === "fulfilled") {
                profile.value = profileRes.value.data?.data ?? null;
            } else {
                const err = profileRes.reason as Error & { status?: number };
                if (err.status === 404) {
                    profile.value = null;
                } else {
                    throw err;
                }
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

    // 学生认证
    const verifyStudent = async (data: SubmitStudentVerificationRequest) => {
        const res = await api.identity.verifyStudent(data);
        profile.value = res.data?.data ?? null;
        return res.data?.data ?? null;
    };

    // 绑定手机
    const bindPhone = async (data: BindPhoneRequest) => {
        await api.identity.bindPhone(data);
    };

    // 重置状态（setup store 不支持 $reset）
    const reset = () => {
        identity.value = null;
        profile.value = null;
        schools.value = [];
        loading.value = false;
    };

    return {
        identity,
        profile,
        schools,
        loading,
        identityVerified,
        studentVerified,
        canViewFullReviews,
        fetchStatus,
        fetchSchools,
        submitIdentity,
        verifyStudent,
        bindPhone,
        reset,
    };
});
