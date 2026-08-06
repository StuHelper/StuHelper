<script setup lang="ts">
import type {
  AdminManualReviewDetail,
  AdminManualReviewSummary,
} from '#/api/admin';

import { computed, onMounted, reactive, ref } from 'vue';

import { useAccessStore } from '@vben/stores';

import {
  ElAlert,
  ElButton,
  ElCard,
  ElDescriptions,
  ElDescriptionsItem,
  ElDialog,
  ElDrawer,
  ElEmpty,
  ElForm,
  ElFormItem,
  ElInput,
  ElInputNumber,
  ElMessage,
  ElOption,
  ElSelect,
  ElSkeleton,
  ElStatistic,
  ElTag,
} from 'element-plus';

import {
  accessManualStudentReviewMaterial,
  decideManualStudentReview,
  getManualStudentReview,
  listManualStudentReviews,
} from '#/api/admin';
import { SCHOOL_SCOPE_REQUIRED_ERROR, useAuthStore } from '#/store/auth';

import PersistentAdminTable from '../../shared/admin-table/PersistentAdminTable.vue';
import PersistentAdminTableColumn from '../../shared/admin-table/PersistentAdminTableColumn.vue';
import AdminContentLayout from '../../shared/AdminContentLayout.vue';
import { formatAdminDateTime } from '../../shared/display';

const READ_CAPABILITY = 'student:manual_review:read';
const DECIDE_CAPABILITY = 'student:manual_review:decide';
const MATERIAL_CAPABILITY = 'student:manual_material:access';
const PAGE_SIZE = 30;

type QueueStatus =
  | ''
  | 'approved'
  | 'pending'
  | 'rejected'
  | 'supplement_required';
type DecisionAction = 'approve' | 'reject' | 'request_supplement';

const accessStore = useAccessStore();
const authStore = useAuthStore();
const loading = ref(false);
const loadError = ref('');
const actionError = ref('');
const rows = ref<AdminManualReviewSummary[]>([]);
const offset = ref(0);
const hasNextPage = ref(false);
const query = reactive({ schoolCode: '', status: '' as QueueStatus });
let requestSequence = 0;

const detailVisible = ref(false);
const detailLoading = ref(false);
const detail = ref<AdminManualReviewDetail | null>(null);
const detailError = ref('');
const materialLoadingID = ref('');

const decisionVisible = ref(false);
const decisionLoading = ref(false);
const decisionTarget = ref<AdminManualReviewSummary | null>(null);
const decision = reactive({
  action: 'approve' as DecisionAction,
  expiresInDays: 365,
  internalRiskNote: '',
  userVisibleReason: '',
});

const canDecide = computed(() =>
  accessStore.accessCodes.includes(DECIDE_CAPABILITY),
);
const canAccessMaterial = computed(() =>
  accessStore.accessCodes.includes(MATERIAL_CAPABILITY),
);
const pendingCount = computed(
  () => rows.value.filter((item) => item.status === 'pending').length,
);
const supplementCount = computed(
  () =>
    rows.value.filter((item) => item.status === 'supplement_required').length,
);
const verifiedEmailCount = computed(
  () => rows.value.filter((item) => item.emailVerified).length,
);

function normalizeSchoolScope(): boolean {
  try {
    const resolved = authStore.resolveScopedSchoolId(
      READ_CAPABILITY,
      query.schoolCode,
    );
    if (resolved) query.schoolCode = resolved;
    if (!query.schoolCode.trim()) {
      loadError.value =
        '请输入要管理的学校代码。全局管理员必须明确选择学校范围。';
      rows.value = [];
      return false;
    }
    return true;
  } catch (error) {
    if (
      error instanceof Error &&
      error.message === SCHOOL_SCOPE_REQUIRED_ERROR
    ) {
      loadError.value = '您管理多个学校，请先输入一个学校代码。';
      rows.value = [];
      return false;
    }
    throw error;
  }
}

