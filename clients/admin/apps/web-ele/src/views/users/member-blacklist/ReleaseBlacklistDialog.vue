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

import { $t } from '#/locales';

import { releaseReasonOptions, scopeLabel, sourceLabel } from './options';

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
  if (props.submitting || !props.target) return;
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
    :title="$t('admin.users.memberBlacklist.releaseDialogTitle')"
    width="520px"
  >
    <div v-if="target" class="mb-3 text-sm text-slate-600">
      <div>
        {{ $t('admin.users.memberBlacklist.releaseSubjectPrefix') }}
        <span class="font-mono">{{ target.subjectID }}</span>
      </div>
      <div>
        {{
          $t('admin.users.memberBlacklist.releaseScopeLine', {
            scope: scopeLabel(target),
          })
        }}
      </div>
      <div>
        {{
          $t('admin.users.memberBlacklist.releaseSourceLine', {
            source: sourceLabel(target),
          })
        }}
      </div>
    </div>
    <ElForm label-position="top">
      <ElFormItem
        :label="$t('admin.users.memberBlacklist.releaseSemanticsLabel')"
      >
        <ElSelect
          v-model="draft.releaseReasonCode"
          data-field="releaseReasonCode"
        >
          <ElOption
            v-for="opt in releaseReasonOptions()"
            :key="opt.value"
            :label="opt.label"
            :value="opt.value"
          />
        </ElSelect>
      </ElFormItem>
      <ElFormItem :label="$t('admin.users.memberBlacklist.remarkLabel')">
        <ElInput
          v-model="draft.releaseReason"
          :rows="2"
          data-field="releaseReason"
          type="textarea"
        />
      </ElFormItem>
    </ElForm>
    <template #footer>
      <ElButton @click="visible = false">
        {{ $t('admin.common.cancel') }}
      </ElButton>
      <ElButton
        :disabled="props.submitting || !target"
        :loading="props.submitting"
        data-action="submitRelease"
        type="warning"
        @click="submit"
      >
        {{ $t('admin.users.memberBlacklist.release') }}
      </ElButton>
    </template>
  </ElDialog>
</template>
