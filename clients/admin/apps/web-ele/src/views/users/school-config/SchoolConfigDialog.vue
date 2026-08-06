<script setup lang="ts">
import type { AdminVerificationSchoolConfig } from '#/api/admin';

import { reactive, watch } from 'vue';

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
  ElSelect,
  ElSwitch,
} from 'element-plus';

export type SchoolVerificationDraft = {
  adapterID: string;
  adapterVersion: string;
  emailDomains: string[];
  enrollmentPolicy: Record<string, unknown>;
  manualFormSchema: Record<string, unknown>;
  nameMatchPolicy: Record<string, unknown>;
  reason: string;
  schoolCode: string;
  snapshotAutoActivate: boolean;
  snapshotGraceSeconds: number;
  snapshotHardExpirySeconds: number;
  snapshotSyncIntervalSeconds: number;
  snapshotWarningAfterSeconds: number;
  studentIDPolicy: Record<string, unknown>;
};

const props = defineProps<{
  school: AdminVerificationSchoolConfig | null;
  submitting: boolean;
}>();
const emit = defineEmits<{
  (event: 'save', draft: SchoolVerificationDraft): void;
}>();
const visible = defineModel<boolean>('visible', { required: true });

const form = reactive({
  adapterID: 'declarative',
  adapterVersion: '1',
  emailDomainsText: '',
  enrollmentPolicyText: '',
  manualFormSchemaText: '{}',
  nameMatchPolicyText: '{\n  "strategy": "exact_trimmed"\n}',
  reason: '',
  schoolCode: '',
  snapshotGraceSeconds: 0,
  snapshotHardExpirySeconds: 1_209_600,
  snapshotAutoActivate: false,
  snapshotSyncIntervalSeconds: 604_800,
  snapshotWarningAfterSeconds: 691_200,
  studentIDPolicyText:
    '{\n  "strategy": "regex",\n  "pattern": "^[0-9]{8}$",\n  "transform": "none"\n}',
});

function defaultEnrollmentPolicy() {
  return JSON.stringify(
    {
      rosterEligibleCodes: ['eligible'],
      rosterKnownEligibilityCodes: ['eligible', 'ineligible'],
      rosterMaximumRowDeltaRatio: 0.25,
      rosterMinimumRows: 1,
      rosterRequireCurrentMarker: true,
    },
    null,
    2,
  );
}

watch(
  () => [props.school, visible.value] as const,
  ([school, isVisible]) => {
    if (!isVisible) return;
    form.reason = '';
    if (!school) {
      form.schoolCode = '';
      form.adapterID = 'declarative';
      form.adapterVersion = '1';
      form.emailDomainsText = '';
      form.studentIDPolicyText =
        '{\n  "strategy": "regex",\n  "pattern": "^[0-9]{8}$",\n  "transform": "none"\n}';
      form.nameMatchPolicyText = '{\n  "strategy": "exact_trimmed"\n}';
      form.enrollmentPolicyText = defaultEnrollmentPolicy();
      form.manualFormSchemaText = '{}';
      form.snapshotSyncIntervalSeconds = 604_800;
      form.snapshotWarningAfterSeconds = 691_200;
      form.snapshotHardExpirySeconds = 1_209_600;
      form.snapshotGraceSeconds = 0;
      form.snapshotAutoActivate = false;
      return;
    }
    form.schoolCode = school.schoolCode;
    form.adapterID = school.adapterID;
    form.adapterVersion = school.adapterVersion;
    form.emailDomainsText = school.emailDomains.join('\n');
    form.studentIDPolicyText = JSON.stringify(school.studentIDPolicy, null, 2);
    form.nameMatchPolicyText = JSON.stringify(school.nameMatchPolicy, null, 2);
    form.enrollmentPolicyText = JSON.stringify(
      school.enrollmentPolicy,
      null,
      2,
    );
    form.manualFormSchemaText = JSON.stringify(
      school.manualFormSchema,
      null,
      2,
    );
    form.snapshotSyncIntervalSeconds = school.snapshotSyncIntervalSeconds;
    form.snapshotWarningAfterSeconds = school.snapshotWarningAfterSeconds;
    form.snapshotHardExpirySeconds = school.snapshotHardExpirySeconds;
    form.snapshotGraceSeconds = school.snapshotGraceSeconds;
    form.snapshotAutoActivate = school.snapshotAutoActivate;
  },
  { immediate: true },
);

