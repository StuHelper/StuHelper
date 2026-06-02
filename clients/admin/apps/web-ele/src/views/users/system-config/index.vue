<script setup lang="ts">
import type { SystemConfig } from '#/api/admin';

import { computed, onMounted, reactive, ref } from 'vue';

import { useAccessStore } from '@vben/stores';

import {
  ElButton,
  ElDialog,
  ElForm,
  ElFormItem,
  ElInput,
  ElInputNumber,
  ElMessage,
  ElOption,
  ElSelect,
  ElSwitch,
  ElTable,
  ElTableColumn,
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

interface EmailPolicyProvider {
  name: string;
  enabled: boolean;
  priority: number;
  weight: number;
}

interface EmailDeliveryPolicy {
  mode: 'priority' | 'weighted';
  maxAttempts: number;
  providers: EmailPolicyProvider[];
}

const emailPolicyKey = 'email.delivery_policy';
const emailPolicyConfig = computed(() =>
  configs.value.find((item) => item.key === emailPolicyKey),
);
const emailPolicyForm = reactive<EmailDeliveryPolicy>({
  mode: 'priority',
  maxAttempts: 2,
  providers: [],
});

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
  if (row.key === emailPolicyKey) {
    hydrateEmailPolicyForm(row.value);
  }
  dialogVisible.value = true;
}

function hydrateEmailPolicyForm(raw: string) {
  let parsed: Partial<EmailDeliveryPolicy> | undefined;
  try {
    parsed = JSON.parse(raw) as Partial<EmailDeliveryPolicy>;
  } catch (_error) {
    parsed = undefined;
  }
  emailPolicyForm.mode = parsed?.mode === 'weighted' ? 'weighted' : 'priority';
  emailPolicyForm.maxAttempts =
    typeof parsed?.maxAttempts === 'number' && parsed.maxAttempts > 0
      ? parsed.maxAttempts
      : 2;
  const providers = Array.isArray(parsed?.providers) ? parsed.providers : [];
  emailPolicyForm.providers = providers
    .map((item) => ({
      name: String((item as Partial<EmailPolicyProvider>).name ?? '').trim(),
      enabled: Boolean((item as Partial<EmailPolicyProvider>).enabled),
      priority: Number((item as Partial<EmailPolicyProvider>).priority ?? 100),
      weight: Number((item as Partial<EmailPolicyProvider>).weight ?? 1),
    }))
    .filter((item) => item.name.length > 0);
}

function syncEmailPolicyValue() {
  form.value = JSON.stringify({
    mode: emailPolicyForm.mode,
    maxAttempts: emailPolicyForm.maxAttempts,
    providers: emailPolicyForm.providers.map((provider) => ({
      name: provider.name,
      enabled: provider.enabled,
      priority: provider.priority,
      weight: provider.weight,
    })),
  });
}

async function handleSubmit() {
  if (!canUpdateSystemConfig() || submitting.value) {
    return;
  }

  submitting.value = true;
  try {
    if (form.key === emailPolicyKey) {
      syncEmailPolicyValue();
    }
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
    <template v-if="emailPolicyConfig" #toolbar>
      <div class="admin-system-config-toolbar">
        <span class="admin-cell-muted">
          {{ $t('admin.users.systemConfig.emailPolicySummary') }}
        </span>
        <ElButton
          v-if="canUpdateSystemConfig()"
          data-email-policy-edit-button
          plain
          size="small"
          type="primary"
          @click="emailPolicyConfig && openEdit(emailPolicyConfig)"
        >
          {{ $t('admin.users.systemConfig.editEmailPolicy') }}
        </ElButton>
      </div>
    </template>

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
        <template v-if="form.key === emailPolicyKey">
          <ElFormItem :label="$t('admin.users.systemConfig.emailPolicyMode')">
            <ElSelect
              v-model="emailPolicyForm.mode"
              data-email-policy-mode-select
            >
              <ElOption
                :label="$t('admin.users.systemConfig.emailPolicyPriority')"
                value="priority"
              />
              <ElOption
                :label="$t('admin.users.systemConfig.emailPolicyWeighted')"
                value="weighted"
              />
            </ElSelect>
          </ElFormItem>
          <ElFormItem :label="$t('admin.users.systemConfig.emailMaxAttempts')">
            <ElInputNumber
              v-model="emailPolicyForm.maxAttempts"
              data-email-policy-max-attempts
              :min="1"
              :max="10"
            />
          </ElFormItem>
          <ElTable :data="emailPolicyForm.providers" border>
            <ElTableColumn
              prop="name"
              :label="$t('admin.users.systemConfig.emailProvider')"
              min-width="140"
            />
            <ElTableColumn
              :label="$t('admin.users.systemConfig.emailProviderEnabled')"
              width="96"
            >
              <template #default="{ row }">
                <ElSwitch
                  v-model="row.enabled"
                  :data-email-policy-provider-enabled="row.name"
                />
              </template>
            </ElTableColumn>
            <ElTableColumn
              :label="$t('admin.users.systemConfig.emailProviderPriority')"
              width="140"
            >
              <template #default="{ row }">
                <ElInputNumber
                  v-model="row.priority"
                  :data-email-policy-provider-priority="row.name"
                  :min="0"
                  :max="999"
                />
              </template>
            </ElTableColumn>
            <ElTableColumn
              :label="$t('admin.users.systemConfig.emailProviderWeight')"
              width="140"
            >
              <template #default="{ row }">
                <ElInputNumber
                  v-model="row.weight"
                  :data-email-policy-provider-weight="row.name"
                  :min="1"
                  :max="1000"
                />
              </template>
            </ElTableColumn>
          </ElTable>
        </template>
        <ElFormItem v-else :label="$t('admin.users.systemConfig.value')">
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

<style scoped>
.admin-system-config-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  width: 100%;
}
</style>