async function fetchQueue(reset = false) {
  if (reset) offset.value = 0;
  if (!normalizeSchoolScope()) return;
  const sequence = ++requestSequence;
  loading.value = true;
  loadError.value = '';
  try {
    const result = await listManualStudentReviews({
      schoolCode: query.schoolCode.trim(),
      limit: PAGE_SIZE,
      offset: offset.value,
      ...(query.status ? { status: query.status } : {}),
    });
    if (sequence !== requestSequence) return;
    rows.value = result;
    hasNextPage.value = result.length === PAGE_SIZE;
  } catch (error) {
    if (sequence !== requestSequence) return;
    loadError.value = errorMessage(error);
    rows.value = [];
  } finally {
    if (sequence === requestSequence) loading.value = false;
  }
}

async function openDetail(row: AdminManualReviewSummary) {
  detailVisible.value = true;
  detailLoading.value = true;
  detailError.value = '';
  detail.value = null;
  try {
    detail.value = await getManualStudentReview(row.id);
  } catch (error) {
    detailError.value = errorMessage(error);
  } finally {
    detailLoading.value = false;
  }
}

async function openMaterial(materialID: string) {
  if (!detail.value || materialLoadingID.value) return;
  materialLoadingID.value = materialID;
  actionError.value = '';
  try {
    const access = await accessManualStudentReviewMaterial(
      detail.value.case.id,
      materialID,
    );
    window.open(access.url, '_blank', 'noopener,noreferrer');
  } catch (error) {
    actionError.value = errorMessage(error);
    ElMessage.error(actionError.value);
  } finally {
    materialLoadingID.value = '';
  }
}

function openDecision(row: AdminManualReviewSummary, action: DecisionAction) {
  decisionTarget.value = row;
  decision.action = action;
  decision.userVisibleReason = '';
  decision.internalRiskNote = '';
  decision.expiresInDays = row.materialType === 'admission_notice' ? 180 : 365;
  decisionVisible.value = true;
}

async function submitDecision() {
  const target = decisionTarget.value;
  const reason = decision.userVisibleReason.trim();
  if (!target || reason.length < 2) {
    ElMessage.warning('请填写至少 2 个字的用户可见处理说明。');
    return;
  }
  decisionLoading.value = true;
  actionError.value = '';
  try {
    await decideManualStudentReview(target.id, {
      action: decision.action,
      userVisibleReason: reason,
      ...(decision.internalRiskNote.trim()
        ? { internalRiskNote: decision.internalRiskNote.trim() }
        : {}),
      ...(decision.action === 'approve'
        ? { expiresInDays: decision.expiresInDays }
        : {}),
    });
    ElMessage.success('审核决定已保存，学生凭据与资格将按新状态重算。');
    decisionVisible.value = false;
    detailVisible.value = false;
    await fetchQueue();
  } catch (error) {
    actionError.value = errorMessage(error);
    ElMessage.error(actionError.value);
  } finally {
    decisionLoading.value = false;
  }
}

function previousPage() {
  offset.value = Math.max(0, offset.value - PAGE_SIZE);
  void fetchQueue();
}

function nextPage() {
  if (!hasNextPage.value) return;
  offset.value += PAGE_SIZE;
  void fetchQueue();
}

function statusLabel(status: string) {
  return (
    {
      approved: '已批准',
      pending: '待审核',
      rejected: '已驳回',
      supplement_required: '待补充',
    }[status] ?? status
  );
}

function statusType(status: string): 'danger' | 'info' | 'success' | 'warning' {
  if (status === 'approved') return 'success';
  if (status === 'rejected') return 'danger';
  if (status === 'pending' || status === 'supplement_required')
    return 'warning';
  return 'info';
}

function materialLabel(type: string) {
  return (
    {
      admission_notice: '录取通知书',
      campus_card: '校园卡',
      other_approved: '学校批准材料',
      student_card: '学生证',
    }[type] ?? type
  );
}

function errorMessage(error: unknown) {
  return error instanceof Error && error.message
    ? error.message
    : '请求失败，请稍后重试。';
}

onMounted(() => fetchQueue(true));
</script>

