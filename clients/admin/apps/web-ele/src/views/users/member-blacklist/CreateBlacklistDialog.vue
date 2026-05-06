<script setup lang="ts">
import type { MemberBlacklistCreateRequest } from '#/api/admin';

import { computed, reactive } from 'vue';

import {
  ElButton,
  ElDatePicker,
  ElDialog,
  ElForm,
  ElFormItem,
  ElInput,
  ElOption,
  ElPopconfirm,
  ElSelect,
} from 'element-plus';

import { type ScopeType, toIsoString } from './options';

const visible = defineModel<boolean>('visible', { required: true });

defineProps<{
  submitting: boolean;
}>();

const emit = defineEmits<{
  (e: 'submit', payload: MemberBlacklistCreateRequest): void;
}>();

const draft = reactive({
  platform: 'qq',
  subjectID: '',
  scopeType: 'guild' as ScopeType,
  guildID: '',
  reasonText: '',
  expiresAt: '' as Date | string,
});

const canSubmit = computed(() => {
  if (!draft.platform.trim() || !draft.subjectID.trim()) return false;
  if (draft.scopeType === 'guild' && !draft.guildID.trim()) return false;
  return Boolean(draft.reasonText.trim());
});

function reset() {
  draft.platform = 'qq';
  draft.subjectID = '';
  draft.scopeType = 'guild';
  draft.guildID = '';
  draft.reasonText = '';
  draft.expiresAt = '';
}

defineExpose({ reset });

function buildPayload(): MemberBlacklistCreateRequest {
  return {
    platform: draft.platform.trim(),
    subjectType: 'qq_user',
    subjectID: draft.subjectID.trim(),
    scopeType: draft.scopeType,
    guildID:
      draft.scopeType === 'guild' ? draft.guildID.trim() : undefined,
    source: 'manual_admin',
    reasonCode: 'manual_blacklist',
    reasonText: draft.reasonText.trim(),
    expiresAt: toIsoString(draft.expiresAt),
    metadata: {
      operatorInput: draft.subjectID.trim(),
      scopeSelectionContext:
        draft.scopeType === 'global'
          ? 'admin_console_form_global'
          : 'admin_console_form_guild',
    },
  };
}

function submit() {
  if (!canSubmit.value) return;
  emit('submit', buildPayload());
}
</script>

<template>
  <ElDialog
    v-model="visible"
    data-dialog="create"
    title="新增成员黑名单"
    width="520px"
  >
    <ElForm label-position="top">
      <ElFormItem label="平台">
        <ElInput v-model="draft.platform" data-field="platform" />
      </ElFormItem>
      <ElFormItem label="主体 ID（QQ 号）">
        <ElInput v-model="draft.subjectID" data-field="subjectID" />
      </ElFormItem>
      <ElFormItem label="范围">
        <ElSelect v-model="draft.scopeType" data-field="scopeType">
          <ElOption label="单群" value="guild" />
          <ElOption label="全局" value="global" />
        </ElSelect>
      </ElFormItem>
      <ElFormItem v-if="draft.scopeType === 'guild'" label="群号">
        <ElInput v-model="draft.guildID" data-field="guildID" />
      </ElFormItem>
      <ElFormItem label="原因（必填）">
        <ElInput
          v-model="draft.reasonText"
          :rows="3"
          data-field="reasonText"
          type="textarea"
        />
      </ElFormItem>
      <ElFormItem label="过期时间（留空表示永久）">
        <ElDatePicker
          v-model="draft.expiresAt"
          data-field="expiresAt"
          type="datetime"
        />
      </ElFormItem>
    </ElForm>
    <template #footer>
      <ElButton @click="visible = false">取消</ElButton>
      <ElPopconfirm
        v-if="draft.scopeType === 'global'"
        title="确认创建全局黑名单？该成员将被所有群拒绝。"
        @confirm="submit"
      >
        <template #reference>
          <ElButton
            :disabled="!canSubmit"
            :loading="submitting"
            data-action="submitCreate"
            type="danger"
          >
            创建全局黑名单
          </ElButton>
        </template>
      </ElPopconfirm>
      <ElButton
        v-else
        :disabled="!canSubmit"
        :loading="submitting"
        data-action="submitCreate"
        type="primary"
        @click="submit"
      >
        创建
      </ElButton>
    </template>
  </ElDialog>
</template>
