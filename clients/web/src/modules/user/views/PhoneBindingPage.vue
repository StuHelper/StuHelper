<template>
    <main class="max-w-[600px] mx-auto p-6 animate-fade-in max-sm:p-4">
        <header class="flex items-center gap-3 mb-6">
            <button
                type="button"
                class="p-2 bg-transparent rounded-lg text-text-muted cursor-pointer transition-colors duration-fast hover:text-text-primary"
                :aria-label="t('common.actions.back')"
                @click="goBack"
            >
                <ArrowLeft class="size-5" aria-hidden="true" />
            </button>
            <h1
                class="font-sans text-xl font-extrabold tracking-tight text-text-primary m-0"
            >
                {{ t("user.verification.phone.title") }}
            </h1>
        </header>

        <section
            class="mb-4 rounded-xl border p-5 shadow-card"
            :class="
                alreadyBound
                    ? 'border-green-500/30 bg-green-500/5'
                    : 'border-border bg-bg-card'
            "
        >
            <div class="flex items-start gap-3">
                <span
                    class="grid size-11 shrink-0 place-items-center rounded-lg"
                    :class="
                        alreadyBound
                            ? 'bg-green-500/10 text-green-600'
                            : 'bg-bg-base text-text-muted'
                    "
                >
                    <CheckCircle2
                        v-if="alreadyBound"
                        class="size-6"
                        aria-hidden="true"
                    />
                    <Phone v-else class="size-6" aria-hidden="true" />
                </span>
                <div class="min-w-0">
                    <h2 class="m-0 text-base font-bold text-text-primary">
                        {{
                            alreadyBound
                                ? t("user.verification.phone.bound")
                                : t("user.verification.phone.unbound")
                        }}
                    </h2>
                    <p
                        v-if="maskedPhone"
                        class="m-0 mt-1 text-sm text-text-muted"
                    >
                        {{ maskedPhone }}
                    </p>
                    <p
                        class="m-0 mt-3 text-sm leading-relaxed text-text-secondary"
                    >
                        {{ t("user.verification.phone.ssoEquivalent") }}
                    </p>
                </div>
            </div>
        </section>

        <section class="bg-bg-card rounded-xl p-5 shadow-card">
            <h2 class="m-0 mb-5 text-base font-bold text-text-primary">
                {{
                    alreadyBound
                        ? t("user.verification.phone.updateTitle")
                        : t("user.verification.phone.bindTitle")
                }}
            </h2>

            <div class="mb-5">
                <label
                    class="block text-sm font-semibold text-text-primary mb-2"
                    for="phone-binding-input"
                >
                    {{ t("user.verification.phone.phoneNumber") }}
                </label>
                <input
                    id="phone-binding-input"
                    v-model="phone"
                    type="tel"
                    maxlength="11"
                    inputmode="numeric"
                    :placeholder="t('user.verification.phone.phoneNumber')"
                    :disabled="loading"
                    class="w-full rounded-lg border border-border bg-bg-base/60 px-3 py-2.5 text-sm text-text-primary outline-none transition-colors duration-fast placeholder-text-muted focus:border-primary focus:ring-2 focus:ring-primary/15"
                />
            </div>

            <div class="mb-5">
                <label
                    class="block text-sm font-semibold text-text-primary mb-2"
                    for="phone-binding-otp"
                >
                    {{ t("user.verification.phone.verifyCode") }}
                </label>
                <div class="flex gap-2">
                    <input
                        id="phone-binding-otp"
                        v-model="otpCode"
                        type="text"
                        inputmode="numeric"
                        maxlength="6"
                        :placeholder="t('user.verification.phone.verifyCode')"
                        :disabled="loading"
                        class="min-w-0 flex-1 rounded-lg border border-border bg-bg-base/60 px-3 py-2.5 text-sm text-text-primary outline-none transition-colors duration-fast placeholder-text-muted focus:border-primary focus:ring-2 focus:ring-primary/15"
                    />
                    <button
                        type="button"
                        class="px-4 py-2 bg-accent text-white rounded-lg text-sm font-medium border-0 cursor-pointer disabled:opacity-60 disabled:cursor-not-allowed whitespace-nowrap"
                        :disabled="!canSendCode"
                        @click="onSendCode"
                    >
                        {{ sendButtonText }}
                    </button>
                </div>
            </div>

            <button
                type="button"
                class="w-full py-2.5 bg-text-primary text-bg-base rounded-lg text-sm font-medium cursor-pointer transition-colors duration-fast hover:bg-accent hover:text-white border-0 disabled:opacity-50 disabled:cursor-not-allowed"
                :disabled="!canSubmit"
                @click="onSubmit"
            >
                {{
                    loading
                        ? t("common.actions.submit")
                        : alreadyBound
                          ? t("user.verification.phone.update")
                          : t("user.verification.phone.bind")
                }}
            </button>
        </section>
    </main>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { ArrowLeft, CheckCircle2, Phone } from "lucide-vue-next";
