<script lang="ts" setup>
import { useRouter } from 'vue-router';

import { AnalysisChartCard, WorkbenchHeader } from '@vben/common-ui';
import { preferences } from '@vben/preferences';
import { useUserStore } from '@vben/stores';

import { ElButton, ElEmpty } from 'element-plus';

import { $t } from '#/locales';

const router = useRouter();
const userStore = useUserStore();

async function goTo(path: string) {
  await router.push(path);
}
</script>

<template>
  <div class="p-5">
    <WorkbenchHeader
      :avatar="userStore.userInfo?.avatar || preferences.app.defaultAvatar"
    >
      <template #title>
        {{ $t('admin.dashboard.workspace.header.title', { name: userStore.userInfo?.realName ?? '' }) }}
      </template>
      <template #description>{{ $t('admin.dashboard.workspace.header.description') }}</template>
    </WorkbenchHeader>

    <AnalysisChartCard class="mt-5" :title="$t('page.dashboard.workspace')">
      <ElEmpty>
        <template #image>
          <div class="mx-auto mb-3 text-4xl">🛠️</div>
        </template>
        <template #default>
          <div class="space-y-4 text-center">
            <p class="text-base font-medium">
              {{ $t('admin.dashboard.workspace.placeholderTitle') }}
            </p>
            <p class="text-[var(--el-text-color-secondary)]">
              {{ $t('admin.dashboard.workspace.placeholderDescription') }}
            </p>
            <div class="flex flex-wrap justify-center gap-3">
              <ElButton type="primary" @click="goTo('/analytics')">
                {{ $t('admin.dashboard.workspace.placeholderPrimaryAction') }}
              </ElButton>
              <ElButton plain @click="goTo('/content/reviews')">
                {{ $t('admin.dashboard.workspace.placeholderSecondaryAction') }}
              </ElButton>
            </div>
          </div>
        </template>
      </ElEmpty>
    </AnalysisChartCard>
  </div>
</template>
