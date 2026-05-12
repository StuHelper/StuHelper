<script setup lang="ts">
import type { MemberBlacklistEntry } from '#/api/admin';

import { ElButton, ElPagination, ElTag } from 'element-plus';

import PersistentAdminTable from '../../shared/admin-table/PersistentAdminTable.vue';
import PersistentAdminTableColumn from '../../shared/admin-table/PersistentAdminTableColumn.vue';

import {
  createdByLabel,
  createdFromLabel,
  entryStatus,
  formatDateTime,
  scopeLabel,
  sourceLabel,
  statusLabel,
  statusType,
} from './options';

defineProps<{
  canManage: boolean;
  items: MemberBlacklistEntry[];
  loading: boolean;
  total: number;
}>();

const emit = defineEmits<{
  (e: 'release', entry: MemberBlacklistEntry): void;
  (e: 'pageChange'): void;
  (e: 'pageSizeChange'): void;
}>();
const page = defineModel<number>('page', { required: true });
const pageSize = defineModel<number>('pageSize', { required: true });
</script>

<template>
  <div>
    <PersistentAdminTable
      table-key="users.memberBlacklist"
      :loading="loading"
      :data="items"
      row-key="id"
      stripe
    >
      <PersistentAdminTableColumn
        column-key="status"
        label="状态"
        :default-width="96"
      >
        <template #default="{ row }">
          <ElTag :type="statusType(entryStatus(row))" data-field="status">
            {{ statusLabel(entryStatus(row)) }}
          </ElTag>
        </template>
      </PersistentAdminTableColumn>
      <PersistentAdminTableColumn
        column-key="subject"
        label="主体"
        :default-min-width="160"
      >
        <template #default="{ row }">
          <div class="font-mono">{{ row.subjectID }}</div>
          <div class="text-xs text-slate-500">{{ row.subjectType }}</div>
        </template>
      </PersistentAdminTableColumn>
      <PersistentAdminTableColumn
        column-key="scope"
        label="范围"
        :default-width="160"
      >
        <template #default="{ row }">{{ scopeLabel(row) }}</template>
      </PersistentAdminTableColumn>
      <PersistentAdminTableColumn
        column-key="source"
        label="来源"
        :default-width="120"
      >
        <template #default="{ row }">{{ sourceLabel(row) }}</template>
      </PersistentAdminTableColumn>
      <PersistentAdminTableColumn
        column-key="reason"
        label="原因"
        :default-min-width="200"
      >
        <template #default="{ row }">
          <div>{{ row.reasonText || '—' }}</div>
          <div class="text-xs text-slate-500">{{ row.reasonCode }}</div>
        </template>
      </PersistentAdminTableColumn>
      <PersistentAdminTableColumn
        column-key="createdFrom"
        label="创建入口"
        :default-width="140"
      >
        <template #default="{ row }">{{ createdFromLabel(row) }}</template>
      </PersistentAdminTableColumn>
      <PersistentAdminTableColumn
        column-key="createdBy"
        label="创建人"
        :default-width="200"
      >
        <template #default="{ row }">
          <span class="font-mono text-xs">{{ createdByLabel(row) }}</span>
        </template>
      </PersistentAdminTableColumn>
      <PersistentAdminTableColumn
        column-key="createdAt"
        label="创建时间"
        :default-width="180"
      >
        <template #default="{ row }">
          {{ formatDateTime(row.createdAt) }}
        </template>
      </PersistentAdminTableColumn>
      <PersistentAdminTableColumn
        column-key="expiresAt"
        label="过期时间"
        :default-width="180"
      >
        <template #default="{ row }">
          {{ row.expiresAt ? formatDateTime(row.expiresAt) : '永久' }}
        </template>
      </PersistentAdminTableColumn>
      <PersistentAdminTableColumn
        column-key="releasedAt"
        label="解除时间"
        :default-width="180"
      >
        <template #default="{ row }">
          {{ formatDateTime(row.releasedAt) }}
        </template>
      </PersistentAdminTableColumn>
      <PersistentAdminTableColumn
        column-key="actions"
        fixed="right"
        label="操作"
        :default-width="120"
      >
        <template #default="{ row }">
          <ElButton
            v-if="canManage && entryStatus(row) === 'active'"
            data-action="release"
            plain
            size="small"
            type="warning"
            @click="emit('release', row)"
          >
            解除
          </ElButton>
          <span v-else class="text-slate-400">—</span>
        </template>
      </PersistentAdminTableColumn>
    </PersistentAdminTable>

    <div class="admin-embedded-pagination">
      <ElPagination
        v-model:current-page="page"
        v-model:page-size="pageSize"
        :total="total"
        layout="total, prev, pager, next, sizes"
        @current-change="emit('pageChange')"
        @size-change="emit('pageSizeChange')"
      />
    </div>
  </div>
</template>
