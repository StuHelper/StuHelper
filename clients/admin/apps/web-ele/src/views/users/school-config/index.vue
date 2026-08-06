<script setup lang="ts">
import type { VerificationMethodDraft } from './MethodConfigDialog.vue';
import type { SchoolVerificationDraft } from './SchoolConfigDialog.vue';

import type {
  AdminCampusConnectorHealth,
  AdminRosterSyncRequest,
  AdminVerificationMethodConfig,
  AdminVerificationSchoolConfig,
  RosterSnapshot,
} from '#/api/admin';

import { computed, onBeforeUnmount, onMounted, ref } from 'vue';

import { useAccessStore } from '@vben/stores';

import {
  ElAlert,
  ElButton,
  ElCard,
  ElDescriptions,
  ElDescriptionsItem,
  ElEmpty,
  ElMessage,
  ElMessageBox,
  ElProgress,
  ElSkeleton,
  ElStatistic,
  ElTabPane,
  ElTabs,
  ElTag,
} from 'element-plus';

import {
  activateRosterSnapshot,
  createRosterSyncRequest,
  createVerificationSchool,
  listCampusConnectorHealth,
  listRosterSnapshots,
  listRosterSyncRequests,
  listVerificationSchools,
  rollbackRosterSnapshot,
  updateVerificationMethod,
  updateVerificationSchool,
  validateVerificationMethod,
  validateVerificationSchool,
} from '#/api/admin';

import PersistentAdminTable from '../../shared/admin-table/PersistentAdminTable.vue';
import PersistentAdminTableColumn from '../../shared/admin-table/PersistentAdminTableColumn.vue';
import AdminContentLayout from '../../shared/AdminContentLayout.vue';
import { formatAdminDateTime } from '../../shared/display';
import MethodConfigDialog from './MethodConfigDialog.vue';
import SchoolConfigDialog from './SchoolConfigDialog.vue';

const CONFIG_UPDATE_CAPABILITY = 'student:verification_config:update';
const ROSTER_ACTIVATE_CAPABILITY = 'student:roster:activate';
const CONNECTOR_MANAGE_CAPABILITY = 'campus_connector:manage';

const accessStore = useAccessStore();
const loading = ref(false);
const contextLoading = ref(false);
const loadError = ref('');
const actionError = ref('');
const schools = ref<AdminVerificationSchoolConfig[]>([]);
const selectedSchoolCode = ref('');
const snapshots = ref<RosterSnapshot[]>([]);
const connectors = ref<AdminCampusConnectorHealth[]>([]);
const rosterSyncRequests = ref<AdminRosterSyncRequest[]>([]);
const activeTab = ref('methods');
let requestSequence = 0;
let rosterSyncPollTimer: ReturnType<typeof setTimeout> | undefined;

const schoolDialogVisible = ref(false);
const schoolDialogSubmitting = ref(false);
const editingSchool = ref<AdminVerificationSchoolConfig | null>(null);
const methodDialogVisible = ref(false);
const methodDialogSubmitting = ref(false);
const editingMethod = ref<AdminVerificationMethodConfig | null>(null);
const actionKey = ref('');

const selectedSchool = computed(
  () =>
    schools.value.find(
      (school) => school.schoolCode === selectedSchoolCode.value,
    ) ?? null,
);
const enabledSchoolCount = computed(
  () => schools.value.filter((school) => school.enabled).length,
);
const healthyMethodCount = computed(
  () =>
    selectedSchool.value?.methods.filter(
      (method) => method.healthStatus === 'healthy',
    ).length ?? 0,
);
const activeSnapshot = computed(
  () => snapshots.value.find((snapshot) => snapshot.isCurrent) ?? null,
);
const latestRosterSync = computed(() => rosterSyncRequests.value[0] ?? null);
const rosterSyncInFlight = computed(
  () =>
    latestRosterSync.value?.status === 'pending' ||
    latestRosterSync.value?.status === 'started',
);
const canUpdateConfig = computed(() =>
  accessStore.accessCodes.includes(CONFIG_UPDATE_CAPABILITY),
);
const canActivateRoster = computed(() =>
  accessStore.accessCodes.includes(ROSTER_ACTIVATE_CAPABILITY),
);
const canManageConnector = computed(() =>
  accessStore.accessCodes.includes(CONNECTOR_MANAGE_CAPABILITY),
);
const hasRunnableRosterConnector = computed(() =>
  connectors.value.some((node) =>
    node.operations.some(
      (operation) =>
        operation.operationType === 'roster_snapshot_upload' &&
        operation.enabled &&
        operation.validationStatus === 'valid' &&
        ['degraded', 'healthy'].includes(operation.healthStatus),
    ),
  ),
);

