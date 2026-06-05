<script setup lang="ts">
import type { SchoolConfig, UpdateSchoolConfigPayload } from '#/api/admin';

import { computed, onMounted, reactive, ref } from 'vue';

import { useAccessStore } from '@vben/stores';

import { ElAlert, ElButton, ElMessage, ElSwitch, ElTag } from 'element-plus';

import { getSchoolConfigList, updateSchoolConfig } from '#/api/admin';
import { $t } from '#/locales';

import PersistentAdminTable from '../../shared/admin-table/PersistentAdminTable.vue';
import PersistentAdminTableColumn from '../../shared/admin-table/PersistentAdminTableColumn.vue';
import AdminContentLayout from '../../shared/AdminContentLayout.vue';
import SchoolConfigDialog from './SchoolConfigDialog.vue';

const loading = ref(false);
const loadError = ref('');
const actionError = ref('');
const submitting = ref(false);
const schools = ref<SchoolConfig[]>([]);
const updatingSchoolIds = ref(new Set<number>());
const accessStore = useAccessStore();
let fetchRequestSeq = 0;
const canUpdateSchoolConfig = () =>
  accessStore.accessCodes.includes('user:school:update');
const enabledSchoolCount = computed(
  () => schools.value.filter((school) => school.enabled).length,
);

async function fetchData() {
  const requestSeq = ++fetchRequestSeq;
  loading.value = true;
  loadError.value = '';
  try {
    const data = await getSchoolConfigList();
    if (requestSeq !== fetchRequestSeq) return;
    schools.value = data;
  } catch (error) {
    if (requestSeq !== fetchRequestSeq) return;
    loadError.value = adminErrorMessage(error);
  } finally {
    if (requestSeq === fetchRequestSeq) {
      loading.value = false;
    }
  }
}

const dialogVisible = ref(false);
const form = reactive({
  academicDbTable: '',
  approvalPolicy: 'manual' as 'auto' | 'manual',
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
  form.approvalPolicy = row.approvalPolicy;
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
    approvalPolicy: submitted.approvalPolicy,
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
  actionError.value = '';
  try {
    await updateSchoolConfig(submitted.schoolID, payload);
    ElMessage.success($t('admin.users.schoolConfig.updated'));
    dialogVisible.value = false;
    await fetchData();
  } catch (error) {
    handleActionError(error);
  } finally {
    submitting.value = false;
  }
}

async function handleToggleEnabled(row: SchoolConfig, enabled: boolean) {
  if (!canUpdateSchoolConfig() || isSchoolUpdating(row.schoolID)) {
    return;
  }
  markSchoolUpdating(row.schoolID, true);
  actionError.value = '';
  try {
    await updateSchoolConfig(row.schoolID, { enabled });
    row.enabled = enabled;
    ElMessage.success($t('admin.users.schoolConfig.updated'));
  } catch (error) {
    handleActionError(error);
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

type TagType = 'danger' | 'info' | 'primary' | 'success' | 'warning';

function admissionCapabilityTags(row: SchoolConfig) {
  const tags: Array<{ label: string; type: TagType }> = [];
  if (row.schoolSsoEnabled) {
    tags.push({
      label: $t('admin.users.schoolConfig.schoolSso'),
      type: 'success',
    });
  }
  if (row.schoolEmailOtpEnabled) {
    tags.push({
      label: $t('admin.users.schoolConfig.schoolEmailOtp'),
      type: 'primary',
    });
  }
  if (tags.length === 0) {
    tags.push({
      label: $t('admin.users.schoolConfig.noAdmissionCapability'),
      type: 'info',
    });
  }
  return tags;
}

function emailPolicySummary(row: SchoolConfig) {
  const policy = row.schoolEmailIdentityPolicy;
  if (!policy) {
    return '';
  }
  const parts = [
    policy.studentIDEmailDomain
      ? $t('admin.users.schoolConfig.emailPolicyDomain', {
          domain: `@${policy.studentIDEmailDomain}`,
        })
      : '',
    policy.requireStudentName
      ? $t('admin.users.schoolConfig.requiresStudentName')
      : '',
  ].filter(Boolean);
  return parts.join(' · ');
}

function adminErrorMessage(error: unknown): string {
  return error instanceof Error && error.message
    ? error.message
    : $t('admin.result.requestFailed');
}

function handleActionError(error: unknown) {
  actionError.value = adminErrorMessage(error);
  ElMessage.error(actionError.value);
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

    <ElAlert
      v-if="loadError"
      class="admin-load-error"
      type="error"
      :closable="false"
      show-icon
      :title="loadError"
    >
      <ElButton size="small" :loading="loading" @click="fetchData">
        {{ $t('admin.common.retry') }}
      </ElButton>
    </ElAlert>

    <ElAlert
      v-if="actionError"
      class="admin-load-error"
      type="error"
      :closable="true"
      show-icon
      :title="actionError"
      @close="actionError = ''"
    />

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
        column-key="approvalPolicy"
        :label="$t('admin.users.schoolConfig.approvalPolicy')"
        :default-width="118"
      >
        <template #default="{ row }">
          <ElTag
            :type="row.approvalPolicy === 'auto' ? 'success' : 'warning'"
            size="small"
          >
            {{
              row.approvalPolicy === 'auto'
                ? $t('admin.users.schoolConfig.approvalPolicies.auto')
                : $t('admin.users.schoolConfig.approvalPolicies.manual')
            }}
          </ElTag>
        </template>
      </PersistentAdminTableColumn>
      <PersistentAdminTableColumn
        column-key="admissionCapabilities"
        :label="$t('admin.users.schoolConfig.admissionCapabilities')"
        :default-min-width="240"
      >
        <template #default="{ row }">
          <div class="school-capabilities">
            <div class="school-capabilities__tags">
              <ElTag
                v-for="tag in admissionCapabilityTags(row)"
                :key="tag.label"
                :type="tag.type"
                size="small"
              >
                {{ tag.label }}
              </ElTag>
            </div>
            <span
              v-if="emailPolicySummary(row)"
              class="admin-cell-muted school-capabilities__policy"
            >
              {{ emailPolicySummary(row) }}
            </span>
          </div>
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

<style scoped>
.school-capabilities {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
}

.school-capabilities__tags {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}

.school-capabilities__policy {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
