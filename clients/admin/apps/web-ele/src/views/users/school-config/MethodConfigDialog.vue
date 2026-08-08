<script setup lang="ts">
import type {
  AdminVerificationMethodConfig,
  VerificationMethod,
} from '#/api/admin';

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
} from 'element-plus';

export type VerificationMethodDraft = {
  adapterID: string;
  adapterVersion: string;
  conditionalPolicy: Record<string, unknown>;
  connectorOperationKey: null | string;
  credentialTTLSeconds: null | number;
  description: string;
  displayName: string;
  method: VerificationMethod;
  privacyNotice: Record<string, unknown>;
  privacyNoticeVersion: string;
  publicFormSchema: Record<string, unknown>;
  reason: string;
  riskPolicy: Record<string, unknown>;
  rosterDependency: 'conditional' | 'independent' | 'required';
};

const props = defineProps<{
  method: AdminVerificationMethodConfig | null;
  submitting: boolean;
}>();
const emit = defineEmits<{
  (event: 'save', draft: VerificationMethodDraft): void;
}>();
const visible = defineModel<boolean>('visible', { required: true });

const form = reactive({
  adapterID: 'shared_manual_review',
  adapterVersion: '1',
  conditionalPolicyText: '{}',
  connectorOperationKey: '',
  credentialTTLSeconds: 31_536_000 as null | number,
  description: '拍摄并提交学校批准的学生材料',
  displayName: '人工材料审核',
  method: 'manual_material_review' as VerificationMethod,
  privacyCategoriesText: '姓名\n学号\n学校邮箱\n实时拍摄的学生材料',
  privacyNoticeVersion: '2026-08-05',
  privacyRetentionSummary:
    '审核材料按学校配置到期删除；学生凭据只保留最小结果。',
  privacySummary: '仅用于核验当前学生身份并处理必要的人工审核。',
  privacyTitle: '学生认证个人信息处理说明',
  publicFormSchemaText: '{}',
  reason: '',
  riskPolicyText: '{}',
  rosterDependency: 'independent' as 'conditional' | 'independent' | 'required',
});

function defaultManualForm() {
  return JSON.stringify(
    {
      fields: [
        {
          inputType: 'text',
          key: 'department',
          label: '学院或院系',
          maxLength: 100,
          required: true,
        },
        {
          inputType: 'text',
          key: 'studentID',
          label: '学号或录取编号',
          maxLength: 64,
          required: true,
        },
        {
          inputType: 'text',
          key: 'name',
          label: '姓名',
          maxLength: 100,
          required: true,
        },
        {
          inputType: 'email',
          key: 'email',
          label: '学校邮箱',
          maxLength: 320,
          required: true,
        },
      ],
    },
    null,
    2,
  );
}

function defaultManualRiskPolicy() {
  return JSON.stringify(
    {
      admissionNoticeMaxCredentialDays: 180,
      allowedMaterialTypes: ['campus_card', 'student_card', 'admission_notice'],
      handoffTTLSeconds: 1800,
      materialRetentionDays: 180,
      maxMaterialBytes: 10_485_760,
      maxMaterials: 3,
      maximumImageDimension: 12_000,
      maximumImagePixels: 40_000_000,
      minimumImageDimension: 320,
      requireEmailVerification: true,
      reviewWindowSeconds: 604_800,
    },
    null,
    2,
  );
}

function setNewMethodDefaults(method: VerificationMethod) {
  form.method = method;
  form.adapterVersion = '1';
  form.conditionalPolicyText = '{}';
  form.connectorOperationKey = '';
  form.credentialTTLSeconds = null;
  form.publicFormSchemaText = '{}';
  form.riskPolicyText = '{}';
  form.rosterDependency = 'required';
  const labels: Record<VerificationMethod, [string, string, string]> = {
    real_name_identity_check: [
      '实名信息校验',
      '使用实名信息完成一次性身份校验',
      'buaa',
    ],
    school_sso: [
      '统一身份认证验证',
      '使用学校统一身份认证账号完成一次性校验',
      'buaa_ldap_bind',
    ],
    student_email_outbound_otp: [
      '学校邮箱接收验证码',
      '向规范学号邮箱发送一次性验证码',
      'buaa',
    ],
    student_email_inbound_challenge: [
      '从学校邮箱发送验证邮件',
      '从规范学号邮箱发送一次性挑战邮件',
      'buaa',
    ],
    manual_material_review: [
      '人工材料审核',
      '拍摄并提交学校批准的学生材料',
      'shared_manual_review',
    ],
  };
  [form.displayName, form.description, form.adapterID] = labels[method];
  if (method === 'school_sso') {
    form.rosterDependency = 'conditional';
    form.conditionalPolicyText = JSON.stringify(
      { requiredAttribute: 'current_student', type: 'adapter_assertion' },
      null,
      2,
    );
    form.connectorOperationKey = 'buaa.ldap.authenticate';
  }
  if (method === 'manual_material_review') {
    form.rosterDependency = 'independent';
    form.credentialTTLSeconds = 31_536_000;
    form.publicFormSchemaText = defaultManualForm();
    form.riskPolicyText = defaultManualRiskPolicy();
  }
}

