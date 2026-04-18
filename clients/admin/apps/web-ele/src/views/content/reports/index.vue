<script setup lang="ts">
import type { Report } from '#/api/admin';

import { onMounted, reactive, ref } from 'vue';

import {
  ElButton,
  ElOption,
  ElPagination,
  ElPopconfirm,
  ElSelect,
  ElTable,
  ElTableColumn,
  ElTag,
} from 'element-plus';

import { getReportList, processReport } from '#/api/admin';
import { $t } from '#/locales';

const loading = ref(false);
const actionLoading = ref(false);
const reports = ref<Report[]>([]);
const total = ref(0);
const query = reactive({
  page: 1,
  pageSize: 20,
  status: 'all' as 'all' | 'pending' | 'rejected' | 'resolved',
});

async function fetchData() {
  loading.value = true;
  try {
    const data = await getReportList(query);
    reports.value = data.items;
    total.value = data.total;
  } finally {
    loading.value = false;
  }
}

async function handleAction(
  reportId: string,
  action: 'delete' | 'hide' | 'reject',
) {
  if (actionLoading.value) {
    return;
  }

  actionLoading.value = true;
  try {
    await processReport(reportId, { action });
    await fetchData();
  } catch (_error) { void _error;
    // unwrapData already displays a toast for failed mutations.
  } finally {
    actionLoading.value = false;
  }
}

type TagType = 'danger' | 'info' | 'success' | 'warning';

const statusTag = (status: string): TagType => {
  const map: Record<string, TagType> = {
    pending: 'warning',
    resolved: 'success',
    rejected: 'info',
  };
  return map[status] || 'info';
};

const statusLabel = (status: string) => {
  return $t(`admin.content.reports.status.${status}`);
};

const reasonLabel = (reason: Report['reason']) => {
  return $t(`admin.content.reports.reason.${reason}`);
};

onMounted(fetchData);
</script>

<template>
  <div class="p-4">
    <div class="mb-4 flex items-center gap-3">
      <ElSelect
        v-model="query.status"
        :placeholder="$t('admin.common.status')"
        style="width: 140px"
        @change="fetchData"
      >
        <ElOption :label="$t('admin.common.all')" value="all" />
        <ElOption
          :label="$t('admin.content.reports.status.pending')"
          value="pending"
        />
        <ElOption
          :label="$t('admin.content.reports.status.resolved')"
          value="resolved"
        />
        <ElOption
          :label="$t('admin.content.reports.status.rejected')"
          value="rejected"
        />
      </ElSelect>
      <ElButton type="primary" @click="fetchData">
        {{ $t('admin.common.query') }}
      </ElButton>
    </div>

    <ElTable v-loading="loading" :data="reports" stripe>
      <ElTableColumn :label="$t('admin.common.id')" prop="id" width="70" />
      <ElTableColumn
        :label="$t('admin.content.reports.reviewId')"
        prop="reviewID"
        min-width="140"
      />
      <ElTableColumn
        :label="$t('admin.content.reports.reviewTitle')"
        min-width="160"
      >
        <template #default="{ row }">
          {{ row.review?.title || $t('admin.content.reports.missingReview') }}
        </template>
      </ElTableColumn>
      <ElTableColumn
        :label="$t('admin.content.reports.reasonLabel')"
        min-width="140"
      >
        <template #default="{ row }">
          {{ reasonLabel(row.reason) }}
        </template>
      </ElTableColumn>
      <ElTableColumn
        :label="$t('admin.common.description')"
        min-width="200"
        show-overflow-tooltip
      >
        <template #default="{ row }">
          {{
            row.description ||
            row.resolutionNote ||
            $t('admin.common.unavailable')
          }}
        </template>
      </ElTableColumn>
      <ElTableColumn :label="$t('admin.common.status')" width="90">
        <template #default="{ row }">
          <ElTag :type="statusTag(row.status)" size="small">
            {{ statusLabel(row.status) }}
          </ElTag>
        </template>
      </ElTableColumn>
      <ElTableColumn
        :label="$t('admin.common.time')"
        prop="createdAt"
        width="170"
      />
      <ElTableColumn
        fixed="right"
        :label="$t('admin.common.actions')"
        width="220"
      >
        <template #default="{ row }">
          <template v-if="row.status === 'pending'">
            <ElPopconfirm
              :title="$t('admin.content.reports.confirmReject')"
              @confirm="handleAction(row.id, 'reject')"
            >
              <template #reference>
                <ElButton
                  link
                  size="small"
                  type="info"
                  :disabled="actionLoading"
                >
                  {{ $t('admin.content.reports.reject') }}
                </ElButton>
              </template>
            </ElPopconfirm>
            <ElPopconfirm
              :title="$t('admin.content.reports.confirmHideReview')"
              @confirm="handleAction(row.id, 'hide')"
            >
              <template #reference>
                <ElButton
                  link
                  size="small"
                  type="warning"
                  :disabled="actionLoading"
                >
                  {{ $t('admin.content.reports.hideReview') }}
                </ElButton>
              </template>
            </ElPopconfirm>
            <ElPopconfirm
              :title="$t('admin.content.reports.confirmDeleteReview')"
              @confirm="handleAction(row.id, 'delete')"
            >
              <template #reference>
                <ElButton
                  link
                  size="small"
                  type="danger"
                  :disabled="actionLoading"
                >
                  {{ $t('admin.content.reports.deleteReview') }}
                </ElButton>
              </template>
            </ElPopconfirm>
          </template>
        </template>
      </ElTableColumn>
    </ElTable>

    <div class="mt-4 flex justify-end">
      <ElPagination
        v-model:current-page="query.page"
        v-model:page-size="query.pageSize"
        :total="total"
        layout="total, prev, pager, next"
        @current-change="fetchData"
      />
    </div>
  </div>
</template>
