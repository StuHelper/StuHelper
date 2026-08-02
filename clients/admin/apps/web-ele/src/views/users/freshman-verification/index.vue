<script setup lang="ts">
import type { FreshmanApplication } from '#/api/admin';

import { onMounted, reactive, ref } from 'vue';

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
  ElSelect,
  ElTag,
} from 'element-plus';

import {
  listFreshmanVerifications,
  reviewFreshmanVerification,
} from '#/api/admin';
import { $t } from '#/locales';

import PersistentAdminTable from '../../shared/admin-table/PersistentAdminTable.vue';
import PersistentAdminTableColumn from '../../shared/admin-table/PersistentAdminTableColumn.vue';
import AdminContentLayout from '../../shared/AdminContentLayout.vue';
import { formatAdminDateTime } from '../../shared/display';
import { isFreshmanReviewPending } from './review-state';

type FreshmanReviewRow = FreshmanApplication & {
  failureCount?: number;
  materialURL?: string;
  qqID?: string;
};
type FreshmanReviewAction = 'approve' | 'approveWithDays' | 'reject';

const loading = ref(false);
const items = ref<FreshmanReviewRow[]>([]);
const total = ref(0);
const loadError = ref('');
const actionError = ref('');
const materialDialogVisible = ref(false);
const materialPreviewURL = ref('');
const query = reactive({
  page: 1,
  pageSize: 20,
  status: 'pending' as 'approved' | 'pending' | 'rejected',
});
const extensionDaysById = reactive<Record<string, number | undefined>>({});
const rejectionReasons = reactive<Record<string, string>>({});
const reviewingActionsById = reactive<
  Record<string, FreshmanReviewAction | undefined>
>({});
let fetchRequestSeq = 0;

async function fetchData() {
  const requestSeq = ++fetchRequestSeq;
  loading.value = true;
  loadError.value = '';
  try {
    const data = await listFreshmanVerifications(query);
    if (requestSeq !== fetchRequestSeq) return;
    items.value = data.items as FreshmanReviewRow[];
    total.value = data.total;
    for (const item of items.value) {
      extensionDaysById[item.id] ??= 0;
    }
  } catch (error) {
    if (requestSeq !== fetchRequestSeq) return;
    loadError.value = adminErrorMessage(error);
  } finally {
    if (requestSeq === fetchRequestSeq) {
      loading.value = false;
    }
  }
}

function resetPageAndFetch() {
  query.page = 1;
  void fetchData();
}

async function approve(row: FreshmanReviewRow, expiresInDays?: number) {
  await handleReview(
    row,
    {
      action: 'approve',
      ...(expiresInDays ? { expiresInDays } : {}),
    },
    expiresInDays ? '已通过新生审核并设置临时认证期限' : '已通过新生审核',
    expiresInDays ? 'approveWithDays' : 'approve',
  );
}

async function reject(row: FreshmanReviewRow) {
  const reason = rejectionReasons[row.id]?.trim();
  if (!reason) {
    ElMessage.error('请填写驳回原因');
    return;
  }

  const submitted = await handleReview(
    row,
    { action: 'reject', reason },
    '已驳回新生审核',
    'reject',
  );
  if (submitted) {
    delete rejectionReasons[row.id];
  }
}

async function handleReview(
  row: FreshmanReviewRow,
  payload: Parameters<typeof reviewFreshmanVerification>[1],
  successMessage: string,
  action: FreshmanReviewAction,
) {
  if (!isFreshmanReviewPending(row.status)) {
    ElMessage.warning('该申请已处理，请刷新列表');
    return false;
  }
  if (rowReviewing(row)) {
    return false;
  }

  reviewingActionsById[row.id] = action;
  actionError.value = '';
  try {
    await reviewFreshmanVerification(row.id, payload);
    ElMessage.success(successMessage);
    delete extensionDaysById[row.id];
    await fetchData();
    return true;
  } catch (error) {
    handleActionError(error);
    return false;
  } finally {
    delete reviewingActionsById[row.id];
  }
}

function rowReviewing(row: FreshmanReviewRow) {
  return Boolean(reviewingActionsById[row.id]);
}

