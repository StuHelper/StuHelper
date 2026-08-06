<script setup lang="ts">
import type { AdmissionPolicy } from '#/api/admin';

import { computed, onMounted, reactive, ref } from 'vue';

import {
  ElAlert,
  ElButton,
  ElDialog,
  ElForm,
  ElFormItem,
  ElInput,
  ElInputNumber,
  ElMessage,
  ElOption,
  ElRadioButton,
  ElRadioGroup,
  ElSelect,
  ElSwitch,
  ElTag,
} from 'element-plus';

import {
  createAdmissionPolicy,
  listAdmissionPolicies,
  updateAdmissionPolicy,
} from '#/api/admin';
import { $t } from '#/locales';

import AdminContentLayout from '../../shared/AdminContentLayout.vue';

const loading = ref(false);
const loadError = ref('');
const actionError = ref('');
const policies = ref<AdmissionPolicy[]>([]);
const savingPolicyIDs = reactive<Record<string, boolean>>({});
const createPolicyDialogVisible = ref(false);
const createPolicySubmitting = ref(false);
const createPolicyForm = reactive({
  guildIDs: '',
  platform: 'qq',
  sourcePolicyID: '',
});
let fetchRequestSeq = 0;

const policyFieldLabels = {
  allowTemporaryFreshman: '允许临时新生凭据入群',
  blacklistDurationSeconds: '自动拉黑时长（秒）',
  failedJoinLimit: '失败入群上限',
  guardEnabled: '启用入群认证守卫',
  initialMuteDurationSeconds: '入群初始禁言（秒）',
  joinHandlingStrategy: '入群处理策略',
  linkWaitSeconds: '学生认证链接等待（秒）',
  manualReviewTimeoutSeconds: '人工审核超时（秒）',
  reminderIntervalSeconds: '提醒间隔（秒）',
  submissionWaitSeconds: '材料提交等待（秒）',
  unverifiedJoinRejectReason: '未认证拒绝理由',
} as const;

const joinHandlingStrategyOptions = [
  { label: '加群后学生认证', value: 'post_join_guard' },
  { label: '申请时学生认证', value: 'join_request_review' },
  { label: '加群后时间验证码', value: 'post_join_time_code' },
] as const;

const createSourceOptions = computed(() =>
  policies.value.map((policy) => ({
    label: `${policy.platform.toUpperCase()} 群 ${policy.guildID}`,
    value: policy.id,
  })),
);

type AdmissionPolicyBoundary = AdmissionPolicy & {
  joinHandlingStrategy?: AdmissionPolicy['joinHandlingStrategy'];
  unverifiedJoinRejectReason?: null | string;
};

