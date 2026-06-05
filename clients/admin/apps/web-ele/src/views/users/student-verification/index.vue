<script setup lang="ts">
import type { StudentVerification } from '#/api/admin';

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

import {
  getStudentVerificationList,
  reviewStudentVerification,
} from '#/api/admin';
import { $t } from '#/locales';
import { SCHOOL_SCOPE_REQUIRED_ERROR, useAuthStore } from '#/store/auth';

import PersistentAdminTable from '../../shared/admin-table/PersistentAdminTable.vue';
import PersistentAdminTableColumn from '../../shared/admin-table/PersistentAdminTableColumn.vue';
import AdminContentLayout from '../../shared/AdminContentLayout.vue';
import { formatAdminDateTime } from '../../shared/display';

const STUDENT_REVIEW_CAPABILITY = 'user:student:review';
type StudentReviewAction = 'approve' | 'reject';

const loading = ref(false);
const items = ref<StudentVerification[]>([]);
const total = ref(0);
const loadError = ref('');
const actionError = ref('');
const authStore = useAuthStore();
const rejectDialogVisible = ref(false);
const rejectTarget = ref<null | StudentVerification>(null);
const rejectionReason = ref('');
const reviewingActionsByUserId = reactive<
  Record<number, StudentReviewAction | undefined>
>({});
let fetchRequestSeq = 0;
const query = reactive({
  page: 1,
  pageSize: 20,
  status: 'all' as 'all' | 'pending' | 'rejected' | 'verified',
  schoolId: '',
});

function normalizeScopedSchoolId(): boolean {
  try {
    const schoolId = authStore.resolveScopedSchoolId(
      STUDENT_REVIEW_CAPABILITY,
      query.schoolId,
    );
    if (schoolId !== query.schoolId) {
      query.schoolId = schoolId;
    }
    return true;
  } catch (error) {
    if (
      error instanceof Error &&
      error.message === SCHOOL_SCOPE_REQUIRED_ERROR
    ) {
      items.value = [];
      total.value = 0;
      ElMessage.warning($t('admin.users.studentVerification.schoolIdRequired'));
      return false;
    }
    throw error;
  }
}

async function fetchData() {
  const requestSeq = ++fetchRequestSeq;
  if (!normalizeScopedSchoolId()) {
    return;
  }
  loading.value = true;
  loadError.value = '';
  try {
    const data = await getStudentVerificationList(query);
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
  const action: StudentReviewAction = approved ? 'approve' : 'reject';
  if (userReviewing(userId)) {
    return false;
  }

  reviewingActionsByUserId[userId] = action;
  actionError.value = '';
  try {
    await reviewStudentVerification(userId, {
      approved,
      ...(rejectionReason ? { rejectionReason } : {}),
    });
    ElMessage.success(
      $t(
        approved
          ? 'admin.users.studentVerification.approveSuccess'
          : 'admin.users.studentVerification.rejectSuccess',
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

function openRejectDialog(row: StudentVerification) {
  rejectTarget.value = row;
  rejectionReason.value = '';
  rejectDialogVisible.value = true;
}

async function submitReject() {
  const reason = rejectionReason.value.trim();
  if (!reason) {
    ElMessage.error($t('admin.users.studentVerification.rejectReasonRequired'));
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

function userActionLoading(userId: number, action: StudentReviewAction) {
  return reviewingActionsByUserId[userId] === action;
}

function rejectTargetReviewing() {
  return rejectTarget.value ? userReviewing(rejectTarget.value.userID) : false;
}

function rejectTargetActionLoading(action: StudentReviewAction) {
  return rejectTarget.value
    ? userActionLoading(rejectTarget.value.userID, action)
    : false;
}

type TagType = 'danger' | 'info' | 'success' | 'warning';

const statusTag = (status: string): TagType => {
  const map: Record<string, TagType> = {
    pending: 'warning',
    verified: 'success',
    rejected: 'danger',
  };
  return map[status] || 'info';
};

const statusLabel = (status: string) => {
  return $t(`admin.users.studentVerification.status.${status}`);
};

const verificationMethodLabel = (
  method: StudentVerification['verificationMethod'],
) => {
  return method
    ? $t(`admin.users.studentVerification.method.${method}`)
    : $t('admin.common.notSet');
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
    :title="$t('admin.routes.userSystem.studentVerification')"
    :total="total"
  >
    <template #toolbar>
      <ElSelect
        v-model="query.status"
        class="admin-toolbar-control"
        :placeholder="$t('admin.users.studentVerification.statusPlaceholder')"
        :teleported="false"
        @change="resetPageAndFetch"
      >
        <ElOption :label="$t('admin.common.all')" value="all" />
        <ElOption
          :label="$t('admin.users.studentVerification.status.pending')"
          value="pending"
        />
        <ElOption
          :label="$t('admin.users.studentVerification.status.verified')"
          value="verified"
        />
        <ElOption
          :label="$t('admin.users.studentVerification.status.rejected')"
          value="rejected"
        />
      </ElSelect>
      <ElInput
        v-model="query.schoolId"
        class="admin-toolbar-control admin-toolbar-control--wide"
        clearable
        :placeholder="$t('admin.users.studentVerification.schoolIdPlaceholder')"
        @clear="resetPageAndFetch"
        @keyup.enter="resetPageAndFetch"
      />
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
      table-key="users.studentVerification"
      :loading="loading"
      :data="items"
      row-key="userID"
      stripe
    >
      <PersistentAdminTableColumn
        column-key="userID"
        :label="$t('admin.users.studentVerification.userId')"
        prop="userID"
        :default-width="96"
      />
      <PersistentAdminTableColumn
        column-key="schoolID"
        :label="$t('admin.users.studentVerification.schoolId')"
        :default-min-width="140"
        prop="schoolID"
      />
      <PersistentAdminTableColumn
        column-key="activeStudentID"
        :label="$t('admin.users.studentVerification.activeStudentId')"
        prop="activeStudentID"
        :default-width="220"
        show-overflow-tooltip
      />
      <PersistentAdminTableColumn
        column-key="status"
        :label="$t('admin.users.studentVerification.statusLabel')"
        :default-width="112"
      >
        <template #default="{ row }">
          <ElTag :type="statusTag(row.verificationStatus)" size="small">
            {{ statusLabel(row.verificationStatus) }}
          </ElTag>
        </template>
      </PersistentAdminTableColumn>
      <PersistentAdminTableColumn
        column-key="verificationMethod"
        :label="$t('admin.users.studentVerification.methodLabel')"
        :default-width="112"
      >
        <template #default="{ row }">
          {{ verificationMethodLabel(row.verificationMethod) }}
        </template>
      </PersistentAdminTableColumn>
      <PersistentAdminTableColumn
        column-key="createdAt"
        :label="$t('admin.users.studentVerification.createdAt')"
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
        :default-width="170"
      >
        <template #default="{ row }">
          <div
            v-if="row.verificationStatus === 'pending'"
            class="admin-action-group"
          >
            <ElPopconfirm
              :title="$t('admin.users.studentVerification.confirmApprove')"
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
                  {{ $t('admin.users.studentVerification.approve') }}
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
              {{ $t('admin.users.studentVerification.reject') }}
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
      :title="$t('admin.users.studentVerification.rejectDialogTitle')"
      width="420px"
    >
      <ElInput
        v-model="rejectionReason"
        :disabled="rejectTargetReviewing()"
        :placeholder="
          $t('admin.users.studentVerification.rejectReasonPlaceholder')
        "
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
