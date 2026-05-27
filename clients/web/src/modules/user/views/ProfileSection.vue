<template>
    <div class="bg-bg-card rounded-xl p-5 shadow-card mb-6">
        <!-- User info header -->
        <div
            class="mb-5 flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between"
        >
            <div class="flex min-w-0 items-center gap-4">
                <div
                    class="p-[3px] bg-gradient-to-br from-primary to-accent rounded-full shrink-0"
                >
                    <div
                        class="size-14 bg-bg-card rounded-full flex items-center justify-center overflow-hidden"
                    >
                        <img
                            v-if="user?.avatar"
                            :src="user.avatar"
                            :alt="displayName"
                            class="size-full object-cover"
                        />
                        <User v-else class="size-7 text-text-muted" />
                    </div>
                </div>
                <div class="min-w-0">
                    <h2
                        class="text-base font-bold text-text-primary m-0 truncate"
                    >
                        {{ displayName }}
                    </h2>
                    <p class="text-sm text-text-muted m-0 mt-0.5 truncate">
                        {{ user?.email ?? "" }}
                    </p>
                </div>
            </div>

            <a
                v-if="accountSettingsUrl"
                :href="accountSettingsUrl"
                class="inline-flex h-9 shrink-0 items-center justify-center gap-2 rounded-md border border-border bg-bg-base px-3 text-sm font-medium text-text-secondary no-underline transition-colors duration-fast hover:border-primary/40 hover:text-primary"
            >
                <KeyRound class="size-4" aria-hidden="true" />
                {{ t("user.identityHome.accountSecurity") }}
            </a>
        </div>

        <!-- Status cards grid -->
        <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
            <!-- Identity verification card -->
            <div class="border-0 rounded-lg p-4 transition-all duration-fast">
                <div class="flex items-center gap-3 mb-3">
                    <div
                        class="p-2 rounded-lg"
                        :class="
                            identityVerified ? 'bg-green-500/10' : 'bg-bg-base'
                        "
                    >
                        <ShieldCheck
                            class="size-5"
                            :class="
                                identityVerified
                                    ? 'text-green-500'
                                    : 'text-text-muted'
                            "
                        />
                    </div>
                    <div class="min-w-0">
                        <p class="text-sm font-semibold text-text-primary m-0">
                            {{ t("user.verification.identity.title") }}
                        </p>
                        <span
                            class="inline-block mt-1 px-2 py-0.5 rounded-full text-xs font-medium"
                            :class="identityStatusClass"
                        >
                            {{ identityStatusLabel }}
                        </span>
                    </div>
                </div>
                <router-link
                    v-if="!identityVerified"
                    to="/user/identity-verification"
                    class="block w-full py-2 bg-text-primary text-bg-base rounded-lg text-xs font-medium text-center no-underline transition-all duration-fast hover:bg-accent hover:text-white"
                >
                    {{ t("user.verification.identity.unverified") }}
                </router-link>
            </div>

            <!-- Student verification card -->
            <div class="border-0 rounded-lg p-4 transition-all duration-fast">
                <div class="flex items-center gap-3 mb-3">
                    <div
                        class="p-2 rounded-lg"
                        :class="
                            studentVerified ? 'bg-green-500/10' : 'bg-bg-base'
                        "
                    >
                        <GraduationCap
                            class="size-5"
                            :class="
                                studentVerified
                                    ? 'text-green-500'
                                    : 'text-text-muted'
                            "
                        />
                    </div>
                    <div class="min-w-0">
                        <p class="text-sm font-semibold text-text-primary m-0">
                            {{ t("user.verification.student.title") }}
                        </p>
                        <span
                            class="inline-block mt-1 px-2 py-0.5 rounded-full text-xs font-medium"
                            :class="studentStatusClass"
                        >
                            {{ studentStatusLabel }}
                        </span>
                    </div>
                </div>
                <template v-if="!studentVerified">
                    <router-link
                        v-if="identityVerified"
                        to="/user/student-verification"
                        class="block w-full py-2 bg-text-primary text-bg-base rounded-lg text-xs font-medium text-center no-underline transition-all duration-fast hover:bg-accent hover:text-white"
                    >
                        {{ t("user.verification.student.unverified") }}
                    </router-link>
                    <p v-else class="text-xs text-text-muted m-0 text-center">
                        {{ t("user.verification.student.identityRequired") }}
                    </p>
                </template>
                <router-link
                    v-if="studentVerified"
                    to="/user/academic-info"
                    class="block w-full py-2 bg-accent/10 text-accent rounded-lg text-xs font-medium text-center no-underline transition-all duration-fast hover:bg-accent hover:text-white"
                >
                    {{ t("user.verification.academic.title") }}
                </router-link>
            </div>

            <!-- QQ binding card -->
            <div class="border-0 rounded-lg p-4 transition-all duration-fast">
                <div class="flex items-center gap-3 mb-3">
                    <div
                        class="p-2 rounded-lg"
                        :class="qqBound ? 'bg-green-500/10' : 'bg-bg-base'"
                    >
                        <Bot
                            class="size-5"
                            :class="
                                qqBound ? 'text-green-500' : 'text-text-muted'
                            "
                        />
                    </div>
                    <div class="min-w-0">
                        <p class="text-sm font-semibold text-text-primary m-0">
                            {{ t("user.verification.qq.title") }}
                        </p>
                        <span
                            class="inline-block mt-1 px-2 py-0.5 rounded-full text-xs font-medium"
                            :class="
                                qqBound
                                    ? 'bg-green-500/10 text-green-600'
                                    : 'bg-bg-base text-text-muted'
                            "
                        >
                            {{
                                qqBound
                                    ? t("user.verification.qq.bound")
                                    : t("user.verification.qq.unbound")
                            }}
                        </span>
                    </div>
                </div>
                <router-link
                    to="/user/qq-binding"
                    class="block w-full py-2 rounded-lg text-xs font-medium text-center no-underline transition-all duration-fast"
                    :class="
                        qqBound
                            ? 'bg-accent/10 text-accent hover:bg-accent hover:text-white'
                            : 'bg-text-primary text-bg-base hover:bg-accent hover:text-white'
                    "
                >
                    {{
                        qqBound
                            ? t("common.actions.more")
                            : t("user.verification.qq.createCode")
                    }}
                </router-link>
            </div>

            <!-- Phone binding card -->
            <div class="border-0 rounded-lg p-4 transition-all duration-fast">
                <div class="flex items-center gap-3 mb-3">
                    <div
                        class="p-2 rounded-lg"
                        :class="phoneBound ? 'bg-green-500/10' : 'bg-bg-base'"
                    >
                        <Phone
                            class="size-5"
                            :class="
                                phoneBound
                                    ? 'text-green-500'
                                    : 'text-text-muted'
                            "
                        />
                    </div>
                    <div class="min-w-0">
                        <p class="text-sm font-semibold text-text-primary m-0">
                            {{ t("user.verification.phone.title") }}
                        </p>
                        <span
                            class="inline-block mt-1 px-2 py-0.5 rounded-full text-xs font-medium"
                            :class="
                                phoneBound
                                    ? 'bg-green-500/10 text-green-600'
                                    : 'bg-bg-base text-text-muted'
                            "
                        >
                            {{
                                phoneBound
                                    ? t("user.verification.phone.bound")
                                    : t("user.verification.phone.unbound")
                            }}
                        </span>
                    </div>
                </div>
                <router-link
                    v-if="!phoneBound"
                    to="/user/phone-binding"
                    class="block w-full py-2 bg-text-primary text-bg-base rounded-lg text-xs font-medium text-center no-underline transition-all duration-fast hover:bg-accent hover:text-white"
                >
                    {{ t("user.verification.phone.bind") }}
                </router-link>
            </div>
        </div>
    </div>
