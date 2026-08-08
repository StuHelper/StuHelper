<script setup lang="ts">
import type {
  AdminStudentCredential,
  AdminStudentSubjectConflict,
} from '#/api/admin';

import { computed, onMounted, reactive, ref } from 'vue';

import { useAccessStore } from '@vben/stores';

import {
  ElAlert,
  ElButton,
  ElCard,
  ElDialog,
  ElForm,
  ElFormItem,
  ElInput,
  ElMessage,
  ElMessageBox,
  ElOption,
  ElSelect,
  ElStatistic,
  ElTabPane,
  ElTabs,
  ElTag,
} from 'element-plus';

import {
  decideStudentSubjectConflict,
  listStudentCredentials,
  listStudentSubjectConflicts,
  revokeStudentCredential,
} from '#/api/admin';
import { SCHOOL_SCOPE_REQUIRED_ERROR, useAuthStore } from '#/store/auth';

import PersistentAdminTable from '../../shared/admin-table/PersistentAdminTable.vue';
import PersistentAdminTableColumn from '../../shared/admin-table/PersistentAdminTableColumn.vue';
import AdminContentLayout from '../../shared/AdminContentLayout.vue';
import { formatAdminDateTime } from '../../shared/display';

const CREDENTIAL_READ = 'student:credential:read';
const CREDENTIAL_REVOKE = 'student:credential:revoke';
const CONFLICT_READ = 'student:subject_conflict:read';
const CONFLICT_RESOLVE = 'student:subject_conflict:resolve';
const PAGE_SIZE = 50;

type ConflictAction =
  | 'dismiss_claim'
  | 'release_subject_for_reverification'
  | 'start_review';

const authStore = useAuthStore();
const accessStore = useAccessStore();
const activeTab = ref('credentials');
const schoolCode = ref('');
const loading = ref(false);
const loadError = ref('');
const actionError = ref('');
const credentials = ref<AdminStudentCredential[]>([]);
const conflicts = ref<AdminStudentSubjectConflict[]>([]);
const credentialOffset = ref(0);
const conflictOffset = ref(0);
const credentialStatus = ref('');
const conflictStatus = ref('open');
const actionKey = ref('');

const conflictDialogVisible = ref(false);
const conflictTarget = ref<AdminStudentSubjectConflict | null>(null);
const conflictDecision = reactive({
  action: 'start_review' as ConflictAction,
  reason: '',
});

const canRevoke = computed(() =>
  accessStore.accessCodes.includes(CREDENTIAL_REVOKE),
);
const canResolve = computed(() =>
  accessStore.accessCodes.includes(CONFLICT_RESOLVE),
);
const activeCredentialCount = computed(
  () => credentials.value.filter((item) => item.status === 'active').length,
);
const reviewCredentialCount = computed(
  () =>
    credentials.value.filter((item) => item.status === 'review_required')
      .length,
);
const openConflictCount = computed(
  () =>
    conflicts.value.filter(
      (item) => item.status === 'open' || item.status === 'under_review',
    ).length,
);

function normalizeSchoolScope() {
  const capability =
    activeTab.value === 'conflicts' ? CONFLICT_READ : CREDENTIAL_READ;
  try {
    const resolved = authStore.resolveScopedSchoolId(
      capability,
      schoolCode.value,
    );
    if (resolved) schoolCode.value = resolved;
    if (!schoolCode.value.trim()) {
      loadError.value = '请输入学校代码；跨校查询被明确禁止。';
      return false;
    }
    return true;
  } catch (error) {
    if (
      error instanceof Error &&
      error.message === SCHOOL_SCOPE_REQUIRED_ERROR
    ) {
      loadError.value = '您管理多个学校，请明确选择一个学校代码。';
      return false;
    }
    throw error;
  }
}