async function fetchSchools(preferredSchoolCode = '') {
  const sequence = ++requestSequence;
  loading.value = true;
  loadError.value = '';
  try {
    const result = await listVerificationSchools();
    if (sequence !== requestSequence) return;
    schools.value = result;
    const preferred = preferredSchoolCode || selectedSchoolCode.value;
    selectedSchoolCode.value = result.some(
      (school) => school.schoolCode === preferred,
    )
      ? preferred
      : (result[0]?.schoolCode ?? '');
    await fetchSchoolContext();
  } catch (error) {
    if (sequence !== requestSequence) return;
    loadError.value = errorMessage(error);
  } finally {
    if (sequence === requestSequence) loading.value = false;
  }
}

async function fetchSchoolContext() {
  const schoolCode = selectedSchoolCode.value;
  if (!schoolCode) {
    snapshots.value = [];
    connectors.value = [];
    rosterSyncRequests.value = [];
    return;
  }
  contextLoading.value = true;
  actionError.value = '';
  try {
    const syncRequest = canManageConnector.value
      ? listRosterSyncRequests(schoolCode).catch((error) => {
          actionError.value = errorMessage(error);
          return [];
        })
      : Promise.resolve([]);
    const [snapshotResult, connectorResult, syncResult] = await Promise.all([
      listRosterSnapshots(schoolCode),
      listCampusConnectorHealth(schoolCode),
      syncRequest,
    ]);
    if (schoolCode !== selectedSchoolCode.value) return;
    snapshots.value = snapshotResult;
    connectors.value = connectorResult;
    rosterSyncRequests.value = syncResult;
    if (rosterSyncInFlight.value && latestRosterSync.value) {
      scheduleRosterSyncPoll(latestRosterSync.value.id);
    }
  } catch (error) {
    if (schoolCode !== selectedSchoolCode.value) return;
    actionError.value = errorMessage(error);
  } finally {
    if (schoolCode === selectedSchoolCode.value) contextLoading.value = false;
  }
}

async function triggerRosterSync() {
  const school = selectedSchool.value;
  if (!school || actionKey.value || !canManageConnector.value) return;
  const reason = await promptReason('立即执行完整学籍同步');
  if (!reason) return;
  actionKey.value = 'roster-sync:create';
  actionError.value = '';
  try {
    const request = await createRosterSyncRequest(school.schoolCode, {
      reason,
    });
    rosterSyncRequests.value = [
      request,
      ...rosterSyncRequests.value.filter((item) => item.id !== request.id),
    ];
    ElMessage.success('完整同步任务已排队，校园连接器将在下一次轮询时领取。');
    scheduleRosterSyncPoll(request.id);
  } catch (error) {
    actionError.value = errorMessage(error);
    ElMessage.error(actionError.value);
  } finally {
    actionKey.value = '';
  }
}

function scheduleRosterSyncPoll(requestID: string) {
  if (rosterSyncPollTimer) clearTimeout(rosterSyncPollTimer);
  rosterSyncPollTimer = setTimeout(async () => {
    const schoolCode = selectedSchoolCode.value;
    if (!schoolCode || !canManageConnector.value) return;
    try {
      const requests = await listRosterSyncRequests(schoolCode);
      if (schoolCode !== selectedSchoolCode.value) return;
      rosterSyncRequests.value = requests;
      const request = requests.find((item) => item.id === requestID);
      if (request?.status === 'pending' || request?.status === 'started') {
        scheduleRosterSyncPoll(requestID);
        return;
      }
      if (request?.status === 'succeeded') {
        ElMessage.success('完整学籍快照已同步并通过中心端导入。');
      } else if (request) {
        ElMessage.error(rosterSyncResultLabel(request));
      }
      await fetchSchoolContext();
    } catch (error) {
      actionError.value = errorMessage(error);
      scheduleRosterSyncPoll(requestID);
    }
  }, 3000);
}

async function refreshSelectedSchool() {
  await fetchSchools(selectedSchoolCode.value);
}

function openCreateSchool() {
  editingSchool.value = null;
  schoolDialogVisible.value = true;
}

function openEditSchool() {
  if (!selectedSchool.value) return;
  editingSchool.value = selectedSchool.value;
  schoolDialogVisible.value = true;
}

async function saveSchool(draft: SchoolVerificationDraft) {
  schoolDialogSubmitting.value = true;
  actionError.value = '';
  try {
    await (editingSchool.value
      ? updateVerificationSchool(draft.schoolCode, {
          adapterID: draft.adapterID,
          adapterVersion: draft.adapterVersion,
          emailDomains: draft.emailDomains,
          enrollmentPolicy: draft.enrollmentPolicy,
          expectedRevision: editingSchool.value.configRevision,
          manualFormSchema: draft.manualFormSchema,
          nameMatchPolicy: draft.nameMatchPolicy,
          reason: draft.reason,
          snapshotAutoActivate: draft.snapshotAutoActivate,
          snapshotGraceSeconds: draft.snapshotGraceSeconds,
          snapshotHardExpirySeconds: draft.snapshotHardExpirySeconds,
          snapshotSyncIntervalSeconds: draft.snapshotSyncIntervalSeconds,
          snapshotWarningAfterSeconds: draft.snapshotWarningAfterSeconds,
          studentIDPolicy: draft.studentIDPolicy,
        })
      : createVerificationSchool(draft));
    ElMessage.success('学校认证配置已保存为停用、待校验草稿。');
    schoolDialogVisible.value = false;
    await fetchSchools(draft.schoolCode);
  } catch (error) {
    actionError.value = errorMessage(error);
    ElMessage.error(actionError.value);
  } finally {
    schoolDialogSubmitting.value = false;
  }
}