function parseObject(label: string, value: string) {
  try {
    const parsed = JSON.parse(value) as unknown;
    if (!parsed || Array.isArray(parsed) || typeof parsed !== 'object') {
      throw new Error('not an object');
    }
    return parsed as Record<string, unknown>;
  } catch {
    ElMessage.warning(`${label}必须是合法的 JSON 对象。`);
    return null;
  }
}

function submit() {
  const reason = form.reason.trim();
  const schoolCode = form.schoolCode.trim();
  if (!/^\d{10}$/.test(schoolCode)) {
    ElMessage.warning('请输入 10 位学校代码，且该学校必须已存在于学校目录。');
    return;
  }
  if (reason.length < 4) {
    ElMessage.warning('请输入至少 4 个字的变更原因或工单编号。');
    return;
  }
  const studentIDPolicy = parseObject('学号策略', form.studentIDPolicyText);
  const nameMatchPolicy = parseObject('姓名策略', form.nameMatchPolicyText);
  const enrollmentPolicy = parseObject(
    '学籍准入策略',
    form.enrollmentPolicyText,
  );
  const manualFormSchema = parseObject(
    '人工表单 schema',
    form.manualFormSchemaText,
  );
  if (
    !studentIDPolicy ||
    !nameMatchPolicy ||
    !enrollmentPolicy ||
    !manualFormSchema
  )
    return;

  emit('save', {
    schoolCode,
    adapterID: form.adapterID.trim(),
    adapterVersion: form.adapterVersion.trim(),
    emailDomains: form.emailDomainsText
      .split(/[\n,]/)
      .map((value) => value.trim().toLowerCase())
      .filter(Boolean),
    studentIDPolicy,
    nameMatchPolicy,
    enrollmentPolicy,
    manualFormSchema,
    snapshotSyncIntervalSeconds: form.snapshotSyncIntervalSeconds,
    snapshotWarningAfterSeconds: form.snapshotWarningAfterSeconds,
    snapshotHardExpirySeconds: form.snapshotHardExpirySeconds,
    snapshotGraceSeconds: form.snapshotGraceSeconds,
    snapshotAutoActivate: form.snapshotAutoActivate,
    reason,
  });
}
</script>

