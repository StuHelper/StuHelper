<script setup lang="ts">
import type { ScopeType, SourceFilter, StatusFilter } from './options';

import { ElButton, ElInput, ElOption, ElSelect } from 'element-plus';

import { SCOPE_OPTIONS, SOURCE_OPTIONS, STATUS_OPTIONS } from './options';

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
  <div class="mb-4 flex flex-wrap items-end gap-3">
    <ElInput
      v-model="subjectID"
      clearable
      data-field="subjectID"
      placeholder="QQ / 主体 ID"
      style="width: 180px"
      @keyup.enter="emit('search')"
    />
    <ElInput
      v-model="guildID"
      clearable
      data-field="guildID"
      placeholder="群号"
      style="width: 160px"
      @keyup.enter="emit('search')"
    />
    <ElInput
      v-model="platform"
      clearable
      data-field="platform"
      placeholder="平台"
      style="width: 120px"
    />
    <ElSelect v-model="scopeType" data-field="scopeType" style="width: 140px">
      <ElOption
        v-for="opt in SCOPE_OPTIONS"
        :key="opt.value || 'all'"
        :label="opt.label"
        :value="opt.value"
      />
    </ElSelect>
    <ElSelect v-model="source" data-field="source" style="width: 160px">
      <ElOption
        v-for="opt in SOURCE_OPTIONS"
        :key="opt.value || 'all'"
        :label="opt.label"
        :value="opt.value"
      />
    </ElSelect>
    <ElSelect v-model="status" data-field="status" style="width: 140px">
      <ElOption
        v-for="opt in STATUS_OPTIONS"
        :key="opt.value"
        :label="opt.label"
        :value="opt.value"
      />
    </ElSelect>
    <ElButton type="primary" @click="emit('search')">查询</ElButton>
    <ElButton @click="emit('reset')">重置</ElButton>
    <div class="flex-1"></div>
    <ElButton
      v-if="canManage"
      data-action="openCreate"
      type="success"
      @click="emit('openCreate')"
    >
      新增黑名单
    </ElButton>
  </div>
</template>
