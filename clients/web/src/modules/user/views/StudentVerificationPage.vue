<template>
    <div class="max-w-[600px] mx-auto p-6 animate-fade-in max-sm:p-4">
        <!-- Back button + title -->
        <header class="flex items-center gap-3 mb-6">
            <button
                class="p-2 bg-transparent rounded-lg text-text-muted cursor-pointer transition-all duration-fast hover:border-text-primary hover:text-text-primary"
                :aria-label="t('common.actions.back')"
                @click="goBack"
            >
                <ArrowLeft class="size-5" />
            </button>
            <h1
                class="font-sans text-xl font-extrabold tracking-tight text-text-primary m-0"
            >
                {{ t("user.verification.student.title") }}
            </h1>
        </header>

        <!-- Loading state -->
        <div v-if="storeLoading && !profile" class="flex justify-center py-12">
            <div
                class="size-8 border-2 border-border border-t-primary rounded-full animate-spin"
                role="status"
                aria-busy="true"
            >
                <span class="sr-only">{{ t("common.actions.loading") }}</span>
            </div>
        </div>

        <!-- Identity not verified warning -->
        <div
            v-else-if="!identityVerified"
            class="bg-bg-card border border-yellow-500/30 rounded-xl p-5 shadow-card"
        >
            <div class="flex items-center gap-3 mb-4">
                <div class="p-2 bg-yellow-500/10 rounded-lg">
                    <ShieldAlert class="size-6 text-yellow-500" />
                </div>
                <div>
                    <h2 class="text-base font-bold text-text-primary m-0">
                        {{ t("user.verification.student.identityRequired") }}
                    </h2>
                    <p class="text-sm text-text-muted m-0 mt-1">
                        {{ t("user.verification.student.desc") }}
                    </p>
                </div>
            </div>
            <router-link
                to="/user/identity-verification"
                class="block w-full py-2.5 bg-text-primary text-bg-base rounded-lg text-sm font-medium text-center no-underline transition-all duration-fast hover:bg-accent hover:text-white"
            >
                {{ t("user.verification.identity.title") }}
            </router-link>
        </div>

        <!-- Verified status card -->
        <div
            v-else-if="profile?.verificationStatus === 'verified' && !showForm"
            class="bg-bg-card border border-green-500/30 rounded-xl p-5 shadow-card"
        >
            <div class="flex items-center gap-3 mb-4">
                <div class="p-2 bg-green-500/10 rounded-lg">
                    <GraduationCap class="size-6 text-green-500" />
                </div>
                <div>
                    <h2 class="text-base font-bold text-text-primary m-0">
                        {{ t("user.verification.student.verified") }}
                    </h2>
                    <p class="text-sm text-text-muted m-0 mt-1">
                        {{ t("user.verification.student.verifySuccess") }}
                    </p>
                </div>
            </div>
            <div v-if="profile.schoolID" class="text-sm text-text-secondary">
                <div class="flex justify-between">
                    <span class="text-text-muted">{{
                        t("user.verification.student.selectSchool")
                    }}</span>
                    <span>{{ schoolName(profile.schoolID) }}</span>
                </div>
            </div>
        </div>

        <!-- Pending status card -->
        <div
            v-else-if="profile?.verificationStatus === 'pending' && !showForm"
            class="bg-bg-card border border-yellow-500/30 rounded-xl p-5 shadow-card"
        >
            <div class="flex items-center gap-3">
                <div class="p-2 bg-yellow-500/10 rounded-lg">
                    <Clock class="size-6 text-yellow-500" />
                </div>
                <div>
                    <h2 class="text-base font-bold text-text-primary m-0">
                        {{ t("user.verification.student.pending") }}
                    </h2>
                    <p class="text-sm text-text-muted m-0 mt-1">
                        {{ t("user.verification.student.verifyPending") }}
                    </p>
                </div>
            </div>
        </div>

        <!-- Rejected status card -->
        <div
            v-else-if="profile?.verificationStatus === 'rejected' && !showForm"
            class="bg-bg-card border border-red-500/30 rounded-xl p-5 shadow-card"
        >
            <div class="flex items-center gap-3 mb-4">
                <div class="p-2 bg-red-500/10 rounded-lg">
                    <XCircle class="size-6 text-red-500" />
                </div>
                <div>
                    <h2 class="text-base font-bold text-text-primary m-0">
                        {{ t("user.verification.student.rejected") }}
                    </h2>
                    <p class="text-sm text-text-muted m-0 mt-1">
                        {{ t("user.verification.student.verifyFailed") }}
                    </p>
                </div>
            </div>
            <button
                class="w-full py-2.5 bg-text-primary text-bg-base rounded-lg text-sm font-medium cursor-pointer transition-all duration-fast hover:bg-accent hover:text-white border-0"
                @click="showForm = true"
            >
                {{ t("user.verification.identity.resubmit") }}
            </button>
        </div>

        <!-- Verification form -->
        <div
            v-else
            class="bg-bg-card rounded-xl p-5 shadow-card"
        >
            <p class="text-sm text-text-muted mb-5 m-0">
                {{ t("user.verification.student.desc") }}
            </p>

            <!-- Step 1: Select school -->
            <div class="mb-5">
                <label
                    class="block text-sm font-semibold text-text-primary mb-2"
                    for="student-school"
                >
                    {{ t("user.verification.student.selectSchool") }}
                </label>
                <select
                    id="student-school"
                    v-model.number="form.schoolID"
                    class="w-full px-3 py-2.5 bg-transparent rounded-lg text-sm text-text-primary outline-none transition-all duration-fast focus:border-primary appearance-none cursor-pointer"
                >
                    <option :value="0" disabled>
                        {{ t("user.verification.student.selectSchool") }}
                    </option>
                    <option
                        v-for="school in schools"
                        :key="school.schoolID"
                        :value="school.schoolID"
                        :disabled="!school.enabled"
                    >
                        {{ school.schoolName }}
                    </option>
                </select>
            </div>

            <!-- Step 2: School-specific form -->
            <template v-if="selectedSchool">
                <!-- Consent text -->
                <div
                    v-if="selectedSchool.consentText"
                    class="mb-5 p-3 bg-primary/5 border border-primary/20 rounded-lg"
                >
                    <p
                        class="text-xs text-text-secondary m-0 whitespace-pre-line"
                    >
                        {{ selectedSchool.consentText }}
                    </p>
                </div>

                <!-- LDAP login form -->
                <template v-if="selectedSchool.verificationMethod === 'ldap'">
                    <div class="mb-4">
                        <label
                            class="block text-sm font-semibold text-text-primary mb-2"
                            for="student-id"
                        >
                            {{ t("user.verification.student.studentId") }}
                        </label>
                        <input
                            id="student-id"
                            v-model="form.studentID"
                            type="text"
                            class="w-full px-3 py-2.5 bg-transparent rounded-lg text-sm text-text-primary placeholder-text-muted outline-none transition-all duration-fast focus:border-primary"
                            :placeholder="
                                t('user.verification.student.studentId')
                            "
                        />
                    </div>

                    <div class="mb-5">
                        <label
                            class="block text-sm font-semibold text-text-primary mb-2"
                            for="student-password"
                        >
                            {{ t("user.verification.student.password") }}
                        </label>
                        <input
                            id="student-password"
                            v-model="form.password"
                            type="password"
                            class="w-full px-3 py-2.5 bg-transparent rounded-lg text-sm text-text-primary placeholder-text-muted outline-none transition-all duration-fast focus:border-primary"
                            :placeholder="
                                t('user.verification.student.password')
                            "
                        />
                    </div>
                </template>

                <!-- Manual verification form -->
                <template
                    v-else-if="selectedSchool.verificationMethod === 'manual'"
                >
                    <div v-if="manualFields.length > 0" class="mb-5 space-y-4">
                        <div v-for="field in manualFields" :key="field.key">
                            <label
                                class="block text-sm font-semibold text-text-primary mb-2"
                                :for="`manual-${field.key}`"
                            >
                                {{ field.label }}
                                <span v-if="field.required" class="text-red-500"
                                    >*</span
                                >
                            </label>

                            <textarea
                                v-if="field.type === 'textarea'"
                                :id="`manual-${field.key}`"
                                v-model="form.manualFormData[field.key]"
                                rows="3"
                                class="w-full px-3 py-2.5 bg-transparent rounded-lg text-sm text-text-primary placeholder-text-muted outline-none transition-all duration-fast focus:border-primary resize-y"
                                :placeholder="field.placeholder || field.label"
                            />

                            <select
                                v-else-if="field.type === 'select'"
                                :id="`manual-${field.key}`"
                                v-model="form.manualFormData[field.key]"
                                class="w-full px-3 py-2.5 bg-transparent rounded-lg text-sm text-text-primary outline-none transition-all duration-fast focus:border-primary appearance-none cursor-pointer"
                            >
                                <option value="" disabled>
                                    {{ field.placeholder || field.label }}
                                </option>
                                <option
                                    v-for="option in field.options ?? []"
                                    :key="option"
                                    :value="option"
                                >
                                    {{ option }}
                                </option>
                            </select>

                            <input
                                v-else
                                :id="`manual-${field.key}`"
                                v-model="form.manualFormData[field.key]"
                                :type="field.type === 'date' ? 'date' : 'text'"
                                class="w-full px-3 py-2.5 bg-transparent rounded-lg text-sm text-text-primary placeholder-text-muted outline-none transition-all duration-fast focus:border-primary"
                                :placeholder="field.placeholder || field.label"
                            />
                        </div>
                    </div>

                    <div
                        v-else
                        class="mb-5 p-4 bg-bg-card rounded-lg"
                    >
                        <div
                            class="flex items-center gap-2 text-sm text-text-muted"
                        >
                            <Clock class="size-4 shrink-0" />
                            <span>{{
                                t("user.verification.student.manualHint")
                            }}</span>
                        </div>
                    </div>
                </template>

                <!-- Consent checkbox -->
                <label
                    v-if="selectedSchool.consentText"
                    class="flex items-start gap-2 mb-5 cursor-pointer text-sm text-text-secondary"
                >
                    <input
                        type="checkbox"
                        v-model="form.consent"
                        class="mt-0.5 accent-primary"
                    />
                    <span>{{ t("user.verification.student.consent") }}</span>
                </label>

                <!-- Submit button -->
                <button
                    class="w-full py-2.5 bg-text-primary text-bg-base rounded-lg text-sm font-medium cursor-pointer transition-all duration-fast hover:bg-accent hover:text-white border-0 disabled:opacity-50 disabled:cursor-not-allowed"
                    :disabled="!canSubmit || submitting"
                    @click="handleSubmit"
                >
                    <span
                        v-if="submitting"
                        class="inline-flex items-center gap-2"
                    >
                        <span
                            class="size-4 border-2 border-bg-base border-t-transparent rounded-full animate-spin"
                        />
                        {{ t("common.actions.loading") }}
                    </span>
                    <span v-else>
                        {{ t("user.verification.student.verify") }}
                    </span>
                </button>
            </template>
        </div>
    </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted, watch } from "vue";
