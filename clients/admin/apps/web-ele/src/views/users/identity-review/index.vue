<script setup lang="ts">
import type {
  IdentityVerification,
  IdentityVerificationReviewDetail,
} from '#/api/admin';

import { computed, onMounted, reactive, ref } from 'vue';

import {
  ElAlert,
  ElButton,
  ElDialog,
  ElEmpty,
  ElImage,
  ElInput,
  ElMessage,
  ElOption,
  ElPagination,
  ElPopconfirm,
  ElSelect,
  ElSkeleton,
  ElTag,
} from 'element-plus';

import {
  getIdentityList,
  getIdentityReviewDetail,
  reviewIdentity,
} from '#/api/admin';
import { $t } from '#/locales';

import PersistentAdminTable from '../../shared/admin-table/PersistentAdminTable.vue';
import PersistentAdminTableColumn from '../../shared/admin-table/PersistentAdminTableColumn.vue';
import AdminContentLayout from '../../shared/AdminContentLayout.vue';
import { formatAdminDateTime } from '../../shared/display';

type IdentityReviewAction = 'approve' | 'reject';
type IdentityEvidenceSlot = 'back' | 'front' | 'selfie';

const loading = ref(false);
const items = ref<IdentityVerification[]>([]);
const total = ref(0);
const loadError = ref('');
const actionError = ref('');
const rejectDialogVisible = ref(false);
const rejectTarget = ref<IdentityVerification | null>(null);
const rejectionReason = ref('');
const detailDialogVisible = ref(false);
const detailLoading = ref(false);
const detailError = ref('');
const detailTarget = ref<IdentityVerification | null>(null);
const detail = ref<IdentityVerificationReviewDetail | null>(null);
const failedEvidenceSlots = reactive<Record<IdentityEvidenceSlot, boolean>>({
  back: false,
  front: false,
  selfie: false,
});
const reviewingActionsByUserId = reactive<
  Record<number, IdentityReviewAction | undefined>
>({});
let fetchRequestSeq = 0;
let detailRequestSeq = 0;
const query = reactive({
  page: 1,
  pageSize: 20,
  status: 'all' as 'all' | 'pending' | 'rejected' | 'verified',
});

const detailPreviewURLs = computed(() => {
  const current = detail.value;
  if (!current) return [];
  return [
    current.docPhotoFrontURL,
    current.docPhotoBackURL,
    current.docPhotoSelfieURL,
  ].filter((url): url is string => url !== null);
});

const detailHasRequiredEvidence = computed(
  () =>
    Boolean(detail.value?.docPhotoFrontURL) &&
    Boolean(detail.value?.docPhotoSelfieURL),
);
const detailHasFailedEvidence = computed(() =>
  Object.values(failedEvidenceSlots).some(Boolean),
);
const detailCanApprove = computed(
  () => detailHasRequiredEvidence.value && !detailHasFailedEvidence.value,
);

function resetFailedEvidenceSlots() {
  failedEvidenceSlots.back = false;
  failedEvidenceSlots.front = false;
  failedEvidenceSlots.selfie = false;
}

function markEvidenceLoadFailed(slot: IdentityEvidenceSlot) {
  failedEvidenceSlots[slot] = true;
}

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

function isPending(row: IdentityVerification) {
  return !row.verified && !row.reviewedAt;
}

async function fetchIdentityDetail() {
  const target = detailTarget.value;
  if (!target) return;

  const userId = target.userID;
  const requestSeq = ++detailRequestSeq;
  detailLoading.value = true;
  detailError.value = '';
  detail.value = null;
  resetFailedEvidenceSlots();
  try {
    const data = await getIdentityReviewDetail(userId);
    if (
      requestSeq !== detailRequestSeq ||
      detailTarget.value?.userID !== userId
    ) {
      return;
    }
    detail.value = data;
  } catch (error) {
    if (requestSeq !== detailRequestSeq) return;
    detailError.value = adminErrorMessage(error);
  } finally {
    if (requestSeq === detailRequestSeq) {
      detailLoading.value = false;
    }
  }
}

