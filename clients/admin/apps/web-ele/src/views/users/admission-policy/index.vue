<script setup lang="ts">
import type { AdmissionPolicy } from '#/api/admin';

import { computed, reactive, ref } from 'vue';

import {
  ElAlert,
  ElButton,
  ElDatePicker,
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
import { useAdminAction } from '#/composables/use-admin-action';
import { adminErrorMessage, useAdminLoad } from '#/composables/use-admin-list';
import { $t } from '#/locales';

import AdminContentLayout from '../../shared/AdminContentLayout.vue';

const managementGuildText = reactive<Record<string, string>>({});
const createPolicyDialogVisible = ref(false);
const createPolicyForm = reactive({
  guildIDs: '',
  platform: 'qq',
  sourcePolicyID: '',
});

const POLICY_DATETIME_FORMAT = 'YYYY-MM-DDTHH:mm:ssZ';

const policyFieldLabels = computed(() => ({
  blacklistDurationSeconds: $t(
    'admin.users.admissionPolicy.fields.blacklistDurationSeconds',
  ),
  failedJoinLimit: $t('admin.users.admissionPolicy.fields.failedJoinLimit'),
  forwardRawMaterialToQQ: $t(
    'admin.users.admissionPolicy.fields.forwardRawMaterialToQQ',
  ),
  freshmanChannelClosesAt: $t(
    'admin.users.admissionPolicy.fields.freshmanChannelClosesAt',
  ),
  freshmanChannelEnabled: $t(
    'admin.users.admissionPolicy.fields.freshmanChannelEnabled',
  ),
  freshmanDefaultExpiresAt: $t(
    'admin.users.admissionPolicy.fields.freshmanDefaultExpiresAt',
  ),
  guardEnabled: $t('admin.users.admissionPolicy.fields.guardEnabled'),
  initialMuteDurationSeconds: $t(
    'admin.users.admissionPolicy.fields.initialMuteDurationSeconds',
  ),
  joinHandlingStrategy: $t(
    'admin.users.admissionPolicy.fields.joinHandlingStrategy',
  ),
  linkWaitSeconds: $t('admin.users.admissionPolicy.fields.linkWaitSeconds'),
  managementGuildIDs: $t(
    'admin.users.admissionPolicy.fields.managementGuildIDs',
  ),
  manualReviewTimeoutSeconds: $t(
    'admin.users.admissionPolicy.fields.manualReviewTimeoutSeconds',
  ),
  maxExtensionDays: $t('admin.users.admissionPolicy.fields.maxExtensionDays'),
  maxMaterialBytes: $t('admin.users.admissionPolicy.fields.maxMaterialBytes'),
  reminderIntervalSeconds: $t(
    'admin.users.admissionPolicy.fields.reminderIntervalSeconds',
  ),
  submissionWaitSeconds: $t(
    'admin.users.admissionPolicy.fields.submissionWaitSeconds',
  ),
  unverifiedJoinRejectReason: $t(
    'admin.users.admissionPolicy.fields.unverifiedJoinRejectReason',
  ),
}));

const joinHandlingStrategyOptions = computed(
  () =>
    [
      {
        label: $t('admin.users.admissionPolicy.strategyPostJoinGuard'),
        value: 'post_join_guard',
      },
      {
        label: $t('admin.users.admissionPolicy.strategyJoinRequestReview'),
        value: 'join_request_review',
      },
      {
        label: $t('admin.users.admissionPolicy.strategyPostJoinTimeCode'),
        value: 'post_join_time_code',
      },
    ] as const,
);

type AdmissionPolicyBoundary = Omit<AdmissionPolicy, 'managementGuildIDs'> & {
  joinHandlingStrategy?: AdmissionPolicy['joinHandlingStrategy'];
  managementGuildIDs?: null | string[];
  unverifiedJoinRejectReason?: null | string;
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
    joinHandlingStrategy: policy.joinHandlingStrategy ?? 'post_join_guard',
    managementGuildIDs: normalizeManagementGuildIDs(policy.managementGuildIDs),
    // F030：留空保持留空，前端不注入默认文案——否则保存会把客户端默认值
    // 静默写回后端。默认提示放输入框 placeholder，由后端落默认。
    unverifiedJoinRejectReason: policy.unverifiedJoinRejectReason ?? '',
  };
}

