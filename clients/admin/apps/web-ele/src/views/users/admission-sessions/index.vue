<script setup lang="ts">
import type { AdmissionSessionAction, StatusFilter } from './options';

import type { ListAdmissionSessionsParams } from '#/api/admin';

import { computed, onMounted, reactive, ref } from 'vue';

import { useAccessStore } from '@vben/stores';

import { ElAlert, ElButton, ElMessage } from 'element-plus';

import {
  cancelAdminAdmissionSession,
  listAdmissionSessions,
  regenerateAdminAdmissionSession,
  resendAdminAdmissionSession,
} from '#/api/admin';
import { $t } from '#/locales';

import AdminContentLayout from '../../shared/AdminContentLayout.vue';
import AdmissionSessionFilters from './AdmissionSessionFilters.vue';
import AdmissionSessionTable from './AdmissionSessionTable.vue';

const loading = ref(false);
const loadError = ref('');
const actionError = ref('');
const actionLoadingById = reactive<
  Record<string, AdmissionSessionAction | undefined>
>({});
const items = ref<Awaited<ReturnType<typeof listAdmissionSessions>>['items']>(
  [],
);
const total = ref(0);
let fetchRequestSeq = 0;
const accessStore = useAccessStore();
const canManage = computed(() =>
  accessStore.accessCodes.includes('admission:session:manage'),
);

const query = reactive({
  page: 1,
  pageSize: 20,
  platform: 'qq',
  botSelfID: '',
  guildID: '',
  qqID: '',
  status: '' as StatusFilter,
});

async function fetchData() {
  const requestSeq = ++fetchRequestSeq;
  loading.value = true;
  loadError.value = '';
  try {
    const params: ListAdmissionSessionsParams = {
      page: query.page,
      pageSize: query.pageSize,
    };
    if (query.status) params.status = query.status;
    if (query.platform.trim()) params.platform = query.platform.trim();
    if (query.botSelfID.trim()) params.botSelfID = query.botSelfID.trim();
    if (query.guildID.trim()) params.guildID = query.guildID.trim();
    if (query.qqID.trim()) params.qqID = query.qqID.trim();

    const data = await listAdmissionSessions(params);
    if (requestSeq !== fetchRequestSeq) return;
    items.value = data.items;
    total.value = data.total;
  } catch (error) {
    if (requestSeq !== fetchRequestSeq) return;
    loadError.value = adminErrorMessage(error);
  } finally {
    if (requestSeq === fetchRequestSeq) {
      loading.value = false;
    }
  }
}

function runSearch() {
  query.page = 1;
  void fetchData();
}

function onPageSizeChange() {
  query.page = 1;
  void fetchData();
}

function resetQuery() {
  query.page = 1;
  query.platform = 'qq';
  query.botSelfID = '';
  query.guildID = '';
  query.qqID = '';
  query.status = '';
  void fetchData();
}

async function copyAuthURL(url: string) {
  await copyToClipboard(url, '认证链接已复制');
}

async function copyReissueCommand(command: string) {
  await copyToClipboard(command, '重生命令已复制');
}

async function copyToClipboard(text: string, successMessage: string) {
  try {
    if (!navigator.clipboard?.writeText) {
      throw new Error('clipboard unavailable');
    }
    await navigator.clipboard.writeText(text);
    ElMessage.success(successMessage);
    return true;
  } catch {
    if (copyTextWithFallback(text)) {
      ElMessage.success(successMessage);
      return true;
    }
    ElMessage.error('复制失败，请手动复制');
    return false;
  }
}

function copyTextWithFallback(text: string): boolean {
  const textarea = document.createElement('textarea');
  try {
    textarea.value = text;
    document.body.append(textarea);
    textarea.focus();
    textarea.select();
    return document.execCommand('copy');
  } catch {
    return false;
  } finally {
    textarea.remove();
  }
}

async function requestResend(id: string) {
  await runSessionAction(id, 'resend', async () => {
    await resendAdminAdmissionSession(id);
    ElMessage.success('已加入机器人重发队列');
  });
}

async function requestRegenerate(id: string) {
  await runSessionAction(id, 'regenerate', async () => {
    await regenerateAdminAdmissionSession(id);
    ElMessage.success('已重新生成并加入机器人提醒队列');
  });
}

async function requestCancel(id: string) {
  await runSessionAction(id, 'cancel', async () => {
    await cancelAdminAdmissionSession(id);
    ElMessage.success('认证会话已取消');
  });
}

async function runSessionAction(
  id: string,
  action: AdmissionSessionAction,
  request: () => Promise<void>,
) {
  if (actionLoadingById[id]) {
    return;
  }

  actionLoadingById[id] = action;
  actionError.value = '';
  try {
    await request();
    await fetchData();
  } catch (error) {
    handleActionError(error);
  } finally {
    delete actionLoadingById[id];
  }
}

function handleActionError(error: unknown) {
  actionError.value = adminErrorMessage(error);
  ElMessage.error(actionError.value);
}

function adminErrorMessage(error: unknown): string {
  return error instanceof Error && error.message
    ? error.message
    : $t('admin.result.requestFailed');
}

onMounted(fetchData);
</script>

<template>
  <AdminContentLayout title="入群认证会话" :total="total">
    <template #toolbar>
      <AdmissionSessionFilters
        v-model:qq-i-d="query.qqID"
        v-model:guild-i-d="query.guildID"
        v-model:bot-self-i-d="query.botSelfID"
        v-model:platform="query.platform"
        v-model:status="query.status"
        @search="runSearch"
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
      @close="actionError = ''"
    />

    <AdmissionSessionTable
      v-model:page="query.page"
      v-model:page-size="query.pageSize"
      :can-manage="canManage"
      :action-loading-by-id="actionLoadingById"
      :loading="loading"
      :items="items"
      :total="total"
      @copy-auth-u-r-l="copyAuthURL"
      @copy-reissue-command="copyReissueCommand"
      @request-resend="requestResend"
      @request-regenerate="requestRegenerate"
      @request-cancel="requestCancel"
      @page-change="fetchData"
      @page-size-change="onPageSizeChange"
    />
  </AdminContentLayout>
</template>
