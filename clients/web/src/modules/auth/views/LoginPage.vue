<template>
    <main class="login-page flex items-center justify-center px-4 py-10">
        <section
            class="w-full max-w-[380px] rounded-xl border border-border bg-bg-card px-4 py-12 text-center shadow-sm sm:px-10"
        >
            <div
                class="h-[3px] bg-gradient-to-r from-primary to-accent -mt-12 -mx-4 sm:-mx-10 mb-6"
                aria-hidden="true"
            />

            <h1
                class="font-sans text-2xl font-extrabold tracking-tight m-0 mb-2 gradient-text"
            >
                {{ t("common.login.title") }}
            </h1>
            <p class="text-text-muted mb-6 text-sm leading-relaxed">
                {{ t("common.login.subtitle") }}
            </p>

            <div class="flex flex-col gap-3">
                <button
                    v-ripple
                    class="rounded-full border-0 bg-gradient-to-br from-primary to-accent px-6 py-3 text-sm font-medium text-white disabled:cursor-not-allowed disabled:opacity-50"
                    :disabled="loading"
                    @click="handleLogin"
                >
                    {{
                        loading
                            ? t("common.login.redirecting")
                            : t("common.login.identityLogin")
                    }}
                </button>

                <button
                    v-ripple
                    class="rounded-full border border-border bg-transparent px-6 py-3 text-sm font-medium text-text-secondary disabled:cursor-not-allowed disabled:opacity-50"
                    :disabled="loading"
                    @click="handleSignup"
                >
                    {{ t("common.login.signup") }}
                </button>

                <p class="mt-2 text-text-muted text-xs">
                    {{ t("common.login.identityHint") }}
                </p>
            </div>

            <button
                v-if="loginError"
                type="button"
                class="mt-4 inline-flex h-10 items-center justify-center rounded-md border border-border bg-bg-card px-4 text-sm font-semibold text-text-primary disabled:opacity-50"
                :disabled="loading"
                @click="handleLogin"
            >
                {{ t("common.actions.retry") }}
            </button>
        </section>
    </main>
</template>

<script setup lang="ts">
import { ref } from "vue";
import { useRoute } from "vue-router";
import { storeToRefs } from "pinia";
import { useI18n } from "vue-i18n";
import { ElMessage } from "element-plus";
import { getErrorMessage } from "@/api/errors";
import { useAuthStore } from "@/stores/auth";
import { storePostLoginRedirect } from "@/utils/auth";
import {
    resolvePostLoginRedirectTarget,
    sanitizePostLoginRedirect,
} from "@/utils/redirect";

const { t } = useI18n();
const route = useRoute();
const authStore = useAuthStore();
const { loading } = storeToRefs(authStore);
const loginError = ref(false);

const isReauthRequest = () => route.query.reauth === "1";

function defaultAuthenticatedRoute(): string {
    return new URL("/", window.location.origin).toString();
}

// 单点登录跳转辅助逻辑
const getRedirectTarget = (): string | undefined => {
    const redirect = sanitizePostLoginRedirect(
        typeof route.query.redirect === "string"
            ? route.query.redirect
            : undefined,
    );
    if (redirect) {
        return resolvePostLoginRedirectTarget(redirect);
    }
    return defaultAuthenticatedRoute();
};

const saveRedirectTarget = () => {
    const redirect = sanitizePostLoginRedirect(
        typeof route.query.redirect === "string"
            ? route.query.redirect
            : undefined,
    );
    if (redirect) {
        storePostLoginRedirect(redirect);
    }
};

const startLoginForCurrentRoute = async () => {
    saveRedirectTarget();
    const startLogin = isReauthRequest()
        ? authStore.reauthenticate
        : authStore.login;
    await startLogin(getRedirectTarget());
};

const handleLogin = async () => {
    try {
        loginError.value = false;
        await startLoginForCurrentRoute();
    } catch (e) {
        loginError.value = true;
        ElMessage.error(getErrorMessage(e, t("common.login.loginFailed")));
    }
};

const handleSignup = async () => {
    try {
        loginError.value = false;
        saveRedirectTarget();
        await authStore.signup(getRedirectTarget());
    } catch (e) {
        loginError.value = true;
        ElMessage.error(getErrorMessage(e, t("common.login.signupFailed")));
    }
};
</script>

<style scoped>
.login-page {
    min-height: calc(100dvh - var(--navbar-height));
    background: linear-gradient(
        135deg,
        var(--color-bg-base) 0%,
        var(--color-bg-card) 48%,
        var(--color-bg-elevated) 100%
    );
}
</style>
