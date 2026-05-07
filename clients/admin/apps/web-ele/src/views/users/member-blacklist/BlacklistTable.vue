<script setup lang="ts">
import type { MemberBlacklistEntry } from '#/api/admin';

import {
  ElButton,
  ElPagination,
  ElTable,
  ElTableColumn,
  ElTag,
} from 'element-plus';

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
  loading: boolean;
  items: MemberBlacklistEntry[];
  total: number;
  canManage: boolean;
}>();

const page = defineModel<number>('page', { required: true });
const pageSize = defineModel<number>('pageSize', { required: true });

const emit = defineEmits<{
  (e: 'release', entry: MemberBlacklistEntry): void;
  (e: 'pageChange'): void;
  (e: 'pageSizeChange'): void;
}>();
</script>

<template>
  <div>
    <ElTable v-loading="loading" :data="items" stripe>
      <ElTableColumn label="状态" width="96">
        <template #default="{ row }">
          <ElTag :type="statusType(entryStatus(row))" data-field="status">
            {{ statusLabel(entryStatus(row)) }}
          </ElTag>
        </template>
      </ElTableColumn>
      <ElTableColumn label="主体" min-width="160">
        <template #default="{ row }">
          <div class="font-mono">{{ row.subjectID }}</div>
          <div class="text-xs text-slate-500">{{ row.subjectType }}</div>
        </template>
      </ElTableColumn>
      <ElTableColumn label="范围" width="160">
        <template #default="{ row }">{{ scopeLabel(row) }}</template>
      </ElTableColumn>
      <ElTableColumn label="来源" width="120">
        <template #default="{ row }">{{ sourceLabel(row) }}</template>
      </ElTableColumn>
      <ElTableColumn label="原因" min-width="200">
        <template #default="{ row }">
          <div>{{ row.reasonText || '—' }}</div>
          <div class="text-xs text-slate-500">{{ row.reasonCode }}</div>
        </template>
      </ElTableColumn>
      <ElTableColumn label="创建入口" width="140">
        <template #default="{ row }">{{ createdFromLabel(row) }}</template>
      </ElTableColumn>
      <ElTableColumn label="创建人" width="200">
        <template #default="{ row }">
          <span class="font-mono text-xs">{{ createdByLabel(row) }}</span>
        </template>
      </ElTableColumn>
      <ElTableColumn label="创建时间" width="180">
        <template #default="{ row }">{{ formatDateTime(row.createdAt) }}</template>
      </ElTableColumn>
      <ElTableColumn label="过期时间" width="180">
        <template #default="{ row }">
          {{ row.expiresAt ? formatDateTime(row.expiresAt) : '永久' }}
        </template>
      </ElTableColumn>
      <ElTableColumn label="解除时间" width="180">
        <template #default="{ row }">{{ formatDateTime(row.releasedAt) }}</template>
      </ElTableColumn>
      <ElTableColumn fixed="right" label="操作" width="120">
        <template #default="{ row }">
          <ElButton
            v-if="canManage && entryStatus(row) === 'active'"
            data-action="release"
            link
            type="warning"
            @click="emit('release', row)"
          >
            解除
          </ElButton>
          <span v-else class="text-slate-400">—</span>
        </template>
      </ElTableColumn>
    </ElTable>

    <div class="mt-4 flex justify-end">
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