async function validateSchool(enable: boolean) {
  const school = selectedSchool.value;
  if (!school || actionKey.value) return;
  const verb = enable ? '校验并启用学校认证白名单' : '校验并保持学校停用';
  const reason = await promptReason(verb);
  if (!reason) return;
  actionKey.value = `school:${enable}`;
  try {
    await validateVerificationSchool(school.schoolCode, {
      enable,
      expectedRevision: school.configRevision,
      reason,
    });
    ElMessage.success(
      enable ? '学校认证白名单已通过校验并启用。' : '校验完成，学校保持停用。',
    );
    await refreshSelectedSchool();
  } catch (error) {
    actionError.value = errorMessage(error);
    ElMessage.error(actionError.value);
  } finally {
    actionKey.value = '';
  }
}

function openCreateMethod() {
  editingMethod.value = null;
  methodDialogVisible.value = true;
}

function openEditMethod(method: AdminVerificationMethodConfig) {
  editingMethod.value = method;
  methodDialogVisible.value = true;
}

async function saveMethod(draft: VerificationMethodDraft) {
  const school = selectedSchool.value;
  if (!school) return;
  methodDialogSubmitting.value = true;
  actionError.value = '';
  try {
    await updateVerificationMethod(school.schoolCode, draft.method, {
      adapterID: draft.adapterID,
      adapterVersion: draft.adapterVersion,
      conditionalPolicy: draft.conditionalPolicy,
      connectorOperationKey: draft.connectorOperationKey,
      credentialTTLSeconds: draft.credentialTTLSeconds,
      description: draft.description,
      displayName: draft.displayName,
      expectedRevision: editingMethod.value?.configRevision ?? 0,
      privacyNotice: draft.privacyNotice,
      privacyNoticeVersion: draft.privacyNoticeVersion,
      publicFormSchema: draft.publicFormSchema,
      reason: draft.reason,
      riskPolicy: draft.riskPolicy,
      rosterDependency: draft.rosterDependency,
    });
    ElMessage.success('认证方法已保存为停用、待校验草稿。');
    methodDialogVisible.value = false;
    await refreshSelectedSchool();
  } catch (error) {
    actionError.value = errorMessage(error);
    ElMessage.error(actionError.value);
  } finally {
    methodDialogSubmitting.value = false;
  }
}

async function validateMethod(
  method: AdminVerificationMethodConfig,
  enable: boolean,
) {
  const school = selectedSchool.value;
  if (!school || actionKey.value) return;
  const reason = await promptReason(
    enable ? '校验实时依赖并启用该方法' : '重新校验并保持该方法停用',
  );
  if (!reason) return;
  actionKey.value = `method:${method.method}:${enable}`;
  try {
    await validateVerificationMethod(school.schoolCode, method.method, {
      enable,
      expectedRevision: method.configRevision,
      reason,
    });
    ElMessage.success(
      enable
        ? '方法结构与实时依赖校验通过，已启用。'
        : '方法校验完成并保持停用。',
    );
    await refreshSelectedSchool();
  } catch (error) {
    actionError.value = errorMessage(error);
    ElMessage.error(actionError.value);
  } finally {
    actionKey.value = '';
  }
}

async function switchSnapshot(
  snapshot: RosterSnapshot,
  action: 'activate' | 'rollback',
) {
  const school = selectedSchool.value;
  if (!school || actionKey.value) return;
  const reason = await promptReason(
    action === 'activate' ? '激活该完整学籍快照' : '回滚到该历史完整快照',
  );
  if (!reason) return;
  actionKey.value = `snapshot:${snapshot.id}:${action}`;
  try {
    const body = { allowSourceRegression: false, reason };
    await (action === 'activate'
      ? activateRosterSnapshot(school.schoolCode, snapshot.id, body)
      : rollbackRosterSnapshot(school.schoolCode, snapshot.id, body));
    ElMessage.success(
      action === 'activate'
        ? '快照已原子激活。'
        : '快照已原子回滚并触发凭据重评估。',
    );
    await refreshSelectedSchool();
  } catch (error) {
    actionError.value = errorMessage(error);
    ElMessage.error(actionError.value);
  } finally {
    actionKey.value = '';
  }
}

async function promptReason(title: string) {
  try {
    const result = await ElMessageBox.prompt(
      '请输入至少 4 个字的操作原因、工单或事件编号。',
      title,
      {
        confirmButtonText: '确认并审计',
        cancelButtonText: '取消',
        inputPattern: /^.{4,500}$/s,
        inputErrorMessage: '操作原因长度必须为 4–500 个字。',
        type: 'warning',
      },
    );
    return result.value.trim();
  } catch {
    return '';
  }
}

