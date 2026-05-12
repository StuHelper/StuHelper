<script setup lang="ts">
import type { Review } from '#/api/admin';

import { ElButton, ElPopconfirm } from 'element-plus';

import { $t } from '#/locales';

defineProps<{
  actionLoading: boolean;
  row: Review;
}>();

const emit = defineEmits<{
  (e: 'action', reviewId: string, action: 'delete' | 'hide' | 'restore'): void;
}>();
</script>

<template>
  <div class="admin-action-group">
    <ElPopconfirm
      v-if="row.status === 'pending_review'"
      :title="$t('admin.content.reviews.confirmApprove')"
      @confirm="emit('action', row.id, 'restore')"
    >
      <template #reference>
        <ElButton plain size="small" type="success" :disabled="actionLoading">
          {{ $t('admin.content.reviews.approve') }}
        </ElButton>
      </template>
    </ElPopconfirm>
    <ElPopconfirm
      v-if="row.status === 'published' || row.status === 'pending_review'"
      :title="$t('admin.content.reviews.confirmHide')"
      @confirm="emit('action', row.id, 'hide')"
    >
      <template #reference>
        <ElButton plain size="small" type="warning" :disabled="actionLoading">
          {{ $t('admin.content.reviews.hide') }}
        </ElButton>
      </template>
    </ElPopconfirm>
    <ElButton
      v-if="row.status === 'hidden'"
      plain
      size="small"
      type="success"
      :disabled="actionLoading"
      @click="emit('action', row.id, 'restore')"
    >
      {{ $t('admin.content.reviews.restore') }}
    </ElButton>
    <ElPopconfirm
      v-if="row.status !== 'deleted'"
      :title="$t('admin.content.reviews.confirmDelete')"
      @confirm="emit('action', row.id, 'delete')"
    >
      <template #reference>
        <ElButton plain size="small" type="danger" :disabled="actionLoading">
          {{ $t('admin.common.delete') }}
        </ElButton>
      </template>
    </ElPopconfirm>
  </div>
</template>
