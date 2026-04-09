<script setup lang="ts">
import type { Review } from '#/api/admin';

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

import { getReviewList, updateReview } from '#/api/admin';
import { $t } from '#/locales';

const loading = ref(false);
const actionLoading = ref(false);
const reviews = ref<Review[]>([]);
const total = ref(0);
const query = reactive({
  page: 1,
  pageSize: 20,
  status: 'all' as 'all' | 'deleted' | 'hidden' | 'published',
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
  } catch {
    // unwrapData already displays a toast for failed mutations.
  } finally {
    actionLoading.value = false;
  }
}

const statusTag = (status: string): 'danger' | 'success' | 'warning' => {
  const map: Record<string, 'danger' | 'success' | 'warning'> = {
    published: 'success',
    hidden: 'warning',
    deleted: 'danger',
  };
  return map[status] || 'warning';
};

onMounted(fetchData);
</script>

<template>
  <div class="p-4">
    <div class="mb-4 flex items-center gap-3">
      <ElSelect
        v-model="query.status"
        :placeholder="$t('admin.common.status')"
        style="width: 120px"
        @change="fetchData"
      >
        <ElOption :label="$t('admin.common.all')" value="all" />
        <ElOption :label="$t('admin.content.reviews.status.published')" value="published" />
        <ElOption :label="$t('admin.content.reviews.status.hidden')" value="hidden" />
        <ElOption :label="$t('admin.content.reviews.status.deleted')" value="deleted" />
      </ElSelect>
      <ElButton type="primary" @click="fetchData">{{ $t('admin.common.query') }}</ElButton>
    </div>

    <ElTable v-loading="loading" :data="reviews" stripe>
      <ElTableColumn :label="$t('admin.common.id')" prop="id" width="70" />
      <ElTableColumn :label="$t('admin.content.reviews.course')" min-width="120" prop="courseName" />
      <ElTableColumn :label="$t('admin.content.reviews.teacher')" min-width="120">
        <template #default="{ row }">
          {{ row.teacherName || $t('admin.content.reviews.teacherUnlinked') }}
        </template>
      </ElTableColumn>
      <ElTableColumn :label="$t('admin.common.title')" min-width="140" prop="title" />
      <ElTableColumn
        :label="$t('admin.common.content')"
        min-width="200"
        prop="content"
        show-overflow-tooltip
      />
      <ElTableColumn :label="$t('admin.content.reviews.rating')" width="70">
        <template #default="{ row }">
          {{ averageRating(row.ratings) }}
        </template>
      </ElTableColumn>
      <ElTableColumn :label="$t('admin.common.status')" width="90">
        <template #default="{ row }">
          <ElTag :type="statusTag(row.status)" size="small">
            {{
              row.status === 'published'
                ? $t('admin.content.reviews.status.published')
                : row.status === 'hidden'
                  ? $t('admin.content.reviews.status.hidden')
                  : $t('admin.content.reviews.status.deleted')
            }}
          </ElTag>
        </template>
      </ElTableColumn>
      <ElTableColumn :label="$t('admin.common.createdAt')" prop="createdAt" width="170" />
      <ElTableColumn fixed="right" :label="$t('admin.common.actions')" width="180">
        <template #default="{ row }">
          <template v-if="row.status === 'published'">
            <ElPopconfirm
              :title="$t('admin.content.reviews.confirmHide')"
              @confirm="handleAction(row.id, 'hide')"
            >
              <template #reference>
                <ElButton link size="small" type="warning" :disabled="actionLoading">
                  {{ $t('admin.content.reviews.hide') }}
                </ElButton>
              </template>
            </ElPopconfirm>
          </template>
          <template v-if="row.status === 'hidden'">
            <ElButton
              link
              size="small"
              type="success"
              :disabled="actionLoading"
              @click="handleAction(row.id, 'restore')"
            >
              {{ $t('admin.content.reviews.restore') }}
            </ElButton>
          </template>
          <ElPopconfirm
            v-if="row.status !== 'deleted'"
            :title="$t('admin.content.reviews.confirmDelete')"
            @confirm="handleAction(row.id, 'delete')"
          >
            <template #reference>
              <ElButton link size="small" type="danger" :disabled="actionLoading">
                {{ $t('admin.common.delete') }}
              </ElButton>
            </template>
          </ElPopconfirm>
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