<template>
  <ElDialog
    v-model="visible"
    :title="school ? `编辑 ${school.schoolName}` : '创建学校认证配置'"
    width="min(820px, 94vw)"
    destroy-on-close
  >
    <ElAlert
      type="info"
      :closable="false"
      show-icon
      title="学校目录不等于认证白名单"
      description="创建或编辑后配置会强制停用并回到待校验状态。这里不接收 LDAP、Oracle 地址或任何口令；内网目标与 secret 由校园连接器基础设施单独管理。"
      class="config-alert"
    />
    <ElForm label-position="top" class="school-form">
      <div class="form-grid form-grid--three">
        <ElFormItem label="学校代码">
          <ElInput
            v-model="form.schoolCode"
            maxlength="10"
            :disabled="Boolean(school)"
            placeholder="10 位教育机构代码"
          />
        </ElFormItem>
        <ElFormItem label="学校适配器">
          <ElSelect v-model="form.adapterID" style="width: 100%">
            <ElOption label="声明式适配器" value="declarative" />
            <ElOption label="北航代码适配器" value="buaa" />
          </ElSelect>
        </ElFormItem>
        <ElFormItem label="适配器版本">
          <ElInput v-model="form.adapterVersion" maxlength="64" />
        </ElFormItem>
      </div>

      <ElFormItem label="学校邮箱域名">
        <ElInput
          v-model="form.emailDomainsText"
          type="textarea"
          :rows="2"
          placeholder="每行一个域名，例如 example.edu.cn；不要填写 @、通配符或 URL"
        />
      </ElFormItem>

      <div class="json-grid">
        <ElFormItem label="学号规范化与校验策略">
          <ElInput
            v-model="form.studentIDPolicyText"
            type="textarea"
            :rows="8"
          />
        </ElFormItem>
        <ElFormItem label="姓名匹配策略">
          <ElInput
            v-model="form.nameMatchPolicyText"
            type="textarea"
            :rows="8"
          />
        </ElFormItem>
        <ElFormItem label="学籍准入与质量门禁策略">
          <ElInput
            v-model="form.enrollmentPolicyText"
            type="textarea"
            :rows="12"
          />
        </ElFormItem>
        <ElFormItem label="共享人工审核表单 schema">
          <ElInput
            v-model="form.manualFormSchemaText"
            type="textarea"
            :rows="12"
          />
        </ElFormItem>
      </div>

      <h3 class="form-section-title">快照新鲜度</h3>
      <div class="form-grid form-grid--four">
        <ElFormItem label="同步周期（秒）">
          <ElInputNumber
            v-model="form.snapshotSyncIntervalSeconds"
            :min="1"
            controls-position="right"
          />
        </ElFormItem>
        <ElFormItem label="告警阈值（秒）">
          <ElInputNumber
            v-model="form.snapshotWarningAfterSeconds"
            :min="1"
            controls-position="right"
          />
        </ElFormItem>
        <ElFormItem label="硬失效阈值（秒）">
          <ElInputNumber
            v-model="form.snapshotHardExpirySeconds"
            :min="1"
            controls-position="right"
          />
        </ElFormItem>
        <ElFormItem label="宽限期（秒）">
          <ElInputNumber
            v-model="form.snapshotGraceSeconds"
            :min="0"
            controls-position="right"
          />
        </ElFormItem>
      </div>

      <ElFormItem label="通过质量门禁后自动激活">
        <ElSwitch
          v-model="form.snapshotAutoActivate"
          inline-prompt
          active-text="自动"
          inactive-text="手动"
        />
        <p class="field-help">
          默认关闭。开启后，只有完整快照通过全部质量检查且来源时间未回退时才会自动切换；失败快照仍保持隔离。
        </p>
      </ElFormItem>

      <ElFormItem label="变更原因（必填）">
        <ElInput
          v-model="form.reason"
          maxlength="500"
          show-word-limit
          placeholder="填写工单、变更目标和影响说明"
        />
      </ElFormItem>
    </ElForm>
    <template #footer>
      <ElButton :disabled="submitting" @click="visible = false">取消</ElButton>
      <ElButton type="primary" :loading="submitting" @click="submit">
        保存为待校验草稿
      </ElButton>
    </template>
  </ElDialog>
</template>

<style scoped>
.config-alert {
  margin-bottom: 20px;
}

.field-help {
  margin: 6px 0 0;
  font-size: 12px;
  line-height: 1.6;
  color: var(--el-text-color-secondary);
}

.form-grid,
.json-grid {
  display: grid;
  gap: 0 16px;
}

.form-grid--three {
  grid-template-columns: 1fr 1fr 0.7fr;
}

.form-grid--four {
  grid-template-columns: repeat(4, minmax(0, 1fr));
}

.json-grid {
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.json-grid :deep(textarea) {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 12px;
  line-height: 1.55;
}

.form-section-title {
  margin: 4px 0 14px;
  font-size: 15px;
  color: var(--el-text-color-primary);
}

@media (max-width: 760px) {
  .form-grid--three,
  .form-grid--four,
  .json-grid {
    grid-template-columns: 1fr;
  }
}
</style>
