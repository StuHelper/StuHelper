<script setup lang="ts">
import type { FreshmanApplication } from '#/api/admin';

import { reactive, ref } from 'vue';

import {
  ElAlert,
  ElButton,
  ElDialog,
  ElImage,
  ElInput,
  ElInputNumber,
  ElMessage,
  ElOption,
  ElPagination,
  ElPopconfirm,
  ElSelect,
  ElTag,
} from 'element-plus';

import {
  listFreshmanVerifications,
  reviewFreshmanVerification,
} from '#/api/admin';
import { useAdminAction } from '#/composables/use-admin-action';
import { useAdminList } from '#/composables/use-admin-list';
import { $t } from '#/locales';

import PersistentAdminTable from '../../shared/admin-table/PersistentAdminTable.vue';
import PersistentAdminTableColumn from '../../shared/admin-table/PersistentAdminTableColumn.vue';
import AdminContentLayout from '../../shared/AdminContentLayout.vue';
import { formatAdminDateTime } from '../../shared/display';
import {
  ADMIN_DEFAULT_PAGE_SIZE,
  ADMIN_PAGE_SIZES,
  ADMIN_PAGINATION_LAYOUT,
} from '../../shared/pagination';

type FreshmanReviewAction = 'approve' | 'approveWithDays' | 'reject';
type FreshmanStatusFilter = 'approved' | 'pending' | 'rejected';

const extensionDaysById = reactive<Record<string, number | undefined>>({});
const rejectionReasons = reactive<Record<string, string>>({});
const materialDialogVisible = ref(false);
const materialPreviewURL = ref('');

const {
  fetchData,
  items,
  loadError,
  loading,
  query,
  resetPageAndFetch,
  total,
} = useAdminList<
  FreshmanApplication,
  { page: number; pageSize: number; status: FreshmanStatusFilter }
>({
  fetcher: async (listQuery) => {
    const data = await listFreshmanVerifications(listQuery);
    for (const item of data.items) {
      extensionDaysById[item.id] ??= 0;
    }
    return data;
  },
  initialQuery: {
    page: 1,
    pageSize: ADMIN_DEFAULT_PAGE_SIZE,
    status: 'pending',
  },
});

const {
  actionError,
  clearActionError,
  isActionPending,
  pendingActionKinds,
  runAction,
} = useAdminAction();

async function approve(row: FreshmanApplication) {
  await handleReview(
    row,
    { action: 'approve' },
    $t('admin.users.freshmanVerification.approveSuccess'),
    'approve',
  );
}

async function approveWithDays(row: FreshmanApplication) {
  const expiresInDays = rowExtensionDays(row);
  if (expiresInDays === undefined) {
    return;
  }
  await handleReview(
    row,
    { action: 'approve', expiresInDays },
    $t('admin.users.freshmanVerification.approveWithDaysSuccess'),
    'approveWithDays',
  );
}

async function reject(row: FreshmanApplication) {
  const reason = rejectionReasons[row.id]?.trim();
  if (!reason) {
    ElMessage.warning(
      $t('admin.users.freshmanVerification.rejectReasonRequired'),
    );
    return;
  }

  const submitted = await handleReview(
    row,
    { action: 'reject', reason },
    $t('admin.users.freshmanVerification.rejectSuccess'),
    'reject',
  );
  if (submitted) {
    delete rejectionReasons[row.id];
  }
}

async function handleReview(
  row: FreshmanApplication,
  payload: Parameters<typeof reviewFreshmanVerification>[1],
  successMessage: string,
  action: FreshmanReviewAction,
) {
  const succeeded = await runAction(
    () => reviewFreshmanVerification(row.id, payload),
    {
      id: row.id,
      kind: action,
      successMessage,
    },
  );
  if (succeeded) {
    delete extensionDaysById[row.id];
    await fetchData();
  }
  return succeeded;
}

function rowActionLoading(
  row: FreshmanApplication,
  action: FreshmanReviewAction,
) {
  return pendingActionKinds.value[row.id] === action;
}

/**
 * 临时认证天数：仅正整数有效。0 是显式的"不要带天数"，对应禁用
 * 带天数通过按钮，而不是静默退化为普通通过。
 */
function rowExtensionDays(row: FreshmanApplication) {
  const days = extensionDaysById[row.id];
  return typeof days === 'number' && Number.isInteger(days) && days > 0
    ? days
    : undefined;
}

function openMaterial(row: FreshmanApplication) {
  if (!row.materialURL) {
    ElMessage.warning($t('admin.users.freshmanVerification.noMaterial'));
    return;
  }
  materialPreviewURL.value = row.materialURL;
  materialDialogVisible.value = true;
}

function statusType(status: FreshmanApplication['status']) {
  if (status === 'approved') return 'success';
  if (status === 'rejected') return 'danger';
  return 'warning';
}

function statusLabel(status: FreshmanApplication['status']) {
  if (status === 'approved') {
    return $t('admin.users.freshmanVerification.statusApproved');
  }
  if (status === 'rejected') {
    return $t('admin.users.freshmanVerification.statusRejected');
  }
  return $t('admin.users.freshmanVerification.statusPending');
}
</script>

