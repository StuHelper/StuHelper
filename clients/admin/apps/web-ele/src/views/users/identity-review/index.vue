<script setup lang="ts">
import type { IdentityVerification } from '#/api/admin';

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

import { getIdentityList, reviewIdentity } from '#/api/admin';
import { $t } from '#/locales';

const loading = ref(false);
const actionLoading = ref(false);
const items = ref<IdentityVerification[]>([]);
const total = ref(0);
const query = reactive({
  page: 1,
  pageSize: 20,
  status: 'all' as 'all' | 'pending' | 'rejected' | 'verified',
});

async function fetchData() {
  loading.value = true;
  try {
    const data = await getIdentityList(query);
    items.value = data.items;
    total.value = data.total;
  } finally {
    loading.value = false;
  }
}

async function handleReview(userId: number, approved: boolean) {
  if (actionLoading.value) {
    return;
  }

  actionLoading.value = true;
  try {
    await reviewIdentity(userId, { approved });
    await fetchData();
  } catch (_error) {
    void _error;
    // 失败提示已由 unwrapData 统一处理。
  } finally {
    actionLoading.value = false;
  }
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

onMounted(fetchData);
</script>

<template>
  <div class="p-4">
    <div class="mb-4 flex items-center gap-3">
      <ElSelect
        v-model="query.status"
        :placeholder="$t('admin.users.identityReview.statusPlaceholder')"
        style="width: 140px"
        @change="fetchData"
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
      <ElButton type="primary" @click="fetchData">
        {{ $t('admin.common.query') }}
      </ElButton>
    </div>

    <ElTable v-loading="loading" :data="items" stripe>
      <ElTableColumn
        :label="$t('admin.users.identityReview.userId')"
        prop="userID"
        width="80"
      />
      <ElTableColumn
        :label="$t('admin.users.identityReview.realName')"
        prop="realName"
        width="120"
      />
      <ElTableColumn
        :label="$t('admin.users.identityReview.docTypeLabel')"
        width="180"
      >
        <template #default="{ row }">
          {{ docTypeLabel(row.docType) }}
        </template>
      </ElTableColumn>
      <ElTableColumn
        :label="$t('admin.users.identityReview.verifyMethodLabel')"
        width="120"
      >
        <template #default="{ row }">
          {{ verifyMethodLabel(row.verifyMethod) }}
        </template>
      </ElTableColumn>
      <ElTableColumn
        :label="$t('admin.users.identityReview.statusLabel')"
        width="100"
      >
        <template #default="{ row }">
          <ElTag :type="statusTag(row)" size="small">
            {{ statusLabel(row) }}
          </ElTag>
        </template>
      </ElTableColumn>
      <ElTableColumn
        :label="$t('admin.users.identityReview.finishedAt')"
        width="170"
      >
        <template #default="{ row }">
          {{
            row.verifiedAt || row.reviewedAt || $t('admin.common.unavailable')
          }}
        </template>
      </ElTableColumn>
      <ElTableColumn
        fixed="right"
        :label="$t('admin.common.actions')"
        width="140"
      >
        <template #default="{ row }">
          <template v-if="!row.verified">
            <ElPopconfirm
              :title="$t('admin.users.identityReview.confirmApprove')"
              @confirm="handleReview(row.userID, true)"
            >
              <template #reference>
                <ElButton
                  link
                  size="small"
                  type="success"
                  :disabled="actionLoading"
                >
                  {{ $t('admin.users.identityReview.approve') }}
                </ElButton>
              </template>
            </ElPopconfirm>
            <ElPopconfirm
              :title="$t('admin.users.identityReview.confirmReject')"
              @confirm="handleReview(row.userID, false)"
            >
              <template #reference>
                <ElButton
                  link
                  size="small"
                  type="danger"
                  :disabled="actionLoading"
                >
                  {{ $t('admin.users.identityReview.reject') }}
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
