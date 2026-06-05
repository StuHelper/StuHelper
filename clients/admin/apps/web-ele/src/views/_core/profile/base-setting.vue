<script setup lang="ts">
import type { AuthApi } from '#/api';

import { computed, onMounted, ref } from 'vue';

import {
  ElAlert,
  ElButton,
  ElDescriptions,
  ElDescriptionsItem,
  ElSkeleton,
  ElTag,
} from 'element-plus';

import { getUserInfoApi } from '#/api';
import { $t } from '#/locales';

const loading = ref(true);
const loadError = ref('');
const profile = ref<AuthApi.MeResult | null>(null);
let fetchRequestSeq = 0;

const roles = computed(() => profile.value?.roles ?? []);

function displayValue(value: null | string | undefined) {
  return value && value.trim() ? value : '-';
}

async function fetchProfile() {
  const requestSeq = ++fetchRequestSeq;
  loading.value = true;
  loadError.value = '';
  try {
    const { me } = await getUserInfoApi();
    if (requestSeq !== fetchRequestSeq) return;
    profile.value = me;
  } catch (error) {
    if (requestSeq !== fetchRequestSeq) return;
    loadError.value = adminErrorMessage(error);
  } finally {
    if (requestSeq === fetchRequestSeq) {
      loading.value = false;
    }
  }
}

function adminErrorMessage(error: unknown): string {
  return error instanceof Error && error.message
    ? error.message
    : $t('admin.result.requestFailed');
}

onMounted(fetchProfile);
</script>
<template>
  <ElSkeleton :loading="loading" animated>
    <ElAlert
      v-if="loadError"
      class="admin-load-error"
      type="error"
      :closable="false"
      show-icon
      :title="loadError"
    >
      <ElButton size="small" :loading="loading" @click="fetchProfile">
        {{ $t('admin.common.retry') }}
      </ElButton>
    </ElAlert>

    <div v-if="profile" class="max-w-2xl">
      <ElDescriptions :column="1" border>
        <ElDescriptionsItem
          :label="$t('admin.profile.baseSetting.fields.realName')"
        >
          {{ displayValue(profile.displayName) }}
        </ElDescriptionsItem>
        <ElDescriptionsItem
          :label="$t('admin.profile.baseSetting.fields.username')"
        >
          {{ displayValue(profile.name) }}
        </ElDescriptionsItem>
        <ElDescriptionsItem
          :label="$t('admin.profile.baseSetting.fields.email')"
        >
          {{ displayValue(profile.email) }}
        </ElDescriptionsItem>
        <ElDescriptionsItem
          :label="$t('admin.profile.baseSetting.fields.roles')"
        >
          <div v-if="roles.length > 0" class="flex flex-wrap gap-2">
            <ElTag v-for="role in roles" :key="role" effect="plain">
              {{ role }}
            </ElTag>
          </div>
          <span v-else>-</span>
        </ElDescriptionsItem>
      </ElDescriptions>
    </div>
  </ElSkeleton>
</template>
