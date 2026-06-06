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
            <div
                v-if="verifiedStudentRows.length > 0"
                class="divide-y divide-border border-y border-border text-sm text-text-secondary"
                data-student-verified-profile
            >
                <div
                    v-for="row in verifiedStudentRows"
                    :key="row.label"
                    class="grid gap-1 py-3 sm:grid-cols-[120px_1fr] sm:gap-4"
                >
                    <span class="text-text-muted">{{ row.label }}</span>
                    <span class="break-all text-text-primary">{{
                        row.value
                    }}</span>
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
        <div v-else class="bg-bg-card rounded-xl p-5 shadow-card">
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
                    v-model="form.schoolCode"
                    data-student-school-select
                    class="w-full px-3 py-2.5 bg-transparent rounded-lg text-sm text-text-primary outline-none transition-all duration-fast focus:border-primary appearance-none cursor-pointer"
                >
                    <option value="" disabled>
                        {{ t("user.verification.student.selectSchool") }}
                    </option>
                    <option
                        v-for="school in schools"
                        :key="school.schoolCode"
                        :value="school.schoolCode"
                        :disabled="!school.enabled"
                    >
                        {{ school.schoolName }}（{{ school.schoolCode }}）
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

                <!-- School email OTP with academic database precheck -->
                <template v-if="selectedSchoolRequiresAcademicEmail">
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
                            data-student-id-input
                            type="text"
                            class="w-full px-3 py-2.5 bg-transparent rounded-lg text-sm text-text-primary placeholder-text-muted outline-none transition-all duration-fast focus:border-primary"
                            :placeholder="
                                t('user.verification.student.studentId')
                            "
                        />
                    </div>

                    <p
                        v-if="academicMatchMessage"
                        :class="academicMatchMessageClass"
                        data-student-academic-match-status
                    >
                        {{ academicMatchMessage }}
                    </p>

                    <div class="mb-4">
                        <label
                            class="block text-sm font-semibold text-text-primary mb-2"
                            for="student-name"
                        >
                            姓名
                        </label>
                        <input
                            id="student-name"
                            v-model="form.studentName"
                            data-student-name-input
                            type="text"
                            class="w-full px-3 py-2.5 bg-transparent rounded-lg text-sm text-text-primary placeholder-text-muted outline-none transition-all duration-fast focus:border-primary"
                            placeholder="请输入学籍库中的姓名"
                        />
                    </div>

                    <div class="mb-4">
                        <label
                            class="block text-sm font-semibold text-text-primary mb-2"
                            for="student-school-email"
                        >
                            学号邮箱
                        </label>
                        <input
                            id="student-school-email"
                            :value="form.email"
                            data-student-school-email-input
                            type="email"
                            readonly
                            class="w-full px-3 py-2.5 bg-transparent rounded-lg text-sm text-text-primary placeholder-text-muted outline-none transition-all duration-fast focus:border-primary"
                            placeholder="学号和姓名校验通过后自动填写"
                        />
                    </div>

                    <div class="mb-5 grid gap-3 sm:grid-cols-[1fr_auto]">
                        <label
                            class="block text-sm font-semibold text-text-primary"
                            for="student-email-code"
                        >
                            邮箱验证码
                            <input
                                id="student-email-code"
                                v-model="form.emailCode"
                                data-student-email-code-input
                                type="text"
                                inputmode="numeric"
                                class="mt-2 w-full px-3 py-2.5 bg-transparent rounded-lg text-sm text-text-primary placeholder-text-muted outline-none transition-all duration-fast focus:border-primary"
                                placeholder="请输入验证码"
                            />
                        </label>
                        <button
                            class="self-end py-2.5 px-4 bg-transparent border border-border rounded-lg text-sm font-medium text-text-primary cursor-pointer transition-all duration-fast hover:border-primary disabled:opacity-50 disabled:cursor-not-allowed"
                            :class="{
                                'opacity-50 cursor-not-allowed':
                                    selectedSchoolRequiresAcademicEmail &&
                                    !canRequestEmailOTP,
                            }"
                            type="button"
                            data-student-email-otp-request
                            :aria-disabled="
                                selectedSchoolRequiresAcademicEmail &&
                                !canRequestEmailOTP
                                    ? 'true'
                                    : undefined
                            "
                            :disabled="submitting"
                            @click="requestEmailOTP"
                        >
                            发送验证码
                        </button>
                    </div>
                </template>

                <!-- LDAP login form -->
                <template
                    v-else-if="selectedSchool.verificationMethod === 'ldap'"
                >
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

                    <div v-else class="mb-5 p-4 bg-bg-card rounded-lg">
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
                    class="flex items-start gap-2 mb-5 cursor-pointer text-sm text-text-secondary"
                >
                    <input
                        type="checkbox"
                        v-model="form.consent"
                        data-student-consent-checkbox
                        class="mt-0.5 accent-primary"
                    />
                    <span>
                        {{
                            selectedSchool.consentText
                                ? t("user.verification.student.consent")
                                : t("user.verification.student.consentPlain")
                        }}
                    </span>
                </label>

                <!-- Submit button -->
                <button
                    class="w-full py-2.5 bg-text-primary text-bg-base rounded-lg text-sm font-medium cursor-pointer transition-all duration-fast hover:bg-accent hover:text-white border-0 disabled:opacity-50 disabled:cursor-not-allowed"
                    data-student-verification-submit
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
import {
    ref,
    reactive,
    computed,
    onBeforeUnmount,
    onMounted,
    watch,
} from "vue";
import { useRouter } from "vue-router";
import { useI18n } from "vue-i18n";
import { GraduationCap, Clock, XCircle, ArrowLeft } from "lucide-vue-next";
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
const schools = computed(() => store.schools);
const storeLoading = computed(() => store.loading);
const showForm = ref(false);
const submitting = ref(false);
const academicMatchState = ref<
    "idle" | "waiting" | "checking" | "matched" | "mismatch" | "error"