const {
  data: loadedPolicies,
  load: fetchData,
  loadError,
  loading,
} = useAdminLoad(async () => {
  const data = await listAdmissionPolicies();
  const normalized = data.map((policy) => normalizePolicy(policy));
  for (const policy of normalized) {
    managementGuildText[policy.id] = policy.managementGuildIDs.join('\n');
  }
  const firstPolicy = normalized[0];
  if (!createPolicyForm.sourcePolicyID && firstPolicy) {
    createPolicyForm.sourcePolicyID = firstPolicy.id;
  }
  return normalized;
});

const policies = computed(() => loadedPolicies.value ?? []);

const { actionError, clearActionError, isActionPending, runAction } =
  useAdminAction();

const createSourceOptions = computed(() =>
  policies.value.map((policy) => ({
    label: $t('admin.users.admissionPolicy.sourceOptionLabel', {
      guild: policy.guildID,
      platform: policy.platform.toUpperCase(),
    }),
    value: policy.id,
  })),
);

function parseManagementGuildIDs(policyID: string) {
  return parseGuildIDText(managementGuildText[policyID] ?? '');
}

function managementGuildCount(policy: AdmissionPolicy) {
  return parseManagementGuildIDs(policy.id).length;
}

function joinHandlingStrategyLabel(policy: AdmissionPolicy) {
  return (
    joinHandlingStrategyOptions.value.find(
      (option) => option.value === policy.joinHandlingStrategy,
    )?.label ?? policy.joinHandlingStrategy
  );
}

function joinHandlingStrategyHelp(policy: AdmissionPolicy) {
  if (policy.joinHandlingStrategy === 'post_join_guard') {
    return $t('admin.users.admissionPolicy.strategyHelpPostJoinGuard');
  }
  if (policy.joinHandlingStrategy === 'post_join_time_code') {
    return $t('admin.users.admissionPolicy.strategyHelpPostJoinTimeCode');
  }
  return $t('admin.users.admissionPolicy.strategyHelpJoinRequestReview');
}

function guardSyncLabel(policy: AdmissionPolicy) {
  if (!policy.guardEnabled) {
    return $t('admin.users.admissionPolicy.syncDisabled');
  }
  if (policy.joinHandlingStrategy === 'post_join_guard') {
    return $t('admin.users.admissionPolicy.syncPostJoinGuard');
  }
  if (policy.joinHandlingStrategy === 'post_join_time_code') {
    return $t('admin.users.admissionPolicy.syncPostJoinTimeCode');
  }
  return $t('admin.users.admissionPolicy.syncJoinRequestReview');
}

function guardSyncTagType(
  policy: AdmissionPolicy,
): 'danger' | 'info' | 'success' | 'warning' {
  if (!policy.guardEnabled) return 'info';
  return isPostJoinLocalStrategy(policy)
    ? 'success'
    : 'warning';
}

function isPostJoinLocalStrategy(policy: AdmissionPolicy) {
  return (
    policy.joinHandlingStrategy === 'post_join_guard' ||
    policy.joinHandlingStrategy === 'post_join_time_code'
  );
}

