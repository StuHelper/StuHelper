<script setup lang="ts">
import type { IdentityVerification } from '#/api/admin';

import { ref } from 'vue';

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

type IdentityReviewAction = 'approve' | 'reject';
type IdentityStatusFilter = 'all' | 'pending' | 'rejected' | 'verified';

const {
  fetchData,
  items,
  loadError,
  loading,
  query,
  resetPageAndFetch,
  total,
} = useAdminList<
  IdentityVerification,
  { page: number; pageSize: number; status: IdentityStatusFilter }
>({
  fetcher: (listQuery) => getIdentityList(listQuery),
  initialQuery: {
    page: 1,
    pageSize: ADMIN_DEFAULT_PAGE_SIZE,
    status: 'all',
  },
});

const {
  actionError,
  clearActionError,
  isActionPending,
  pendingActionKinds,
  runAction,
} = useAdminAction();

const rejectDialogVisible = ref(false);
const rejectTarget = ref<IdentityVerification | null>(null);
const rejectionReason = ref('');

async function handleReview(
  userId: number,
  approved: boolean,
  rejectionReason?: string,
) {
  const action: IdentityReviewAction = approved ? 'approve' : 'reject';
  const succeeded = await runAction(
    () =>
      reviewIdentity(userId, {
        approved,
        ...(rejectionReason ? { rejectionReason } : {}),
      }),
    {
      id: userId,
      kind: action,
      successMessage: $t(
        approved
          ? 'admin.users.identityReview.approveSuccess'
          : 'admin.users.identityReview.rejectSuccess',
      ),
    },
  );
  if (succeeded) {
    await fetchData();
  }
  return succeeded;
}

function openRejectDialog(row: IdentityVerification) {
  rejectTarget.value = row;
  rejectionReason.value = '';
  rejectDialogVisible.value = true;
}

async function submitReject() {
  const reason = rejectionReason.value.trim();
  if (!reason) {
    ElMessage.warning($t('admin.users.identityReview.rejectReasonRequired'));
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

function userActionLoading(userId: number, action: IdentityReviewAction) {
  return pendingActionKinds.value[String(userId)] === action;
}

function rejectTargetReviewing() {
  return rejectTarget.value
    ? isActionPending(rejectTarget.value.userID)
    : false;
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
      @close="clearActionError"
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
                  :disabled="isActionPending(row.userID)"
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
              :disabled="isActionPending(row.userID)"
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
        background
        :layout="ADMIN_PAGINATION_LAYOUT"
        :page-sizes="ADMIN_PAGE_SIZES"
        :total="total"
        @current-change="fetchData"
        @size-change="resetPageAndFetch"
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
