<script setup lang="ts">
import type { ScopeType, SourceFilter, StatusFilter } from './options';

import type {
  ListMemberBlacklistParams,
  MemberBlacklistCreateRequest,
  MemberBlacklistEntry,
  MemberBlacklistReleaseRequest,
} from '#/api/admin';

import { computed, onMounted, reactive, ref, useTemplateRef } from 'vue';

import { useAccessStore } from '@vben/stores';

import { ElMessage } from 'element-plus';

import {
  createMemberBlacklist,
  listMemberBlacklist,
  releaseMemberBlacklist,
} from '#/api/admin';

import AdminContentLayout from '../../shared/AdminContentLayout.vue';
import BlacklistFilters from './BlacklistFilters.vue';
import BlacklistTable from './BlacklistTable.vue';
import CreateBlacklistDialog from './CreateBlacklistDialog.vue';
import ReleaseBlacklistDialog from './ReleaseBlacklistDialog.vue';

const loading = ref(false);
const items = ref<MemberBlacklistEntry[]>([]);
const total = ref(0);

const accessStore = useAccessStore();
const canManage = computed(() =>
  accessStore.accessCodes.includes('member_blacklist:manage'),
);

const query = reactive({
  page: 1,
  pageSize: 20,
  platform: '',
  scopeType: '' as '' | ScopeType,
  source: '' as SourceFilter,
  status: 'active' as StatusFilter,
  guildID: '',
  subjectID: '',
});

const createDialogVisible = ref(false);
const creating = ref(false);
const createDialog = useTemplateRef<{ reset: () => void }>('createDialog');

const releaseDialogVisible = ref(false);
const releasing = ref(false);
const releaseTarget = ref<MemberBlacklistEntry | null>(null);

async function fetchData() {
  loading.value = true;
  try {
    const params: ListMemberBlacklistParams = {
      page: query.page,
      pageSize: query.pageSize,
      status: query.status,
    };
    if (query.platform.trim()) params.platform = query.platform.trim();
    if (query.scopeType) params.scopeType = query.scopeType;
    if (query.source) params.source = query.source;
    if (query.guildID.trim()) params.guildID = query.guildID.trim();
    if (query.subjectID.trim()) params.subjectID = query.subjectID.trim();

    const data = await listMemberBlacklist(params);
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
  query.platform = '';
  query.scopeType = '';
  query.source = '';
  query.status = 'active';
  query.guildID = '';
  query.subjectID = '';
  query.page = 1;
  void fetchData();
}

function openCreateDialog() {
  createDialog.value?.reset();
  createDialogVisible.value = true;
}

async function submitCreate(payload: MemberBlacklistCreateRequest) {
  creating.value = true;
  try {
    await createMemberBlacklist(payload);
    ElMessage.success(`已将 ${payload.subjectID} 加入黑名单`);
    createDialogVisible.value = false;
    query.page = 1;
    await fetchData();
  } finally {
    creating.value = false;
  }
}

function openReleaseDialog(entry: MemberBlacklistEntry) {
  releaseTarget.value = entry;
  releaseDialogVisible.value = true;
}

async function submitRelease(payload: {
  id: string;
  request: MemberBlacklistReleaseRequest;
}) {
  releasing.value = true;
  try {
    await releaseMemberBlacklist(payload.id, payload.request);
    ElMessage.success(`已解除黑名单：${releaseTarget.value?.subjectID ?? ''}`);
    releaseDialogVisible.value = false;
    releaseTarget.value = null;
    await fetchData();
  } finally {
    releasing.value = false;
  }
}

onMounted(fetchData);
</script>

<template>
  <AdminContentLayout title="成员黑名单" :total="total">
    <template #toolbar>
      <BlacklistFilters
        v-model:platform="query.platform"
        v-model:scope-type="query.scopeType"
        v-model:source="query.source"
        v-model:status="query.status"
        v-model:guild-i-d="query.guildID"
        v-model:subject-i-d="query.subjectID"
        :can-manage="canManage"
        @search="runSearch"
        @reset="resetQuery"
        @open-create="openCreateDialog"
      />
    </template>

    <BlacklistTable
      v-model:page="query.page"
      v-model:page-size="query.pageSize"
      :loading="loading"
      :items="items"
      :total="total"
      :can-manage="canManage"
      @release="openReleaseDialog"
      @page-change="fetchData"
      @page-size-change="onPageSizeChange"
    />

    <CreateBlacklistDialog
      ref="createDialog"
      v-model:visible="createDialogVisible"
      :submitting="creating"
      @submit="submitCreate"
    />

    <ReleaseBlacklistDialog
      v-model:visible="releaseDialogVisible"
      :target="releaseTarget"
      :submitting="releasing"
      @submit="submitRelease"
    />
  </AdminContentLayout>
</template>
