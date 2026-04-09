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

// After OIDC completes, always land on an actual admin route (never back
// to /auth/login) so the guard runs initSession() and evaluates access.
// Respect ?redirect= from deep links; fall back to admin home.
const postLoginPath = computed(
  () => (route.query.redirect as string) || preferences.app.defaultHomePath,
);

onMounted(async () => {
  if (!isForbidden.value) {
    await authStore.redirectToLogin(postLoginPath.value);
  }
});

async function handleTryDifferentAccount() {
  // Clear the existing forbidden session cookie first, then re-authenticate.
  // Redirect to admin home — the guard will evaluate the new session.
  await authStore.logout(false);
  await authStore.redirectToLogin(preferences.app.defaultHomePath);
}
</script>

<template>
  <div v-if="isForbidden" class="flex h-full items-center justify-center">
    <div class="text-center">
      <p class="text-destructive mb-2 text-2xl font-bold">
        {{ $t('authentication.accessDenied') }}
      </p>
      <p class="text-muted-foreground mb-6 text-sm">
        {{ $t('authentication.accessDeniedDesc') }}
      </p>
      <button
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
