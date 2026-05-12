<script setup lang="ts">
import type { IdentityVerification } from '#/api/admin';

import { onMounted, reactive, ref } from 'vue';

import {
  ElButton,
  ElOption,
  ElPagination,
  ElPopconfirm,
  ElSelect,
  ElTag,
} from 'element-plus';

import { getIdentityList, reviewIdentity } from '#/api/admin';
import { $t } from '#/locales';

import AdminContentLayout from '../../shared/AdminContentLayout.vue';
import { formatAdminDateTime } from '../../shared/display';
import PersistentAdminTable from '../../shared/admin-table/PersistentAdminTable.vue';
import PersistentAdminTableColumn from '../../shared/admin-table/PersistentAdminTableColumn.vue';

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
    </template>

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
                  plain
                  size="small"
                  type="danger"
                  :disabled="actionLoading"
                >
                  {{ $t('admin.users.identityReview.reject') }}
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
        :total="total"
        layout="total, prev, pager, next"
        @current-change="fetchData"
      />
    </template>
  </AdminContentLayout>
</template>
