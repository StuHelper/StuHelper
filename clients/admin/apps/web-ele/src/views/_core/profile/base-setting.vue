<script setup lang="ts">
import type { BasicOption } from '@vben/types';

import type { VbenFormSchema } from '#/adapter/form';

import { computed, onMounted, ref } from 'vue';

import { ProfileBaseSetting } from '@vben/common-ui';

import { getUserInfoApi } from '#/api';
import { $t } from '#/locales';

const profileBaseSettingRef = ref();

const MOCK_ROLES_OPTIONS: BasicOption[] = [
  {
    label: $t('admin.profile.baseSetting.roles.admin'),
    value: 'super',
  },
  {
    label: $t('admin.profile.baseSetting.roles.user'),
    value: 'user',
  },
  {
    label: $t('admin.profile.baseSetting.roles.test'),
    value: 'test',
  },
];

const formSchema = computed((): VbenFormSchema[] => {
  return [
    {
      fieldName: 'realName',
      component: 'Input',
      label: $t('admin.profile.baseSetting.fields.realName'),
    },
    {
      fieldName: 'username',
      component: 'Input',
      label: $t('admin.profile.baseSetting.fields.username'),
    },
    {
      fieldName: 'roles',
      component: 'Select',
      componentProps: {
        mode: 'tags',
        options: MOCK_ROLES_OPTIONS,
      },
      label: $t('admin.profile.baseSetting.fields.roles'),
    },
    {
      fieldName: 'introduction',
      component: 'Textarea',
      label: $t('admin.profile.baseSetting.fields.introduction'),
    },
  ];
});

onMounted(async () => {
  const { userInfo } = await getUserInfoApi();
  profileBaseSettingRef.value.getFormApi().setValues(userInfo);
});
</script>
<template>
  <ProfileBaseSetting ref="profileBaseSettingRef" :form-schema="formSchema" />
</template>
