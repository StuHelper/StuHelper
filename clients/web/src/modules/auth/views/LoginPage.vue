<template>
  <div class="login-page flex items-center justify-center px-4 py-10 relative overflow-hidden">
    <div class="login-grid" aria-hidden="true" />
    <div class="login-sheen" aria-hidden="true" />
    <div class="relative z-10 glass-card glow-border py-12 px-4 sm:px-10 rounded-xl shadow-lg text-center max-w-[380px] w-full animate-scale-in overflow-hidden">
      <!-- 渐变条 -->
      <div class="h-[3px] bg-gradient-to-r from-primary to-accent -mt-12 -mx-4 sm:-mx-10 mb-6"></div>

      <h1 class="font-sans text-2xl font-extrabold tracking-tight m-0 mb-2 gradient-text">StuHelper</h1>
      <p class="text-text-muted mb-6 text-sm leading-relaxed">{{ $t('common.login.subtitle') }}</p>

      <div class="flex flex-col gap-3">
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
import { useRoute } from "vue-router";
import { storeToRefs } from "pinia";
import { useI18n } from "vue-i18n";
import { ElMessage } from "element-plus";
import { getErrorMessage } from "@/api/errors";
import { useAuthStore } from "@/stores/auth";

const { t } = useI18n();
const route = useRoute();
const authStore = useAuthStore();
const { loading } = storeToRefs(authStore);

const isReauthRequest = () => route.query.reauth === "1";

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

function defaultAuthenticatedRoute(): string {
    return new URL("/", window.location.origin).toString();
}

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
        const startLogin = isReauthRequest() ? authStore.reauthenticate : authStore.login;
        await startLogin(getRedirectTarget());
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
