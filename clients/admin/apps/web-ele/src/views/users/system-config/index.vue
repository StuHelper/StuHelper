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
} from 'element-plus';

import { getSystemConfigList, updateSystemConfig } from '#/api/admin';
import { $t } from '#/locales';

import PersistentAdminTable from '../../shared/admin-table/PersistentAdminTable.vue';
import PersistentAdminTableColumn from '../../shared/admin-table/PersistentAdminTableColumn.vue';
import AdminContentLayout from '../../shared/AdminContentLayout.vue';
import { formatAdminDateTime } from '../../shared/display';

const loading = ref(false);
const submitting = ref(false);
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
  if (!canUpdateSystemConfig() || submitting.value) {
    return;
  }

  submitting.value = true;
  try {
    await updateSystemConfig(form.key, { value: form.value });
    ElMessage.success($t('admin.users.systemConfig.updated'));
    dialogVisible.value = false;
    await fetchData();
  } catch (_error) {
    void _error;
    // shared-result already displays the backend error message.
  } finally {
    submitting.value = false;
  }
}

onMounted(fetchData);
</script>

<template>
  <AdminContentLayout
    :title="$t('admin.routes.userSystem.systemConfig')"
    :total="configs.length"
  >
    <PersistentAdminTable
      table-key="users.systemConfig"
      :loading="loading"
      :data="configs"
      row-key="key"
      stripe
    >
      <PersistentAdminTableColumn
        column-key="key"
        :label="$t('admin.users.systemConfig.key')"
        :default-min-width="220"
        prop="key"
        show-overflow-tooltip
      />
      <PersistentAdminTableColumn
        column-key="value"
        :label="$t('admin.users.systemConfig.value')"
        :default-min-width="220"
        prop="value"
        show-overflow-tooltip
      />
      <PersistentAdminTableColumn
        column-key="description"
        :label="$t('admin.common.description')"
        :default-min-width="240"
        show-overflow-tooltip
      >
        <template #default="{ row }">
          {{ row.description || $t('admin.common.unavailable') }}
        </template>
      </PersistentAdminTableColumn>
      <PersistentAdminTableColumn
        column-key="updatedAt"
        :label="$t('admin.common.updatedAt')"
        :default-width="148"
      >
        <template #default="{ row }">
          <span class="admin-cell-muted">
            {{ formatAdminDateTime(row.updatedAt) }}
          </span>
        </template>
      </PersistentAdminTableColumn>
      <PersistentAdminTableColumn
        v-if="canUpdateSystemConfig()"
        column-key="actions"
        fixed="right"
        :label="$t('admin.common.actions')"
        :default-width="100"
      >
        <template #default="{ row }">
          <ElButton plain size="small" type="primary" @click="openEdit(row)">
            {{ $t('admin.common.edit') }}
          </ElButton>
        </template>
      </PersistentAdminTableColumn>
    </PersistentAdminTable>

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
          :loading="submitting"
          type="primary"
          @click="handleSubmit"
        >
          {{ $t('admin.common.save') }}
        </ElButton>
      </template>
    </ElDialog>
  </AdminContentLayout>
</template>
