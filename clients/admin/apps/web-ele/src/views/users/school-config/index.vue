<script setup lang="ts">
import type { SchoolConfig, UpdateSchoolConfigPayload } from '#/api/admin';

import { computed, onMounted, reactive, ref } from 'vue';

import { useAccessStore } from '@vben/stores';

import { ElButton, ElMessage, ElSwitch, ElTag } from 'element-plus';

import { getSchoolConfigList, updateSchoolConfig } from '#/api/admin';
import { $t } from '#/locales';

import PersistentAdminTable from '../../shared/admin-table/PersistentAdminTable.vue';
import PersistentAdminTableColumn from '../../shared/admin-table/PersistentAdminTableColumn.vue';
import AdminContentLayout from '../../shared/AdminContentLayout.vue';
import SchoolConfigDialog from './SchoolConfigDialog.vue';

const loading = ref(false);
const submitting = ref(false);
const schools = ref<SchoolConfig[]>([]);
const updatingSchoolIds = ref(new Set<number>());
const accessStore = useAccessStore();
const canUpdateSchoolConfig = () =>
  accessStore.accessCodes.includes('user:school:update');
const enabledSchoolCount = computed(
  () => schools.value.filter((school) => school.enabled).length,
);

async function fetchData() {
  loading.value = true;
  try {
    const data = await getSchoolConfigList();
    schools.value = data;
  } finally {
    loading.value = false;
  }
}

const dialogVisible = ref(false);
const form = reactive({
  academicDbTable: '',
  consentText: '',
  enabled: true,
  ldapBaseDN: '',
  ldapURL: '',
  schoolID: 0,
  schoolCode: '',
  schoolName: '',
  systemBindDN: '',
  systemBindPassword: '',
  useTLS: true,
  verificationMethod: 'manual' as 'ldap' | 'manual',
});

type SchoolConfigForm = typeof form;

function openEdit(row: SchoolConfig) {
  if (!canUpdateSchoolConfig()) {
    return;
  }
  form.schoolID = row.schoolID;
  form.schoolCode = row.schoolCode;
  form.schoolName = row.schoolName;
  form.verificationMethod = row.verificationMethod;
  form.enabled = row.enabled;
  form.academicDbTable = row.academicDbTable ?? '';
  form.consentText = row.consentText ?? '';
  form.ldapURL = row.ldapConfig?.url ?? '';
  form.ldapBaseDN = row.ldapConfig?.baseDN ?? '';
  form.systemBindDN = row.ldapConfig?.systemBindDN ?? '';
  form.systemBindPassword = '';
  form.useTLS = row.ldapConfig?.useTLS ?? true;
  dialogVisible.value = true;
}

async function handleSubmit(submitted: SchoolConfigForm) {
  if (!canUpdateSchoolConfig() || submitting.value) {
    return;
  }

  const payload: UpdateSchoolConfigPayload = {
    academicDbTable: submitted.academicDbTable || undefined,
    consentText: submitted.consentText || undefined,
    enabled: submitted.enabled,
    ldapConfig:
      submitted.verificationMethod === 'ldap'
        ? {
            baseDN: submitted.ldapBaseDN || undefined,
            systemBindDN: submitted.systemBindDN || undefined,
            systemBindPassword: submitted.systemBindPassword || undefined,
            url: submitted.ldapURL || undefined,
            useTLS: submitted.useTLS,
          }
        : undefined,
    schoolName: submitted.schoolName,
    verificationMethod: submitted.verificationMethod,
  };

  submitting.value = true;
  try {
    await updateSchoolConfig(submitted.schoolID, payload);
    ElMessage.success($t('admin.users.schoolConfig.updated'));
    dialogVisible.value = false;
    await fetchData();
  } catch (_error) {
    void _error;
    // shared-result already displays the backend error message.
  } finally {
    submitting.value = false;
  }
}

