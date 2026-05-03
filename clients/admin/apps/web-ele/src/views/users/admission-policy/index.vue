<script setup lang="ts">
import type { AdmissionPolicy } from '#/api/admin';

import { onMounted, reactive, ref } from 'vue';

import {
  ElButton,
  ElDatePicker,
  ElForm,
  ElFormItem,
  ElInput,
  ElInputNumber,
  ElMessage,
  ElPopconfirm,
  ElSwitch,
} from 'element-plus';

import {
  listAdmissionPolicies,
  releaseAdmissionBlacklist,
  updateAdmissionPolicy,
} from '#/api/admin';

const loading = ref(false);
const releasing = ref(false);
const policies = ref<AdmissionPolicy[]>([]);
const blacklistQQ = ref('');
const managementGuildText = reactive<Record<string, string>>({});

async function fetchData() {
  loading.value = true;
  try {
    policies.value = await listAdmissionPolicies();
    for (const policy of policies.value) {
      managementGuildText[policy.id] = policy.managementGuildIDs.join('\n');
    }
  } finally {
    loading.value = false;
  }
}

function parseManagementGuildIDs(policyID: string) {
  return (managementGuildText[policyID] ?? '')
    .split('\n')
    .map((value) => value.trim())
    .filter(Boolean);
}

async function savePolicy(policy: AdmissionPolicy) {
  await updateAdmissionPolicy({
    ...policy,
    managementGuildIDs: parseManagementGuildIDs(policy.id),
  });
  await fetchData();
}

async function releaseBlacklist() {
  const qqID = blacklistQQ.value.trim();
  if (!qqID) {
    ElMessage.error('请输入 QQ 号');
    return;
  }

  releasing.value = true;
  try {
    await releaseAdmissionBlacklist(qqID);
    blacklistQQ.value = '';
    ElMessage.success('已解除黑名单');
  } finally {
    releasing.value = false;
  }
}

onMounted(fetchData);
</script>

<template>
  <div v-loading="loading" class="grid gap-4 p-4">
    <ElForm
      v-for="policy in policies"
      :key="policy.id"
      class="rounded border border-slate-200 bg-white p-4"
      label-width="180px"
    >
      <h2 class="mb-4 text-base font-semibold">
        {{ policy.platform }} / {{ policy.guildID }}
      </h2>
      <ElFormItem label="freshmanChannelEnabled">
        <ElSwitch v-model="policy.freshmanChannelEnabled" />
      </ElFormItem>
      <ElFormItem label="freshmanChannelClosesAt">
        <ElDatePicker v-model="policy.freshmanChannelClosesAt" type="datetime" />
      </ElFormItem>
      <ElFormItem label="freshmanDefaultExpiresAt">
        <ElDatePicker v-model="policy.freshmanDefaultExpiresAt" type="datetime" />
      </ElFormItem>
      <ElFormItem label="initialMuteDurationSeconds">
        <ElInputNumber v-model="policy.initialMuteDurationSeconds" />
      </ElFormItem>
      <ElFormItem label="linkWaitSeconds">
        <ElInputNumber v-model="policy.linkWaitSeconds" />
      </ElFormItem>
      <ElFormItem label="submissionWaitSeconds">
        <ElInputNumber v-model="policy.submissionWaitSeconds" />
      </ElFormItem>
      <ElFormItem label="manualReviewTimeoutSeconds">
        <ElInputNumber v-model="policy.manualReviewTimeoutSeconds" />
      </ElFormItem>
      <ElFormItem label="reminderIntervalSeconds">
        <ElInputNumber v-model="policy.reminderIntervalSeconds" />
      </ElFormItem>
      <ElFormItem label="failedJoinLimit">
        <ElInputNumber v-model="policy.failedJoinLimit" />
      </ElFormItem>
      <ElFormItem label="blacklistDurationSeconds">
        <ElInputNumber v-model="policy.blacklistDurationSeconds" />
      </ElFormItem>
      <ElFormItem label="maxMaterialBytes">
        <ElInputNumber v-model="policy.maxMaterialBytes" />
      </ElFormItem>
      <ElFormItem label="maxExtensionDays">
        <ElInputNumber v-model="policy.maxExtensionDays" />
      </ElFormItem>
      <ElFormItem label="managementGuildIDs">
        <ElInput
          v-model="managementGuildText[policy.id]"
          :rows="3"
          type="textarea"
        />
      </ElFormItem>
      <ElFormItem label="forwardRawMaterialToQQ">
        <ElSwitch v-model="policy.forwardRawMaterialToQQ" />
      </ElFormItem>
      <ElButton type="primary" @click="savePolicy(policy)">保存</ElButton>
    </ElForm>

    <section class="rounded border border-slate-200 bg-white p-4">
      <h2 class="mb-4 text-base font-semibold">黑名单解除</h2>
      <div class="flex items-center gap-3">
        <ElInput
          v-model="blacklistQQ"
          data-field="blacklistQQ"
          placeholder="QQ 号"
          style="width: 220px"
        />
        <ElPopconfirm
          title="确认解除该 QQ 的入群黑名单？"
          @confirm="releaseBlacklist"
        >
          <template #reference>
            <ElButton
              data-action="releaseBlacklist"
              :loading="releasing"
              type="warning"
            >
              解除黑名单
            </ElButton>
          </template>
        </ElPopconfirm>
      </div>
    </section>
  </div>
</template>
