<script setup lang="ts">
import type { SystemConfig } from '#/api/admin';

import { onMounted, reactive, ref } from 'vue';

import { useAccessStore } from '@vben/stores';

import {
  ElButton,
  ElDialog,
  ElForm,
  ElFormItem,
  ElInput,
  ElMessage,
  ElTable,
  ElTableColumn,
} from 'element-plus';

import { getSystemConfigList, updateSystemConfig } from '#/api/admin';
import { $t } from '#/locales';

const loading = ref(false);
const configs = ref<SystemConfig[]>([]);
const accessStore = useAccessStore();
const canUpdateSystemConfig = () =>
  accessStore.accessCodes.includes('user:system:update');

async function fetchData() {
  loading.value = true;
  try {
    const data = await getSystemConfigList();
    configs.value = data;
  } finally {
    loading.value = false;
  }
}

// ── 编辑弹窗 ──

const dialogVisible = ref(false);
const form = reactive({
  key: '',
  value: '',
  description: '',
});

function openEdit(row: SystemConfig) {
  if (!canUpdateSystemConfig()) {
    return;
  }
  form.key = row.key;
  form.value = row.value;
  form.description = row.description ?? '';
  dialogVisible.value = true;
}

async function handleSubmit() {
  if (!canUpdateSystemConfig()) {
    return;
  }

  await updateSystemConfig(form.key, { value: form.value });
  ElMessage.success($t('admin.users.systemConfig.updated'));
  dialogVisible.value = false;
  await fetchData();
}

onMounted(fetchData);
</script>

<template>
  <div class="p-4">
    <ElTable v-loading="loading" :data="configs" stripe>
      <ElTableColumn
        :label="$t('admin.users.systemConfig.key')"
        min-width="160"
        prop="key"
      />
      <ElTableColumn
        :label="$t('admin.users.systemConfig.value')"
        min-width="200"
        prop="value"
        show-overflow-tooltip
      />
      <ElTableColumn
        :label="$t('admin.common.description')"
        min-width="200"
        show-overflow-tooltip
      >
        <template #default="{ row }">
          {{ row.description || $t('admin.common.unavailable') }}
        </template>
      </ElTableColumn>
      <ElTableColumn
        :label="$t('admin.common.updatedAt')"
        prop="updatedAt"
        width="170"
      />
      <ElTableColumn
        v-if="canUpdateSystemConfig()"
        fixed="right"
        :label="$t('admin.common.actions')"
        width="90"
      >
        <template #default="{ row }">
          <ElButton link size="small" type="primary" @click="openEdit(row)">
            {{ $t('admin.common.edit') }}
          </ElButton>
        </template>
      </ElTableColumn>
    </ElTable>

    <!-- 编辑弹窗 -->
    <ElDialog
      v-model="dialogVisible"
      :title="$t('admin.users.systemConfig.dialogTitle', { key: form.key })"
      width="500px"
    >
      <ElForm label-width="80px">
        <ElFormItem :label="$t('admin.users.systemConfig.key')">
          <ElInput :model-value="form.key" disabled />
        </ElFormItem>
        <ElFormItem :label="$t('admin.common.description')">
          <ElInput :model-value="form.description" disabled />
        </ElFormItem>
        <ElFormItem :label="$t('admin.users.systemConfig.value')">
          <ElInput
            v-model="form.value"
            :rows="4"
            :placeholder="$t('admin.users.systemConfig.valuePlaceholder')"
            type="textarea"
          />
        </ElFormItem>
      </ElForm>
      <template #footer>
        <ElButton @click="dialogVisible = false">
          {{ $t('admin.common.cancel') }}
        </ElButton>
        <ElButton
          v-if="canUpdateSystemConfig()"
          type="primary"
          @click="handleSubmit"
        >
          {{ $t('admin.common.save') }}
        </ElButton>
      </template>
    </ElDialog>
  </div>
</template>
