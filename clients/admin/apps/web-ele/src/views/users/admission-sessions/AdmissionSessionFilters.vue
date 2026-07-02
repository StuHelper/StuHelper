<script setup lang="ts">
import type { StatusFilter } from './options';

import { ElButton, ElInput, ElOption, ElSelect } from 'element-plus';

import { $t } from '#/locales';

import { statusOptions } from './options';

const emit = defineEmits<{
  (e: 'reset'): void;
  (e: 'search'): void;
}>();

const qqID = defineModel<string>('qqID', { required: true });
const guildID = defineModel<string>('guildID', { required: true });
const botSelfID = defineModel<string>('botSelfID', { required: true });
const platform = defineModel<string>('platform', { required: true });
const status = defineModel<StatusFilter>('status', { required: true });
</script>

<template>
  <div class="flex flex-wrap items-end gap-3">
    <ElInput
      v-model="qqID"
      clearable
      data-field="qqID"
      :placeholder="$t('admin.users.admissionSessions.qqPlaceholder')"
      style="width: 180px"
      @keyup.enter="emit('search')"
    />
    <ElInput
      v-model="guildID"
      clearable
      data-field="guildID"
      :placeholder="$t('admin.users.admissionSessions.guildPlaceholder')"
      style="width: 160px"
      @keyup.enter="emit('search')"
    />
    <ElInput
      v-model="botSelfID"
      clearable
      data-field="botSelfID"
      :placeholder="$t('admin.users.admissionSessions.botPlaceholder')"
      style="width: 160px"
      @keyup.enter="emit('search')"
    />
    <ElInput
      v-model="platform"
      clearable
      data-field="platform"
      :placeholder="$t('admin.users.admissionSessions.platformPlaceholder')"
      style="width: 120px"
      @keyup.enter="emit('search')"
    />
    <ElSelect
      v-model="status"
      data-field="status"
      style="width: 150px"
      :teleported="false"
      @change="emit('search')"
    >
      <ElOption
        v-for="opt in statusOptions()"
        :key="opt.value || 'all'"
        :label="opt.label"
        :value="opt.value"
      />
    </ElSelect>
    <ElButton type="primary" @click="emit('search')">
      {{ $t('admin.common.query') }}
    </ElButton>
    <ElButton @click="emit('reset')">
      {{ $t('admin.common.reset') }}
    </ElButton>
  </div>
</template>
