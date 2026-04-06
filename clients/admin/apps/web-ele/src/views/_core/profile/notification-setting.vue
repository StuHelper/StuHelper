<script setup lang="ts">
import { computed } from 'vue';

import { ProfileNotificationSetting } from '@vben/common-ui';

import { $t } from '#/locales';

const formSchema = computed(() => {
  return [
    {
      value: true,
      fieldName: 'accountPassword',
      label: $t('admin.profile.notification.accountPassword.label'),
      description: $t('admin.profile.notification.accountPassword.description'),
    },
    {
      value: true,
      fieldName: 'systemMessage',
      label: $t('admin.profile.notification.systemMessage.label'),
      description: $t('admin.profile.notification.systemMessage.description'),
    },
    {
      value: true,
      fieldName: 'todoTask',
      label: $t('admin.profile.notification.todoTask.label'),
      description: $t('admin.profile.notification.todoTask.description'),
    },
  ];
});
</script>
<template>
  <ProfileNotificationSetting :form-schema="formSchema" />
</template>