function rowActionLoading(
  row: FreshmanReviewRow,
  action: FreshmanReviewAction,
) {
  return reviewingActionsById[row.id] === action;
}

function rowExtensionDays(row: FreshmanReviewRow) {
  const days = extensionDaysById[row.id];
  return typeof days === 'number' && days > 0 ? days : undefined;
}

function openMaterial(row: FreshmanReviewRow) {
  if (!row.materialURL) {
    ElMessage.warning('暂无材料预览');
    return;
  }
  materialPreviewURL.value = row.materialURL;
  materialDialogVisible.value = true;
}

function statusType(status: FreshmanReviewRow['status']) {
  if (status === 'approved') return 'success';
  if (status === 'rejected') return 'danger';
  return 'warning';
}

function statusLabel(status: FreshmanReviewRow['status']) {
  if (status === 'approved') return '已通过';
  if (status === 'rejected') return '已驳回';
  return '待审核';
}

function handleActionError(error: unknown) {
  actionError.value = adminErrorMessage(error);
  ElMessage.error(actionError.value);
}

function adminErrorMessage(error: unknown): string {
  return error instanceof Error && error.message
    ? error.message
    : $t('admin.result.requestFailed');
}

onMounted(fetchData);
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
        <ElOption label="待审核" value="pending" />
        <ElOption label="已通过" value="approved" />
        <ElOption label="已驳回" value="rejected" />
      </ElSelect>
      <ElButton type="primary" @click="resetPageAndFetch">查询</ElButton>
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
      @close="actionError = ''"
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
        label="状态"
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
        label="学校"
        prop="schoolID"
        :default-width="120"
      />
      <PersistentAdminTableColumn
        column-key="qqID"
        data-field="qqID"
        label="QQ"
        :default-width="140"
      >
        <template #default="{ row }">{{ row.qqID || '—' }}</template>
      </PersistentAdminTableColumn>
      <PersistentAdminTableColumn
        column-key="applicant"
        label="申请人"
        prop="applicantNameMasked"
        :default-width="120"
      />
      <PersistentAdminTableColumn
        column-key="createdAt"
        data-field="createdAt"
        label="申请时间"
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
        label="失败次数"
        :default-width="100"
      >
        <template #default="{ row }">{{ row.failureCount ?? '—' }}</template>
      </PersistentAdminTableColumn>
      <PersistentAdminTableColumn
        column-key="actions"
        fixed="right"
        label="操作"
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
              材料预览
            </ElButton>
            <template v-if="isFreshmanReviewPending(row.status)">
              <ElButton
                plain
                size="small"
                type="success"
                data-action="approve"
                :disabled="rowReviewing(row)"
                :loading="rowActionLoading(row, 'approve')"
                @click="approve(row)"
              >
                通过
              </ElButton>
              <ElInputNumber
                v-model="extensionDaysById[row.id]"
                class="freshman-action-number"
                :disabled="rowReviewing(row)"
                :min="0"
                size="small"
              />
              <ElButton
                plain
                size="small"
                type="success"
                data-action="approveWithDays"
                :disabled="rowReviewing(row)"
                :loading="rowActionLoading(row, 'approveWithDays')"
                @click="approve(row, rowExtensionDays(row))"
              >
                带天数通过
              </ElButton>
              <ElInput
                v-model="rejectionReasons[row.id]"
                class="freshman-action-reason"
                :disabled="rowReviewing(row)"
                placeholder="驳回原因"
                size="small"
              />
              <ElButton
                plain
                size="small"
                type="danger"
                data-action="reject"
                :disabled="rowReviewing(row)"
                :loading="rowActionLoading(row, 'reject')"
                @click="reject(row)"
              >
                驳回
              </ElButton>
            </template>
            <span v-else class="admin-cell-muted" data-review-complete>
              已处理
            </span>
          </div>
        </template>
      </PersistentAdminTableColumn>
    </PersistentAdminTable>

    <template #pagination>
      <ElPagination
        v-model:current-page="query.page"
        v-model:page-size="query.pageSize"
        :total="total"
        layout="total, prev, pager, next"
        @current-change="fetchData"
      />
    </template>

    <ElDialog v-model="materialDialogVisible" title="材料预览" width="720px">
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