function normalizePolicy(policy: AdmissionPolicyBoundary): AdmissionPolicy {
  return {
    ...policy,
    joinHandlingStrategy: policy.joinHandlingStrategy ?? 'post_join_guard',
    unverifiedJoinRejectReason:
      policy.unverifiedJoinRejectReason?.trim() ||
      '请先完成 StuHelper 学生认证后再申请入群。',
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
    if (!createPolicyForm.sourcePolicyID) {
      createPolicyForm.sourcePolicyID = policies.value[0]?.id ?? '';
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

function joinHandlingStrategyLabel(policy: AdmissionPolicy) {
  return (
    joinHandlingStrategyOptions.find(
      (option) => option.value === policy.joinHandlingStrategy,
    )?.label ?? policy.joinHandlingStrategy
  );
}

function joinHandlingStrategyHelp(policy: AdmissionPolicy) {
  if (policy.joinHandlingStrategy === 'post_join_guard') {
    return 'Koishi 会在目标群成员入群后执行禁言、发认证链接、通过后解除禁言。';
  }
  if (policy.joinHandlingStrategy === 'post_join_time_code') {
    return 'Koishi 会先同意入群申请，成员入群后在群内发送动态验证码完成验证；超时未验证将自动移出群聊。';
  }
  return 'Koishi 不启用入群后守卫绑定，后端按 StuHelper 学生认证状态处理加群申请。';
}

function guardSyncLabel(policy: AdmissionPolicy) {
  if (!policy.guardEnabled) return '目标群停用';
  return isPostJoinLocalStrategy(policy)
    ? '同步为 Koishi 入群后守卫'
    : '同步为申请阶段策略';
}

function guardSyncTagType(
  policy: AdmissionPolicy,
): 'danger' | 'info' | 'success' | 'warning' {
  if (!policy.guardEnabled) return 'info';
  return isPostJoinLocalStrategy(policy) ? 'success' : 'warning';
}

function isPostJoinLocalStrategy(policy: AdmissionPolicy) {
  return (
    policy.joinHandlingStrategy === 'post_join_guard' ||
    policy.joinHandlingStrategy === 'post_join_time_code'
  );
}

function isPostJoinGuardStrategy(policy: AdmissionPolicy) {
  return policy.joinHandlingStrategy === 'post_join_guard';
}

function isPostJoinTimeCodeStrategy(policy: AdmissionPolicy) {
  return policy.joinHandlingStrategy === 'post_join_time_code';
}

function usesStudentVerificationFlow(policy: AdmissionPolicy) {
  return isPostJoinGuardStrategy(policy);
}

function linkWaitLabel(policy: AdmissionPolicy) {
  return isPostJoinTimeCodeStrategy(policy)
    ? '验证码等待（秒）'
    : policyFieldLabels.linkWaitSeconds;
}

function usesPostJoinWait(policy: AdmissionPolicy) {
  return isPostJoinGuardStrategy(policy) || isPostJoinTimeCodeStrategy(policy);
}

function saveImpactSummary(policy: AdmissionPolicy) {
  const summary = [
    `目标认证群：${policy.platform.toUpperCase()} ${policy.guildID}`,
    `执行状态：${guardSyncLabel(policy)}`,
    `临时新生凭据：${policy.allowTemporaryFreshman ? '允许' : '不允许'}`,
  ];
  return summary.join('；');
}

function parseCreateGuildIDs() {
  return parseGuildIDText(createPolicyForm.guildIDs);
}

function parseGuildIDText(value: string) {
  return value
    .split('\n')
    .map((value) => value.trim())
    .filter(
      (value, index, values) =>
        value.length > 0 && values.indexOf(value) === index,
    );
}

function openCreatePolicyDialog() {
  createPolicyForm.platform ||= 'qq';
  if (!createPolicyForm.sourcePolicyID) {
    createPolicyForm.sourcePolicyID = policies.value[0]?.id ?? '';
  }
  createPolicyForm.guildIDs = '';
  createPolicyDialogVisible.value = true;
}

async function submitCreatePolicies() {
  if (createPolicySubmitting.value) {
    return;
  }
  const guildIDs = parseCreateGuildIDs();
  if (
    !createPolicyForm.sourcePolicyID ||
    !createPolicyForm.platform ||
    guildIDs.length === 0
  ) {
    handleActionError(new Error('请填写目标认证群号并选择要复制的策略'));
    return;
  }

  createPolicySubmitting.value = true;
  actionError.value = '';
  try {
    for (const guildID of guildIDs) {
      await createAdmissionPolicy({
        guildID,
        platform: createPolicyForm.platform,
        sourcePolicyID: createPolicyForm.sourcePolicyID,
      });
    }
    ElMessage.success(`已创建 ${guildIDs.length} 个入群认证群策略`);
    createPolicyDialogVisible.value = false;
    await fetchData();
  } catch (error) {
    handleActionError(error);
  } finally {
    createPolicySubmitting.value = false;
  }
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
      autoApproveJoin: isPostJoinLocalStrategy(policy),
      autoApproveVerifiedJoin: true,
      autoApproveUnverifiedJoin: isPostJoinLocalStrategy(policy),
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
    description="控制目标群入群处理方式、入群后等待时长和学生认证审核行为。"
    title="入群认证策略"
    :total="policies.length"
  >
    <template #actions>
      <ElButton
        type="primary"
        :disabled="loading || policies.length === 0"
        @click="openCreatePolicyDialog"
      >
        新增目标认证群
      </ElButton>
    </template>

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
        <header
          class="flex flex-col gap-3 border-b border-slate-200 pb-4 lg:flex-row lg:items-start lg:justify-between"
        >
          <div>
            <div class="flex flex-wrap items-center gap-2">
              <h2 class="text-base font-semibold text-slate-900">
                {{ policy.platform.toUpperCase() }} 群 {{ policy.guildID }}
              </h2>
              <ElTag
                size="small"
                :type="guardSyncTagType(policy)"
                data-policy-sync-source="admin"
              >
                {{ guardSyncLabel(policy) }}
              </ElTag>
            </div>
            <p class="mt-1 text-sm text-slate-500">
              Admin 是入群认证策略权威源。Koishi WebUI
              只显示同步后的执行态和现场队列。
            </p>
          </div>
          <div
            class="grid min-w-[220px] gap-1 text-sm text-slate-600"
            data-policy-summary
          >
            <span>目标认证群：{{ policy.guildID }}</span>
            <span>入群处理：{{ joinHandlingStrategyLabel(policy) }}</span>
            <span>
              临时新生凭据：{{
                policy.allowTemporaryFreshman ? '允许入群' : '不允许入群'
              }}
            </span>
            <span v-if="isPostJoinTimeCodeStrategy(policy)">
              验证码等待：{{ policy.linkWaitSeconds }} 秒
            </span>
          </div>
        </header>

        <section class="grid gap-4 border-b border-slate-200 py-4">
          <div>
            <h3 class="text-sm font-semibold text-slate-900">入群处理</h3>
            <p class="mt-1 text-sm text-slate-500">
              这里配置目标认证群本身。不要把材料审核通知群填到目标认证群里。
            </p>
          </div>
          <div class="grid gap-4 lg:grid-cols-2">
            <ElFormItem :label="policyFieldLabels.guardEnabled">
              <ElSwitch
                v-model="policy.guardEnabled"
                active-text="启用"
                inactive-text="停用目标群"
              />
            </ElFormItem>
            <ElFormItem :label="policyFieldLabels.joinHandlingStrategy">
              <ElRadioGroup v-model="policy.joinHandlingStrategy">
                <ElRadioButton
                  v-for="option in joinHandlingStrategyOptions"
                  :key="option.value"
                  :label="option.value"
                >
                  {{ option.label }}
                </ElRadioButton>
              </ElRadioGroup>
              <p class="mt-2 text-xs leading-5 text-slate-500">
                {{ joinHandlingStrategyHelp(policy) }}
              </p>
            </ElFormItem>
          </div>
          <ElFormItem
            v-if="policy.joinHandlingStrategy === 'join_request_review'"
            :label="policyFieldLabels.unverifiedJoinRejectReason"
          >
            <ElInput
              v-model="policy.unverifiedJoinRejectReason"
              maxlength="120"
              show-word-limit
            />
          </ElFormItem>
        </section>

        <section
          v-if="usesPostJoinWait(policy)"
          class="grid gap-4 border-b border-slate-200 py-4"
        >
          <div>
            <h3 class="text-sm font-semibold text-slate-900">时间与提醒</h3>
            <p class="mt-1 text-sm text-slate-500">
              {{
                isPostJoinTimeCodeStrategy(policy)
                  ? '验证码等待时间控制成员入群后必须完成动态验证码的期限，超时后 Koishi 会移出成员。'
                  : '这些时间控制入群后禁言、链接绑定、材料提交、人工审核和提醒节奏。'
              }}
            </p>
          </div>
          <div
            class="grid gap-4 md:grid-cols-2"
            :class="
              isPostJoinTimeCodeStrategy(policy)
                ? 'xl:grid-cols-2'
                : 'xl:grid-cols-5'
            "
          >
            <ElFormItem
              v-if="isPostJoinGuardStrategy(policy)"
              :label="policyFieldLabels.initialMuteDurationSeconds"
            >
              <ElInputNumber
                v-model="policy.initialMuteDurationSeconds"
                :min="1"
              />
            </ElFormItem>
            <ElFormItem :label="linkWaitLabel(policy)">
              <ElInputNumber v-model="policy.linkWaitSeconds" :min="1" />
              <p
                v-if="isPostJoinTimeCodeStrategy(policy)"
                class="mt-2 text-xs leading-5 text-slate-500"
              >
                Koishi 同步后会按该值创建群内验证码挑战的超时踢出期限。
              </p>
            </ElFormItem>
            <ElFormItem
              v-if="isPostJoinGuardStrategy(policy)"
              :label="policyFieldLabels.submissionWaitSeconds"
            >
              <ElInputNumber v-model="policy.submissionWaitSeconds" :min="1" />
            </ElFormItem>
            <ElFormItem
              v-if="isPostJoinGuardStrategy(policy)"
              :label="policyFieldLabels.manualReviewTimeoutSeconds"
            >
              <ElInputNumber
                v-model="policy.manualReviewTimeoutSeconds"
                :min="1"
              />
            </ElFormItem>
            <ElFormItem
              v-if="isPostJoinGuardStrategy(policy)"
              :label="policyFieldLabels.reminderIntervalSeconds"
            >
              <ElInputNumber
                v-model="policy.reminderIntervalSeconds"
                :min="1"
              />
            </ElFormItem>
          </div>
        </section>

        <section class="grid gap-4 border-b border-slate-200 py-4">
          <div>
            <h3 class="text-sm font-semibold text-slate-900">学生资格策略</h3>
            <p class="mt-1 text-sm text-slate-500">
              正式学生凭据始终可满足学生资格。临时新生凭据仅在本开关启用且凭据仍有效时可入群；人工材料审核在独立学生认证模块完成。
            </p>
          </div>
          <ElFormItem :label="policyFieldLabels.allowTemporaryFreshman">
            <ElSwitch
              v-model="policy.allowTemporaryFreshman"
              active-text="允许"
              inactive-text="仅正式学生"
            />
          </ElFormItem>
        </section>

        <section
          v-if="usesStudentVerificationFlow(policy)"
          class="grid gap-4 border-b border-slate-200 py-4"
        >
          <div>
            <h3 class="text-sm font-semibold text-slate-900">失败与限制</h3>
            <p class="mt-1 text-sm text-slate-500">
              控制多次未认证和自动黑名单。凭据延期与人工材料由学生认证管理面负责。
            </p>
          </div>
          <div class="grid gap-4 md:grid-cols-2">
            <ElFormItem :label="policyFieldLabels.failedJoinLimit">
              <ElInputNumber v-model="policy.failedJoinLimit" :min="1" />
            </ElFormItem>
            <ElFormItem :label="policyFieldLabels.blacklistDurationSeconds">
              <ElInputNumber
                v-model="policy.blacklistDurationSeconds"
                :min="0"
              />
            </ElFormItem>
          </div>
        </section>

        <footer
          class="flex flex-col gap-3 pt-4 lg:flex-row lg:items-center lg:justify-between"
        >
          <p class="text-sm leading-6 text-slate-500" data-policy-save-impact>
            保存影响：{{ saveImpactSummary(policy) }}。Koishi
            会在下次同步后显示执行态。
          </p>
          <ElButton
            type="primary"
            :disabled="savingPolicyIDs[policy.id]"
            :loading="savingPolicyIDs[policy.id]"
            @click="savePolicy(policy)"
          >
            保存
          </ElButton>
        </footer>
      </ElForm>

      <p class="text-sm text-slate-500">
        成员黑名单管理已迁移至独立页面：「用户系统 → 成员黑名单」。
      </p>
    </div>

    <ElDialog
      v-model="createPolicyDialogVisible"
      title="新增目标认证群"
      width="520px"
      :teleported="false"
    >
      <ElForm label-position="top">
        <ElFormItem label="复制策略">
          <ElSelect
            v-model="createPolicyForm.sourcePolicyID"
            placeholder="选择已有策略"
            class="w-full"
          >
            <ElOption
              v-for="option in createSourceOptions"
              :key="option.value"
              :label="option.label"
              :value="option.value"
            />
          </ElSelect>
        </ElFormItem>
        <ElFormItem label="平台">
          <ElInput v-model="createPolicyForm.platform" placeholder="qq" />
        </ElFormItem>
        <ElFormItem label="目标认证群号">
          <ElInput
            v-model="createPolicyForm.guildIDs"
            placeholder="每行一个需要开启入群认证的 QQ 群号"
            :rows="4"
            type="textarea"
          />
        </ElFormItem>
      </ElForm>
      <template #footer>
        <ElButton @click="createPolicyDialogVisible = false">取消</ElButton>
        <ElButton
          type="primary"
          :loading="createPolicySubmitting"
          @click="submitCreatePolicies"
        >
          创建
        </ElButton>
      </template>
    </ElDialog>
  </AdminContentLayout>
</template>