async function handleToggleEnabled(row: SchoolConfig, enabled: boolean) {
  if (!canUpdateSchoolConfig() || isSchoolUpdating(row.schoolID)) {
    return;
  }
  markSchoolUpdating(row.schoolID, true);
  try {
    await updateSchoolConfig(row.schoolID, { enabled });
    row.enabled = enabled;
    ElMessage.success($t('admin.users.schoolConfig.updated'));
  } catch (_error) {
    void _error;
    // shared-result already displays the backend error message.
  } finally {
    markSchoolUpdating(row.schoolID, false);
  }
}

function isSchoolUpdating(schoolID: number) {
  return updatingSchoolIds.value.has(schoolID);
}

function markSchoolUpdating(schoolID: number, updating: boolean) {
  const next = new Set(updatingSchoolIds.value);
  if (updating) {
    next.add(schoolID);
  } else {
    next.delete(schoolID);
  }
  updatingSchoolIds.value = next;
}

onMounted(fetchData);
</script>

<template>
  <AdminContentLayout
    :title="$t('admin.routes.userSystem.schoolConfig')"
    :total="schools.length"
  >
    <template #toolbar>
      <span class="admin-cell-muted" data-school-directory-summary>
        {{
          $t('admin.users.schoolConfig.enabledSummary', {
            enabled: enabledSchoolCount,
            total: schools.length,
          })
        }}
      </span>
    </template>

    <PersistentAdminTable
      table-key="users.schoolConfig"
      :loading="loading"
      :data="schools"
      row-key="schoolCode"
      stripe
    >
      <PersistentAdminTableColumn
        column-key="schoolCode"
        :label="$t('admin.users.schoolConfig.schoolCode')"
        prop="schoolCode"
        :default-width="148"
      />
      <PersistentAdminTableColumn
        column-key="schoolName"
        :label="$t('admin.users.schoolConfig.schoolName')"
        :default-min-width="200"
        prop="schoolName"
        show-overflow-tooltip
      />
      <PersistentAdminTableColumn
        column-key="verificationMethod"
        :label="$t('admin.users.schoolConfig.verificationMethod')"
        :default-width="128"
      >
        <template #default="{ row }">
          <ElTag
            :type="row.verificationMethod === 'ldap' ? 'success' : 'info'"
            size="small"
          >
            {{
              row.verificationMethod === 'ldap'
                ? $t('admin.users.schoolConfig.methods.ldap')
                : $t('admin.users.schoolConfig.methods.manual')
            }}
          </ElTag>
        </template>
      </PersistentAdminTableColumn>
      <PersistentAdminTableColumn
        column-key="enabled"
        :label="$t('admin.users.schoolConfig.enabledStatus')"
        :default-width="132"
      >
        <template #default="{ row }">
          <ElSwitch
            v-if="canUpdateSchoolConfig()"
            :model-value="row.enabled"
            :loading="isSchoolUpdating(row.schoolID)"
            data-school-enabled-switch
            @change="(value) => handleToggleEnabled(row, Boolean(value))"
          />
          <ElTag v-else :type="row.enabled ? 'success' : 'danger'" size="small">
            {{
              row.enabled
                ? $t('admin.users.schoolConfig.enabled')
                : $t('admin.users.schoolConfig.disabled')
            }}
          </ElTag>
        </template>
      </PersistentAdminTableColumn>
      <PersistentAdminTableColumn
        column-key="ldapURL"
        :label="$t('admin.users.schoolConfig.ldapUrl')"
        :default-min-width="220"
        show-overflow-tooltip
      >
        <template #default="{ row }">
          {{ row.ldapConfig?.url || $t('admin.common.unavailable') }}
        </template>
      </PersistentAdminTableColumn>
      <PersistentAdminTableColumn
        v-if="canUpdateSchoolConfig()"
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

    <SchoolConfigDialog
      v-model:visible="dialogVisible"
      :can-update="canUpdateSchoolConfig()"
      :form="form"
      :submitting="submitting"
      @submit="handleSubmit"
    />
  </AdminContentLayout>
</template>
