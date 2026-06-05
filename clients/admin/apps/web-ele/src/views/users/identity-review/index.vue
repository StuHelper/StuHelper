<script setup lang="ts">
import type { IdentityVerification } from '#/api/admin';

import { onMounted, reactive, ref } from 'vue';

import {
  ElAlert,
  ElButton,
  ElDialog,
  ElInput,
  ElMessage,
  ElOption,
  ElPagination,
  ElPopconfirm,
  ElSelect,
  ElTag,
} from 'element-plus';

import { getIdentityList, reviewIdentity } from '#/api/admin';
import { $t } from '#/locales';

import PersistentAdminTable from '../../shared/admin-table/PersistentAdminTable.vue';
import PersistentAdminTableColumn from '../../shared/admin-table/PersistentAdminTableColumn.vue';
import AdminContentLayout from '../../shared/AdminContentLayout.vue';
import { formatAdminDateTime } from '../../shared/display';

type IdentityReviewAction = 'approve' | 'reject';

const loading = ref(false);
const items = ref<IdentityVerification[]>([]);
const total = ref(0);
const loadError = ref('');
const actionError = ref('');
const rejectDialogVisible = ref(false);
const rejectTarget = ref<IdentityVerification | null>(null);
const rejectionReason = ref('');
const reviewingActionsByUserId = reactive<
  Record<number, IdentityReviewAction | undefined>
>({});
let fetchRequestSeq = 0;
const query = reactive({
  page: 1,
  pageSize: 20,
  status: 'all' as 'all' | 'pending' | 'rejected' | 'verified',
});

