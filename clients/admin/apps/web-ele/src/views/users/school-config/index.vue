<script setup lang="ts">
import type { SchoolConfig, UpdateSchoolConfigPayload } from '#/api/admin';

import { onMounted, reactive, ref } from 'vue';

import { useAccessStore } from '@vben/stores';

import { ElButton, ElMessage, ElTag } from 'element-plus';

import { getSchoolConfigList, updateSchoolConfig } from '#/api/admin';
import { $t } from '#/locales';

import AdminContentLayout from '../../shared/AdminContentLayout.vue';
import PersistentAdminTable from '../../shared/admin-table/PersistentAdminTable.vue';
import PersistentAdminTableColumn from '../../shared/admin-table/PersistentAdminTableColumn.vue';
import SchoolConfigDialog from './SchoolConfigDialog.vue';

const loading = ref(false);
const submitting = ref(false);
const schools = ref<SchoolConfig[]>([]);
const accessStore = useAccessStore();
const canUpdateSchoolConfig = () =>
  accessStore.accessCodes.includes('user:school:update');

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
  schoolName: '',
  systemBindDN: '',
  systemBindPassword: '',
  useTLS: true,
  verificationMethod: 'manual' as 'ldap' | 'manual',
});

function openEdit(row: SchoolConfig) {
  if (!canUpdateSchoolConfig()) {
    return;
  }
  form.schoolID = row.schoolID;
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

async function handleSubmit() {
  if (!canUpdateSchoolConfig() || submitting.value) {
    return;
  }

  const payload: UpdateSchoolConfigPayload = {
    academicDbTable: form.academicDbTable || undefined,
    consentText: form.consentText || undefined,
    enabled: form.enabled,
    ldapConfig:
      form.verificationMethod === 'ldap'
        ? {
            baseDN: form.ldapBaseDN || undefined,
            systemBindDN: form.systemBindDN || undefined,
            systemBindPassword: form.systemBindPassword || undefined,
            url: form.ldapURL || undefined,
            useTLS: form.useTLS,
          }
        : undefined,
    schoolName: form.schoolName,
    verificationMethod: form.verificationMethod,
  };

  submitting.value = true;
  try {
    await updateSchoolConfig(form.schoolID, payload);
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

onMounted(fetchData);
</script>

<template>
  <AdminContentLayout
    :title="$t('admin.routes.userSystem.schoolConfig')"
    :total="schools.length"
  >
    <PersistentAdminTable
      table-key="users.schoolConfig"
      :loading="loading"
      :data="schools"
      row-key="schoolID"
      stripe
    >
      <PersistentAdminTableColumn
        column-key="schoolID"
        :label="$t('admin.users.schoolConfig.schoolId')"
        prop="schoolID"
        :default-width="112"
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
        :default-width="112"
      >
        <template #default="{ row }">
          <ElTag :type="row.enabled ? 'success' : 'danger'" size="small">
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