async function fetchCurrent(reset = false) {
  if (!normalizeSchoolScope()) return;
  if (reset) {
    if (activeTab.value === 'credentials') credentialOffset.value = 0;
    else conflictOffset.value = 0;
  }
  loading.value = true;
  loadError.value = '';
  try {
    if (activeTab.value === 'credentials') {
      credentials.value = await listStudentCredentials({
        schoolCode: schoolCode.value.trim(),
        limit: PAGE_SIZE,
        offset: credentialOffset.value,
        ...(credentialStatus.value
          ? {
              status: credentialStatus.value as
                | 'active'
                | 'expired'
                | 'pending'
                | 'rejected'
                | 'review_required'
                | 'revoked',
            }
          : {}),
      });
    } else {
      conflicts.value = await listStudentSubjectConflicts({
        schoolCode: schoolCode.value.trim(),
        limit: PAGE_SIZE,
        offset: conflictOffset.value,
        ...(conflictStatus.value
          ? {
              status: conflictStatus.value as
                | 'dismissed'
                | 'open'
                | 'resolved'
                | 'under_review',
            }
          : {}),
      });
    }
  } catch (error) {
    loadError.value = errorMessage(error);
  } finally {
    loading.value = false;
  }
}

async function revokeCredential(row: AdminStudentCredential) {
  if (actionKey.value) return;
  try {
    const result = await ElMessageBox.prompt(
      '撤销会立即提高资格 revision，并使依赖该凭据的准入判断失效。请输入原因。',
      '撤销学生凭据',
      {
        confirmButtonText: '撤销并审计',
        cancelButtonText: '取消',
        inputPattern: /^.{4,500}$/s,
        inputErrorMessage: '请输入 4–500 个字的原因。',
        type: 'warning',
      },
    );
    actionKey.value = `credential:${row.id}`;
    await revokeStudentCredential(row.id, {
      expectedRevision: row.revision,
      reason: result.value.trim(),
    });
    ElMessage.success('凭据已撤销，学生资格正在按新 revision 重算。');
    await fetchCurrent();
  } catch (error) {
    if (error === 'cancel' || error === 'close') return;
    actionError.value = errorMessage(error);
    ElMessage.error(actionError.value);
  } finally {
    actionKey.value = '';
  }
}

function openConflictDecision(row: AdminStudentSubjectConflict) {
  conflictTarget.value = row;
  conflictDecision.action =
    row.status === 'open' ? 'start_review' : 'dismiss_claim';
  conflictDecision.reason = '';
  conflictDialogVisible.value = true;
}

async function submitConflictDecision() {
  const target = conflictTarget.value;
  const reason = conflictDecision.reason.trim();
  if (!target || reason.length < 4 || actionKey.value) {
    ElMessage.warning('请输入至少 4 个字的处置原因。');
    return;
  }
  actionKey.value = `conflict:${target.id}`;
  try {
    await decideStudentSubjectConflict(target.id, {
      action: conflictDecision.action,
      reason,
    });
    ElMessage.success(
      conflictDecision.action === 'release_subject_for_reverification'
        ? '主体已释放，相关活动凭据已撤销，双方需要重新认证。'
        : '冲突处置状态已更新。',
    );
    conflictDialogVisible.value = false;
    await fetchCurrent();
  } catch (error) {
    actionError.value = errorMessage(error);
    ElMessage.error(actionError.value);
  } finally {
    actionKey.value = '';
  }
}

function statusType(status: string): 'danger' | 'info' | 'success' | 'warning' {
  if (status === 'active' || status === 'resolved') return 'success';
  if (
    status === 'open' ||
    status === 'pending' ||
    status === 'review_required' ||
    status === 'under_review'
  )
    return 'warning';
  if (status === 'rejected' || status === 'revoked') return 'danger';
  return 'info';
}

function methodLabel(method: string) {
  return (
    {
      manual_material_review: '人工材料审核',
      real_name_identity_check: '实名信息校验',
      school_sso: '统一身份认证',
      student_email_inbound_challenge: '发送验证邮件',
      student_email_outbound_otp: '邮箱验证码',
    }[method] ?? method
  );
}

function errorMessage(error: unknown) {
  return error instanceof Error && error.message
    ? error.message
    : '请求失败，请稍后重试。';
}

onMounted(() => fetchCurrent(true));
</script>

