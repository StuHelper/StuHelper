<script setup lang="ts">
import type { StudentVerification } from '#/api/admin';

import { onMounted, reactive, ref } from 'vue';

import {
  ElButton,
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

const loading = ref(false);
const actionLoading = ref(false);
const items = ref<StudentVerification[]>([]);
const total = ref(0);
const authStore = useAuthStore();
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
  if (!normalizeScopedSchoolId()) {
    return;
  }
  loading.value = true;
  try {
    const data = await getStudentVerificationList(query);
    items.value = data.items;
    total.value = data.total;
  } catch (_error) {
    void _error;
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
  } catch (_error) {
    void _error;
    // 失败提示已由 unwrapData 统一处理。
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
        class="admin-toolbar-control admin-toolbar-control--wide"
        clearable
        :placeholder="$t('admin.users.studentVerification.schoolIdPlaceholder')"
        @clear="fetchData"
        @keyup.enter="fetchData"
      />
      <ElButton type="primary" @click="fetchData">
        {{ $t('admin.common.query') }}
      </ElButton>
    </template>

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
                  plain
                  size="small"
                  type="danger"
                  :disabled="actionLoading"
                >
                  {{ $t('admin.users.studentVerification.reject') }}
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