function saveImpactSummary(policy: AdmissionPolicy) {
  return [
    $t('admin.users.admissionPolicy.impactTarget', {
      guild: policy.guildID,
      platform: policy.platform.toUpperCase(),
    }),
    $t('admin.users.admissionPolicy.impactState', {
      state: guardSyncLabel(policy),
    }),
    $t('admin.users.admissionPolicy.impactReviewGuilds', {
      count: managementGuildCount(policy),
    }),
  ].join('；');
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
  const firstPolicy = policies.value[0];
  if (!createPolicyForm.sourcePolicyID && firstPolicy) {
    createPolicyForm.sourcePolicyID = firstPolicy.id;
  }
  createPolicyForm.guildIDs = '';
  createPolicyDialogVisible.value = true;
}

async function submitCreatePolicies() {
  const guildIDs = parseCreateGuildIDs();
  if (
    !createPolicyForm.sourcePolicyID ||
    !createPolicyForm.platform ||
    guildIDs.length === 0
  ) {
    ElMessage.warning($t('admin.users.admissionPolicy.formIncomplete'));
    return;
  }

  const failures: Array<{ guildID: string; message: string }> = [];
  const succeeded = await runAction(
    async () => {
      for (const guildID of guildIDs) {
        try {
          await createAdmissionPolicy({
            guildID,
            platform: createPolicyForm.platform,
            sourcePolicyID: createPolicyForm.sourcePolicyID,
          });
        } catch (error) {
          failures.push({ guildID, message: adminErrorMessage(error) });
        }
      }
      if (failures.length > 0) {
        // F029：失败群号回填输入框，重试只补失败项；已创建的群号
        // 不会被重复提交，错误信息逐群列出失败原因。
        createPolicyForm.guildIDs = failures
          .map((failure) => failure.guildID)
          .join('\n');
        throw new Error(
          $t('admin.users.admissionPolicy.createPartialFailure', {
            created: guildIDs.length - failures.length,
            failed: failures.length,
            failures: failures
              .map((failure) => `${failure.guildID}（${failure.message}）`)
              .join('；'),
          }),
        );
      }
    },
    {
      successMessage: $t('admin.users.admissionPolicy.createSuccess', {
        count: guildIDs.length,
      }),
    },
  );

  if (succeeded) {
    createPolicyDialogVisible.value = false;
  }
  if (succeeded || failures.length < guildIDs.length) {
    await fetchData();
  }
}

async function savePolicy(policy: AdmissionPolicy) {
  const succeeded = await runAction(
    () =>
      updateAdmissionPolicy({
        ...policy,
        autoApproveJoin: isPostJoinLocalStrategy(policy),
        autoApproveVerifiedJoin: true,
        autoApproveUnverifiedJoin: isPostJoinLocalStrategy(policy),
        managementGuildIDs: parseManagementGuildIDs(policy.id),
        unverifiedJoinRejectReason:
          policy.unverifiedJoinRejectReason?.trim() ?? '',
      }),
    {
      id: policy.id,
      successMessage: $t('admin.users.admissionPolicy.saveSuccess', {
        guild: policy.guildID,
        platform: policy.platform.toUpperCase(),
      }),
    },
  );
  if (succeeded) {
    await fetchData();
  }
}
</script>

