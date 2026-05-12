<script setup lang="ts">
import type { FreshmanApplication } from '#/api/admin';

import { onMounted, reactive, ref } from 'vue';

import {
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

import AdminContentLayout from '../../shared/AdminContentLayout.vue';
import { formatAdminDateTime } from '../../shared/display';
import PersistentAdminTable from '../../shared/admin-table/PersistentAdminTable.vue';
import PersistentAdminTableColumn from '../../shared/admin-table/PersistentAdminTableColumn.vue';

type FreshmanReviewRow = FreshmanApplication & {
  failureCount?: number;
  materialURL?: string;
  qqID?: string;
};

const loading = ref(false);
const actionLoading = ref(false);
const items = ref<FreshmanReviewRow[]>([]);
const total = ref(0);
const extensionDays = ref(0);
const materialDialogVisible = ref(false);
const materialPreviewURL = ref('');
const query = reactive({
  page: 1,
  pageSize: 20,
  status: 'pending' as 'approved' | 'pending' | 'rejected',
});
const rejectionReasons = reactive<Record<string, string>>({});

async function fetchData() {
  loading.value = true;
  try {
    const data = await listFreshmanVerifications(query);
    items.value = data.items as FreshmanReviewRow[];
    total.value = data.total;
  } finally {
    loading.value = false;
  }
}

async function approve(row: FreshmanReviewRow, expiresInDays?: number) {
  if (actionLoading.value) return;
  actionLoading.value = true;
  try {
    await reviewFreshmanVerification(row.id, {
      action: 'approve',
      ...(expiresInDays ? { expiresInDays } : {}),
    });
    await fetchData();
  } finally {
    actionLoading.value = false;
  }
}

async function reject(row: FreshmanReviewRow) {
  const reason = rejectionReasons[row.id]?.trim();
  if (!reason) {
    ElMessage.error('请填写驳回原因');
    return;
  }
  await reviewFreshmanVerification(row.id, { action: 'reject', reason });
  await fetchData();
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
        @change="fetchData"
      >
        <ElOption label="待审核" value="pending" />
        <ElOption label="已通过" value="approved" />
        <ElOption label="已驳回" value="rejected" />
      </ElSelect>
      <ElButton type="primary" @click="fetchData">查询</ElButton>
    </template>

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
            <ElButton
              plain
              size="small"
              type="success"
              data-action="approve"
              :disabled="actionLoading"
              @click="approve(row)"
            >
              通过
            </ElButton>
            <ElInputNumber
              v-model="extensionDays"
              class="freshman-action-number"
              :min="0"
              size="small"
            />
            <ElButton
              plain
              size="small"
              type="success"
              data-action="approveWithDays"
              :disabled="actionLoading"
              @click="approve(row, extensionDays || undefined)"
            >
              带天数通过
            </ElButton>
            <ElInput
              v-model="rejectionReasons[row.id]"
              class="freshman-action-reason"
              placeholder="驳回原因"
              size="small"
            />
            <ElButton
              plain
              size="small"
              type="danger"
              data-action="reject"
              @click="reject(row)"
            >
              驳回
            </ElButton>
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
  align-items: center;
  gap: 8px;
  flex-wrap: nowrap;
}

.freshman-action-number {
  width: 96px;
}

.freshman-action-reason {
  width: 140px;
}
</style>