function methodLabel(method: string) {
  return (
    {
      manual_material_review: '人工材料审核',
      real_name_identity_check: '实名信息校验',
      school_sso: '学校统一身份认证',
      student_email_inbound_challenge: '学校邮箱发送验证邮件',
      student_email_outbound_otp: '学校邮箱接收验证码',
    }[method] ?? method
  );
}

function statusType(status: string): 'danger' | 'info' | 'success' | 'warning' {
  if (
    status === 'active' ||
    status === 'succeeded' ||
    status === 'healthy' ||
    status === 'ready' ||
    status === 'valid'
  )
    return 'success';
  if (
    status === 'degraded' ||
    status === 'pending' ||
    status === 'started' ||
    status === 'staging' ||
    status === 'validating'
  )
    return 'warning';
  if (
    status === 'failed' ||
    status === 'timed_out' ||
    status === 'invalid' ||
    status === 'offline' ||
    status === 'revoked' ||
    status === 'unavailable'
  )
    return 'danger';
  return 'info';
}

function freshnessHours(seconds: number) {
  if (seconds % 86_400 === 0) return `${seconds / 86_400} 天`;
  const hours = seconds / 3600;
  return Number.isInteger(hours) ? `${hours} 小时` : `${hours.toFixed(1)} 小时`;
}

function rosterSyncResultLabel(request: AdminRosterSyncRequest) {
  const result = request.resultCode || request.status;
  return (
    {
      cancelled: '同步任务已取消。',
      delivery_attempts_exhausted:
        '连接器多次领取后仍未完成，请检查节点和上游状态。',
      delivery_deadline_exceeded: '连接器未在 24 小时内领取并完成任务。',
      schema_unknown: 'Oracle 返回结构与批准的字段映射不一致。',
      secret_unavailable: '校园连接器缺少 Oracle 只读账号 secret。',
      snapshot_encryption_failed: '连接器无法加密快照，请检查快照公钥。',
      tls_failure: 'Oracle TLS 或校园 CA 校验失败。',
      upstream_unavailable: 'Oracle 暂时不可用或查询失败。',
    }[result] ?? `同步未完成（${result}）。`
  );
}

function qualityPercentage(snapshot: RosterSnapshot) {
  if (snapshot.qualityChecks.length === 0) return 0;
  const passed = snapshot.qualityChecks.filter(
    (check) => check.status === 'passed',
  ).length;
  return Math.round((passed * 100) / snapshot.qualityChecks.length);
}

function qualityProgressStatus(snapshot: RosterSnapshot) {
  return snapshot.qualityChecks.some((check) => check.status === 'failed')
    ? ('exception' as const)
    : undefined;
}

function errorMessage(error: unknown) {
  return error instanceof Error && error.message
    ? error.message
    : '请求失败，请稍后重试。';
}

onMounted(() => fetchSchools());
onBeforeUnmount(() => {
  if (rosterSyncPollTimer) clearTimeout(rosterSyncPollTimer);
});
</script>

