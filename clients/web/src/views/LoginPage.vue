<template>
  <div class="min-h-screen flex items-center justify-center bg-bg-base p-4">
    <div class="bg-bg-glass backdrop-blur-[20px] py-12 px-10 border border-border rounded-xl shadow-card text-center max-w-[380px] w-full animate-fade-in overflow-hidden">
      <!-- 渐变条 -->
      <div class="h-[3px] bg-gradient-to-r from-primary to-accent -mt-12 -mx-10 mb-6"></div>

      <h1 class="font-sans text-2xl font-extrabold tracking-tight m-0 mb-2 gradient-text">StuHelper</h1>
      <p class="text-text-muted mb-8 text-sm leading-relaxed">{{ $t('common.login.subtitle') }}</p>

      <div class="flex flex-col gap-3">
        <button
          class="py-3 px-6 rounded-full text-sm font-medium cursor-pointer transition-all duration-fast ease-out bg-gradient-to-br from-primary to-accent text-white border-none hover:enabled:opacity-90 hover:enabled:-translate-y-px disabled:opacity-50 disabled:cursor-not-allowed"
          @click="handleLogin"
          :disabled="loading"
        >
          {{ loading ? $t('common.login.redirecting') : $t('common.login.ssoLogin') }}
        </button>

        <button
          class="py-3 px-6 rounded-full text-sm font-medium cursor-pointer transition-all duration-fast ease-out bg-transparent text-text-secondary border border-border hover:enabled:text-text-primary hover:enabled:border-text-primary disabled:opacity-50 disabled:cursor-not-allowed"
          @click="handleSignup"
          :disabled="loading"
        >
          {{ $t('common.login.signup') }}
        </button>
      </div>

      <p class="mt-6 text-text-muted text-xs">{{ $t('common.login.hint') }}</p>
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
