<script setup lang="ts">
import type { AdmissionPolicy } from '#/api/admin';

import { onMounted, reactive, ref } from 'vue';

import {
  ElAlert,
  ElButton,
  ElDatePicker,
  ElForm,
  ElFormItem,
  ElInput,
  ElInputNumber,
  ElMessage,
  ElSwitch,
} from 'element-plus';

import { listAdmissionPolicies, updateAdmissionPolicy } from '#/api/admin';
import { $t } from '#/locales';

import AdminContentLayout from '../../shared/AdminContentLayout.vue';

const loading = ref(false);
const loadError = ref('');
const actionError = ref('');
const policies = ref<AdmissionPolicy[]>([]);
const managementGuildText = reactive<Record<string, string>>({});
const savingPolicyIDs = reactive<Record<string, boolean>>({});
let fetchRequestSeq = 0;

const POLICY_DATETIME_FORMAT = 'YYYY-MM-DDTHH:mm:ssZ';

const policyFieldLabels = {
  blacklistDurationSeconds: '自动拉黑时长（秒）',
  failedJoinLimit: '失败入群上限',
  forwardRawMaterialToQQ: '转发原始材料到 QQ',
  freshmanChannelClosesAt: '新生通道关闭时间',
  freshmanChannelEnabled: '启用新生入群通道',
  freshmanDefaultExpiresAt: '默认临时认证到期时间',
  initialMuteDurationSeconds: '入群初始禁言（秒）',
  linkWaitSeconds: '绑定链接等待（秒）',
  managementGuildIDs: '管理群号',
  manualReviewTimeoutSeconds: '人工审核超时（秒）',
  maxExtensionDays: '最大延期天数',
  maxMaterialBytes: '材料大小上限（字节）',
  reminderIntervalSeconds: '提醒间隔（秒）',
  submissionWaitSeconds: '材料提交等待（秒）',
} as const;

type AdmissionPolicyBoundary = Omit<AdmissionPolicy, 'managementGuildIDs'> & {
  managementGuildIDs?: null | string[];
};

function normalizeManagementGuildIDs(values?: null | string[]) {
  if (!Array.isArray(values)) {
    return [];
  }
  return values
    .map((value) => value.trim())
    .filter((value) => value.length > 0);
}

function normalizePolicy(policy: AdmissionPolicyBoundary): AdmissionPolicy {
  return {
    ...policy,
    managementGuildIDs: normalizeManagementGuildIDs(policy.managementGuildIDs),
  };
}

async function fetchData() {
  const requestSeq = ++fetchRequestSeq;
  loading.value = true;
  loadError.value = '';
  try {
    const data = await listAdmissionPolicies();
    if (requestSeq !== fetchRequestSeq) return;
    policies.value = data.map((policy) => normalizePolicy(policy));
    for (const policy of policies.value) {
      managementGuildText[policy.id] = policy.managementGuildIDs.join('\n');
    }
  } catch (error) {
    if (requestSeq !== fetchRequestSeq) return;
    loadError.value = adminErrorMessage(error);
  } finally {
    if (requestSeq === fetchRequestSeq) {
      loading.value = false;
    }
  }
}

function parseManagementGuildIDs(policyID: string) {
  return (managementGuildText[policyID] ?? '')
    .split('\n')
    .map((value) => value.trim())
    .filter(Boolean);
}

async function savePolicy(policy: AdmissionPolicy) {
  if (savingPolicyIDs[policy.id]) {
    return;
  }

  savingPolicyIDs[policy.id] = true;
  actionError.value = '';
  try {
    await updateAdmissionPolicy({
      ...policy,
      managementGuildIDs: parseManagementGuildIDs(policy.id),
    });
    ElMessage.success(
      `已保存 ${policy.platform.toUpperCase()} 群 ${policy.guildID} 入群认证策略`,
    );
    await fetchData();
  } catch (error) {
    handleActionError(error);
  } finally {
    savingPolicyIDs[policy.id] = false;
  }
}

function handleActionError(error: unknown) {
  actionError.value = adminErrorMessage(error);
  ElMessage.error(actionError.value);
}

function adminErrorMessage(error: unknown): string {
  return error instanceof Error && error.message
    ? error.message
    : $t('admin.result.requestFailed');
}

onMounted(fetchData);
</script>

