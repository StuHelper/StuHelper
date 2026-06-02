<script setup lang="ts">
import type { StatusFilter } from './options';

import { ElButton, ElInput, ElOption, ElSelect } from 'element-plus';

import { STATUS_OPTIONS } from './options';

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
      placeholder="QQ 号"
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
      v-model="botSelfID"
      clearable
      data-field="botSelfID"
      placeholder="Bot QQ"
      style="width: 160px"
      @keyup.enter="emit('search')"
    />
    <ElInput
      v-model="platform"
      clearable
      data-field="platform"
      placeholder="平台"
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
        v-for="opt in STATUS_OPTIONS"
        :key="opt.value || 'all'"
        :label="opt.label"
        :value="opt.value"
      />
    </ElSelect>
    <ElButton type="primary" @click="emit('search')">查询</ElButton>
    <ElButton @click="emit('reset')">重置</ElButton>
  </div>
</template>