<template>
  <AdminContentLayout title="学校认证配置" :total="schools.length">
    <template #toolbar>
      <ElButton v-if="canUpdateConfig" plain @click="openCreateSchool">
        新增学校配置
      </ElButton>
      <ElButton
        type="primary"
        :loading="loading"
        @click="refreshSelectedSchool"
      >
        刷新状态
      </ElButton>
    </template>

    <div class="control-overview">
      <ElCard shadow="never" class="control-stat">
        <ElStatistic title="认证白名单学校" :value="enabledSchoolCount" />
        <span>目录学校不会自动进入白名单</span>
      </ElCard>
      <ElCard shadow="never" class="control-stat">
        <ElStatistic title="当前学校健康方法" :value="healthyMethodCount" />
        <span>方法之间独立降级，不做全校级联熔断</span>
      </ElCard>
      <ElCard shadow="never" class="control-stat">
        <ElStatistic
          title="当前完整快照"
          :value="activeSnapshot?.rowCount ?? 0"
        />
        <span>{{
          activeSnapshot
            ? `revision ${activeSnapshot.activationRevision ?? '—'}`
            : '尚未激活名册'
        }}</span>
      </ElCard>
    </div>

    <ElAlert
      v-if="loadError"
      :title="loadError"
      type="error"
      :closable="false"
      show-icon
      class="admin-load-error"
    />
    <ElAlert
      v-if="actionError"
      :title="actionError"
      type="error"
      closable
      show-icon
      class="admin-load-error"
      @close="actionError = ''"
    />

    <div class="control-layout">
      <aside class="school-rail">
        <div class="rail-heading">
          <div>
            <strong>学校范围</strong>
            <span>只有已创建配置的目录项</span>
          </div>
        </div>
        <button
          v-for="school in schools"
          :key="school.schoolCode"
          type="button"
          class="school-option"
          :class="{
            'school-option--active': school.schoolCode === selectedSchoolCode,
          }"
          @click="
            selectedSchoolCode = school.schoolCode;
            fetchSchoolContext();
          "
        >
          <span class="school-option__name">{{ school.schoolName }}</span>
          <small>{{ school.schoolCode }}</small>
          <ElTag :type="school.enabled ? 'success' : 'info'" size="small">
            {{ school.enabled ? '已启用' : '停用' }}
          </ElTag>
        </button>
        <ElEmpty
          v-if="!loading && schools.length === 0"
          description="尚无学校认证配置"
          :image-size="72"
        />
      </aside>

      <main class="school-workspace">
        <ElSkeleton v-if="loading && !selectedSchool" :rows="10" animated />
        <ElEmpty
          v-else-if="!selectedSchool"
          description="选择或创建一个学校认证配置"
        />
        <template v-else>
          <header class="school-hero">
            <div>
              <div class="school-hero__eyebrow">
                {{ selectedSchool.schoolCode }} · revision
                {{ selectedSchool.configRevision }}
              </div>
              <h2>{{ selectedSchool.schoolName }}</h2>
              <div class="hero-tags">
                <ElTag :type="selectedSchool.enabled ? 'success' : 'info'">
                  {{
                    selectedSchool.enabled
                      ? '认证白名单已启用'
                      : '认证白名单停用'
                  }}
                </ElTag>
                <ElTag :type="statusType(selectedSchool.validationStatus)">
                  配置 {{ selectedSchool.validationStatus }}
                </ElTag>
                <ElTag type="info">
                  {{ selectedSchool.adapterID }}@{{
                    selectedSchool.adapterVersion
                  }}
                </ElTag>
              </div>
            </div>
            <div v-if="canUpdateConfig" class="hero-actions">
              <ElButton plain @click="openEditSchool">编辑草稿</ElButton>
              <ElButton
                v-if="!selectedSchool.enabled"
                type="primary"
                :loading="actionKey === 'school:true'"
                @click="validateSchool(true)"
              >
                校验并启用
              </ElButton>
              <ElButton
                v-else
                type="warning"
                plain
                :loading="actionKey === 'school:false'"
                @click="validateSchool(false)"
              >
                停用并重验
              </ElButton>
            </div>
          </header>

          <ElAlert
            v-if="selectedSchool.validationCode"
            :title="`最近校验结果：${selectedSchool.validationCode}`"
            type="warning"
            :closable="false"
            show-icon
            class="validation-alert"
          />

          <ElTabs v-model="activeTab" class="control-tabs">
            <ElTabPane label="认证方法" name="methods">
              <div class="tab-heading">
                <div>
                  <h3>学校能力注册表</h3>
                  <p>每个方法独立配置适配器、名册依赖、用户说明与健康状态。</p>
                </div>
                <ElButton
                  v-if="canUpdateConfig"
                  type="primary"
                  plain
                  @click="openCreateMethod"
                >
                  新增方法
                </ElButton>
              </div>
              <div class="method-list">
                <ElCard
                  v-for="method in selectedSchool.methods"
                  :key="method.method"
                  shadow="never"
                  class="method-card"
                >
                  <div class="method-card__header">
                    <div>
                      <span class="method-name">{{
                        method.displayName || methodLabel(method.method)
                      }}</span>
                      <small>{{ method.method }}</small>
                    </div>
                    <div class="hero-tags">
                      <ElTag :type="method.enabled ? 'success' : 'info'">
                        {{ method.enabled ? '已启用' : '停用' }}
                      </ElTag>
                      <ElTag :type="statusType(method.healthStatus)">
                        {{ method.healthStatus }}
                      </ElTag>
                    </div>
                  </div>
                  <p>{{ method.description }}</p>
                  <ElDescriptions :column="2" size="small">
                    <ElDescriptionsItem label="适配器">
                      {{ method.adapterID }}@{{ method.adapterVersion }}
                    </ElDescriptionsItem>
                    <ElDescriptionsItem label="名册依赖">
                      {{ method.rosterDependency }}
                    </ElDescriptionsItem>
                    <ElDescriptionsItem label="连接器操作">
                      {{ method.connectorOperationKey || '不依赖连接器' }}
                    </ElDescriptionsItem>
                    <ElDescriptionsItem label="凭据有效期">
                      {{
                        method.credentialTTLSeconds
                          ? freshnessHours(method.credentialTTLSeconds)
                          : '按策略派生'
                      }}
                    </ElDescriptionsItem>
                  </ElDescriptions>
                  <ElAlert
                    v-if="method.validationCode || method.healthCode"
                    :title="method.healthCode || method.validationCode || ''"
                    type="warning"
                    :closable="false"
                    class="method-warning"
                  />
                  <div v-if="canUpdateConfig" class="method-actions">
                    <ElButton
                      size="small"
                      plain
                      @click="openEditMethod(method)"
                    >
                      编辑草稿
                    </ElButton>
                    <ElButton
                      v-if="!method.enabled"
                      size="small"
                      type="primary"
                      :loading="actionKey === `method:${method.method}:true`"
                      @click="validateMethod(method, true)"
                    >
                      校验并启用
                    </ElButton>
                    <ElButton
                      v-else
                      size="small"
                      type="warning"
                      plain
                      :loading="actionKey === `method:${method.method}:false`"
                      @click="validateMethod(method, false)"
                    >
                      停用并重验
                    </ElButton>
                  </div>
                </ElCard>
              </div>
              <ElEmpty
                v-if="selectedSchool.methods.length === 0"
                description="尚未配置认证方法"
              />
            </ElTabPane>

            <ElTabPane label="学籍快照" name="snapshots">
              <div class="tab-heading">
                <div>
                  <h3>版本化完整快照</h3>
                  <p>
                    同步结果先进入 staging，经质量门禁后原子激活；仅 upsert
                    不能替代完整快照。
                  </p>
                </div>
                <div class="tab-actions">
                  <ElButton
                    v-if="canManageConnector"
                    type="primary"
                    :disabled="!hasRunnableRosterConnector"
                    :loading="
                      actionKey === 'roster-sync:create' || rosterSyncInFlight
                    "
                    @click="triggerRosterSync"
                  >
                    {{ rosterSyncInFlight ? '同步执行中' : '立即完整同步' }}
                  </ElButton>
                  <ElButton
                    :loading="contextLoading"
                    plain
                    @click="fetchSchoolContext"
                  >
                    刷新
                  </ElButton>
                </div>
              </div>
              <ElAlert
                v-if="latestRosterSync"
                :title="`最近手动同步：${latestRosterSync.status} · ${formatAdminDateTime(latestRosterSync.createdAt)}`"
                :description="
                  rosterSyncInFlight
                    ? `连接器领取次数 ${latestRosterSync.claimAttempts}；任务最多保留 24 小时。`
                    : latestRosterSync.status === 'succeeded'
                      ? '快照已经由中心端校验、导入；是否自动激活取决于当前学校策略。'
                      : rosterSyncResultLabel(latestRosterSync)
                "
                :type="
                  latestRosterSync.status === 'succeeded'
                    ? 'success'
                    : rosterSyncInFlight
                      ? 'info'
                      : 'warning'
                "
                :closable="false"
                show-icon
                class="validation-alert"
              />
              <ElAlert
                v-else-if="canManageConnector && !hasRunnableRosterConnector"
                title="没有可执行完整同步的校园连接器"
                description="请先完成节点心跳、Oracle TLS、操作 allowlist 和 secret reference 配置。"
                type="warning"
                :closable="false"
                show-icon
                class="validation-alert"
              />
              <ElAlert
                v-if="!activeSnapshot"
                title="尚无活动快照，所有依赖名册的方法都应保持不可用。"
                type="warning"
                :closable="false"
                show-icon
                class="validation-alert"
              />
              <PersistentAdminTable
                table-key="users.targetRosterSnapshots"
                :data="snapshots"
                :loading="contextLoading"
                row-key="id"
                stripe
              >
                <PersistentAdminTableColumn
                  column-key="version"
                  label="源版本"
                  prop="sourceVersion"
                  :default-min-width="180"
                />
                <PersistentAdminTableColumn
                  column-key="status"
                  label="状态"
                  :default-width="116"
                >
                  <template #default="{ row }">
                    <ElTag :type="statusType(row.status)">
                      {{ row.isCurrent ? 'current · ' : '' }}{{ row.status }}
                    </ElTag>
                  </template>
                </PersistentAdminTableColumn>
                <PersistentAdminTableColumn
                  column-key="rows"
                  label="行数 / 可准入"
                  :default-width="148"
                >
                  <template #default="{ row }">
                    {{ row.rowCount }} / {{ row.eligibleRowCount }}
                  </template>
                </PersistentAdminTableColumn>
                <PersistentAdminTableColumn
                  column-key="mode"
                  label="导入模式"
                  prop="importMode"
                  :default-width="132"
                />
                <PersistentAdminTableColumn
                  column-key="cutoff"
                  label="源截止时间"
                  :default-width="176"
                >
                  <template #default="{ row }">
                    {{ formatAdminDateTime(row.sourceCutoffAt) }}
                  </template>
                </PersistentAdminTableColumn>
                <PersistentAdminTableColumn
                  column-key="quality"
                  label="质量门禁"
                  :default-width="168"
                >
                  <template #default="{ row }">
                    <ElProgress
                      :percentage="qualityPercentage(row)"
                      :status="qualityProgressStatus(row)"
                      :stroke-width="8"
                    />
                  </template>
                </PersistentAdminTableColumn>
                <PersistentAdminTableColumn
                  v-if="canActivateRoster"
                  column-key="actions"
                  label="操作"
                  fixed="right"
                  :default-width="180"
                >
                  <template #default="{ row }">
                    <ElButton
                      v-if="row.status === 'ready'"
                      size="small"
                      type="primary"
                      :loading="actionKey === `snapshot:${row.id}:activate`"
                      @click="switchSnapshot(row, 'activate')"
                    >
                      激活
                    </ElButton>
                    <ElButton
                      v-if="
                        row.status === 'superseded' ||
                        row.status === 'rolled_back'
                      "
                      size="small"
                      type="warning"
                      plain
                      :loading="actionKey === `snapshot:${row.id}:rollback`"
                      @click="switchSnapshot(row, 'rollback')"
                    >
                      回滚到此版本
                    </ElButton>
                  </template>
                </PersistentAdminTableColumn>
              </PersistentAdminTable>
            </ElTabPane>

            <ElTabPane label="连接器健康" name="connectors">
              <div class="tab-heading">
                <div>
                  <h3>校园连接器只读健康</h3>
                  <p>
                    本视图不会返回证书指纹、公钥、目标主机、端口、路由或 secret
                    reference。
                  </p>
                </div>
                <ElButton
                  :loading="contextLoading"
                  plain
                  @click="fetchSchoolContext"
                >
                  刷新心跳
                </ElButton>
              </div>
              <div class="connector-list">
                <ElCard
                  v-for="node in connectors"
                  :key="node.id"
                  shadow="never"
                  class="connector-card"
                >
                  <div class="connector-card__header">
                    <div>
                      <strong>{{ node.displayName }}</strong>
                      <span>
                        协议 {{ node.protocolVersion }} · 软件
                        {{ node.softwareVersion }}
                      </span>
                    </div>
                    <ElTag :type="statusType(node.status)">
                      {{ node.status }}
                    </ElTag>
                  </div>
                  <ElDescriptions :column="2" border size="small">
                    <ElDescriptionsItem label="最近心跳">
                      {{
                        node.lastHeartbeatAt
                          ? formatAdminDateTime(node.lastHeartbeatAt)
                          : '从未上报'
                      }}
                    </ElDescriptionsItem>
                    <ElDescriptionsItem label="证书到期">
                      {{ formatAdminDateTime(node.certificateNotAfter) }}
                    </ElDescriptionsItem>
                    <ElDescriptionsItem label="并发上限">
                      {{ node.maxConcurrency }}
                    </ElDescriptionsItem>
                    <ElDescriptionsItem label="健康分类">
                      {{ node.lastHealthCode || '正常' }}
                    </ElDescriptionsItem>
                  </ElDescriptions>
                  <div class="operation-chips">
                    <ElTag
                      v-for="operation in node.operations"
                      :key="operation.operationKey"
                      :type="statusType(operation.healthStatus)"
                      effect="plain"
                    >
                      {{ operation.operationKey }} ·
                      {{ operation.healthStatus }}
                    </ElTag>
                  </div>
                </ElCard>
              </div>
              <ElEmpty
                v-if="!contextLoading && connectors.length === 0"
                description="该学校尚未绑定已批准连接器"
              />
            </ElTabPane>

            <ElTabPane label="策略摘要" name="policy">
              <ElDescriptions :column="2" border>
                <ElDescriptionsItem label="学校适配器">
                  {{ selectedSchool.adapterID }}@{{
                    selectedSchool.adapterVersion
                  }}
                </ElDescriptionsItem>
                <ElDescriptionsItem label="邮箱域">
                  {{ selectedSchool.emailDomains.join('、') || '未配置' }}
                </ElDescriptionsItem>
                <ElDescriptionsItem label="同步周期">
                  {{
                    freshnessHours(selectedSchool.snapshotSyncIntervalSeconds)
                  }}
                </ElDescriptionsItem>
                <ElDescriptionsItem label="告警阈值">
                  {{
                    freshnessHours(selectedSchool.snapshotWarningAfterSeconds)
                  }}
                </ElDescriptionsItem>
                <ElDescriptionsItem label="硬失效阈值">
                  {{ freshnessHours(selectedSchool.snapshotHardExpirySeconds) }}
                </ElDescriptionsItem>
                <ElDescriptionsItem label="宽限期">
                  {{ freshnessHours(selectedSchool.snapshotGraceSeconds) }}
                </ElDescriptionsItem>
                <ElDescriptionsItem label="质量通过后自动激活">
                  {{
                    selectedSchool.snapshotAutoActivate ? '已开启' : '手动激活'
                  }}
                </ElDescriptionsItem>
              </ElDescriptions>
              <div class="policy-json-grid">
                <ElCard shadow="never">
                  <strong>学号策略</strong>
                  <pre>{{
                    JSON.stringify(selectedSchool.studentIDPolicy, null, 2)
                  }}</pre>
                </ElCard>
                <ElCard shadow="never">
                  <strong>姓名策略</strong>
                  <pre>{{
                    JSON.stringify(selectedSchool.nameMatchPolicy, null, 2)
                  }}</pre>
                </ElCard>
                <ElCard shadow="never">
                  <strong>学籍准入策略</strong>
                  <pre>{{
                    JSON.stringify(selectedSchool.enrollmentPolicy, null, 2)
                  }}</pre>
                </ElCard>
                <ElCard shadow="never">
                  <strong>人工表单 schema</strong>
                  <pre>{{
                    JSON.stringify(selectedSchool.manualFormSchema, null, 2)
                  }}</pre>
                </ElCard>
              </div>
            </ElTabPane>
          </ElTabs>
        </template>
      </main>
    </div>

    <SchoolConfigDialog
      v-model:visible="schoolDialogVisible"
      :school="editingSchool"
      :submitting="schoolDialogSubmitting"
      @save="saveSchool"
    />
    <MethodConfigDialog
      v-model:visible="methodDialogVisible"
      :method="editingMethod"
      :submitting="methodDialogSubmitting"
      @save="saveMethod"
    />
  </AdminContentLayout>
