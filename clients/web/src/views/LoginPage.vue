<template>
    <div class="login-page">
        <div class="login-card">
            <!-- 渐变条 -->
            <div class="card-gradient-bar"></div>

            <h1 class="brand-name gradient-text">StuHelper</h1>
            <p class="subtitle">{{ $t('login.subtitle') }}</p>

            <div class="login-buttons">
                <button
                    class="btn-primary"
                    @click="handleLogin"
                    :disabled="loading"
                >
                    {{ loading ? $t('login.redirecting') : $t('login.ssoLogin') }}
                </button>

                <button
                    class="btn-ghost"
                    @click="handleSignup"
                    :disabled="loading"
                >
                    {{ $t('login.signup') }}
                </button>
            </div>

            <p class="hint">{{ $t('login.hint') }}</p>
        </div>
    </div>
</template>

<script setup lang="ts">
import { storeToRefs } from "pinia";
import { useI18n } from "vue-i18n";
import { ElMessage } from "element-plus";
import { useAuthStore } from "@/stores/auth";

const { t } = useI18n();
const authStore = useAuthStore();
const { loading } = storeToRefs(authStore);

const handleLogin = async () => {
    try {
        await authStore.login();
    } catch (e) {
        const message = e instanceof Error ? e.message : t("common.login.loginFailed");
        ElMessage.error(message);
    }
};

const handleSignup = async () => {
    try {
        await authStore.signup();
    } catch (e) {
        const message = e instanceof Error ? e.message : t("common.login.signupFailed");
        ElMessage.error(message);
    }
};
</script>

<style scoped>
.login-page {
    min-height: 100vh;
    display: flex;
    align-items: center;
    justify-content: center;
    background: var(--bg-base);
    padding: var(--space-4);
}

.login-card {
    background: var(--bg-glass);
    backdrop-filter: blur(20px);
    padding: var(--space-12) var(--space-10);
    border: 1px solid var(--border);
    border-radius: var(--radius-xl);
    box-shadow: var(--shadow-card);
    text-align: center;
    max-width: 380px;
    width: 100%;
    animation: fadeIn var(--duration-slower) var(--ease-out);
    overflow: hidden;
}

/* 渐变条 */
.card-gradient-bar {
    height: 3px;
    background: var(--gradient-brand);
    margin: calc(-1 * var(--space-12)) calc(-1 * var(--space-10)) var(--space-6);
}

.brand-name {
    font-family: var(--font-sans);
    font-size: var(--text-2xl);
    font-weight: var(--weight-extrabold);
    letter-spacing: var(--tracking-tight);
    margin: 0 0 var(--space-2) 0;
}

.subtitle {
    color: var(--text-muted);
    margin-bottom: var(--space-8);
    font-size: var(--text-sm);
    line-height: var(--leading-relaxed);
}

.login-buttons {
    display: flex;
    flex-direction: column;
    gap: var(--space-3);
}

.btn-primary,
.btn-ghost {
    padding: var(--space-3) var(--space-6);
    border-radius: var(--radius-full);
    font-size: var(--text-sm);
    font-weight: var(--weight-medium);
    cursor: pointer;
    transition: all var(--duration-fast) var(--ease-out);
}

.btn-primary {
    background: var(--gradient-brand);
    color: white;
    border: none;
}

.btn-primary:hover:not(:disabled) {
    opacity: 0.9;
    transform: translateY(-1px);
}

.btn-ghost {
    background: transparent;
    color: var(--text-secondary);
    border: 1px solid var(--border);
}

.btn-ghost:hover:not(:disabled) {
    color: var(--text-primary);
    border-color: var(--text-primary);
}

.btn-primary:disabled,
.btn-ghost:disabled {
    opacity: 0.5;
    cursor: not-allowed;
}

.hint {
    margin-top: var(--space-6);
    color: var(--text-muted);
    font-size: var(--text-xs);
}
</style>