import { useRouter } from "vue-router";
import { useI18n } from "vue-i18n";
import {
    GraduationCap,
    ShieldAlert,
    Clock,
    XCircle,
    ArrowLeft,
} from "lucide-vue-next";
import { useVerificationStore } from "@/stores/verification";
import { useToast } from "@/composables/useToast";
import {
    isManualSubmitReady,
    normalizeManualFields,
    normalizeManualFormData,
} from "@/modules/user/utils/manualVerification";

const router = useRouter();
const { t } = useI18n();
const store = useVerificationStore();
const toast = useToast();

const profile = computed(() => store.profile);
const identityVerified = computed(() => store.identityVerified);
const schools = computed(() => store.schools);
const storeLoading = computed(() => store.loading);
const showForm = ref(false);
const submitting = ref(false);

const form = reactive({
    schoolID: 0,
    studentID: "",
    password: "",
    manualFormData: {} as Record<string, string>,
    consent: false,
});

function goBack() {
    void router.push("/identity");
}

const selectedSchool = computed(() => {
    if (!form.schoolID) return null;
    return schools.value.find((s) => s.schoolID === form.schoolID) ?? null;
});

function schoolName(schoolID: number): string {
    const school = schools.value.find((s) => s.schoolID === schoolID);
    return school ? school.schoolName : String(schoolID);
}