</template>

<style scoped>
.control-overview {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 16px;
  margin-bottom: 16px;
}

.control-stat {
  background: linear-gradient(
    145deg,
    var(--el-bg-color),
    var(--el-fill-color-extra-light)
  );
  border: 1px solid color-mix(in srgb, var(--el-border-color) 76%, transparent);
  border-radius: 16px;
}

.control-stat span {
  display: block;
  margin-top: 8px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.admin-load-error,
.validation-alert {
  margin-bottom: 16px;
}

.control-layout {
  display: grid;
  grid-template-columns: 248px minmax(0, 1fr);
  gap: 18px;
  align-items: start;
}

.school-rail {
  position: sticky;
  top: 16px;
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 12px;
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 16px;
}

.rail-heading {
  padding: 8px 8px 12px;
}

.rail-heading div {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.rail-heading span,
.school-option small {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.school-option {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 4px 8px;
  width: 100%;
  padding: 12px;
  color: var(--el-text-color-primary);
  text-align: left;
  cursor: pointer;
  background: transparent;
  border: 1px solid transparent;
  border-radius: 12px;
  transition: 160ms ease;
}

.school-option:hover,
.school-option--active {
  background: var(--el-fill-color-light);
  border-color: color-mix(
    in srgb,
    var(--el-color-primary) 38%,
    var(--el-border-color)
  );
}

.school-option__name {
  overflow: hidden;
  text-overflow: ellipsis;
  font-weight: 650;
  white-space: nowrap;
}

.school-option :deep(.el-tag) {
  grid-row: 1 / span 2;
  grid-column: 2;
  align-self: center;
}

.school-workspace {
  min-width: 0;
  padding: 20px;
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 18px;
}

.school-hero,
.tab-heading,
.connector-card__header,
.method-card__header {
  display: flex;
  gap: 16px;
  align-items: flex-start;
  justify-content: space-between;
}

.school-hero h2 {
  margin: 6px 0 10px;
  font-size: clamp(22px, 3vw, 30px);
  letter-spacing: -0.02em;
}

.school-hero__eyebrow {
  font-size: 12px;
  font-weight: 600;
  color: var(--el-text-color-secondary);
  text-transform: uppercase;
  letter-spacing: 0.04em;
}

.hero-tags,
.hero-actions,
.tab-actions,
.method-actions,
.operation-chips {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.control-tabs {
  margin-top: 18px;
}

.tab-heading {
  margin-bottom: 16px;
}

.tab-heading h3 {
  margin: 0;
  font-size: 18px;
}

.tab-heading p,
.method-card p {
  margin: 6px 0 0;
  font-size: 13px;
  color: var(--el-text-color-secondary);
}

.method-list,
.connector-list {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 14px;
}

.method-card,
.connector-card {
  border-radius: 14px;
}

.method-card__header > div:first-child,
.connector-card__header > div:first-child {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.method-name,
.connector-card__header strong {
  font-size: 16px;
  font-weight: 700;
}

.method-card__header small,
.connector-card__header span {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.method-card :deep(.el-descriptions) {
  margin: 16px 0;
}

.method-warning {
  margin-bottom: 12px;
}

.connector-card :deep(.el-descriptions) {
  margin: 16px 0;
}

.policy-json-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 14px;
  margin-top: 16px;
}

.policy-json-grid pre {
  max-height: 280px;
  padding: 12px;
  overflow: auto;
  font-size: 12px;
  line-height: 1.55;
  color: var(--el-text-color-regular);
  background: var(--el-fill-color-extra-light);
  border-radius: 10px;
}

@media (max-width: 1100px) {
  .control-layout {
    grid-template-columns: 1fr;
  }

  .school-rail {
    position: static;
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .rail-heading {
    grid-column: 1 / -1;
  }
}

@media (max-width: 780px) {
  .control-overview,
  .method-list,
  .connector-list,
  .policy-json-grid,
  .school-rail {
    grid-template-columns: 1fr;
  }

  .school-hero,
  .tab-heading {
    flex-direction: column;
  }
}
</style>
