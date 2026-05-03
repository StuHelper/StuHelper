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
  ElPagination,
  ElSelect,
  ElOption,
  ElTable,
  ElTableColumn,
  ElTag,
} from 'element-plus';

import {
  listFreshmanVerifications,
  reviewFreshmanVerification,
} from '#/api/admin';

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

onMounted(fetchData);
</script>

<template>
  <div class="p-4">
    <div class="mb-4 flex items-center gap-3">
      <ElSelect v-model="query.status" style="width: 140px" @change="fetchData">
        <ElOption label="待审核" value="pending" />
        <ElOption label="已通过" value="approved" />
        <ElOption label="已驳回" value="rejected" />
      </ElSelect>
      <ElButton type="primary" @click="fetchData">查询</ElButton>
    </div>

    <ElTable v-loading="loading" :data="items" stripe>
      <ElTableColumn data-field="status" label="状态" width="100">
        <template #default="{ row }">
          <ElTag :type="statusType(row.status)" data-state="pendingReview">
            {{ row.status }}
          </ElTag>
        </template>
      </ElTableColumn>
      <ElTableColumn data-field="schoolID" label="学校" prop="schoolID" />
      <ElTableColumn data-field="qqID" label="QQ">
        <template #default="{ row }">{{ row.qqID || '—' }}</template>
      </ElTableColumn>
      <ElTableColumn label="申请人" prop="applicantNameMasked" />
      <ElTableColumn data-field="createdAt" label="申请时间" prop="createdAt" />
      <ElTableColumn data-field="failureCount" label="失败次数">
        <template #default="{ row }">{{ row.failureCount ?? '—' }}</template>
      </ElTableColumn>
      <ElTableColumn fixed="right" label="操作" width="320">
        <template #default="{ row }">
          <ElButton
            link
            type="primary"
            data-material-preview
            @click="openMaterial(row)"
          >
            材料预览
          </ElButton>
          <ElButton
            link
            type="success"
            data-action="approve"
            :disabled="actionLoading"
            @click="approve(row)"
          >
            通过
          </ElButton>
          <ElInputNumber v-model="extensionDays" :min="0" size="small" />
          <ElButton
            link
            type="success"
            data-action="approveWithDays"
            :disabled="actionLoading"
            @click="approve(row, extensionDays || undefined)"
          >
            带天数通过
          </ElButton>
          <ElInput
            v-model="rejectionReasons[row.id]"
            placeholder="驳回原因"
            size="small"
          />
          <ElButton link type="danger" data-action="reject" @click="reject(row)">
            驳回
          </ElButton>
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

    <ElDialog v-model="materialDialogVisible" title="材料预览" width="720px">
      <ElImage :src="materialPreviewURL" fit="contain" />
    </ElDialog>
  </div>
</template>
