<script setup lang="ts">
import { computed } from 'vue';

import { ElAlert } from 'element-plus';

import { $t } from '#/locales';
import { useAuthStore } from '#/store';

const authStore = useAuthStore();

// URL comes from the backend /auth/me response — the backend derives it
// from the OIDC issuer config, so the frontend never needs to know
// where the identity provider lives.
const accountSettingsUrl = computed(() => authStore.accountSettingsUrl);

function handleGoToAccountSettings() {
  if (accountSettingsUrl.value) {
    window.open(accountSettingsUrl.value, '_blank', 'noopener,noreferrer');
  }
}
</script>
<template>
  <div class="w-1/3">
    <ElAlert
      :title="$t('admin.profile.password.managedExternally')"
      type="info"
      :closable="false"
      show-icon
    >
      <template #default>
        <p class="mb-3">
          {{ $t('admin.profile.password.managedExternallyDesc') }}
        </p>
        <a
          v-if="accountSettingsUrl"
          class="text-primary cursor-pointer underline"
          @click="handleGoToAccountSettings"
        >
          {{ $t('admin.profile.password.goToAccountSettings') }}
        </a>
      </template>
    </ElAlert>
  </div>
</template>