<template>
  <AdminContentLayout
    :title="$t('admin.routes.userSystem.freshmanVerification')"
    :total="total"
  >
    <template #toolbar>
      <ElSelect
        v-model="query.status"
        class="admin-toolbar-control"
        :teleported="false"
        @change="resetPageAndFetch"
      >
        <ElOption
          :label="$t('admin.users.freshmanVerification.statusPending')"
          value="pending"
        />
        <ElOption
          :label="$t('admin.users.freshmanVerification.statusApproved')"
          value="approved"
        />
        <ElOption
          :label="$t('admin.users.freshmanVerification.statusRejected')"
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
      table-key="users.freshmanVerification"
      :loading="loading"
      :data="items"
      row-key="id"
      stripe
    >
      <PersistentAdminTableColumn
        column-key="status"
        data-field="status"
        :label="$t('admin.common.status')"
        :default-width="100"
      >
        <template #default="{ row }">
          <ElTag :type="statusType(row.status)" data-state="pendingReview">
            {{ statusLabel(row.status) }}
          </ElTag>
        </template>
      </PersistentAdminTableColumn>
      <PersistentAdminTableColumn
        column-key="schoolID"
        data-field="schoolID"
        :label="$t('admin.users.freshmanVerification.schoolColumn')"
        prop="schoolID"
        :default-width="120"
      />
      <PersistentAdminTableColumn
        column-key="qqID"
        data-field="qqID"
        :label="$t('admin.users.freshmanVerification.qqColumn')"
        :default-width="140"
      >
        <template #default="{ row }">{{ row.qqID || '—' }}</template>
      </PersistentAdminTableColumn>
      <PersistentAdminTableColumn
        column-key="applicant"
        :label="$t('admin.users.freshmanVerification.applicantColumn')"
        prop="applicantNameMasked"
        :default-width="120"
      />
      <PersistentAdminTableColumn
        column-key="createdAt"
        data-field="createdAt"
        :label="$t('admin.users.freshmanVerification.createdAtColumn')"
        :default-width="148"
      >
        <template #default="{ row }">
          <span class="admin-cell-muted">
            {{ formatAdminDateTime(row.createdAt) }}
          </span>
        </template>
      </PersistentAdminTableColumn>
      <PersistentAdminTableColumn
        column-key="failureCount"
        data-field="failureCount"
        :label="$t('admin.users.freshmanVerification.failureCountColumn')"
        :default-width="100"
      >
        <template #default="{ row }">{{ row.failureCount ?? '—' }}</template>
      </PersistentAdminTableColumn>
      <PersistentAdminTableColumn
        column-key="actions"
        fixed="right"
        :label="$t('admin.common.actions')"
        :default-width="420"
      >
        <template #default="{ row }">
          <div class="freshman-action-group">
            <ElButton
              plain
              size="small"
              type="primary"
              data-material-preview
              @click="openMaterial(row)"
            >
              {{ $t('admin.users.freshmanVerification.materialPreview') }}
            </ElButton>
            <ElPopconfirm
              :title="$t('admin.users.freshmanVerification.confirmApprove')"
              @confirm="approve(row)"
            >
              <template #reference>
                <ElButton
                  plain
                  size="small"
                  type="success"
                  data-action="approve"
                  :disabled="isActionPending(row.id)"
                  :loading="rowActionLoading(row, 'approve')"
                >
                  {{ $t('admin.users.freshmanVerification.approve') }}
                </ElButton>
              </template>
            </ElPopconfirm>
            <ElInputNumber
              v-model="extensionDaysById[row.id]"
              class="freshman-action-number"
              :disabled="isActionPending(row.id)"
              :min="0"
              size="small"
            />
            <ElPopconfirm
              :title="
                $t('admin.users.freshmanVerification.confirmApproveWithDays', {
                  days: rowExtensionDays(row) ?? 0,
                })
              "
              @confirm="approveWithDays(row)"
            >
              <template #reference>
                <ElButton
                  plain
                  size="small"
                  type="success"
                  data-action="approveWithDays"
                  :disabled="
                    isActionPending(row.id) ||
                    rowExtensionDays(row) === undefined
                  "
                  :loading="rowActionLoading(row, 'approveWithDays')"
                >
                  {{ $t('admin.users.freshmanVerification.approveWithDays') }}
                </ElButton>
              </template>
            </ElPopconfirm>
            <ElInput
              v-model="rejectionReasons[row.id]"
              class="freshman-action-reason"
              :disabled="isActionPending(row.id)"
              :placeholder="
                $t('admin.users.freshmanVerification.rejectReasonPlaceholder')
              "
              size="small"
            />
            <ElButton
              plain
              size="small"
              type="danger"
              data-action="reject"
              :disabled="isActionPending(row.id)"
              :loading="rowActionLoading(row, 'reject')"
              @click="reject(row)"
            >
              {{ $t('admin.users.freshmanVerification.reject') }}
            </ElButton>
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

    <ElDialog
      v-model="materialDialogVisible"
      :title="$t('admin.users.freshmanVerification.materialPreview')"
      width="720px"
    >
      <ElImage :src="materialPreviewURL" fit="contain" />
    </ElDialog>
  </AdminContentLayout>
</template>

<style scoped>
.freshman-action-group {
  display: flex;
  flex-wrap: nowrap;
  gap: 8px;
  align-items: center;
}

.freshman-action-number {
  width: 96px;
}

.freshman-action-reason {
  width: 140px;
}
</style>
