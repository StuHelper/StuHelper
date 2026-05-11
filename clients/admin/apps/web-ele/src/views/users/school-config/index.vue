<script setup lang="ts">
import type { SchoolConfig, UpdateSchoolConfigPayload } from '#/api/admin';

import { onMounted, reactive, ref } from 'vue';

import { useAccessStore } from '@vben/stores';

import {
  ElButton,
  ElDialog,
  ElForm,
  ElFormItem,
  ElInput,
  ElMessage,
  ElOption,
  ElSelect,
  ElSwitch,
  ElTable,
  ElTableColumn,
  ElTag,
} from 'element-plus';

import { getSchoolConfigList, updateSchoolConfig } from '#/api/admin';
import { $t } from '#/locales';

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
  <div class="p-4">
    <ElTable v-loading="loading" :data="schools" stripe>
      <ElTableColumn
        :label="$t('admin.users.schoolConfig.schoolId')"
        prop="schoolID"
        width="100"
      />
      <ElTableColumn
        :label="$t('admin.users.schoolConfig.schoolName')"
        min-width="180"
        prop="schoolName"
      />
      <ElTableColumn
        :label="$t('admin.users.schoolConfig.verificationMethod')"
        width="110"
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
      </ElTableColumn>
      <ElTableColumn
        :label="$t('admin.users.schoolConfig.enabledStatus')"
        width="100"
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
      </ElTableColumn>
      <ElTableColumn
        :label="$t('admin.users.schoolConfig.ldapUrl')"
        min-width="200"
      >
        <template #default="{ row }">
          {{ row.ldapConfig?.url || $t('admin.common.unavailable') }}
        </template>
      </ElTableColumn>
      <ElTableColumn
        v-if="canUpdateSchoolConfig()"
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

    <ElDialog
      v-model="dialogVisible"
      :title="
        $t('admin.users.schoolConfig.dialogTitle', {
          schoolName: form.schoolName,
        })
      "
      width="620px"
    >
      <ElForm label-width="120px">
        <ElFormItem :label="$t('admin.users.schoolConfig.schoolName')">
          <ElInput
            v-model="form.schoolName"
            :placeholder="$t('admin.users.schoolConfig.schoolNamePlaceholder')"
          />
        </ElFormItem>
        <ElFormItem :label="$t('admin.users.schoolConfig.verificationMethod')">
          <ElSelect v-model="form.verificationMethod" style="width: 100%">
            <ElOption
              :label="$t('admin.users.schoolConfig.methods.ldap')"
              value="ldap"
            />
            <ElOption
              :label="$t('admin.users.schoolConfig.methods.manual')"
              value="manual"
            />
          </ElSelect>
        </ElFormItem>
        <ElFormItem :label="$t('admin.users.schoolConfig.enableSchool')">
          <ElSwitch v-model="form.enabled" />
        </ElFormItem>
        <template v-if="form.verificationMethod === 'ldap'">
          <ElFormItem :label="$t('admin.users.schoolConfig.ldapUrl')">
            <ElInput
              v-model="form.ldapURL"
              :placeholder="$t('admin.users.schoolConfig.ldapUrlPlaceholder')"
            />
          </ElFormItem>
          <ElFormItem :label="$t('admin.users.schoolConfig.baseDn')">
            <ElInput
              v-model="form.ldapBaseDN"
              :placeholder="$t('admin.users.schoolConfig.baseDnPlaceholder')"
            />
          </ElFormItem>
          <ElFormItem :label="$t('admin.users.schoolConfig.systemBindDn')">
            <ElInput
              v-model="form.systemBindDN"
              :placeholder="
                $t('admin.users.schoolConfig.systemBindDnPlaceholder')
              "
            />
          </ElFormItem>
          <ElFormItem
            :label="$t('admin.users.schoolConfig.systemBindPassword')"
          >
            <ElInput
              v-model="form.systemBindPassword"
              :placeholder="
                $t('admin.users.schoolConfig.systemBindPasswordPlaceholder')
              "
              show-password
              type="password"
            />
          </ElFormItem>
          <ElFormItem :label="$t('admin.users.schoolConfig.useTls')">
            <ElSwitch v-model="form.useTLS" />
          </ElFormItem>
        </template>
        <ElFormItem :label="$t('admin.users.schoolConfig.academicDbTable')">
          <ElInput
            v-model="form.academicDbTable"
            :placeholder="
              $t('admin.users.schoolConfig.academicDbTablePlaceholder')
            "
          />
        </ElFormItem>
        <ElFormItem :label="$t('admin.users.schoolConfig.consentText')">
          <ElInput
            v-model="form.consentText"
            :rows="3"
            :placeholder="$t('admin.users.schoolConfig.consentTextPlaceholder')"
            type="textarea"
          />
        </ElFormItem>
      </ElForm>
      <template #footer>
        <ElButton @click="dialogVisible = false">
          {{ $t('admin.common.cancel') }}
        </ElButton>
        <ElButton
          v-if="canUpdateSchoolConfig()"
          :loading="submitting"
          type="primary"
          @click="handleSubmit"
        >
          {{ $t('admin.common.save') }}
        </ElButton>
      </template>
    </ElDialog>
  </div>
</template>
