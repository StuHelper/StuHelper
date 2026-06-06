<template>
    <main class="login-page flex items-center justify-center px-4 py-10">
        <section class="text-center" aria-live="polite">
            <div
                v-if="!loginError"
                class="mx-auto mb-4 size-9 rounded-full border-2 border-border border-t-primary animate-spin"
                role="status"
                aria-busy="true"
            >
                <span class="sr-only">{{ t("common.login.redirecting") }}</span>
            </div>
            <p class="m-0 text-sm font-medium text-text-secondary">
                {{
                    loginError
                        ? t("common.login.loginFailed")
                        : t("common.login.redirecting")
                }}
            </p>
            <button
                v-if="loginError"
                type="button"
                class="mt-4 inline-flex h-10 items-center justify-center rounded-md border border-border bg-bg-card px-4 text-sm font-semibold text-text-primary transition-colors duration-fast hover:border-primary/40 hover:text-primary disabled:opacity-50"
                :disabled="loading"
                @click="handleLogin"
            >
                {{ t("common.actions.retry") }}
            </button>
        </section>
    </main>
</template>

<script setup lang="ts">
import { onMounted, ref } from "vue";
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

onMounted(() => {
    void handleLogin();
});
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
