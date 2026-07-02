<script setup lang="ts">
import type { ScopeType, SourceFilter, StatusFilter } from './options';

import type {
  ListMemberBlacklistParams,
  MemberBlacklistCreateRequest,
  MemberBlacklistEntry,
  MemberBlacklistReleaseRequest,
} from '#/api/admin';

import { computed, ref, useTemplateRef } from 'vue';

import { useAccessStore } from '@vben/stores';

import { ElAlert, ElButton } from 'element-plus';

import {
  createMemberBlacklist,
  listMemberBlacklist,
  releaseMemberBlacklist,
} from '#/api/admin';
import { useAdminAction } from '#/composables/use-admin-action';
import { useAdminList } from '#/composables/use-admin-list';
import { $t } from '#/locales';

import AdminContentLayout from '../../shared/AdminContentLayout.vue';
import { ADMIN_DEFAULT_PAGE_SIZE } from '../../shared/pagination';
import BlacklistFilters from './BlacklistFilters.vue';
import BlacklistTable from './BlacklistTable.vue';
import CreateBlacklistDialog from './CreateBlacklistDialog.vue';
import ReleaseBlacklistDialog from './ReleaseBlacklistDialog.vue';

const accessStore = useAccessStore();
const canManage = computed(() =>
  accessStore.accessCodes.includes('member_blacklist:manage'),
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
  MemberBlacklistEntry,
  {
    guildID: string;
    page: number;
    pageSize: number;
    platform: string;
    scopeType: '' | ScopeType;
    source: SourceFilter;
    status: StatusFilter;
    subjectID: string;
  }
>({
  fetcher: (listQuery) => {
    const params: ListMemberBlacklistParams = {
      page: listQuery.page,
      pageSize: listQuery.pageSize,
      status: listQuery.status,
    };
    if (listQuery.platform.trim()) {
      params.platform = listQuery.platform.trim();
    }
    if (listQuery.scopeType) params.scopeType = listQuery.scopeType;
    if (listQuery.source) params.source = listQuery.source;
    if (listQuery.guildID.trim()) params.guildID = listQuery.guildID.trim();
    if (listQuery.subjectID.trim()) {
      params.subjectID = listQuery.subjectID.trim();
    }
    return listMemberBlacklist(params);
  },
  initialQuery: {
    guildID: '',
    page: 1,
    pageSize: ADMIN_DEFAULT_PAGE_SIZE,
    platform: '',
    scopeType: '',
    source: '',
    status: 'active',
    subjectID: '',
  },
});

const { actionError, clearActionError, isActionPending, runAction } =
  useAdminAction();

const createDialogVisible = ref(false);
const createDialog = useTemplateRef<{ reset: () => void }>('createDialog');

const releaseDialogVisible = ref(false);
const releaseTarget = ref<MemberBlacklistEntry | null>(null);
const releaseSubmitting = computed(() =>
  releaseTarget.value ? isActionPending(releaseTarget.value.id) : false,
);

function resetQuery() {
  query.platform = '';
  query.scopeType = '';
  query.source = '';
  query.status = 'active';
  query.guildID = '';
  query.subjectID = '';
  resetPageAndFetch();
}

function openCreateDialog() {
  if (!canManage.value) {
    return;
  }
  createDialog.value?.reset();
  createDialogVisible.value = true;
}

async function submitCreate(payload: MemberBlacklistCreateRequest) {
  if (!canManage.value) {
    return;
  }

  const succeeded = await runAction(() => createMemberBlacklist(payload), {
    successMessage: $t('admin.users.memberBlacklist.createSuccess', {
      subject: payload.subjectID,
    }),
  });
  if (succeeded) {
    createDialogVisible.value = false;
    resetPageAndFetch();
  }
}

function openReleaseDialog(entry: MemberBlacklistEntry) {
  if (!canManage.value || isActionPending(entry.id)) {
    return;
  }

  releaseTarget.value = entry;
  releaseDialogVisible.value = true;
}

async function submitRelease(payload: {
  id: string;
  request: MemberBlacklistReleaseRequest;
}) {
  const target = releaseTarget.value;
  if (!canManage.value || !target || target.id !== payload.id) {
    return;
  }

  const succeeded = await runAction(
    () => releaseMemberBlacklist(payload.id, payload.request),
    {
      id: payload.id,
      successMessage: $t('admin.users.memberBlacklist.releaseSuccess', {
        subject: target.subjectID,
      }),
    },
  );
  if (succeeded) {
    releaseDialogVisible.value = false;
    releaseTarget.value = null;
    await fetchData();
  }
}
</script>

<template>
  <AdminContentLayout
    :title="$t('admin.routes.userSystem.memberBlacklist')"
    :total="total"
  >
    <template #toolbar>
      <BlacklistFilters
        v-model:platform="query.platform"
        v-model:scope-type="query.scopeType"
        v-model:source="query.source"
        v-model:status="query.status"
        v-model:guild-i-d="query.guildID"
        v-model:subject-i-d="query.subjectID"
        :can-manage="canManage"
        @search="resetPageAndFetch"
        @reset="resetQuery"
        @open-create="openCreateDialog"
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

    <BlacklistTable
      v-model:page="query.page"
      v-model:page-size="query.pageSize"
      :loading="loading"
      :items="items"
      :total="total"
      :can-manage="canManage"
      @release="openReleaseDialog"
      @page-change="fetchData"
      @page-size-change="resetPageAndFetch"
    />

    <CreateBlacklistDialog
      ref="createDialog"
      v-model:visible="createDialogVisible"
      :submitting="isActionPending()"
      @submit="submitCreate"
    />

    <ReleaseBlacklistDialog
      v-model:visible="releaseDialogVisible"
      :target="releaseTarget"
      :submitting="releaseSubmitting"
      @submit="submitRelease"
    />
  </AdminContentLayout>
</template>
