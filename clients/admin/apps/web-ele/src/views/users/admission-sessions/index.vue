<script setup lang="ts">
import type { AdmissionSessionAction, StatusFilter } from './options';

import type {
  AdmissionSession,
  ListAdmissionSessionsParams,
} from '#/api/admin';

import { computed } from 'vue';

import { useAccessStore } from '@vben/stores';

import { ElAlert, ElButton, ElMessage } from 'element-plus';

import {
  cancelAdminAdmissionSession,
  listAdmissionSessions,
  regenerateAdminAdmissionSession,
  resendAdminAdmissionSession,
} from '#/api/admin';
import { useAdminAction } from '#/composables/use-admin-action';
import { useAdminList } from '#/composables/use-admin-list';
import { $t } from '#/locales';
import { copyTextToClipboard } from '#/utils/clipboard';

import AdminContentLayout from '../../shared/AdminContentLayout.vue';
import { ADMIN_DEFAULT_PAGE_SIZE } from '../../shared/pagination';
import AdmissionSessionFilters from './AdmissionSessionFilters.vue';
import AdmissionSessionTable from './AdmissionSessionTable.vue';

const accessStore = useAccessStore();
const canManage = computed(() =>
  accessStore.accessCodes.includes('admission:session:manage'),
);

const {
  fetchData,
  items,
  loadError,
  loading,
  query,
  resetPageAndFetch,
  total,
} = useAdminList<
  AdmissionSession,
  {
    botSelfID: string;
    guildID: string;
    page: number;
    pageSize: number;
    platform: string;
    qqID: string;
    status: StatusFilter;
  }
>({
  fetcher: (listQuery) => {
    const params: ListAdmissionSessionsParams = {
      page: listQuery.page,
      pageSize: listQuery.pageSize,
    };
    if (listQuery.status) params.status = listQuery.status;
    if (listQuery.platform.trim()) params.platform = listQuery.platform.trim();
    if (listQuery.botSelfID.trim()) {
      params.botSelfID = listQuery.botSelfID.trim();
    }
    if (listQuery.guildID.trim()) params.guildID = listQuery.guildID.trim();
    if (listQuery.qqID.trim()) params.qqID = listQuery.qqID.trim();
    return listAdmissionSessions(params);
  },
  initialQuery: {
    botSelfID: '',
    guildID: '',
    page: 1,
    pageSize: ADMIN_DEFAULT_PAGE_SIZE,
    platform: 'qq',
    qqID: '',
    status: '',
  },
});

const { actionError, clearActionError, pendingActionKinds, runAction } =
  useAdminAction();

function resetQuery() {
  query.platform = 'qq';
  query.botSelfID = '';
  query.guildID = '';
  query.qqID = '';
  query.status = '';
  resetPageAndFetch();
}

async function copyAuthURL(url: string) {
  await copyToClipboard(url, $t('admin.users.admissionSessions.authURLCopied'));
}

async function copyReissueCommand(command: string) {
  await copyToClipboard(
    command,
    $t('admin.users.admissionSessions.reissueCopied'),
  );
}

async function copyToClipboard(text: string, successMessage: string) {
  const copied = await copyTextToClipboard(text);
  if (copied) {
    ElMessage.success(successMessage);
    return true;
  }
  ElMessage.error($t('admin.users.admissionSessions.copyFailed'));
  return false;
}

async function requestResend(id: string) {
  await runSessionAction(
    id,
    'resend',
    () => resendAdminAdmissionSession(id),
    $t('admin.users.admissionSessions.resendQueued'),
  );
}

async function requestRegenerate(id: string) {
  await runSessionAction(
    id,
    'regenerate',
    () => regenerateAdminAdmissionSession(id),
    $t('admin.users.admissionSessions.regenerated'),
  );
}

async function requestCancel(id: string) {
  await runSessionAction(
    id,
    'cancel',
    () => cancelAdminAdmissionSession(id),
    $t('admin.users.admissionSessions.sessionCancelled'),
  );
}

async function runSessionAction(
  id: string,
  kind: AdmissionSessionAction,
  request: () => Promise<unknown>,
  successMessage: string,
) {
  const succeeded = await runAction(request, { id, kind, successMessage });
  if (succeeded) {
    await fetchData();
  }
}
</script>

<template>
  <AdminContentLayout
    :title="$t('admin.routes.userSystem.admissionSessions')"
    :total="total"
  >
    <template #toolbar>
      <AdmissionSessionFilters
        v-model:qq-i-d="query.qqID"
        v-model:guild-i-d="query.guildID"
        v-model:bot-self-i-d="query.botSelfID"
        v-model:platform="query.platform"
        v-model:status="query.status"
        @search="resetPageAndFetch"
        @reset="resetQuery"
      />
    </template>

    <ElAlert
      v-if="loadError"
      class="admin-load-error"
      type="error"
      :closable="false"
      show-icon
      :title="loadError"
    >
      <ElButton size="small" :loading="loading" @click="fetchData">
        {{ $t('admin.common.retry') }}
      </ElButton>
    </ElAlert>

    <ElAlert
      v-if="actionError"
      class="admin-load-error"
      type="error"
      :closable="true"
      show-icon
      :title="actionError"
      @close="clearActionError"
    />

    <AdmissionSessionTable
      v-model:page="query.page"
      v-model:page-size="query.pageSize"
      :can-manage="canManage"
      :action-loading-by-id="pendingActionKinds"
      :loading="loading"
      :items="items"
      :total="total"
      @copy-auth-u-r-l="copyAuthURL"
      @copy-reissue-command="copyReissueCommand"
      @request-resend="requestResend"
      @request-regenerate="requestRegenerate"
      @request-cancel="requestCancel"
      @page-change="fetchData"
      @page-size-change="resetPageAndFetch"
    />
  </AdminContentLayout>
</template>
