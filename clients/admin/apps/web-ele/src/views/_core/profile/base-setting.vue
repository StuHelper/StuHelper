<script setup lang="ts">
import type { AuthApi } from '#/api';

import { computed, onMounted, ref } from 'vue';

import {
  ElDescriptions,
  ElDescriptionsItem,
  ElSkeleton,
  ElTag,
} from 'element-plus';

import { getUserInfoApi } from '#/api';
import { $t } from '#/locales';

const loading = ref(true);
const profile = ref<AuthApi.MeResult | null>(null);

const roles = computed(() => profile.value?.roles ?? []);

function displayValue(value: null | string | undefined) {
  return value && value.trim() ? value : '-';
}

onMounted(async () => {
  try {
    const { me } = await getUserInfoApi();
    profile.value = me;
  } finally {
    loading.value = false;
  }
});
</script>
<template>
  <ElSkeleton :loading="loading" animated>
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