<template>
  <AdminContentLayout
    description="控制新生入群验证、临时认证期限和人工审核转发行为。"
    title="入群认证策略"
    :total="policies.length"
  >
    <ElAlert
      v-if="loadError"
      class="admin-load-error"
      type="error"
      :closable="false"
      show-icon
      :title="loadError"
    >
      <ElButton size="small" :loading="loading" @click="fetchData">
        {{ $t('admin.common.retry') }}
      </ElButton>
    </ElAlert>

    <ElAlert
      v-if="actionError"
      class="admin-load-error"
      type="error"
      :closable="true"
      show-icon
      :title="actionError"
      @close="actionError = ''"
    />

    <div v-loading="loading" class="grid gap-4">
      <ElForm
        v-for="policy in policies"
        :key="policy.id"
        class="rounded border border-slate-200 bg-white p-5 shadow-sm"
        label-position="top"
      >
        <div class="mb-4">
          <h2 class="text-base font-semibold text-slate-900">
            {{ policy.platform.toUpperCase() }} 群 {{ policy.guildID }}
          </h2>
          <p class="mt-1 text-sm text-slate-500">
            控制新生入群验证、临时认证期限和人工审核转发行为。
          </p>
        </div>
        <ElFormItem :label="policyFieldLabels.freshmanChannelEnabled">
          <ElSwitch v-model="policy.freshmanChannelEnabled" />
        </ElFormItem>
        <ElFormItem :label="policyFieldLabels.freshmanChannelClosesAt">
          <ElDatePicker
            v-model="policy.freshmanChannelClosesAt"
            type="datetime"
            :value-format="POLICY_DATETIME_FORMAT"
          />
        </ElFormItem>
        <ElFormItem :label="policyFieldLabels.freshmanDefaultExpiresAt">
          <ElDatePicker
            v-model="policy.freshmanDefaultExpiresAt"
            type="datetime"
            :value-format="POLICY_DATETIME_FORMAT"
          />
        </ElFormItem>
        <ElFormItem :label="policyFieldLabels.initialMuteDurationSeconds">
          <ElInputNumber v-model="policy.initialMuteDurationSeconds" :min="1" />
        </ElFormItem>
        <ElFormItem :label="policyFieldLabels.linkWaitSeconds">
          <ElInputNumber v-model="policy.linkWaitSeconds" :min="1" />
        </ElFormItem>
        <ElFormItem :label="policyFieldLabels.submissionWaitSeconds">
          <ElInputNumber v-model="policy.submissionWaitSeconds" :min="1" />
        </ElFormItem>
        <ElFormItem :label="policyFieldLabels.manualReviewTimeoutSeconds">
          <ElInputNumber v-model="policy.manualReviewTimeoutSeconds" :min="1" />
        </ElFormItem>
        <ElFormItem :label="policyFieldLabels.reminderIntervalSeconds">
          <ElInputNumber v-model="policy.reminderIntervalSeconds" :min="1" />
        </ElFormItem>
        <ElFormItem :label="policyFieldLabels.failedJoinLimit">
          <ElInputNumber v-model="policy.failedJoinLimit" :min="1" />
        </ElFormItem>
        <ElFormItem :label="policyFieldLabels.blacklistDurationSeconds">
          <ElInputNumber v-model="policy.blacklistDurationSeconds" :min="0" />
        </ElFormItem>
        <ElFormItem :label="policyFieldLabels.maxMaterialBytes">
          <ElInputNumber v-model="policy.maxMaterialBytes" :min="1" />
        </ElFormItem>
        <ElFormItem :label="policyFieldLabels.maxExtensionDays">
          <ElInputNumber v-model="policy.maxExtensionDays" :min="1" />
        </ElFormItem>
        <ElFormItem :label="policyFieldLabels.managementGuildIDs">
          <ElInput
            v-model="managementGuildText[policy.id]"
            placeholder="每行一个群号"
            :rows="3"
            type="textarea"
          />
        </ElFormItem>
        <ElFormItem :label="policyFieldLabels.forwardRawMaterialToQQ">
          <ElSwitch v-model="policy.forwardRawMaterialToQQ" />
        </ElFormItem>
        <ElButton
          type="primary"
          :disabled="savingPolicyIDs[policy.id]"
          :loading="savingPolicyIDs[policy.id]"
          @click="savePolicy(policy)"
        >
          保存
        </ElButton>
      </ElForm>

      <p class="text-sm text-slate-500">
        成员黑名单管理已迁移至独立页面：「用户系统 → 成员黑名单」。
      </p>
    </div>
  </AdminContentLayout>
</template>