</template>

<script setup lang="ts">
import { computed, onMounted } from "vue";
import { useI18n } from "vue-i18n";
import {
    Bot,
    ShieldCheck,
    GraduationCap,
    KeyRound,
    User,
    Phone,
} from "lucide-vue-next";
import { useAuthStore } from "@/stores/auth";
import { useVerificationStore } from "@/stores/verification";

const { t } = useI18n();
const authStore = useAuthStore();
const verificationStore = useVerificationStore();

const user = computed(() => authStore.user);
const displayName = computed(
    () => user.value?.displayName ?? user.value?.name ?? "",
);
const accountSettingsUrl = computed(() => user.value?.accountSettingsUrl ?? "");
const identityVerified = computed(() => verificationStore.identityVerified);
const studentVerified = computed(() => verificationStore.studentVerified);
const qqBound = computed(() => verificationStore.qqBound);
const phoneBound = computed(() => profile.value?.phoneVerified === true);

const identity = computed(() => verificationStore.identity);
const profile = computed(() => verificationStore.profile);

// 身份认证状态派生
type StatusVariant = "verified" | "pending" | "rejected" | "unverified";

const identityStatus = computed((): StatusVariant => {
    if (!identity.value) return "unverified";
    if (identity.value.verified) return "verified";
    if (identity.value.reviewedAt) return "rejected";
    return "pending";
});

const identityStatusLabel = computed(() => {
    return t(`user.verification.identity.${identityStatus.value}`);
});

const studentStatus = computed((): StatusVariant => {
    if (!profile.value) return "unverified";
    const status = profile.value.verificationStatus;
    if (status === "verified") return "verified";
    if (status === "rejected") return "rejected";
    if (status === "pending") return "pending";
    return "unverified";
});

const studentStatusLabel = computed(() => {
    return t(`user.verification.student.${studentStatus.value}`);
});

const statusClassMap: Record<StatusVariant, string> = {
    verified: "bg-green-500/10 text-green-600",
    pending: "bg-yellow-500/10 text-yellow-600",
    rejected: "bg-red-500/10 text-red-600",
    unverified: "bg-bg-base text-text-muted",
};

const identityStatusClass = computed(
    () => statusClassMap[identityStatus.value],
);
const studentStatusClass = computed(() => statusClassMap[studentStatus.value]);

onMounted(() => {
    if (authStore.isAuthenticated) {
        void verificationStore.fetchStatus().catch((error) => {
            if (import.meta.env.DEV) {
                console.warn(
                    "[ProfileSection] failed to fetch verification status",
                    error,
                );
            }
        });
    }
});
</script>
