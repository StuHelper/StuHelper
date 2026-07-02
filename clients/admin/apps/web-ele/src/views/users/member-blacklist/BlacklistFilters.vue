<script setup lang="ts">
import type { ScopeType, SourceFilter, StatusFilter } from './options';

import { ElButton, ElInput, ElOption, ElSelect } from 'element-plus';

import { $t } from '#/locales';

import { scopeOptions, sourceOptions, statusOptions } from './options';

defineProps<{
  canManage: boolean;
}>();

const emit = defineEmits<{
  (e: 'search'): void;
  (e: 'reset'): void;
  (e: 'openCreate'): void;
}>();
const platform = defineModel<string>('platform', { required: true });
const scopeType = defineModel<'' | ScopeType>('scopeType', { required: true });
const source = defineModel<SourceFilter>('source', { required: true });
const status = defineModel<StatusFilter>('status', { required: true });
const guildID = defineModel<string>('guildID', { required: true });
const subjectID = defineModel<string>('subjectID', { required: true });
</script>

<template>
  <div class="flex flex-wrap items-end gap-3">
    <ElInput
      v-model="subjectID"
      clearable
      data-field="subjectID"
      :placeholder="$t('admin.users.memberBlacklist.subjectPlaceholder')"
      style="width: 180px"
      @keyup.enter="emit('search')"
    />
    <ElInput
      v-model="guildID"
      clearable
      data-field="guildID"
      :placeholder="$t('admin.users.memberBlacklist.guildPlaceholder')"
      style="width: 160px"
      @keyup.enter="emit('search')"
    />
    <ElInput
      v-model="platform"
      clearable
      data-field="platform"
      :placeholder="$t('admin.users.memberBlacklist.platformPlaceholder')"
      style="width: 120px"
    />
    <ElSelect
      v-model="scopeType"
      data-field="scopeType"
      style="width: 140px"
      :teleported="false"
      @change="emit('search')"
    >
      <ElOption
        v-for="opt in scopeOptions()"
        :key="opt.value || 'all'"
        :label="opt.label"
        :value="opt.value"
      />
    </ElSelect>
    <ElSelect
      v-model="source"
      data-field="source"
      style="width: 160px"
      :teleported="false"
      @change="emit('search')"
    >
      <ElOption
        v-for="opt in sourceOptions()"
        :key="opt.value || 'all'"
        :label="opt.label"
        :value="opt.value"
      />
    </ElSelect>
    <ElSelect
      v-model="status"
      data-field="status"
      style="width: 140px"
      :teleported="false"
      @change="emit('search')"
    >
      <ElOption
        v-for="opt in statusOptions()"
        :key="opt.value"
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
    <div class="flex-1"></div>
    <ElButton
      v-if="canManage"
      data-action="openCreate"
      type="success"
      @click="emit('openCreate')"
    >
      {{ $t('admin.users.memberBlacklist.createButton') }}
    </ElButton>
  </div>
</template>