>("idle");
const academicMatchMessage = ref("");
let academicMatchTimer: ReturnType<typeof setTimeout> | undefined;
let academicMatchRunID = 0;

const form = reactive({
    schoolCode: "",
    studentID: "",
    studentName: "",
    email: "",
    emailCode: "",
    password: "",
    manualFormData: {} as Record<string, string>,
    consent: false,
});

function goBack() {
    void router.push("/identity");
}

const selectedSchool = computed(() => {
    if (!form.schoolCode) return null;
    return schools.value.find((s) => s.schoolCode === form.schoolCode) ?? null;
});

const verifiedStudentRows = computed(() => {
    if (!profile.value || profile.value.verificationStatus !== "verified")
        return [];
    const rows: Array<{ label: string; value: string }> = [];
    if (profile.value.schoolID) {
        rows.push({
            label: t("user.verification.student.boundSchool"),
            value: schoolName(profile.value.schoolID),
        });
    }
    if (profile.value.activeStudentID) {
        rows.push({
            label: t("user.verification.student.activeStudentId"),
            value: profile.value.activeStudentID,
        });
    }
    const studentIDs = profile.value.studentIDs?.filter(Boolean) ?? [];
    if (studentIDs.length > 0) {
        rows.push({
            label: t("user.verification.student.studentIds"),
            value: studentIDs.join(" / "),
        });
    }
    if (profile.value.verificationMethod) {
        rows.push({
            label: t("user.verification.student.method"),
            value: t(
                `user.verification.student.methods.${profile.value.verificationMethod}`,
            ),
        });
    }
    if (profile.value.verifiedAt) {
        rows.push({
            label: t("user.verification.student.verifiedAt"),
            value: new Date(profile.value.verifiedAt).toLocaleString(),
        });
    }
    return rows;
});

function schoolName(schoolID: number): string {
    const school = schools.value.find((s) => s.schoolID === schoolID);
    return school
        ? `${school.schoolName}（${school.schoolCode}）`
        : String(schoolID);
}

const manualFields = computed(() =>
    normalizeManualFields(selectedSchool.value?.manualFormFields),
);
const selectedSchoolRequiresAcademicEmail = computed(
    () =>
        selectedSchool.value?.schoolEmailIdentityPolicy?.type ===
        "academic_student_email",
);
const canRequestEmailOTP = computed(() => {
    if (!selectedSchool.value || !selectedSchoolRequiresAcademicEmail.value)
        return false;
    return academicMatchState.value === "matched" && form.email.trim() !== "";
});
const academicMatchMessageClass = computed(() => {
    if (academicMatchState.value === "matched")
        return "mb-4 text-sm text-green-700";
    if (academicMatchState.value === "checking")
        return "mb-4 text-sm text-text-secondary";
    return "mb-4 text-sm text-red-600";
});