<template>
  <AdminContentLayout title="学生认证审核" :total="rows.length">
    <template #toolbar>
      <ElInput
        v-model="query.schoolCode"
        class="admin-toolbar-control admin-toolbar-control--wide"
        clearable
        maxlength="10"
        placeholder="学校代码，例如 4111010006"
        @keyup.enter="fetchQueue(true)"
      />
      <ElSelect
        v-model="query.status"
        class="admin-toolbar-control"
        placeholder="审核状态"
        :teleported="false"
        @change="fetchQueue(true)"
      >
        <ElOption label="全部状态" value="" />
        <ElOption label="待审核" value="pending" />
        <ElOption label="待补充" value="supplement_required" />
        <ElOption label="已批准" value="approved" />
        <ElOption label="已驳回" value="rejected" />
      </ElSelect>
      <ElButton type="primary" :loading="loading" @click="fetchQueue(true)">
        刷新队列
      </ElButton>
    </template>

    <div class="review-overview" aria-label="当前页审核概览">
      <ElCard shadow="never" class="review-stat review-stat--attention">
        <ElStatistic title="当前页待审核" :value="pendingCount" />
        <span>优先处理已提交且材料完整的申请</span>
      </ElCard>
      <ElCard shadow="never" class="review-stat">
        <ElStatistic title="等待用户补充" :value="supplementCount" />
        <span>补充后会重新进入待审核队列</span>
      </ElCard>
      <ElCard shadow="never" class="review-stat">
        <ElStatistic title="邮箱辅助证据通过" :value="verifiedEmailCount" />
        <span>邮箱通过不单独授予学生身份</span>
      </ElCard>
    </div>

    <ElAlert
      title="最小化审核"
      description="列表只显示脱敏字段和材料元数据。查看表单详情与原图需要近期 MFA；审核决定必须填写用户可见原因。"
      type="info"
      :closable="false"
      show-icon
      class="review-guidance"
    />

    <ElAlert
      v-if="loadError"
      :title="loadError"
      type="error"
      :closable="false"
      show-icon
      class="admin-load-error"
    >
      <ElButton size="small" :loading="loading" @click="fetchQueue()">
        重试
      </ElButton>
    </ElAlert>

    <PersistentAdminTable
      table-key="users.targetStudentManualReview"
      :loading="loading"
      :data="rows"
      row-key="id"
      stripe
    >
      <PersistentAdminTableColumn
        column-key="applicant"
        label="申请人"
        :default-min-width="180"
      >
        <template #default="{ row }">
          <div class="identity-cell">
            <strong>{{ row.applicantNameMasked }}</strong>
            <span>{{ row.studentIDMasked }}</span>
          </div>
        </template>
      </PersistentAdminTableColumn>
      <PersistentAdminTableColumn
        column-key="materialType"
        label="材料"
        :default-min-width="148"
      >
        <template #default="{ row }">
          <div class="material-cell">
            <span>{{ materialLabel(row.materialType) }}</span>
            <small>{{ row.materials.length }} 张实时拍摄材料</small>
          </div>
        </template>
      </PersistentAdminTableColumn>
      <PersistentAdminTableColumn
        column-key="email"
        label="联系邮箱"
        :default-min-width="220"
      >
        <template #default="{ row }">
          <div class="email-cell">
            <span>{{ row.emailMasked || '未提供' }}</span>
            <ElTag v-if="row.emailVerified" size="small" type="success">
              已验证
            </ElTag>
            <ElTag
              v-else-if="row.emailVerificationRequired"
              size="small"
              type="warning"
            >
              待验证
            </ElTag>
          </div>
        </template>
      </PersistentAdminTableColumn>
      <PersistentAdminTableColumn
        column-key="status"
        label="状态"
        :default-width="112"
      >
        <template #default="{ row }">
          <ElTag :type="statusType(row.status)" effect="light">
            {{ statusLabel(row.status) }}
          </ElTag>
        </template>
      </PersistentAdminTableColumn>
      <PersistentAdminTableColumn
        column-key="submittedAt"
        label="提交时间"
        :default-width="170"
      >
        <template #default="{ row }">
          <span class="admin-cell-muted">
            {{
              row.submittedAt
                ? formatAdminDateTime(row.submittedAt)
                : '尚未提交'
            }}
          </span>
        </template>
      </PersistentAdminTableColumn>
      <PersistentAdminTableColumn
        column-key="actions"
        label="操作"
        fixed="right"
        :default-width="260"
      >
        <template #default="{ row }">
          <div class="admin-action-group">
            <ElButton plain size="small" @click="openDetail(row)">
              查看详情
            </ElButton>
            <template v-if="canDecide && row.status === 'pending'">
              <ElButton
                size="small"
                type="success"
                plain
                @click="openDecision(row, 'approve')"
              >
                批准
              </ElButton>
              <ElButton
                size="small"
                type="warning"
                plain
                @click="openDecision(row, 'request_supplement')"
              >
                补充
              </ElButton>
              <ElButton
                size="small"
                type="danger"
                plain
                @click="openDecision(row, 'reject')"
              >
                驳回
              </ElButton>
            </template>
          </div>
        </template>
      </PersistentAdminTableColumn>
    </PersistentAdminTable>

    <template #pagination>
      <div class="queue-pagination">
        <span>
          第 {{ Math.floor(offset / PAGE_SIZE) + 1 }} 页 · 本页
          {{ rows.length }} 条
        </span>
        <div>
          <ElButton :disabled="offset === 0 || loading" @click="previousPage">
            上一页
          </ElButton>
          <ElButton :disabled="!hasNextPage || loading" @click="nextPage">
            下一页
          </ElButton>
        </div>
      </div>
    </template>

    <ElDrawer
      v-model="detailVisible"
      title="人工审核详情"
      size="min(720px, 94vw)"
    >
      <ElSkeleton v-if="detailLoading" :rows="8" animated />
      <ElAlert
        v-else-if="detailError"
        :title="detailError"
        type="error"
        :closable="false"
        show-icon
      />
      <template v-else-if="detail">
        <ElDescriptions :column="2" border class="detail-block">
          <ElDescriptionsItem label="学校">
            {{ detail.case.school.name }}
          </ElDescriptionsItem>
          <ElDescriptionsItem label="申请状态">
            <ElTag :type="statusType(detail.case.status)">
              {{ statusLabel(detail.case.status) }}
            </ElTag>
          </ElDescriptionsItem>
          <ElDescriptionsItem label="申请人">
            {{ detail.case.applicantNameMasked }}
          </ElDescriptionsItem>
          <ElDescriptionsItem label="学号">
            {{ detail.case.studentIDMasked }}
          </ElDescriptionsItem>
          <ElDescriptionsItem label="邮箱">
            {{ detail.case.emailMasked || '未提供' }}
          </ElDescriptionsItem>
          <ElDescriptionsItem label="邮箱证据">
            {{ detail.case.emailVerified ? '已验证（辅助证据）' : '未验证' }}
          </ElDescriptionsItem>
        </ElDescriptions>

        <section class="detail-section">
          <div class="section-heading">
            <div>
              <h3>申请表字段</h3>
              <p>字段值仅在本次近期 MFA 会话中展示。</p>
            </div>
          </div>
          <ElDescriptions
            v-if="Object.keys(detail.formValues).length > 0"
            :column="1"
            border
          >
            <ElDescriptionsItem
              v-for="[key, value] in Object.entries(detail.formValues)"
              :key="key"
              :label="key"
            >
              {{ value }}
            </ElDescriptionsItem>
          </ElDescriptions>
          <ElEmpty v-else description="没有额外表单字段" :image-size="72" />
        </section>

        <section class="detail-section">
          <div class="section-heading">
            <div>
              <h3>拍摄材料</h3>
              <p>访问地址短时有效且每次访问均记录安全审计。</p>
            </div>
          </div>
          <div class="material-grid">
            <ElCard
              v-for="material in detail.case.materials"
              :key="material.id"
              shadow="never"
              class="material-card"
            >
              <strong>{{ material.width }} × {{ material.height }}</strong>
              <span>
                {{ Math.ceil(material.sizeBytes / 1024) }} KB ·
                {{ material.contentType }}
              </span>
              <span>{{ formatAdminDateTime(material.capturedAt) }}</span>
              <ElButton
                v-if="canAccessMaterial"
                type="primary"
                plain
                :loading="materialLoadingID === material.id"
                @click="openMaterial(material.id)"
              >
                安全查看
              </ElButton>
            </ElCard>
          </div>
        </section>

        <div
          v-if="canDecide && detail.case.status === 'pending'"
          class="drawer-actions"
        >
          <ElButton
            type="success"
            @click="openDecision(detail.case, 'approve')"
          >
            批准并签发凭据
          </ElButton>
          <ElButton
            type="warning"
            @click="openDecision(detail.case, 'request_supplement')"
          >
            要求补充
          </ElButton>
          <ElButton type="danger" @click="openDecision(detail.case, 'reject')">
            驳回
          </ElButton>
        </div>
      </template>
    </ElDrawer>

    <ElDialog
      v-model="decisionVisible"
      title="记录审核决定"
      width="min(560px, 92vw)"
    >
      <ElAlert
        type="warning"
        :closable="false"
        show-icon
        title="该操作需要近期 MFA，并会改变学生凭据与资格。"
        class="decision-alert"
      />
      <ElForm label-position="top">
        <ElFormItem label="处理动作">
          <ElSelect v-model="decision.action" style="width: 100%">
            <ElOption label="批准并签发有期限凭据" value="approve" />
            <ElOption label="要求用户补充材料" value="request_supplement" />
            <ElOption label="驳回申请" value="reject" />
          </ElSelect>
        </ElFormItem>
        <ElFormItem v-if="decision.action === 'approve'" label="凭据有效天数">
          <ElInputNumber v-model="decision.expiresInDays" :min="1" :max="366" />
        </ElFormItem>
        <ElFormItem label="用户可见说明（必填）">
          <ElInput
            v-model="decision.userVisibleReason"
            type="textarea"
            :rows="4"
            maxlength="500"
            show-word-limit
            placeholder="说明通过依据、需要补充的材料，或可执行的申诉/重试路径"
          />
        </ElFormItem>
        <ElFormItem label="内部风险备注（可选，不向用户展示）">
          <ElInput
            v-model="decision.internalRiskNote"
            type="textarea"
            :rows="3"
            maxlength="2000"
            show-word-limit
          />
        </ElFormItem>
      </ElForm>
      <template #footer>
        <ElButton :disabled="decisionLoading" @click="decisionVisible = false">
          取消
        </ElButton>
        <ElButton
          type="primary"
          :loading="decisionLoading"
          @click="submitDecision"
        >
          确认并审计
        </ElButton>
      </template>
    </ElDialog>
  </AdminContentLayout>
