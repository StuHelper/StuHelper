<script setup lang="ts">
import type { OperationLog } from '#/api/admin';

import { onMounted, reactive, ref } from 'vue';

import {
  ElPagination,
  ElTable,
  ElTableColumn,
  ElTag,
} from 'element-plus';

import { getOperationLogs } from '#/api/admin';

const loading = ref(false);
const logs = ref<OperationLog[]>([]);
const total = ref(0);
const query = reactive({
  page: 1,
  pageSize: 20,
});

async function fetchData() {
  loading.value = true;
  try {
    const data = await getOperationLogs(query);
    logs.value = data.items;
    total.value = data.total;
  } finally {
    loading.value = false;
  }
}

function formatTime(value: string) {
  return new Date(value).toLocaleString('zh-CN', { hour12: false });
}

function formatJSON(value?: Record<string, unknown>) {
  if (!value || Object.keys(value).length === 0) return '—';
  return JSON.stringify(value);
}

function refreshPage(page: number) {
  query.page = page;
  void fetchData();
}

onMounted(fetchData);
</script>

<template>
  <div v-loading="loading" class="p-4">
    <ElTable :data="logs" border>
      <ElTableColumn label="时间" min-width="180">
        <template #default="{ row }">
          {{ formatTime(row.createdAt) }}
        </template>
      </ElTableColumn>
      <ElTableColumn label="管理员" min-width="160">
        <template #default="{ row }">
          <div>{{ row.adminUsername }}</div>
          <div class="text-xs text-gray-500">{{ row.adminUserID }}</div>
        </template>
      </ElTableColumn>
      <ElTableColumn label="动作" min-width="160">
        <template #default="{ row }">
          <ElTag type="info">{{ row.action }}</ElTag>
        </template>
      </ElTableColumn>
      <ElTableColumn label="资源" min-width="180">
        <template #default="{ row }">
          <div>{{ row.resourceType }}</div>
          <div class="text-xs text-gray-500">{{ row.resourceID }}</div>
        </template>
      </ElTableColumn>
      <ElTableColumn label="变更前" min-width="220" show-overflow-tooltip>
        <template #default="{ row }">
          {{ formatJSON(row.oldValue) }}
        </template>
      </ElTableColumn>
      <ElTableColumn label="变更后" min-width="220" show-overflow-tooltip>
        <template #default="{ row }">
          {{ formatJSON(row.newValue) }}
        </template>
      </ElTableColumn>
      <ElTableColumn label="请求" min-width="220" show-overflow-tooltip>
        <template #default="{ row }">
          <div>{{ row.ipAddress || '—' }}</div>
          <div class="text-xs text-gray-500">{{ row.userAgent || '—' }}</div>
        </template>
      </ElTableColumn>
    </ElTable>

    <div class="mt-4 flex justify-end">
      <ElPagination
        v-model:current-page="query.page"
        v-model:page-size="query.pageSize"
        background
        layout="prev, pager, next, sizes, total"
        :page-sizes="[10, 20, 50, 100]"
        :total="total"
        @current-change="refreshPage"
        @size-change="refreshPage(1)"
      />
    </div>
  </div>
</template>