<template>
  <AdminContentLayout
    :description="$t('admin.users.admissionPolicy.description')"
    :title="$t('admin.routes.userSystem.admissionPolicy')"
    :total="policies.length"
  >
    <template #actions>
      <ElButton
        type="primary"
        :disabled="loading || policies.length === 0"
        @click="openCreatePolicyDialog"
      >
        {{ $t('admin.users.admissionPolicy.createButton') }}
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
      @close="clearActionError"
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
                {{
                  $t('admin.users.admissionPolicy.policyCardTitle', {
                    guild: policy.guildID,
                    platform: policy.platform.toUpperCase(),
                  })
                }}
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
              {{ $t('admin.users.admissionPolicy.authorityNote') }}
            </p>
          </div>
          <div
            class="grid min-w-[220px] gap-1 text-sm text-slate-600"
            data-policy-summary
          >
            <span>
              {{
                $t('admin.users.admissionPolicy.summaryTargetGuild', {
                  guild: policy.guildID,
                })
              }}
            </span>
            <span>
              {{
                $t('admin.users.admissionPolicy.summaryStrategy', {
                  label: joinHandlingStrategyLabel(policy),
                })
              }}
            </span>
            <span>
              {{
                $t('admin.users.admissionPolicy.summaryReviewGuilds', {
                  count: managementGuildCount(policy),
                })
              }}
            </span>
          </div>
        </header>

        <section class="grid gap-4 border-b border-slate-200 py-4">
          <div>
            <h3 class="text-sm font-semibold text-slate-900">
              {{ $t('admin.users.admissionPolicy.sectionJoinHandling') }}
            </h3>
            <p class="mt-1 text-sm text-slate-500">
              {{ $t('admin.users.admissionPolicy.sectionJoinHandlingNote') }}
            </p>
          </div>
          <div class="grid gap-4 lg:grid-cols-2">
            <ElFormItem :label="policyFieldLabels.guardEnabled">
              <ElSwitch
                v-model="policy.guardEnabled"
                :active-text="$t('admin.users.admissionPolicy.guardActive')"
                :inactive-text="$t('admin.users.admissionPolicy.guardInactive')"
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
              :placeholder="
                $t('admin.users.admissionPolicy.unverifiedRejectPlaceholder')
              "
              show-word-limit
            />
          </ElFormItem>
        </section>

        <section class="grid gap-4 border-b border-slate-200 py-4">
          <div>
            <h3 class="text-sm font-semibold text-slate-900">
              {{ $t('admin.users.admissionPolicy.sectionTiming') }}
            </h3>
            <p class="mt-1 text-sm text-slate-500">
              {{ $t('admin.users.admissionPolicy.sectionTimingNote') }}
            </p>
          </div>
          <div class="grid gap-4 md:grid-cols-2 xl:grid-cols-5">
            <ElFormItem :label="policyFieldLabels.initialMuteDurationSeconds">
              <ElInputNumber
                v-model="policy.initialMuteDurationSeconds"
                :min="1"
              />
            </ElFormItem>
            <ElFormItem :label="policyFieldLabels.linkWaitSeconds">
              <ElInputNumber v-model="policy.linkWaitSeconds" :min="1" />
            </ElFormItem>
            <ElFormItem :label="policyFieldLabels.submissionWaitSeconds">
              <ElInputNumber v-model="policy.submissionWaitSeconds" :min="1" />
            </ElFormItem>
            <ElFormItem :label="policyFieldLabels.manualReviewTimeoutSeconds">
              <ElInputNumber
                v-model="policy.manualReviewTimeoutSeconds"
                :min="1"
              />
            </ElFormItem>
            <ElFormItem :label="policyFieldLabels.reminderIntervalSeconds">
              <ElInputNumber
                v-model="policy.reminderIntervalSeconds"
                :min="1"
              />
            </ElFormItem>
          </div>
        </section>

        <section class="grid gap-4 border-b border-slate-200 py-4">
          <div>
            <h3 class="text-sm font-semibold text-slate-900">
              {{ $t('admin.users.admissionPolicy.sectionMaterial') }}
            </h3>
            <p class="mt-1 text-sm text-slate-500">
              {{ $t('admin.users.admissionPolicy.sectionMaterialNote') }}
            </p>
          </div>
          <div class="grid gap-4 lg:grid-cols-3">
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
          </div>
          <div
            class="grid gap-4 lg:grid-cols-[minmax(0,2fr)_minmax(260px,1fr)]"
          >
            <ElFormItem :label="policyFieldLabels.managementGuildIDs">
              <ElInput
                v-model="managementGuildText[policy.id]"
                :placeholder="
                  $t('admin.users.admissionPolicy.managementGuildPlaceholder')
                "
                :rows="3"
                type="textarea"
              />
              <p class="mt-2 text-xs leading-5 text-slate-500">
                {{
                  $t('admin.users.admissionPolicy.managementGuildHint', {
                    count: managementGuildCount(policy),
                  })
                }}
              </p>
            </ElFormItem>
            <ElFormItem :label="policyFieldLabels.forwardRawMaterialToQQ">
              <ElSwitch v-model="policy.forwardRawMaterialToQQ" />
            </ElFormItem>
          </div>
        </section>

        <section class="grid gap-4 border-b border-slate-200 py-4">
          <div>
            <h3 class="text-sm font-semibold text-slate-900">
              {{ $t('admin.users.admissionPolicy.sectionLimits') }}
            </h3>
            <p class="mt-1 text-sm text-slate-500">
              {{ $t('admin.users.admissionPolicy.sectionLimitsNote') }}
            </p>
          </div>
          <div class="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
            <ElFormItem :label="policyFieldLabels.failedJoinLimit">
              <ElInputNumber v-model="policy.failedJoinLimit" :min="1" />
            </ElFormItem>
            <ElFormItem :label="policyFieldLabels.blacklistDurationSeconds">
              <ElInputNumber
                v-model="policy.blacklistDurationSeconds"
                :min="0"
              />
            </ElFormItem>
            <ElFormItem :label="policyFieldLabels.maxMaterialBytes">
              <ElInputNumber v-model="policy.maxMaterialBytes" :min="1" />
            </ElFormItem>
            <ElFormItem :label="policyFieldLabels.maxExtensionDays">
              <ElInputNumber v-model="policy.maxExtensionDays" :min="1" />
            </ElFormItem>
          </div>
        </section>

        <footer
          class="flex flex-col gap-3 pt-4 lg:flex-row lg:items-center lg:justify-between"
        >
          <p class="text-sm leading-6 text-slate-500" data-policy-save-impact>
            {{
              $t('admin.users.admissionPolicy.saveImpact', {
                summary: saveImpactSummary(policy),
              })
            }}
          </p>
          <ElButton
            type="primary"
            :disabled="isActionPending(policy.id)"
            :loading="isActionPending(policy.id)"
            @click="savePolicy(policy)"
          >
            {{ $t('admin.common.save') }}
          </ElButton>
        </footer>
      </ElForm>

      <p class="text-sm text-slate-500">
        {{ $t('admin.users.admissionPolicy.blacklistMovedNote') }}
      </p>
    </div>

    <ElDialog
      v-model="createPolicyDialogVisible"
      :title="$t('admin.users.admissionPolicy.dialogTitle')"
      width="520px"
      :teleported="false"
    >
      <ElForm label-position="top">
        <ElFormItem
          :label="$t('admin.users.admissionPolicy.dialogSourcePolicy')"
        >
          <ElSelect
            v-model="createPolicyForm.sourcePolicyID"
            :placeholder="
              $t('admin.users.admissionPolicy.dialogSourcePlaceholder')
            "
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
        <ElFormItem :label="$t('admin.users.admissionPolicy.dialogPlatform')">
          <ElInput v-model="createPolicyForm.platform" placeholder="qq" />
        </ElFormItem>
        <ElFormItem :label="$t('admin.users.admissionPolicy.dialogGuildIDs')">
          <ElInput
            v-model="createPolicyForm.guildIDs"
            :placeholder="
              $t('admin.users.admissionPolicy.dialogGuildIDsPlaceholder')
            "
            :rows="4"
            type="textarea"
          />
        </ElFormItem>
      </ElForm>
      <template #footer>
        <ElButton @click="createPolicyDialogVisible = false">
          {{ $t('admin.common.cancel') }}
        </ElButton>
        <ElButton
          type="primary"
          :loading="isActionPending()"
          @click="submitCreatePolicies"
        >
          {{ $t('admin.users.admissionPolicy.dialogCreate') }}
        </ElButton>
      </template>
    </ElDialog>
  </AdminContentLayout>
</template>
