<script lang="ts" setup>
import { computed, onMounted } from 'vue';
import { useRoute } from 'vue-router';

import { preferences } from '@vben/preferences';

import { $t } from '#/locales';
import { useAuthStore } from '#/store';

defineOptions({ name: 'Login' });

const route = useRoute();
const authStore = useAuthStore();

const isForbidden = computed(() => route.query.error === 'forbidden');

// OIDC 完成后必须落到真实后台路由，不能回到 /auth/login，
// 这样守卫才能执行 initSession() 并重新判断权限。
// 优先使用深链传入的 ?redirect=，否则回到后台首页。
const postLoginPath = computed(
  () => (route.query.redirect as string) || preferences.app.defaultHomePath,
);

onMounted(async () => {
  if (!isForbidden.value) {
    await authStore.redirectToLogin(postLoginPath.value);
  }
});

async function handleTryDifferentAccount() {
  await authStore.switchAccount(preferences.app.defaultHomePath);
}
</script>

<template>
  <div v-if="isForbidden" class="flex h-full items-center justify-center">
    <div class="text-center">
      <h1 class="text-destructive mb-2 text-2xl font-bold">
        {{ $t('authentication.accessDenied') }}
      </h1>
      <p class="text-muted-foreground mb-6 text-sm">
        {{ $t('authentication.accessDeniedDesc') }}
      </p>
      <button
        type="button"
        class="bg-primary text-primary-foreground rounded-md px-4 py-2 text-sm"
        @click="handleTryDifferentAccount"
      >
        {{ $t('authentication.tryDifferentAccount') }}
      </button>
    </div>
  </div>

  <div v-else class="flex h-full items-center justify-center">
    <p class="text-muted-foreground animate-pulse text-lg">
      Redirecting to login...
    </p>
  </div>
</template>
