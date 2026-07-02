<script setup lang="ts">
import type { ScopeType } from './options';

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

import { $t } from '#/locales';

import { toIsoString } from './options';

const props = defineProps<{
  submitting: boolean;
}>();

const emit = defineEmits<{
  (e: 'submit', payload: MemberBlacklistCreateRequest): void;
}>();

const visible = defineModel<boolean>('visible', { required: true });

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
    guildID: draft.scopeType === 'guild' ? draft.guildID.trim() : undefined,
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
  if (props.submitting || !canSubmit.value) return;
  emit('submit', buildPayload());
}
</script>

<template>
  <ElDialog
    v-model="visible"
    data-dialog="create"
    :title="$t('admin.users.memberBlacklist.createDialogTitle')"
    width="520px"
  >
    <ElForm label-position="top">
      <ElFormItem :label="$t('admin.users.memberBlacklist.platformLabel')">
        <ElInput v-model="draft.platform" data-field="platform" />
      </ElFormItem>
      <ElFormItem :label="$t('admin.users.memberBlacklist.subjectLabel')">
        <ElInput v-model="draft.subjectID" data-field="subjectID" />
      </ElFormItem>
      <ElFormItem :label="$t('admin.users.memberBlacklist.scopeLabel')">
        <ElSelect v-model="draft.scopeType" data-field="scopeType">
          <ElOption
            :label="$t('admin.users.memberBlacklist.scopeFilter.guild')"
            value="guild"
          />
          <ElOption
            :label="$t('admin.users.memberBlacklist.scopeFilter.global')"
            value="global"
          />
        </ElSelect>
      </ElFormItem>
      <ElFormItem
        v-if="draft.scopeType === 'guild'"
        :label="$t('admin.users.memberBlacklist.guildLabel')"
      >
        <ElInput v-model="draft.guildID" data-field="guildID" />
      </ElFormItem>
      <ElFormItem :label="$t('admin.users.memberBlacklist.reasonLabel')">
        <ElInput
          v-model="draft.reasonText"
          :rows="3"
          data-field="reasonText"
          type="textarea"
        />
      </ElFormItem>
      <ElFormItem :label="$t('admin.users.memberBlacklist.expiresLabel')">
        <ElDatePicker
          v-model="draft.expiresAt"
          data-field="expiresAt"
          type="datetime"
        />
      </ElFormItem>
    </ElForm>
    <template #footer>
      <ElButton @click="visible = false">
        {{ $t('admin.common.cancel') }}
      </ElButton>
      <ElPopconfirm
        v-if="draft.scopeType === 'global'"
        :title="$t('admin.users.memberBlacklist.globalConfirm')"
        @confirm="submit"
      >
        <template #reference>
          <ElButton
            :disabled="props.submitting || !canSubmit"
            :loading="props.submitting"
            data-action="submitCreate"
            type="danger"
          >
            {{ $t('admin.users.memberBlacklist.createGlobal') }}
          </ElButton>
        </template>
      </ElPopconfirm>
      <ElButton
        v-else
        :disabled="props.submitting || !canSubmit"
        :loading="props.submitting"
        data-action="submitCreate"
        type="primary"
        @click="submit"
      >
        {{ $t('admin.users.memberBlacklist.create') }}
      </ElButton>
    </template>
  </ElDialog>
</template>
