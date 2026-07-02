<script setup lang="ts">
import type { OperationLog } from '#/api/admin';

import { ElAlert, ElButton, ElPagination, ElTag } from 'element-plus';

import { getOperationLogs } from '#/api/admin';
import { useAdminList } from '#/composables/use-admin-list';
import { $t } from '#/locales';

import PersistentAdminTable from '../../shared/admin-table/PersistentAdminTable.vue';
import PersistentAdminTableColumn from '../../shared/admin-table/PersistentAdminTableColumn.vue';
import AdminContentLayout from '../../shared/AdminContentLayout.vue';
import { compactID, formatAdminDateTime } from '../../shared/display';
import {
  ADMIN_DEFAULT_PAGE_SIZE,
  ADMIN_PAGE_SIZES,
  ADMIN_PAGINATION_LAYOUT,
} from '../../shared/pagination';

const USER_AGENT_PREVIEW_LENGTH = 64;

const {
  fetchData,
  items: logs,
  loadError,
  loading,
  query,
  resetPageAndFetch,
  total,
} = useAdminList<OperationLog, { page: number; pageSize: number }>({
  fetcher: (listQuery) => getOperationLogs(listQuery),
  initialQuery: {
    page: 1,
    pageSize: ADMIN_DEFAULT_PAGE_SIZE,
  },
});

function formatJSON(value?: Record<string, unknown>) {
  if (!value || Object.keys(value).length === 0) return '—';
  return JSON.stringify(value);
}

function compactText(value?: null | string) {
  if (!value) return '—';
  if (value.length <= USER_AGENT_PREVIEW_LENGTH) return value;
  return `${value.slice(0, USER_AGENT_PREVIEW_LENGTH)}...`;
}
</script>

<template>
  <AdminContentLayout :title="$t('admin.routes.content.logs')" :total="total">
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

    <PersistentAdminTable
      table-key="content.operationLogs"
      :loading="loading"
      :data="logs"
      row-key="id"
    >
      <PersistentAdminTableColumn
        column-key="createdAt"
        label="时间"
        :default-min-width="180"
      >
        <template #default="{ row }">
          {{ formatAdminDateTime(row.createdAt) }}
        </template>
      </PersistentAdminTableColumn>
      <PersistentAdminTableColumn
        column-key="admin"
        label="管理员"
        :default-min-width="160"
      >
        <template #default="{ row }">
          <div>{{ row.adminUsername }}</div>
          <div class="admin-id-token" :title="row.adminUserID">
            {{ compactID(row.adminUserID) }}
          </div>
        </template>
      </PersistentAdminTableColumn>
      <PersistentAdminTableColumn
        column-key="action"
        label="动作"
        :default-min-width="160"
      >
        <template #default="{ row }">
          <ElTag type="info">{{ row.action }}</ElTag>
        </template>
      </PersistentAdminTableColumn>
      <PersistentAdminTableColumn
        column-key="resource"
        label="资源"
        :default-min-width="180"
      >
        <template #default="{ row }">
          <div>{{ row.resourceType }}</div>
          <div class="admin-id-token" :title="row.resourceID">
            {{ compactID(row.resourceID) }}
          </div>
        </template>
      </PersistentAdminTableColumn>
      <PersistentAdminTableColumn
        column-key="oldValue"
        label="变更前"
        :default-min-width="220"
        show-overflow-tooltip
      >
        <template #default="{ row }">
          {{ formatJSON(row.oldValue) }}
        </template>
      </PersistentAdminTableColumn>
      <PersistentAdminTableColumn
        column-key="newValue"
        label="变更后"
        :default-min-width="220"
        show-overflow-tooltip
      >
        <template #default="{ row }">
          {{ formatJSON(row.newValue) }}
        </template>
      </PersistentAdminTableColumn>
      <PersistentAdminTableColumn
        column-key="request"
        label="请求"
        :default-min-width="220"
        show-overflow-tooltip
      >
        <template #default="{ row }">
          <div>{{ row.ipAddress || '—' }}</div>
          <div class="admin-id-token" :title="row.userAgent || ''">
            {{ compactText(row.userAgent) }}
          </div>
        </template>
      </PersistentAdminTableColumn>
    </PersistentAdminTable>

    <template #pagination>
      <ElPagination
        v-model:current-page="query.page"
        v-model:page-size="query.pageSize"
        background
        :layout="ADMIN_PAGINATION_LAYOUT"
        :page-sizes="ADMIN_PAGE_SIZES"
        :total="total"
        @current-change="fetchData"
        @size-change="resetPageAndFetch"
      />
    </template>
  </AdminContentLayout>
</template>
