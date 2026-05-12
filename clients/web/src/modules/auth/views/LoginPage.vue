<template>
  <div class="login-page flex items-center justify-center px-4 py-10 relative overflow-hidden">
    <div class="login-grid" aria-hidden="true" />
    <div class="login-sheen" aria-hidden="true" />
    <div class="relative z-10 glass-card glow-border py-12 px-4 sm:px-10 rounded-xl shadow-lg text-center max-w-[380px] w-full animate-scale-in overflow-hidden">
      <!-- 渐变条 -->
      <div class="h-[3px] bg-gradient-to-r from-primary to-accent -mt-12 -mx-4 sm:-mx-10 mb-6"></div>

      <h1 class="font-sans text-2xl font-extrabold tracking-tight m-0 mb-2 gradient-text">StuHelper</h1>
      <p class="text-text-muted mb-6 text-sm leading-relaxed">{{ $t('common.login.subtitle') }}</p>

      <!-- 登录方式切换 -->
      <div class="flex border-b border-border mb-6">
        <button
          v-for="tab in tabs"
          :key="tab.key"
          class="flex-1 py-2.5 text-sm font-medium border-b-2 transition-colors duration-fast cursor-pointer bg-transparent"
          :class="activeTab === tab.key
            ? 'border-primary text-primary'
            : 'border-transparent text-text-muted hover:text-text-secondary'"
          @click="activeTab = tab.key"
        >
          {{ tab.label }}
        </button>
      </div>

      <!-- 手机验证码登录 -->
      <div v-if="activeTab === 'phone'" class="flex flex-col gap-3">
        <div class="grid grid-cols-[minmax(0,1fr)_auto] items-start gap-2">
          <input
            v-model="phone"
            type="tel"
            maxlength="11"
            :placeholder="$t('common.login.phonePlaceholder')"
            class="min-w-0 w-full py-3 px-4 rounded-lg text-sm bg-bg-card border border-border text-text-primary placeholder:text-text-muted focus:outline-none focus:border-primary transition-colors duration-fast"
            :disabled="loading"
            @keydown.enter="handlePhoneLogin"
          />
          <button
            class="min-w-[96px] shrink-0 py-3 px-4 rounded-lg text-sm font-medium cursor-pointer transition-all duration-fast bg-bg-card border border-border text-primary hover:enabled:bg-primary/10 disabled:opacity-50 disabled:cursor-not-allowed whitespace-nowrap sm:min-w-[110px]"
            :disabled="loading || cooldown > 0 || !isValidPhone"
            @click="handleSendOTP"
          >
            {{ cooldown > 0 ? $t('common.login.resendCode', { n: cooldown }) : $t('common.login.sendCode') }}
          </button>
        </div>

        <OtpCodeInput
          v-model="otpCode"
          :disabled="loading"
          :aria-label="$t('common.login.codePlaceholder')"
          @complete="handleOtpComplete"
        />

        <button
          v-ripple
          class="py-3 px-6 rounded-full text-sm font-medium cursor-pointer transition-all duration-fast ease-out bg-gradient-to-br from-primary to-accent text-white border-none hover:enabled:opacity-90 hover:enabled:-translate-y-px hover:enabled:shadow-glow-primary disabled:opacity-50 disabled:cursor-not-allowed press-spring mt-1"
          :disabled="loading || !canVerify"
          @click="handlePhoneLogin"
        >
          {{ loading ? $t('common.login.verifying') : $t('common.login.verifyAndLogin') }}
        </button>

        <p class="mt-2 text-text-muted text-xs">{{ $t('common.login.phoneHint') }}</p>
      </div>

      <!-- 单点登录 -->
      <div v-else class="flex flex-col gap-3">
        <button
          v-ripple
          class="py-3 px-6 rounded-full text-sm font-medium cursor-pointer transition-all duration-fast ease-out bg-gradient-to-br from-primary to-accent text-white border-none hover:enabled:opacity-90 hover:enabled:-translate-y-px hover:enabled:shadow-glow-primary disabled:opacity-50 disabled:cursor-not-allowed press-spring"
          @click="handleLogin"
          :disabled="loading"
        >
          {{ loading ? $t('common.login.redirecting') : $t('common.login.ssoLogin') }}
        </button>

        <button
          v-ripple
          class="py-3 px-6 rounded-full text-sm font-medium cursor-pointer transition-all duration-fast ease-out bg-transparent text-text-secondary hover:enabled:text-text-primary hover:enabled:border-text-primary disabled:opacity-50 disabled:cursor-not-allowed press-spring"
          @click="handleSignup"
          :disabled="loading"
        >
          {{ $t('common.login.signup') }}
        </button>

        <p class="mt-2 text-text-muted text-xs">{{ $t('common.login.ssoHint') }}</p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onUnmounted } from "vue";
import OtpCodeInput from "@/components/common/OtpCodeInput.vue";
import { useRoute, useRouter } from "vue-router";
import { storeToRefs } from "pinia";
import { useI18n } from "vue-i18n";
import { ElMessage } from "element-plus";
import { getErrorMessage } from "@/api/errors";
import { useAuthStore } from "@/stores/auth";

const { t } = useI18n();
const route = useRoute();
const router = useRouter();
const authStore = useAuthStore();
const { loading } = storeToRefs(authStore);