</template>

<style scoped>
.review-overview {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 16px;
  margin-bottom: 16px;
}

.review-stat {
  background: linear-gradient(
    145deg,
    var(--el-bg-color),
    var(--el-fill-color-lighter)
  );
  border: 1px solid color-mix(in srgb, var(--el-border-color) 72%, transparent);
  border-radius: 16px;
}

.review-stat--attention {
  background: linear-gradient(
    145deg,
    color-mix(in srgb, var(--el-color-primary-light-9) 72%, var(--el-bg-color)),
    var(--el-bg-color)
  );
  border-color: color-mix(
    in srgb,
    var(--el-color-primary) 36%,
    var(--el-border-color)
  );
}

.review-stat span {
  display: block;
  margin-top: 8px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.review-guidance,
.admin-load-error {
  margin-bottom: 16px;
}

.identity-cell,
.material-cell {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.identity-cell span,
.material-cell small {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.email-cell {
  display: flex;
  gap: 8px;
  align-items: center;
}

.queue-pagination {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
  color: var(--el-text-color-secondary);
}

.detail-block {
  margin-bottom: 24px;
}

.detail-section {
  padding: 20px 0;
  border-top: 1px solid var(--el-border-color-lighter);
}

.section-heading h3 {
  margin: 0;
  font-size: 16px;
  color: var(--el-text-color-primary);
}

.section-heading p {
  margin: 6px 0 16px;
  font-size: 13px;
  color: var(--el-text-color-secondary);
}

.material-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}

.material-card :deep(.el-card__body) {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.material-card span {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.drawer-actions {
  position: sticky;
  bottom: 0;
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  padding: 16px 0 4px;
  background: var(--el-bg-color);
  border-top: 1px solid var(--el-border-color-lighter);
}

.decision-alert {
  margin-bottom: 18px;
}

@media (max-width: 900px) {
  .review-overview,
  .material-grid {
    grid-template-columns: 1fr;
  }
}
</style>
