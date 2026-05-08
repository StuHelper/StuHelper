<script setup lang="ts">
import type { ReleaseReasonCode } from './options';

import type {
  MemberBlacklistEntry,
  MemberBlacklistReleaseRequest,
} from '#/api/admin';

import { reactive, watch } from 'vue';

import {
  ElButton,
  ElDialog,
  ElForm,
  ElFormItem,
  ElInput,
  ElOption,
  ElSelect,
} from 'element-plus';

import { RELEASE_REASON_OPTIONS, scopeLabel, sourceLabel } from './options';

const props = defineProps<{
  submitting: boolean;
  target: MemberBlacklistEntry | null;
}>();

const emit = defineEmits<{
  (
    e: 'submit',
    payload: { id: string; request: MemberBlacklistReleaseRequest },
  ): void;
}>();

const visible = defineModel<boolean>('visible', { required: true });

const draft = reactive({
  releaseReasonCode: 'manual_pardon' as ReleaseReasonCode,
  releaseReason: '',
});

// Reset draft whenever the dialog is opened (visible flips true) or the
// target swaps while it is open. Watching only `props.target` would miss
// the cancel-then-reopen-same-entry case because the target ref does not
// change between the two opens, leaving stale draft state behind.
watch(
  [visible, () => props.target],
  ([nextVisible]) => {
    if (!nextVisible) return;
    const target = props.target;
    if (!target) return;
    draft.releaseReasonCode =
      target.source === 'admission_failure' ? 'manual_pardon' : 'release_only';
    draft.releaseReason = '';
  },
  { immediate: true },
);

function submit() {
  if (!props.target) return;
  const request: MemberBlacklistReleaseRequest = {
    releaseReasonCode: draft.releaseReasonCode,
    ...(draft.releaseReason.trim()
      ? { releaseReason: draft.releaseReason.trim() }
      : {}),
  };
  emit('submit', { id: props.target.id, request });
}
</script>

<template>
  <ElDialog
    v-model="visible"
    data-dialog="release"
    title="解除成员黑名单"
    width="520px"
  >
    <div v-if="target" class="mb-3 text-sm text-slate-600">
      <div>
        主体：<span class="font-mono">{{ target.subjectID }}</span>
      </div>
      <div>范围：{{ scopeLabel(target) }}</div>
      <div>来源：{{ sourceLabel(target) }}</div>
    </div>
    <ElForm label-position="top">
      <ElFormItem label="解除语义">
        <ElSelect
          v-model="draft.releaseReasonCode"
          data-field="releaseReasonCode"
        >
          <ElOption
            v-for="opt in RELEASE_REASON_OPTIONS"
            :key="opt.value"
            :label="opt.label"
            :value="opt.value"
          />
        </ElSelect>
      </ElFormItem>
      <ElFormItem label="备注（可选）">
        <ElInput
          v-model="draft.releaseReason"
          :rows="2"
          data-field="releaseReason"
          type="textarea"
        />
      </ElFormItem>
    </ElForm>
    <template #footer>
      <ElButton @click="visible = false">取消</ElButton>
      <ElButton
        :disabled="!target"
        :loading="submitting"
        data-action="submitRelease"
        type="warning"
        @click="submit"
      >
        解除
      </ElButton>
    </template>
  </ElDialog>
</template>