<template>
  <AdminContentLayout
    title="学生凭据与主体冲突"
    :total="activeTab === 'credentials' ? credentials.length : conflicts.length"
  >
    <template #toolbar>
      <ElInput
        v-model="schoolCode"
        class="admin-toolbar-control admin-toolbar-control--wide"
        maxlength="10"
        placeholder="学校代码"
        @keyup.enter="fetchCurrent(true)"
      />
      <ElButton type="primary" :loading="loading" @click="fetchCurrent(true)">
        刷新
      </ElButton>
    </template>

    <div class="governance-overview">
      <ElCard shadow="never">
        <ElStatistic title="当前页活动凭据" :value="activeCredentialCount" />
      </ElCard>
      <ElCard shadow="never">
        <ElStatistic title="需要复核的凭据" :value="reviewCredentialCount" />
      </ElCard>
      <ElCard shadow="never">
        <ElStatistic title="开放主体冲突" :value="openConflictCount" />
      </ElCard>
    </div>

    <ElAlert
      title="隐私与权限边界"
      description="本页不返回姓名、证件号、原始材料、主体 HMAC 或学校密码。撤销凭据和释放主体都需要学校范围授权、近期 MFA 与审计原因。"
      type="info"
      :closable="false"
      show-icon
      class="governance-alert"
    />
    <ElAlert
      v-if="loadError"
      :title="loadError"
      type="error"
      :closable="false"
      show-icon
      class="governance-alert"
    />
    <ElAlert
      v-if="actionError"
      :title="actionError"
      type="error"
      closable
      show-icon
      class="governance-alert"
      @close="actionError = ''"
    />

    <ElTabs v-model="activeTab" @tab-change="fetchCurrent(true)">
      <ElTabPane label="学生凭据" name="credentials">
        <div class="filter-row">
          <ElSelect
            v-model="credentialStatus"
            placeholder="凭据状态"
            @change="fetchCurrent(true)"
          >
            <ElOption label="全部状态" value="" />
            <ElOption label="活动" value="active" />
            <ElOption label="需要复核" value="review_required" />
            <ElOption label="已过期" value="expired" />
            <ElOption label="已撤销" value="revoked" />
          </ElSelect>
        </div>
        <PersistentAdminTable
          table-key="users.targetStudentCredentials"
          :data="credentials"
          :loading="loading"
          row-key="id"
          stripe
        >
          <PersistentAdminTableColumn
            column-key="user"
            label="用户 ID"
            prop="userID"
            :default-width="110"
          />
          <PersistentAdminTableColumn
            column-key="subject"
            label="学籍主体"
            prop="subjectDisplay"
            :default-min-width="150"
          />
          <PersistentAdminTableColumn
            column-key="method"
            label="认证方法"
            :default-min-width="180"
          >
            <template #default="{ row }">
              {{ methodLabel(row.method) }}
            </template>
          </PersistentAdminTableColumn>
          <PersistentAdminTableColumn
            column-key="status"
            label="状态"
            :default-width="120"
          >
            <template #default="{ row }">
              <ElTag :type="statusType(row.status)">
                {{ row.status }}
              </ElTag>
            </template>
          </PersistentAdminTableColumn>
          <PersistentAdminTableColumn
            column-key="dependency"
            label="名册依赖 / 保证"
            :default-width="168"
          >
            <template #default="{ row }">
              {{ row.rosterDependency }} · {{ row.assurance }}
            </template>
          </PersistentAdminTableColumn>
          <PersistentAdminTableColumn
            column-key="verified"
            label="通过时间"
            :default-width="176"
          >
            <template #default="{ row }">
              {{ formatAdminDateTime(row.verifiedAt) }}
            </template>
          </PersistentAdminTableColumn>
          <PersistentAdminTableColumn
            column-key="expires"
            label="到期时间"
            :default-width="176"
          >
            <template #default="{ row }">
              {{
                row.expiresAt
                  ? formatAdminDateTime(row.expiresAt)
                  : '按学校策略'
              }}
            </template>
          </PersistentAdminTableColumn>
          <PersistentAdminTableColumn
            v-if="canRevoke"
            column-key="actions"
            label="操作"
            fixed="right"
            :default-width="112"
          >
            <template #default="{ row }">
              <ElButton
                v-if="
                  row.status === 'active' || row.status === 'review_required'
                "
                size="small"
                type="danger"
                plain
                :loading="actionKey === `credential:${row.id}`"
                @click="revokeCredential(row)"
              >
                撤销
              </ElButton>
            </template>
          </PersistentAdminTableColumn>
        </PersistentAdminTable>
      </ElTabPane>

      <ElTabPane label="主体冲突" name="conflicts">
        <div class="filter-row">
          <ElSelect
            v-model="conflictStatus"
            placeholder="冲突状态"
            @change="fetchCurrent(true)"
          >
            <ElOption label="全部状态" value="" />
            <ElOption label="待处置" value="open" />
            <ElOption label="复核中" value="under_review" />
            <ElOption label="已解决" value="resolved" />
            <ElOption label="已驳回申请" value="dismissed" />
          </ElSelect>
        </div>
        <PersistentAdminTable
          table-key="users.targetStudentSubjectConflicts"
          :data="conflicts"
          :loading="loading"
          row-key="id"
          stripe
        >
          <PersistentAdminTableColumn
            column-key="claimant"
            label="申领用户"
            prop="claimantUserID"
            :default-width="116"
          />
          <PersistentAdminTableColumn
            column-key="incumbent"
            label="当前用户"
            :default-width="116"
          >
            <template #default="{ row }">
              {{ row.incumbentUserID ?? '无' }}
            </template>
          </PersistentAdminTableColumn>
          <PersistentAdminTableColumn
            column-key="reason"
            label="内部分类"
            prop="reasonCode"
            :default-min-width="220"
          />
          <PersistentAdminTableColumn
            column-key="status"
            label="状态"
            :default-width="128"
          >
            <template #default="{ row }">
              <ElTag :type="statusType(row.status)">
                {{ row.status }}
              </ElTag>
            </template>
          </PersistentAdminTableColumn>
          <PersistentAdminTableColumn
            column-key="created"
            label="发现时间"
            :default-width="176"
          >
            <template #default="{ row }">
              {{ formatAdminDateTime(row.createdAt) }}
            </template>
          </PersistentAdminTableColumn>
          <PersistentAdminTableColumn
            v-if="canResolve"
            column-key="actions"
            label="操作"
            fixed="right"
            :default-width="120"
          >
            <template #default="{ row }">
              <ElButton
                v-if="row.status === 'open' || row.status === 'under_review'"
                size="small"
                type="primary"
                plain
                @click="openConflictDecision(row)"
              >
                处置
              </ElButton>
            </template>
          </PersistentAdminTableColumn>
        </PersistentAdminTable>
      </ElTabPane>
    </ElTabs>

    <ElDialog
      v-model="conflictDialogVisible"
      title="主体冲突处置"
      width="min(580px, 92vw)"
    >
      <ElAlert
        v-if="conflictDecision.action === 'release_subject_for_reverification'"
        title="高影响操作：释放主体会撤销该主体的活动学生凭据，并要求相关账号重新认证。"
        type="warning"
        :closable="false"
        show-icon
        class="governance-alert"
      />
      <ElForm label-position="top">
        <ElFormItem label="处置动作">
          <ElSelect v-model="conflictDecision.action" style="width: 100%">
            <ElOption label="标记进入人工复核" value="start_review" />
            <ElOption label="驳回本次申领" value="dismiss_claim" />
            <ElOption
              label="释放主体并要求重新认证"
              value="release_subject_for_reverification"
            />
          </ElSelect>
        </ElFormItem>
        <ElFormItem label="处置原因（必填）">
          <ElInput
            v-model="conflictDecision.reason"
            type="textarea"
            :rows="4"
            maxlength="500"
            show-word-limit
            placeholder="填写核查依据、工单和后续可执行路径"
          />
        </ElFormItem>
      </ElForm>
      <template #footer>
        <ElButton
          :disabled="Boolean(actionKey)"
          @click="conflictDialogVisible = false"
        >
          取消
        </ElButton>
        <ElButton
          type="primary"
          :loading="Boolean(actionKey)"
          @click="submitConflictDecision"
        >
          确认并审计
        </ElButton>
      </template>
    </ElDialog>
  </AdminContentLayout>
</template>

<style scoped>
.governance-overview {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 16px;
  margin-bottom: 16px;
}

.governance-overview :deep(.el-card) {
  background: linear-gradient(
    145deg,
    var(--el-bg-color),
    var(--el-fill-color-extra-light)
  );
  border-radius: 16px;
}

.governance-alert {
  margin-bottom: 16px;
}

.filter-row {
  display: flex;
  justify-content: flex-end;
  margin-bottom: 14px;
}

.filter-row :deep(.el-select) {
  width: 220px;
}

@media (max-width: 720px) {
  .governance-overview {
    grid-template-columns: 1fr;
  }
}
</style>