watch(
  () => [props.method, visible.value] as const,
  ([method, isVisible]) => {
    if (!isVisible) return;
    form.reason = '';
    if (!method) {
      setNewMethodDefaults('manual_material_review');
      return;
    }
    form.method = method.method;
    form.displayName = method.displayName;
    form.description = method.description;
    form.adapterID = method.adapterID;
    form.adapterVersion = method.adapterVersion;
    form.rosterDependency = method.rosterDependency;
    form.conditionalPolicyText = JSON.stringify(
      method.conditionalPolicy,
      null,
      2,
    );
    form.publicFormSchemaText = JSON.stringify(
      method.publicFormSchema,
      null,
      2,
    );
    form.riskPolicyText = JSON.stringify(method.riskPolicy, null, 2);
    form.credentialTTLSeconds = method.credentialTTLSeconds ?? null;
    form.connectorOperationKey = method.connectorOperationKey ?? '';
    form.privacyNoticeVersion = method.privacyNoticeVersion ?? '2026-08-05';
    form.privacyTitle = String(
      method.privacyNotice.title ?? '学生认证个人信息处理说明',
    );
    form.privacySummary = String(
      method.privacyNotice.summary ?? '仅用于本次学生身份校验。',
    );
    form.privacyRetentionSummary = String(
      method.privacyNotice.retentionSummary ??
        '请求原文不持久化；仅保留最小认证结果。',
    );
    const categories = method.privacyNotice.dataCategories;
    form.privacyCategoriesText = Array.isArray(categories)
      ? categories.join('\n')
      : '';
  },
  { immediate: true },
);

function parseObject(label: string, value: string) {
  try {
    const parsed = JSON.parse(value) as unknown;
    if (!parsed || Array.isArray(parsed) || typeof parsed !== 'object')
      throw new Error(`${label} must be a JSON object`);
    return parsed as Record<string, unknown>;
  } catch {
    ElMessage.warning(`${label}必须是合法的 JSON 对象。`);
    return null;
  }
}

function submit() {
  const reason = form.reason.trim();
  if (reason.length < 4) {
    ElMessage.warning('请输入至少 4 个字的变更原因。');
    return;
  }
  const conditionalPolicy = parseObject(
    '名册条件策略',
    form.conditionalPolicyText,
  );
  const publicFormSchema = parseObject(
    '公开表单 schema',
    form.publicFormSchemaText,
  );
  const riskPolicy = parseObject('风险策略', form.riskPolicyText);
  if (!conditionalPolicy || !publicFormSchema || !riskPolicy) return;
  const dataCategories = form.privacyCategoriesText
    .split('\n')
    .map((value) => value.trim())
    .filter(Boolean);
  if (
    !form.privacyNoticeVersion.trim() ||
    !form.privacyTitle.trim() ||
    !form.privacySummary.trim() ||
    !form.privacyRetentionSummary.trim() ||
    dataCategories.length === 0
  ) {
    ElMessage.warning('请完整填写用户可见的个人信息处理说明。');
    return;
  }
  emit('save', {
    method: form.method,
    displayName: form.displayName.trim(),
    description: form.description.trim(),
    adapterID: form.adapterID.trim(),
    adapterVersion: form.adapterVersion.trim(),
    rosterDependency: form.rosterDependency,
    conditionalPolicy,
    publicFormSchema,
    riskPolicy,
    credentialTTLSeconds: form.credentialTTLSeconds,
    connectorOperationKey: form.connectorOperationKey.trim() || null,
    privacyNoticeVersion: form.privacyNoticeVersion.trim(),
    privacyNotice: {
      dataCategories,
      retentionSummary: form.privacyRetentionSummary.trim(),
      summary: form.privacySummary.trim(),
      title: form.privacyTitle.trim(),
    },
    reason,
  });
}
</script>