const manualFields = computed(() =>
    normalizeManualFields(selectedSchool.value?.manualFormFields),
);

const canSubmit = computed(() => {
    if (!selectedSchool.value) return false;
    if (selectedSchool.value.consentText && !form.consent) return false;

    if (selectedSchool.value.verificationMethod === "ldap") {
        return form.studentID.trim() !== "" && form.password !== "";
    }

    if (selectedSchool.value.verificationMethod === "manual") {
        return isManualSubmitReady(manualFields.value, form.manualFormData);
    }

    return true;
});

async function handleSubmit() {
    if (!canSubmit.value || submitting.value) return;

    submitting.value = true;
    try {
        const manualFormData = normalizeManualFormData(form.manualFormData);
        await store.verifyStudent({
            schoolID: form.schoolID,
            studentID: form.studentID.trim() || undefined,
            password: form.password || undefined,
            manualFormData,
            consent: form.consent,
        });
        await store.fetchStatus();
        showForm.value = false;
    } catch (err) {
        toast.error(
            err instanceof Error
                ? err.message
                : t("user.verification.student.verifyFailed"),
        );
    } finally {
        submitting.value = false;
    }
}

watch(
    () => form.schoolID,
    () => {
        form.studentID = "";
        form.password = "";
        form.manualFormData = {};
        form.consent = false;
    },
);

onMounted(() => {
    store.fetchStatus().catch(() => {});
    store.fetchSchools().catch(() => {});
});
</script>