import { getErrorStatus } from "@/api/errors";
import { useToast } from "@/composables/useToast";
import { useVerificationStore } from "@/stores/verification";

const { t } = useI18n();
const router = useRouter();
const toast = useToast();
const verificationStore = useVerificationStore();

const phone = ref("");
const otpCode = ref("");
const loading = ref(false);
const sending = ref(false);
const cooldown = ref(0);
let cooldownTimer: ReturnType<typeof setInterval> | null = null;

const phonePattern = /^1[3-9]\d{9}$/;

const alreadyBound = computed(
    () =>
        verificationStore.profile?.phoneVerified === true ||
        verificationStore.userSurface?.phoneBound === true,
);
const maskedPhone = computed(
    () =>
        verificationStore.profile?.phone ??
        verificationStore.userSurface?.phone ??
        "",
);

const isPhoneValid = computed(() => phonePattern.test(phone.value));
const isCodeValid = computed(() => /^\d{6}$/.test(otpCode.value));
const canSendCode = computed(
    () =>
        isPhoneValid.value &&
        cooldown.value === 0 &&
        !sending.value &&
        !loading.value,
);
const canSubmit = computed(
    () => isPhoneValid.value && isCodeValid.value && !loading.value,
);

const sendButtonText = computed(() => {
    if (sending.value) return t("user.verification.phone.sending");
    if (cooldown.value > 0) {
        return t("user.verification.phone.resend", {
            seconds: cooldown.value,
        });
    }
    return t("user.verification.phone.sendCode");
});

function startCooldown() {
    cooldown.value = 60;
    cooldownTimer = setInterval(() => {
        cooldown.value -= 1;
        if (cooldown.value <= 0 && cooldownTimer) {
            clearInterval(cooldownTimer);
            cooldownTimer = null;
        }
    }, 1000);
}

async function onSendCode() {
    if (!isPhoneValid.value) {
        toast.error(t("user.verification.phone.invalidPhone"));
        return;
    }
    sending.value = true;
    try {
        await verificationStore.requestBindPhoneOTP(phone.value);
        toast.success(t("user.verification.phone.codeSent"));
        startCooldown();
    } catch (err: unknown) {
        const status = getErrorStatus(err);
        if (status === 429) {
            toast.error(t("user.verification.phone.tooManyRequests"));
        } else if (status === 400) {
            toast.error(t("user.verification.phone.invalidPhone"));
        } else if (status === 503) {
            toast.error(t("user.verification.phone.serviceUnavailable"));
        } else {
            toast.error(t("common.actions.operationFailed"));
        }
    } finally {
        sending.value = false;
    }
}

async function onSubmit() {
    if (!canSubmit.value) return;
    const wasBound = alreadyBound.value;
    loading.value = true;
    try {
        await verificationStore.bindPhone({
            phone: phone.value,
            otpCode: otpCode.value,
        });
        phone.value = "";
        otpCode.value = "";
        toast.success(
            wasBound
                ? t("user.verification.phone.updateSuccess")
                : t("user.verification.phone.bindSuccess"),
        );
    } catch (err: unknown) {
        const status = getErrorStatus(err);
        if (status === 401) {
            toast.error(t("user.verification.phone.invalidCode"));
        } else if (status === 409) {
            toast.error(t("user.verification.phone.alreadyBound"));
        } else if (status === 429) {
            toast.error(t("user.verification.phone.tooManyAttempts"));
        } else if (status === 503) {
            toast.error(t("user.verification.phone.serviceUnavailable"));
        } else {
            toast.error(t("common.actions.operationFailed"));
        }
    } finally {
        loading.value = false;
    }
}

function goBack() {
    void router.push("/identity");
}

onMounted(() => {
    void Promise.all([
        verificationStore.fetchStatus(),
        verificationStore.fetchUserSurface(),
    ]).catch(() => {
        toast.error(t("common.loadFailed"));
    });
});

onUnmounted(() => {
    if (cooldownTimer) clearInterval(cooldownTimer);
});
</script>