async function fetchData() {
  const requestSeq = ++fetchRequestSeq;
  loading.value = true;
  loadError.value = '';
  try {
    const data = await getIdentityList(query);
    if (requestSeq !== fetchRequestSeq) return;
    items.value = data.items;
    total.value = data.total;
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

async function handleReview(
  userId: number,
  approved: boolean,
  rejectionReason?: string,
) {
  const action: IdentityReviewAction = approved ? 'approve' : 'reject';
  if (userReviewing(userId)) {
    return false;
  }

  reviewingActionsByUserId[userId] = action;
  actionError.value = '';
  try {
    await reviewIdentity(userId, {
      approved,
      ...(rejectionReason ? { rejectionReason } : {}),
    });
    ElMessage.success(
      $t(
        approved
          ? 'admin.users.identityReview.approveSuccess'
          : 'admin.users.identityReview.rejectSuccess',
      ),
    );
    await fetchData();
    return true;
  } catch (error) {
    handleActionError(error);
    return false;
  } finally {
    delete reviewingActionsByUserId[userId];
  }
}

function openRejectDialog(row: IdentityVerification) {
  rejectTarget.value = row;
  rejectionReason.value = '';
  rejectDialogVisible.value = true;
}

async function submitReject() {
  const reason = rejectionReason.value.trim();
  if (!reason) {
    ElMessage.error($t('admin.users.identityReview.rejectReasonRequired'));
    return;
  }

  const target = rejectTarget.value;
  if (!target) {
    return;
  }

  const submitted = await handleReview(target.userID, false, reason);
  if (!submitted) {
    return;
  }

  rejectDialogVisible.value = false;
  rejectTarget.value = null;
  rejectionReason.value = '';
}

function userReviewing(userId: number) {
  return Boolean(reviewingActionsByUserId[userId]);
}

function userActionLoading(userId: number, action: IdentityReviewAction) {
  return reviewingActionsByUserId[userId] === action;
}

function rejectTargetReviewing() {
  return rejectTarget.value ? userReviewing(rejectTarget.value.userID) : false;
}

function rejectTargetActionLoading(action: IdentityReviewAction) {
  return rejectTarget.value
    ? userActionLoading(rejectTarget.value.userID, action)
    : false;
}

const statusTag = (row: IdentityVerification) => {
  if (row.verified) return 'success';
  return 'warning';
};

const statusLabel = (row: IdentityVerification) => {
  if (row.verified) return $t('admin.users.identityReview.status.verified');
  if (row.reviewedAt) return $t('admin.users.identityReview.status.rejected');
  return $t('admin.users.identityReview.status.pending');
};

const docTypeLabel = (docType: IdentityVerification['docType']) => {
  if (!docType) {
    return $t('admin.common.unavailable');
  }

  return $t(`admin.users.identityReview.docType.${docType}`);
};

const verifyMethodLabel = (method: IdentityVerification['verifyMethod']) => {
  return method
    ? $t(`admin.users.identityReview.verifyMethod.${method}`)
    : $t('admin.users.identityReview.status.pending');
};

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
    :title="$t('admin.routes.userSystem.identityReview')"
    :total="total"
  >
    <template #toolbar>
      <ElSelect
        v-model="query.status"
        class="admin-toolbar-control"
        :placeholder="$t('admin.users.identityReview.statusPlaceholder')"
        :teleported="false"
        @change="resetPageAndFetch"
      >
        <ElOption :label="$t('admin.common.all')" value="all" />
        <ElOption
          :label="$t('admin.users.identityReview.status.pending')"
          value="pending"
        />
        <ElOption
          :label="$t('admin.users.identityReview.status.verified')"
          value="verified"
        />
        <ElOption
          :label="$t('admin.users.identityReview.status.rejected')"
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
      @close="actionError = ''"
    />

    <PersistentAdminTable
      table-key="users.identityReview"
      :loading="loading"
      :data="items"
      row-key="userID"
      stripe
    >
      <PersistentAdminTableColumn
        column-key="userID"
        :label="$t('admin.users.identityReview.userId')"
        prop="userID"
        :default-width="96"
      />
      <PersistentAdminTableColumn
        column-key="realName"
        :label="$t('admin.users.identityReview.realName')"
        prop="realName"
        :default-width="160"
        show-overflow-tooltip
      />
      <PersistentAdminTableColumn
        column-key="docType"
        :label="$t('admin.users.identityReview.docTypeLabel')"
        :default-width="180"
      >
        <template #default="{ row }">
          {{ docTypeLabel(row.docType) }}
        </template>
      </PersistentAdminTableColumn>
      <PersistentAdminTableColumn
        column-key="verifyMethod"
        :label="$t('admin.users.identityReview.verifyMethodLabel')"
        :default-width="128"
      >
        <template #default="{ row }">
          {{ verifyMethodLabel(row.verifyMethod) }}
        </template>
      </PersistentAdminTableColumn>
      <PersistentAdminTableColumn
        column-key="status"
        :label="$t('admin.users.identityReview.statusLabel')"
        :default-width="112"
      >
        <template #default="{ row }">
          <ElTag :type="statusTag(row)" size="small">
            {{ statusLabel(row) }}
          </ElTag>
        </template>
      </PersistentAdminTableColumn>
      <PersistentAdminTableColumn
        column-key="finishedAt"
        :label="$t('admin.users.identityReview.finishedAt')"
        :default-width="148"
      >
        <template #default="{ row }">
          <span class="admin-cell-muted">
            {{ formatAdminDateTime(row.verifiedAt || row.reviewedAt) }}
          </span>
        </template>
      </PersistentAdminTableColumn>
      <PersistentAdminTableColumn
        column-key="actions"
        fixed="right"
        :label="$t('admin.common.actions')"
        :default-width="170"
      >
        <template #default="{ row }">
          <div v-if="!row.verified" class="admin-action-group">
            <ElPopconfirm
              :title="$t('admin.users.identityReview.confirmApprove')"
              @confirm="handleReview(row.userID, true)"
            >
              <template #reference>
                <ElButton
                  plain
                  size="small"
                  type="success"
                  data-action="approve"
                  :disabled="userReviewing(row.userID)"
                  :loading="userActionLoading(row.userID, 'approve')"
                >
                  {{ $t('admin.users.identityReview.approve') }}
                </ElButton>
              </template>
            </ElPopconfirm>
            <ElButton
              plain
              size="small"
              type="danger"
              data-action="reject"
              :disabled="userReviewing(row.userID)"
              :loading="userActionLoading(row.userID, 'reject')"
              @click="openRejectDialog(row)"
            >
              {{ $t('admin.users.identityReview.reject') }}
            </ElButton>
          </div>
          <span v-else class="admin-cell-muted">—</span>
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

    <ElDialog
      v-model="rejectDialogVisible"
      :title="$t('admin.users.identityReview.rejectDialogTitle')"
      width="420px"
    >
      <ElInput
        v-model="rejectionReason"
        :disabled="rejectTargetReviewing()"
        :placeholder="$t('admin.users.identityReview.rejectReasonPlaceholder')"
        :rows="4"
        type="textarea"
      />
      <template #footer>
        <ElButton
          :disabled="rejectTargetReviewing()"
          @click="rejectDialogVisible = false"
        >
          {{ $t('admin.common.cancel') }}
        </ElButton>
        <ElButton
          :loading="rejectTargetActionLoading('reject')"
          type="primary"
          @click="submitReject"
        >
          {{ $t('admin.common.confirm') }}
        </ElButton>
      </template>
    </ElDialog>
  </AdminContentLayout>
</template>
