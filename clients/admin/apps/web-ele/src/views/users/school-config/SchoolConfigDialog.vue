<script setup lang="ts">
import {
  ElButton,
  ElDialog,
  ElForm,
  ElFormItem,
  ElInput,
  ElOption,
  ElSelect,
  ElSwitch,
} from 'element-plus';

import { $t } from '#/locales';

defineProps<{
  canUpdate: boolean;
  form: {
    academicDbTable: string;
    consentText: string;
    enabled: boolean;
    ldapBaseDN: string;
    ldapURL: string;
    schoolName: string;
    systemBindDN: string;
    systemBindPassword: string;
    useTLS: boolean;
    verificationMethod: 'ldap' | 'manual';
  };
  submitting: boolean;
}>();

const emit = defineEmits<{
  (e: 'submit'): void;
}>();
const visible = defineModel<boolean>('visible', { required: true });
</script>

<template>
  <ElDialog
    v-model="visible"
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
        <ElFormItem :label="$t('admin.users.schoolConfig.systemBindPassword')">
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
      <ElButton @click="visible = false">
        {{ $t('admin.common.cancel') }}
      </ElButton>
      <ElButton
        v-if="canUpdate"
        :loading="submitting"
        type="primary"
        @click="emit('submit')"
      >
        {{ $t('admin.common.save') }}
      </ElButton>
    </template>
  </ElDialog>
</template>
