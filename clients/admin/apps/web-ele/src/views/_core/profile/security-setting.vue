<script setup lang="ts">
import { computed } from 'vue';

import { ProfileSecuritySetting } from '@vben/common-ui';

import { $t } from '#/locales';

const formSchema = computed(() => {
  return [
    {
      value: true,
      fieldName: 'accountPassword',
      label: $t('admin.profile.security.accountPassword.label'),
      description: $t('admin.profile.security.accountPassword.description'),
    },
    {
      value: true,
      fieldName: 'securityPhone',
      label: $t('admin.profile.security.securityPhone.label'),
      description: $t('admin.profile.security.securityPhone.description'),
    },
    {
      value: true,
      fieldName: 'securityQuestion',
      label: $t('admin.profile.security.securityQuestion.label'),
      description: $t('admin.profile.security.securityQuestion.description'),
    },
    {
      value: true,
      fieldName: 'securityEmail',
      label: $t('admin.profile.security.securityEmail.label'),
      description: $t('admin.profile.security.securityEmail.description'),
    },
    {
      value: false,
      fieldName: 'securityMfa',
      label: $t('admin.profile.security.securityMfa.label'),
      description: $t('admin.profile.security.securityMfa.description'),
    },
  ];
});
</script>
<template>
  <ProfileSecuritySetting :form-schema="formSchema" />
</template>