// 当前登录方式
const activeTab = ref<"phone" | "sso">("phone");
const tabs = computed(() => [
    { key: "phone" as const, label: t("common.login.phoneLogin") },
    { key: "sso" as const, label: t("common.login.ssoLogin") },
]);

// 手机验证码登录表单状态
const phone = ref("");
const otpCode = ref("");
const cooldown = ref(0);

let cooldownTimer: ReturnType<typeof setInterval> | null = null;

const phonePattern = /^1[3-9]\d{9}$/;
const isValidPhone = computed(() => phonePattern.test(phone.value));
const canVerify = computed(() => isValidPhone.value && otpCode.value.length === 6);

function sanitizeInternalRedirect(redirect: string | null | undefined): string | undefined {
    if (!redirect) return undefined;
    if (redirect.startsWith("/") && !redirect.startsWith("//")) {
        return redirect;
    }

    try {
        const parsed = new URL(redirect, window.location.origin);
        if (parsed.origin !== window.location.origin) return undefined;
        return `${parsed.pathname}${parsed.search}${parsed.hash}`;
    } catch (_error) { void _error;
        return undefined;
    }
}

function getPostLoginRedirect(): string | undefined {
    return (
        sanitizeInternalRedirect(
            typeof route.query.redirect === "string" ? route.query.redirect : undefined,
        ) ||
        sanitizeInternalRedirect(sessionStorage.getItem("post_login_redirect"))
    );
}

function defaultAuthenticatedRoute(): string {
    return new URL("/", window.location.origin).toString();
}

function clearStoredRedirects() {
    sessionStorage.removeItem("post_login_redirect");
}

const startCooldown = (seconds: number) => {
    cooldown.value = seconds;
    if (cooldownTimer) clearInterval(cooldownTimer);
    cooldownTimer = setInterval(() => {
        cooldown.value -= 1;
        if (cooldown.value <= 0) {
            if (cooldownTimer) clearInterval(cooldownTimer);
            cooldownTimer = null;
        }
    }, 1000);
};

onUnmounted(() => {
    if (cooldownTimer) clearInterval(cooldownTimer);
});

const handleSendOTP = async () => {
    if (!isValidPhone.value) {
        ElMessage.warning(t("common.login.invalidPhone"));
        return;
    }

    try {
        const data = await authStore.requestPhoneOTP(phone.value);
        ElMessage.success(t("common.login.otpSent"));
        startCooldown(data?.cooldown ?? 60);
    } catch (e) {
        ElMessage.error(getErrorMessage(e, t("common.login.otpSendFailed")));
    }
};

const handlePhoneLogin = async () => {
    if (!canVerify.value) return;

    try {
        await authStore.verifyPhoneOTP(phone.value, otpCode.value);

        // 登录成功后跳转
        const redirect = getPostLoginRedirect();
        clearStoredRedirects();
        if (redirect) {
            router.replace(redirect);
        } else {
            router.replace({ name: "home" });
        }
    } catch (e) {
        ElMessage.error(getErrorMessage(e, t("common.login.otpVerifyFailed")));
    }
};

const handleOtpComplete = async (value: string) => {
    otpCode.value = value;
    await handlePhoneLogin();
};

// 单点登录跳转辅助逻辑
const getRedirectTarget = (): string | undefined => {
    const redirect = sanitizeInternalRedirect(
        typeof route.query.redirect === "string" ? route.query.redirect : undefined,
    );
    if (redirect) {
        return new URL(redirect, window.location.origin).toString();
    }
    return defaultAuthenticatedRoute();
};

const saveRedirectTarget = () => {
    const redirect = sanitizeInternalRedirect(
        typeof route.query.redirect === "string" ? route.query.redirect : undefined,
    );
    if (redirect) {
        sessionStorage.setItem("post_login_redirect", redirect);
    }
};

const handleLogin = async () => {
    try {
        saveRedirectTarget();
        await authStore.login(getRedirectTarget());
    } catch (e) {
        ElMessage.error(getErrorMessage(e, t("common.login.loginFailed")));
    }
};

const handleSignup = async () => {
    try {
        saveRedirectTarget();
        await authStore.signup(getRedirectTarget());
    } catch (e) {
        ElMessage.error(getErrorMessage(e, t("common.login.signupFailed")));
    }
};
</script>

<style scoped>
.login-page {
    min-height: calc(100dvh - var(--navbar-height));
    isolation: isolate;
    background:
        linear-gradient(135deg, var(--color-bg-base) 0%, var(--color-bg-card) 48%, var(--color-bg-elevated) 100%);
}

.login-grid,
.login-sheen {
    position: absolute;
    inset: 0;
    pointer-events: none;
}

.login-grid {
    background-image:
        linear-gradient(color-mix(in srgb, var(--color-text-primary) 6%, transparent) 1px, transparent 1px),
        linear-gradient(90deg, color-mix(in srgb, var(--color-text-primary) 6%, transparent) 1px, transparent 1px);
    background-size: 48px 48px;
    mask-image: linear-gradient(to bottom, transparent 0%, black 18%, black 82%, transparent 100%);
    opacity: 0.34;
}

.login-sheen {
    background:
        linear-gradient(115deg, transparent 0%, color-mix(in srgb, var(--color-primary) 10%, transparent) 34%, transparent 56%),
        linear-gradient(250deg, transparent 8%, color-mix(in srgb, var(--color-accent) 8%, transparent) 42%, transparent 68%);
    opacity: 0.72;
}
</style>