<template>
  <ElDialog
    v-model="visible"
    :title="method ? `编辑 ${method.displayName}` : '新增认证方法'"
    width="min(840px, 94vw)"
    destroy-on-close
  >
    <ElAlert
      type="info"
      :closable="false"
      show-icon
      title="方法配置与校园连接器解耦"
      description="这里只选择已批准的 operation key，不展示或修改内网地址、端口、证书、公钥和 secret reference。保存后方法会自动停用，必须通过结构与实时健康校验才能启用。"
      class="method-alert"
    />
    <ElForm label-position="top">
      <div class="method-grid method-grid--three">
        <ElFormItem label="方法类型">
          <ElSelect
            v-model="form.method"
            style="width: 100%"
            :disabled="Boolean(method)"
            @change="setNewMethodDefaults"
          >
            <ElOption label="实名信息校验" value="real_name_identity_check" />
            <ElOption label="学校统一身份认证" value="school_sso" />
            <ElOption
              label="学校邮箱接收验证码"
              value="student_email_outbound_otp"
            />
            <ElOption
              label="从学校邮箱发送验证邮件"
              value="student_email_inbound_challenge"
            />
            <ElOption label="人工材料审核" value="manual_material_review" />
          </ElSelect>
        </ElFormItem>
        <ElFormItem label="适配器 ID">
          <ElInput v-model="form.adapterID" />
        </ElFormItem>
        <ElFormItem label="适配器版本">
          <ElInput v-model="form.adapterVersion" />
        </ElFormItem>
      </div>
      <div class="method-grid method-grid--two">
        <ElFormItem label="用户可见名称">
          <ElInput v-model="form.displayName" maxlength="100" />
        </ElFormItem>
        <ElFormItem label="用户可见说明">
          <ElInput v-model="form.description" maxlength="500" />
        </ElFormItem>
      </div>
      <div class="method-grid method-grid--three">
        <ElFormItem label="名册依赖">
          <ElSelect v-model="form.rosterDependency" style="width: 100%">
            <ElOption label="必须命中当前名册" value="required" />
            <ElOption label="独立证据" value="independent" />
            <ElOption label="按适配器条件判定" value="conditional" />
          </ElSelect>
        </ElFormItem>
        <ElFormItem label="凭据有效期（秒，可空）">
          <ElInputNumber
            v-model="form.credentialTTLSeconds"
            :min="60"
            :max="157680000"
            controls-position="right"
          />
        </ElFormItem>
        <ElFormItem label="连接器 operation key（可空）">
          <ElInput
            v-model="form.connectorOperationKey"
            placeholder="例如 buaa.ldap.authenticate"
          />
        </ElFormItem>
      </div>

      <div class="json-grid">
        <ElFormItem label="名册条件策略">
          <ElInput
            v-model="form.conditionalPolicyText"
            type="textarea"
            :rows="9"
          />
        </ElFormItem>
        <ElFormItem label="风险与限流策略">
          <ElInput v-model="form.riskPolicyText" type="textarea" :rows="9" />
        </ElFormItem>
        <ElFormItem label="受控公开表单 schema">
          <ElInput
            v-model="form.publicFormSchemaText"
            type="textarea"
            :rows="12"
          />
        </ElFormItem>
        <div class="notice-panel">
          <strong>用户可见个人信息处理说明</strong>
          <ElFormItem label="版本">
            <ElInput v-model="form.privacyNoticeVersion" />
          </ElFormItem>
          <ElFormItem label="标题">
            <ElInput v-model="form.privacyTitle" />
          </ElFormItem>
          <ElFormItem label="用途说明">
            <ElInput v-model="form.privacySummary" type="textarea" :rows="2" />
          </ElFormItem>
          <ElFormItem label="保留期限说明">
            <ElInput
              v-model="form.privacyRetentionSummary"
              type="textarea"
              :rows="2"
            />
          </ElFormItem>
          <ElFormItem label="数据类型（每行一个）">
            <ElInput
              v-model="form.privacyCategoriesText"
              type="textarea"
              :rows="4"
            />
          </ElFormItem>
        </div>
      </div>

      <ElFormItem label="变更原因（必填）">
        <ElInput
          v-model="form.reason"
          maxlength="500"
          show-word-limit
          placeholder="填写工单、适配器版本变更和影响说明"
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
.method-alert {
  margin-bottom: 20px;
}

.method-grid,
.json-grid {
  display: grid;
  gap: 0 16px;
}

.method-grid--three {
  grid-template-columns: 1.2fr 1fr 0.7fr;
}

.method-grid--two,
.json-grid {
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.json-grid :deep(textarea) {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 12px;
  line-height: 1.5;
}

.notice-panel {
  padding: 16px;
  background: var(--el-fill-color-extra-light);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 12px;
}

.notice-panel > strong {
  display: block;
  margin-bottom: 14px;
}

@media (max-width: 780px) {
  .method-grid--three,
  .method-grid--two,
  .json-grid {
    grid-template-columns: 1fr;
  }
}
</style>