function openEvidenceDialog(row: IdentityVerification) {
  if (!isPending(row)) return;
  detailTarget.value = row;
  detail.value = null;
  detailError.value = '';
  detailDialogVisible.value = true;
  void fetchIdentityDetail();
}

function resetIdentityDetail() {
  detailRequestSeq += 1;
  detailLoading.value = false;
  detailError.value = '';
  detail.value = null;
  detailTarget.value = null;
  resetFailedEvidenceSlots();
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
  if (!isPending(row)) return;
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
  detailDialogVisible.value = false;
  rejectTarget.value = null;
  rejectionReason.value = '';
}

async function approveDetail() {
  const target = detailTarget.value;
  if (
    !target ||
    !detail.value ||
    !isPending(target) ||
    !detailCanApprove.value
  ) {
    return;
  }

  const submitted = await handleReview(target.userID, true);
  if (submitted) {
    detailDialogVisible.value = false;
  }
}

function rejectDetail() {
  const target = detailTarget.value;
  if (target) {
    openRejectDialog(target);
  }
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

function detailTargetReviewing() {
  return detailTarget.value ? userReviewing(detailTarget.value.userID) : false;
}

function detailTargetActionLoading(action: IdentityReviewAction) {
  return detailTarget.value
    ? userActionLoading(detailTarget.value.userID, action)
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
        :default-width="132"
      >
        <template #default="{ row }">
          <div v-if="isPending(row)" class="admin-action-group">
            <ElButton
              plain
              size="small"
              type="primary"
              data-action="review-evidence"
              :disabled="userReviewing(row.userID)"
              @click="openEvidenceDialog(row)"
            >
              {{ $t('admin.users.identityReview.reviewEvidence') }}
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
      v-model="detailDialogVisible"
      destroy-on-close
      :title="$t('admin.users.identityReview.evidenceDialogTitle')"
      width="min(920px, 92vw)"
      @closed="resetIdentityDetail"
    >
      <ElSkeleton v-if="detailLoading" animated :rows="5" />

      <ElAlert
        v-else-if="detailError"
        type="error"
        :closable="false"
        show-icon
        :title="detailError"
      >
        <ElButton size="small" @click="fetchIdentityDetail">
          {{ $t('admin.common.retry') }}
        </ElButton>
      </ElAlert>

      <div v-else-if="detail" class="identity-evidence-detail">
        <dl class="identity-evidence-meta">
          <div>
            <dt>{{ $t('admin.users.identityReview.userId') }}</dt>
            <dd>{{ detail.userID }}</dd>
          </div>
          <div>
            <dt>{{ $t('admin.users.identityReview.realName') }}</dt>
            <dd>{{ detail.realName }}</dd>
          </div>
          <div>
            <dt>{{ $t('admin.users.identityReview.docTypeLabel') }}</dt>
            <dd>{{ docTypeLabel(detail.docType) }}</dd>
          </div>
        </dl>

        <ElAlert
          v-if="!detailHasRequiredEvidence"
          type="warning"
          :closable="false"
          show-icon
          :title="$t('admin.users.identityReview.incompleteEvidence')"
        />

        <ElAlert
          v-else-if="detailHasFailedEvidence"
          type="error"
          :closable="false"
          show-icon
          :title="$t('admin.users.identityReview.evidenceLoadFailed')"
        >
          <ElButton size="small" @click="fetchIdentityDetail">
            {{ $t('admin.users.identityReview.refreshEvidence') }}
          </ElButton>
        </ElAlert>

        <div class="identity-evidence-grid">
          <section class="identity-evidence-card">
            <h3>{{ $t('admin.users.identityReview.photoFront') }}</h3>
            <ElImage
              v-if="detail.docPhotoFrontURL"
              :alt="$t('admin.users.identityReview.photoFront')"
              fit="contain"
              :preview-src-list="detailPreviewURLs"
              :src="detail.docPhotoFrontURL"
              @error="markEvidenceLoadFailed('front')"
            />
            <p v-else class="identity-evidence-missing">
              {{ $t('admin.users.identityReview.missingEvidence') }}
            </p>
          </section>

          <section class="identity-evidence-card">
            <h3>{{ $t('admin.users.identityReview.photoBack') }}</h3>
            <ElImage
              v-if="detail.docPhotoBackURL"
              :alt="$t('admin.users.identityReview.photoBack')"
              fit="contain"
              :preview-src-list="detailPreviewURLs"
              :src="detail.docPhotoBackURL"
              @error="markEvidenceLoadFailed('back')"
            />
            <p v-else class="identity-evidence-missing">
              {{ $t('admin.users.identityReview.optionalEvidenceMissing') }}
            </p>
          </section>

          <section class="identity-evidence-card">
            <h3>{{ $t('admin.users.identityReview.photoSelfie') }}</h3>
            <ElImage
              v-if="detail.docPhotoSelfieURL"
              :alt="$t('admin.users.identityReview.photoSelfie')"
              fit="contain"
              :preview-src-list="detailPreviewURLs"
              :src="detail.docPhotoSelfieURL"
              @error="markEvidenceLoadFailed('selfie')"
            />
            <p v-else class="identity-evidence-missing">
              {{ $t('admin.users.identityReview.missingEvidence') }}
            </p>
          </section>
        </div>
      </div>

      <ElEmpty
        v-else
        :description="$t('admin.users.identityReview.noEvidenceDetail')"
      />

      <template #footer>
        <ElButton
          :disabled="detailTargetReviewing()"
          @click="detailDialogVisible = false"
        >
          {{ $t('admin.common.cancel') }}
        </ElButton>
        <ElButton
          data-action="reject"
          :disabled="
            !detail ||
            !detailTarget ||
            !isPending(detailTarget) ||
            detailTargetReviewing()
          "
          :loading="detailTargetActionLoading('reject')"
          type="danger"
          @click="rejectDetail"
        >
          {{ $t('admin.users.identityReview.reject') }}
        </ElButton>
        <ElPopconfirm
          :title="$t('admin.users.identityReview.confirmApprove')"
          @confirm="approveDetail"
        >
          <template #reference>
            <ElButton
              data-action="approve"
              :disabled="
                !detail ||
                !detailTarget ||
                !isPending(detailTarget) ||
                !detailCanApprove ||
                detailTargetReviewing()
              "
              :loading="detailTargetActionLoading('approve')"
              type="success"
            >
              {{ $t('admin.users.identityReview.approve') }}
            </ElButton>
          </template>
        </ElPopconfirm>
      </template>
    </ElDialog>

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

<style scoped>
.identity-evidence-detail {
  display: grid;
  gap: 20px;
}

.identity-evidence-meta {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 12px;
  margin: 0;
}

.identity-evidence-meta > div,
.identity-evidence-card {
  min-width: 0;
  padding: 14px;
  background: var(--el-fill-color-extra-light);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 10px;
}

.identity-evidence-meta dt {
  margin-bottom: 6px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.identity-evidence-meta dd {
  margin: 0;
  font-weight: 600;
  color: var(--el-text-color-primary);
  overflow-wrap: anywhere;
}

.identity-evidence-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 12px;
}

.identity-evidence-card {
  display: grid;
  gap: 10px;
  align-content: start;
  min-height: 240px;
}

.identity-evidence-card h3 {
  margin: 0;
  font-size: 14px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.identity-evidence-card :deep(.el-image) {
  width: 100%;
  height: 190px;
  background: var(--el-bg-color);
  border-radius: 8px;
}

.identity-evidence-missing {
  display: grid;
  place-items: center;
  min-height: 190px;
  margin: 0;
  color: var(--el-text-color-secondary);
  text-align: center;
}

@media (max-width: 760px) {
  .identity-evidence-meta,
  .identity-evidence-grid {
    grid-template-columns: 1fr;
  }
}
</style>
