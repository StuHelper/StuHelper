<script setup lang="ts">
import { reactive, watch } from 'vue';

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

type SchoolConfigForm = {
  academicDbTable: string;
  approvalPolicy: 'auto' | 'manual';
  consentText: string;
  enabled: boolean;
  ldapBaseDN: string;
  ldapURL: string;
  schoolCode: string;
  schoolID: number;
  schoolName: string;
  systemBindDN: string;
  systemBindPassword: string;
  useTLS: boolean;
  verificationMethod: 'ldap' | 'manual';
};

const props = defineProps<{
  canUpdate: boolean;
  form: SchoolConfigForm;
  submitting: boolean;
}>();

const emit = defineEmits<{
  (e: 'submit', form: SchoolConfigForm): void;
}>();
const visible = defineModel<boolean>('visible', { required: true });
const draft = reactive<SchoolConfigForm>({
  academicDbTable: '',
  approvalPolicy: 'manual',
  consentText: '',
  enabled: true,
  ldapBaseDN: '',
  ldapURL: '',
  schoolCode: '',
  schoolID: 0,
  schoolName: '',
  systemBindDN: '',
  systemBindPassword: '',
  useTLS: true,
  verificationMethod: 'manual',
});

watch(
  () => props.form,
  (form) => {
    Object.assign(draft, form);
  },
  { deep: true, immediate: true },
);

function submit() {
  emit('submit', { ...draft });
}
</script>

<template>
  <ElDialog
    v-model="visible"
    :title="
      $t('admin.users.schoolConfig.dialogTitle', {
        schoolName: draft.schoolName,
      })
    "
    width="620px"
  >
    <ElForm label-width="120px">
      <ElFormItem :label="$t('admin.users.schoolConfig.schoolCode')">
        <ElInput :model-value="draft.schoolCode" disabled />
      </ElFormItem>
      <ElFormItem :label="$t('admin.users.schoolConfig.schoolName')">
        <ElInput
          v-model="draft.schoolName"
          :placeholder="$t('admin.users.schoolConfig.schoolNamePlaceholder')"
        />
      </ElFormItem>
      <ElFormItem :label="$t('admin.users.schoolConfig.verificationMethod')">
        <ElSelect v-model="draft.verificationMethod" style="width: 100%">
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
      <ElFormItem :label="$t('admin.users.schoolConfig.approvalPolicy')">
        <ElSelect v-model="draft.approvalPolicy" style="width: 100%">
          <ElOption
            :label="$t('admin.users.schoolConfig.approvalPolicies.auto')"
            value="auto"
          />
          <ElOption
            :label="$t('admin.users.schoolConfig.approvalPolicies.manual')"
            value="manual"
          />
        </ElSelect>
      </ElFormItem>
      <ElFormItem :label="$t('admin.users.schoolConfig.enableSchool')">
        <ElSwitch v-model="draft.enabled" />
      </ElFormItem>
      <template v-if="draft.verificationMethod === 'ldap'">
        <ElFormItem :label="$t('admin.users.schoolConfig.ldapUrl')">
          <ElInput
            v-model="draft.ldapURL"
            :placeholder="$t('admin.users.schoolConfig.ldapUrlPlaceholder')"
          />
        </ElFormItem>
        <ElFormItem :label="$t('admin.users.schoolConfig.baseDn')">
          <ElInput
            v-model="draft.ldapBaseDN"
            :placeholder="$t('admin.users.schoolConfig.baseDnPlaceholder')"
          />
        </ElFormItem>
        <ElFormItem :label="$t('admin.users.schoolConfig.systemBindDn')">
          <ElInput
            v-model="draft.systemBindDN"
            :placeholder="
              $t('admin.users.schoolConfig.systemBindDnPlaceholder')
            "
          />
        </ElFormItem>
        <ElFormItem :label="$t('admin.users.schoolConfig.systemBindPassword')">
          <ElInput
            v-model="draft.systemBindPassword"
            :placeholder="
              $t('admin.users.schoolConfig.systemBindPasswordPlaceholder')
            "
            show-password
            type="password"
          />
        </ElFormItem>
        <ElFormItem :label="$t('admin.users.schoolConfig.useTls')">
          <ElSwitch v-model="draft.useTLS" />
        </ElFormItem>
      </template>
      <ElFormItem :label="$t('admin.users.schoolConfig.academicDbTable')">
        <ElInput
          v-model="draft.academicDbTable"
          :placeholder="
            $t('admin.users.schoolConfig.academicDbTablePlaceholder')
          "
        />
      </ElFormItem>
      <ElFormItem :label="$t('admin.users.schoolConfig.consentText')">
        <ElInput
          v-model="draft.consentText"
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
        @click="submit"
      >
        {{ $t('admin.common.save') }}
      </ElButton>
    </template>
  </ElDialog>
</template>
