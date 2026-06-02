<script setup lang="ts">
import type { ListAdmissionSessionsParams } from '#/api/admin';

import { onMounted, reactive, ref } from 'vue';

import { ElMessage } from 'element-plus';

import { listAdmissionSessions } from '#/api/admin';

import AdminContentLayout from '../../shared/AdminContentLayout.vue';
import AdmissionSessionFilters from './AdmissionSessionFilters.vue';
import AdmissionSessionTable from './AdmissionSessionTable.vue';
import type { StatusFilter } from './options';

const loading = ref(false);
const items = ref<Awaited<ReturnType<typeof listAdmissionSessions>>['items']>(
  [],
);
const total = ref(0);

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
  loading.value = true;
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
    items.value = data.items;
    total.value = data.total;
  } finally {
    loading.value = false;
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
  await navigator.clipboard.writeText(url);
  ElMessage.success('认证链接已复制');
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

    <AdmissionSessionTable
      v-model:page="query.page"
      v-model:page-size="query.pageSize"
      :loading="loading"
      :items="items"
      :total="total"
      @copy-auth-u-r-l="copyAuthURL"
      @page-change="fetchData"
      @page-size-change="onPageSizeChange"
    />
  </AdminContentLayout>
</template>
