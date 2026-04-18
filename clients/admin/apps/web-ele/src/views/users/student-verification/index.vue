<script setup lang="ts">
import type { StudentVerification } from '#/api/admin';

import { onMounted, reactive, ref } from 'vue';

import {
  ElButton,
  ElInput,
  ElOption,
  ElPagination,
  ElPopconfirm,
  ElSelect,
  ElTable,
  ElTableColumn,
  ElTag,
} from 'element-plus';

import {
  getStudentVerificationList,
  reviewStudentVerification,
} from '#/api/admin';
import { $t } from '#/locales';

const loading = ref(false);
const actionLoading = ref(false);
const items = ref<StudentVerification[]>([]);
const total = ref(0);
const query = reactive({
  page: 1,
  pageSize: 20,
  status: 'all' as 'all' | 'pending' | 'rejected' | 'verified',
  schoolId: '',
});

async function fetchData() {
  loading.value = true;
  try {
    const data = await getStudentVerificationList(query);
    items.value = data.items;
    total.value = data.total;
  } catch (_error) { void _error;
    items.value = [];
    total.value = 0;
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
    await reviewStudentVerification(userId, { approved });
    await fetchData();
  } catch (_error) { void _error;
    // unwrapData already displays a toast for failed mutations.
  } finally {
    actionLoading.value = false;
  }
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

onMounted(fetchData);
</script>

<template>
  <div class="p-4">
    <div class="mb-4 flex items-center gap-3">
      <ElSelect
        v-model="query.status"
        :placeholder="$t('admin.users.studentVerification.statusPlaceholder')"
        style="width: 140px"
        @change="fetchData"
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
        clearable
        :placeholder="$t('admin.users.studentVerification.schoolIdPlaceholder')"
        style="width: 180px"
        @clear="fetchData"
        @keyup.enter="fetchData"
      />
      <ElButton type="primary" @click="fetchData">
        {{ $t('admin.common.query') }}
      </ElButton>
    </div>

    <ElTable v-loading="loading" :data="items" stripe>
      <ElTableColumn
        :label="$t('admin.users.studentVerification.userId')"
        prop="userID"
        width="80"
      />
      <ElTableColumn
        :label="$t('admin.users.studentVerification.schoolId')"
        min-width="140"
        prop="schoolID"
      />
      <ElTableColumn
        :label="$t('admin.users.studentVerification.activeStudentId')"
        prop="activeStudentID"
        width="140"
      />
      <ElTableColumn
        :label="$t('admin.users.studentVerification.statusLabel')"
        width="100"
      >
        <template #default="{ row }">
          <ElTag :type="statusTag(row.verificationStatus)" size="small">
            {{ statusLabel(row.verificationStatus) }}
          </ElTag>
        </template>
      </ElTableColumn>
      <ElTableColumn
        :label="$t('admin.users.studentVerification.methodLabel')"
        width="100"
      >
        <template #default="{ row }">
          {{ verificationMethodLabel(row.verificationMethod) }}
        </template>
      </ElTableColumn>
      <ElTableColumn
        :label="$t('admin.users.studentVerification.createdAt')"
        prop="createdAt"
        width="170"
      />
      <ElTableColumn
        fixed="right"
        :label="$t('admin.common.actions')"
        width="140"
      >
        <template #default="{ row }">
          <template v-if="row.verificationStatus === 'pending'">
            <ElPopconfirm
              :title="$t('admin.users.studentVerification.confirmApprove')"
              @confirm="handleReview(row.userID, true)"
            >
              <template #reference>
                <ElButton
                  link
                  size="small"
                  type="success"
                  :disabled="actionLoading"
                >
                  {{ $t('admin.users.studentVerification.approve') }}
                </ElButton>
              </template>
            </ElPopconfirm>
            <ElPopconfirm
              :title="$t('admin.users.studentVerification.confirmReject')"
              @confirm="handleReview(row.userID, false)"
            >
              <template #reference>
                <ElButton
                  link
                  size="small"
                  type="danger"
                  :disabled="actionLoading"
                >
                  {{ $t('admin.users.studentVerification.reject') }}
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
