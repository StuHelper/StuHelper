<script setup lang="ts">
import type { Review } from '#/api/admin';

import { onMounted, reactive, ref } from 'vue';

import {
  ElButton,
  ElOption,
  ElPagination,
  ElSelect,
  ElTag,
} from 'element-plus';

import { getReviewList, updateReview } from '#/api/admin';
import { $t } from '#/locales';

import AdminContentLayout from '../../shared/AdminContentLayout.vue';
import PersistentAdminTable from '../../shared/admin-table/PersistentAdminTable.vue';
import PersistentAdminTableColumn from '../../shared/admin-table/PersistentAdminTableColumn.vue';
import {
  compactID,
  formatAdminDateTime,
  formatNullableText,
} from '../../shared/display';
import ReviewActions from './ReviewActions.vue';

const loading = ref(false);
const actionLoading = ref(false);
const reviews = ref<Review[]>([]);
const total = ref(0);
const query = reactive({
  page: 1,
  pageSize: 20,
  status: 'all' as
    | 'all'
    | 'deleted'
    | 'hidden'
    | 'pending_review'
    | 'published',
});

function averageRating(ratings: Record<string, number>) {
  const values = Object.values(ratings);
  if (values.length === 0) return '—';
  const totalScore = values.reduce((sum, value) => sum + value, 0);
  return (totalScore / values.length).toFixed(1);
}

async function fetchData() {
  loading.value = true;
  try {
    const data = await getReviewList(query);
    reviews.value = data.items;
    total.value = data.total;
  } finally {
    loading.value = false;
  }
}

async function handleAction(
  reviewId: string,
  action: 'delete' | 'hide' | 'restore',
) {
  if (actionLoading.value) {
    return;
  }

  actionLoading.value = true;
  try {
    await updateReview(reviewId, { action });
    await fetchData();
  } catch (_error) {
    void _error;
    // 失败提示已由 unwrapData 统一处理。
  } finally {
    actionLoading.value = false;
  }
}

const statusTag = (
  status: string,
): 'danger' | 'info' | 'success' | 'warning' => {
  const map: Record<string, 'danger' | 'info' | 'success' | 'warning'> = {
    published: 'success',
    pending_review: 'warning',
    hidden: 'info',
    deleted: 'danger',
  };
  return map[status] || 'warning';
};

const statusLabel = (status: string) => {
  const map: Record<string, string> = {
    published: $t('admin.content.reviews.status.published'),
    pending_review: $t('admin.content.reviews.status.pendingReview'),
    hidden: $t('admin.content.reviews.status.hidden'),
    deleted: $t('admin.content.reviews.status.deleted'),
  };
  return map[status] || status;
};

onMounted(fetchData);
</script>

<template>
  <AdminContentLayout
    :title="$t('admin.routes.content.reviews')"
    :total="total"
  >
    <template #toolbar>
      <ElSelect
        v-model="query.status"
        class="admin-toolbar-control"
        :placeholder="$t('admin.common.status')"
        :teleported="false"
        @change="fetchData"
      >
        <ElOption :label="$t('admin.common.all')" value="all" />
        <ElOption
          :label="$t('admin.content.reviews.status.published')"
          value="published"
        />
        <ElOption
          :label="$t('admin.content.reviews.status.pendingReview')"
          value="pending_review"
        />
        <ElOption
          :label="$t('admin.content.reviews.status.hidden')"
          value="hidden"
        />
        <ElOption
          :label="$t('admin.content.reviews.status.deleted')"
          value="deleted"
        />
      </ElSelect>
      <ElButton type="primary" @click="fetchData">
        {{ $t('admin.common.query') }}
      </ElButton>
    </template>

    <PersistentAdminTable
      table-key="content.reviews"
      :loading="loading"
      :data="reviews"
      row-key="id"
      stripe
    >
      <PersistentAdminTableColumn
        column-key="id"
        :label="$t('admin.common.id')"
        :default-width="148"
      >
        <template #default="{ row }">
          <span class="admin-id-token" :title="row.id">
            {{ compactID(row.id) }}
          </span>
        </template>
      </PersistentAdminTableColumn>
      <PersistentAdminTableColumn
        column-key="courseName"
        :label="$t('admin.content.reviews.course')"
        :default-min-width="140"
        prop="courseName"
        show-overflow-tooltip
      />
      <PersistentAdminTableColumn
        column-key="teacherName"
        :label="$t('admin.content.reviews.teacher')"
        :default-min-width="120"
        show-overflow-tooltip
      >
        <template #default="{ row }">
          {{ row.teacherName || $t('admin.content.reviews.teacherUnlinked') }}
        </template>
      </PersistentAdminTableColumn>
      <PersistentAdminTableColumn
        column-key="title"
        :label="$t('admin.common.title')"
        :default-min-width="180"
        show-overflow-tooltip
      >
        <template #default="{ row }">
          <div class="admin-cell-title" :title="row.title">
            {{ formatNullableText(row.title) }}
          </div>
        </template>
      </PersistentAdminTableColumn>
      <PersistentAdminTableColumn
        column-key="content"
        :label="$t('admin.common.content')"
        :default-min-width="260"
        show-overflow-tooltip
      >
        <template #default="{ row }">
          <div class="admin-cell-body" :title="row.content">
            {{ formatNullableText(row.content) }}
          </div>
        </template>
      </PersistentAdminTableColumn>
      <PersistentAdminTableColumn
        align="center"
        column-key="rating"
        :label="$t('admin.content.reviews.rating')"
        :default-width="82"
      >
        <template #default="{ row }">
          <span class="admin-number">{{ averageRating(row.ratings) }}</span>
        </template>
      </PersistentAdminTableColumn>
      <PersistentAdminTableColumn
        column-key="status"
        :label="$t('admin.common.status')"
        :default-width="104"
      >
        <template #default="{ row }">
          <ElTag :type="statusTag(row.status)" size="small">
            {{ statusLabel(row.status) }}
          </ElTag>
        </template>
      </PersistentAdminTableColumn>
      <PersistentAdminTableColumn
        column-key="createdAt"
        :label="$t('admin.common.createdAt')"
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
        :default-width="190"
      >
        <template #default="{ row }">
          <ReviewActions
            :row="row"
            :action-loading="actionLoading"
            @action="handleAction"
          />
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
