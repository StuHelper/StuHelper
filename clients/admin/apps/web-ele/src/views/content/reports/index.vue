<script setup lang="ts">
import type { Report } from '#/api/admin';

import {
  ElAlert,
  ElButton,
  ElOption,
  ElPagination,
  ElPopconfirm,
  ElSelect,
  ElTag,
} from 'element-plus';

import { getReportList, processReport } from '#/api/admin';
import { useAdminAction } from '#/composables/use-admin-action';
import { useAdminList } from '#/composables/use-admin-list';
import { $t } from '#/locales';

import PersistentAdminTable from '../../shared/admin-table/PersistentAdminTable.vue';
import PersistentAdminTableColumn from '../../shared/admin-table/PersistentAdminTableColumn.vue';
import AdminContentLayout from '../../shared/AdminContentLayout.vue';
import {
  compactID,
  formatAdminDateTime,
  formatNullableText,
} from '../../shared/display';
import {
  ADMIN_DEFAULT_PAGE_SIZE,
  ADMIN_PAGE_SIZES,
  ADMIN_PAGINATION_LAYOUT,
} from '../../shared/pagination';

type ReportStatusFilter = 'all' | 'pending' | 'rejected' | 'resolved';

const {
  fetchData,
  items: reports,
  loadError,
  loading,
  query,
  resetPageAndFetch,
  total,
} = useAdminList<
  Report,
  { page: number; pageSize: number; status: ReportStatusFilter }
>({
  fetcher: (listQuery) => getReportList(listQuery),
  initialQuery: {
    page: 1,
    pageSize: ADMIN_DEFAULT_PAGE_SIZE,
    status: 'all',
  },
});

const { actionError, clearActionError, isActionPending, runAction } =
  useAdminAction();

async function handleAction(
  reportId: string,
  action: 'delete' | 'hide' | 'reject',
) {
  const succeeded = await runAction(() => processReport(reportId, { action }), {
    id: reportId,
  });
  if (succeeded) {
    await fetchData();
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
  if (typeof reason !== 'string' || reason.trim() === '') {
    return $t('admin.content.reports.reason.unknown');
  }
  const key = `admin.content.reports.reason.${reason}`;
  const label = $t(key);
  return label === key ? reason : label;
};
</script>

<template>
  <AdminContentLayout
    :title="$t('admin.routes.content.reports')"
    :total="total"
  >
    <template #toolbar>
      <ElSelect
        v-model="query.status"
        class="admin-toolbar-control"
        :placeholder="$t('admin.common.status')"
        :teleported="false"
        @change="resetPageAndFetch"
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
      <ElButton type="primary" @click="resetPageAndFetch">
        {{ $t('admin.common.query') }}
      </ElButton>
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

    <PersistentAdminTable
      table-key="content.reports"
      :loading="loading"
      :data="reports"
      row-key="id"
      stripe
    >
      <PersistentAdminTableColumn
        column-key="id"
        :label="$t('admin.common.id')"
        :default-width="148"
      >
        <template #default="{ row }">
          <span class="admin-id-token" :title="row.id">
            {{ compactID(row.id) }}
          </span>
        </template>
      </PersistentAdminTableColumn>
      <PersistentAdminTableColumn
        column-key="reviewID"
        :label="$t('admin.content.reports.reviewId')"
        :default-width="148"
      >
        <template #default="{ row }">
          <span class="admin-sub-id" :title="row.reviewID">
            {{ compactID(row.reviewID) }}
          </span>
        </template>
      </PersistentAdminTableColumn>
      <PersistentAdminTableColumn
        column-key="reviewTitle"
        :label="$t('admin.content.reports.reviewTitle')"
        :default-min-width="180"
        show-overflow-tooltip
      >
        <template #default="{ row }">
          <div class="admin-cell-title" :title="row.review?.title">
            {{ row.review?.title || $t('admin.content.reports.missingReview') }}
          </div>
        </template>
      </PersistentAdminTableColumn>
      <PersistentAdminTableColumn
        column-key="reason"
        :label="$t('admin.content.reports.reasonLabel')"
        :default-min-width="130"
        show-overflow-tooltip
      >
        <template #default="{ row }">
          {{ reasonLabel(row.reason) }}
        </template>
      </PersistentAdminTableColumn>
      <PersistentAdminTableColumn
        column-key="description"
        :label="$t('admin.common.description')"
        :default-min-width="240"
        show-overflow-tooltip
      >
        <template #default="{ row }">
          <div
            class="admin-cell-body"
            :title="row.description || row.resolutionNote"
          >
            {{
              formatNullableText(
                row.description ||
                  row.resolutionNote ||
                  $t('admin.common.unavailable'),
              )
            }}
          </div>
        </template>
      </PersistentAdminTableColumn>
      <PersistentAdminTableColumn
        column-key="status"
        :label="$t('admin.common.status')"
        :default-width="104"
      >
        <template #default="{ row }">
          <ElTag :type="statusTag(row.status)" size="small">
            {{ statusLabel(row.status) }}
          </ElTag>
        </template>
      </PersistentAdminTableColumn>
      <PersistentAdminTableColumn
        column-key="createdAt"
        :label="$t('admin.common.time')"
        :default-width="148"
      >
        <template #default="{ row }">
          <span class="admin-cell-muted">
            {{ formatAdminDateTime(row.createdAt) }}
          </span>
        </template>
      </PersistentAdminTableColumn>
      <PersistentAdminTableColumn
        column-key="actions"
        fixed="right"
        :label="$t('admin.common.actions')"
        :default-width="260"
      >
        <template #default="{ row }">
          <div v-if="row.status === 'pending'" class="admin-action-group">
            <ElPopconfirm
              :title="$t('admin.content.reports.confirmReject')"
              @confirm="handleAction(row.id, 'reject')"
            >
              <template #reference>
                <ElButton
                  plain
                  size="small"
                  type="info"
                  :disabled="isActionPending(row.id)"
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
                  plain
                  size="small"
                  type="warning"
                  :disabled="isActionPending(row.id)"
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
                  plain
                  size="small"
                  type="danger"
                  :disabled="isActionPending(row.id)"
                >
                  {{ $t('admin.content.reports.deleteReview') }}
                </ElButton>
              </template>
            </ElPopconfirm>
          </div>
          <span v-else class="admin-cell-muted">—</span>
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