const canSubmit = computed(() => {
    if (!selectedSchool.value) return false;
    if (!form.consent) return false;

    if (selectedSchoolRequiresAcademicEmail.value) {
        return (
            canRequestEmailOTP.value &&
            form.email.trim() !== "" &&
            form.emailCode.trim() !== ""
        );
    }

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
        if (selectedSchoolRequiresAcademicEmail.value) {
            await store.verifyStudentEmailOTP({
                schoolCode: form.schoolCode,
                email: form.email,
                code: form.emailCode.trim(),
                consent: form.consent,
            });
            await store.fetchStatus();
            showForm.value = false;
            return;
        }
        const manualFormData = normalizeManualFormData(form.manualFormData);
        await store.verifyStudent({
            schoolCode: form.schoolCode,
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

async function requestEmailOTP() {
    if (submitting.value) return;
    if (!canRequestEmailOTP.value) {
        toast.error(academicRequestBlockedMessage());
        return;
    }

    submitting.value = true;
    try {
        const result = await store.requestStudentEmailOTP({
            schoolCode: form.schoolCode,
            studentID: form.studentID.trim(),
            studentName: form.studentName.trim(),
        });
        form.email = result.email;
        toast.success("验证码已发送");
    } catch (err) {
        toast.error(err instanceof Error ? err.message : "验证码发送失败");
    } finally {
        submitting.value = false;
    }
}

watch(
    () => form.schoolCode,
    () => {
        resetAcademicMatch();
        form.studentID = "";
        form.studentName = "";
        form.email = "";
        form.emailCode = "";
        form.password = "";
        form.manualFormData = {};
        form.consent = false;
    },
);

watch(
    () => [form.studentID, form.studentName],
    () => {
        form.email = "";
        form.emailCode = "";
    },
);

watch(
    () => [
        selectedSchoolRequiresAcademicEmail.value,
        form.schoolCode,
        form.studentID,
        form.studentName,
    ],
    () => {
        scheduleAcademicMatch();
    },
);

onMounted(async () => {
    const results = await Promise.allSettled([
        store.fetchStatus(),
        store.fetchSchools(),
    ]);
    if (results.some((result) => result.status === "rejected")) {
        toast.error(t("common.loadFailed"));
    }
});

onBeforeUnmount(() => {
    clearAcademicMatchTimer();
    academicMatchRunID += 1;
});

function scheduleAcademicMatch() {
    clearAcademicMatchTimer();
    academicMatchRunID += 1;
    if (!selectedSchoolRequiresAcademicEmail.value) {
        resetAcademicMatch();
        return;
    }
    form.email = "";
    form.emailCode = "";
    const studentID = form.studentID.trim();
    const studentName = form.studentName.trim();
    if (!studentID && !studentName) {
        academicMatchState.value = "idle";
        academicMatchMessage.value = "";
        return;
    }
    if (!studentID || !studentName) {
        academicMatchState.value = "waiting";
        academicMatchMessage.value = "请先输入学号和姓名。";
        return;
    }
    academicMatchState.value = "checking";
    academicMatchMessage.value = "正在匹配学号和姓名...";
    const runID = academicMatchRunID;
    academicMatchTimer = setTimeout(() => {
        void runAcademicMatch(runID, studentID, studentName);
    }, 300);
}

async function runAcademicMatch(
    runID: number,
    studentID: string,
    studentName: string,
) {
    try {
        const result = await store.matchStudentEmailAcademicStudent({
            schoolCode: form.schoolCode,
            studentID,
            studentName,
        });
        if (runID !== academicMatchRunID) return;
        if (result.matched && result.email) {
            academicMatchState.value = "matched";
            academicMatchMessage.value = result.message || "学号和姓名已匹配。";
            form.email = result.email;
            return;
        }
        academicMatchState.value = "mismatch";
        academicMatchMessage.value =
            result.message || "学号和姓名不匹配，请核对后再发送验证码。";
        form.email = "";
    } catch (err) {
        if (runID !== academicMatchRunID) return;
        academicMatchState.value = "error";
        academicMatchMessage.value =
            err instanceof Error
                ? err.message
                : "学籍匹配暂时不可用，请稍后重试。";
        form.email = "";
    }
}

function resetAcademicMatch() {
    clearAcademicMatchTimer();
    academicMatchRunID += 1;
    academicMatchState.value = "idle";
    academicMatchMessage.value = "";
}

function clearAcademicMatchTimer() {
    if (academicMatchTimer === undefined) return;
    clearTimeout(academicMatchTimer);
    academicMatchTimer = undefined;
}

function academicRequestBlockedMessage(): string {
    if (!form.studentID.trim() || !form.studentName.trim()) {
        return "请先输入学号和姓名。";
    }
    if (academicMatchState.value === "checking") {
        return "请等待学号和姓名匹配完成。";
    }
    return "请先通过学号和姓名匹配。";
}
</script>
