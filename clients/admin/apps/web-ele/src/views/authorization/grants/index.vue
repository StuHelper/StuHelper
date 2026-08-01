<script setup lang="ts">
import type {
  AssignableAuthorizationRole,
  AuthorizationDesiredState,
  AuthorizationGrant,
  AuthorizationProjectionStatus,
  AuthorizationRole,
  CreateAuthorizationGrantRequest,
  ListAuthorizationGrantsParams,
} from '#/api/admin';

import { onMounted, reactive, ref } from 'vue';

import {
  ElAlert,
  ElButton,
  ElDialog,
  ElForm,
  ElFormItem,
  ElInput,
  ElInputNumber,
  ElMessage,
  ElMessageBox,
  ElOption,
  ElPagination,
  ElSelect,
  ElTag,
  ElTooltip,
} from 'element-plus';

import {
  createAuthorizationGrant,
  listAuthorizationGrants,
  reconcileAllAuthorizationProjections,
  reconcileAuthorizationGrant,
  revokeAuthorizationGrant,
} from '#/api/admin';
import { $t } from '#/locales';

import PersistentAdminTable from '../../shared/admin-table/PersistentAdminTable.vue';
import PersistentAdminTableColumn from '../../shared/admin-table/PersistentAdminTableColumn.vue';
import AdminContentLayout from '../../shared/AdminContentLayout.vue';
import { formatAdminDateTime } from '../../shared/display';

const roleOptions: AuthorizationRole[] = [
  'super_admin',
  'school_admin',
  'section_admin',
  'section_moderator',
  'section_reviewer',
];
const assignableRoleOptions: AssignableAuthorizationRole[] = [
  'school_admin',
  'section_admin',
  'section_moderator',
  'section_reviewer',
];
const sectionRoles = new Set<AuthorizationRole>([
  'section_admin',
  'section_moderator',
  'section_reviewer',
]);

const loading = ref(false);
const loadError = ref('');
const actionError = ref('');
const items = ref<AuthorizationGrant[]>([]);
const total = ref(0);
const actionGrantID = ref<number>();
const rebuilding = ref(false);
let fetchRequestSeq = 0;

const query = reactive<{
  desiredState: '' | AuthorizationDesiredState;
  page: number;
  pageSize: number;
  projectionStatus: '' | AuthorizationProjectionStatus;
  role: '' | AuthorizationRole;
  subjectUserID?: number;
}>({
  desiredState: '',
  page: 1,
  pageSize: 20,
  projectionStatus: '',
  role: '',
  subjectUserID: undefined,
});

const createVisible = ref(false);
const creating = ref(false);
const createForm = reactive<{
  reason: string;
  role: AssignableAuthorizationRole;
  schoolID?: number;
  subjectUserID?: number;
}>({
  reason: '',
  role: 'school_admin',
  schoolID: undefined,
  subjectUserID: undefined,
});

function roleLabel(role: AuthorizationRole) {
  return $t(`admin.authorization.roles.${role}`);
}

function stateLabel(
  state: AuthorizationDesiredState | AuthorizationProjectionStatus,
) {
  return $t(`admin.authorization.states.${state}`);
}

function sourceLabel(source: AuthorizationGrant['source']) {
  return $t(`admin.authorization.sources.${source}`);
}

function stateTagType(
  state: AuthorizationDesiredState | AuthorizationProjectionStatus,
) {
  switch (state) {
    case 'applied':
    case 'granted': {
      return 'success';
    }
    case 'failed':
    case 'revoked': {
      return 'danger';
    }
    default: {
      return 'warning';
    }
  }
}

function scopeLabel(grant: AuthorizationGrant) {
  if (grant.schoolID === undefined) {
    return $t('admin.authorization.global');
  }
  if (grant.sectionID) {
    return `${grant.schoolID} / ${grant.sectionID}`;
  }
  return String(grant.schoolID);
}

function buildListParams(): ListAuthorizationGrantsParams {
  return {
    page: query.page,
    pageSize: query.pageSize,
    ...(query.subjectUserID ? { subjectUserID: query.subjectUserID } : {}),
    ...(query.role ? { role: query.role } : {}),
    ...(query.desiredState ? { desiredState: query.desiredState } : {}),
    ...(query.projectionStatus
      ? { projectionStatus: query.projectionStatus }
      : {}),
  };
}

async function fetchData() {
  const requestSeq = ++fetchRequestSeq;
  loading.value = true;
  loadError.value = '';
  try {
    const data = await listAuthorizationGrants(buildListParams());
    if (requestSeq !== fetchRequestSeq) return;
    items.value = data.items;
    total.value = data.total;
  } catch (error) {
    if (requestSeq !== fetchRequestSeq) return;
    loadError.value = adminErrorMessage(error);
  } finally {
    if (requestSeq === fetchRequestSeq) {
      loading.value = false;
    }
  }
}

function resetPageAndFetch() {
  query.page = 1;
  void fetchData();
}

function resetFilters() {
  query.subjectUserID = undefined;
  query.role = '';
  query.desiredState = '';
  query.projectionStatus = '';
  resetPageAndFetch();
}

function openCreate() {
  createForm.subjectUserID = undefined;
  createForm.role = 'school_admin';
  createForm.schoolID = undefined;
  createForm.reason = '';
  createVisible.value = true;
}

async function submitCreate() {
  if (!createForm.subjectUserID || createForm.subjectUserID <= 0) {
    ElMessage.warning($t('admin.authorization.validation.subjectRequired'));
    return;
  }
  if (!createForm.schoolID || createForm.schoolID <= 0) {
    ElMessage.warning($t('admin.authorization.validation.schoolRequired'));
    return;
  }
  const reason = createForm.reason.trim();
  if (!reason) {
    ElMessage.warning($t('admin.authorization.validation.reasonRequired'));
    return;
  }

  const request: CreateAuthorizationGrantRequest = {
    reason,
    role: createForm.role,
    subjectUserID: createForm.subjectUserID,
  };
  if (createForm.schoolID) {
    request.schoolID = createForm.schoolID;
    if (sectionRoles.has(createForm.role)) {
      request.sectionID = `school_${createForm.schoolID}_review_moderation`;
    }
  }

  creating.value = true;
  actionError.value = '';
  try {
    await createAuthorizationGrant(request);
    ElMessage.success($t('admin.authorization.created'));
    createVisible.value = false;
    query.page = 1;
    await fetchData();
  } catch (error) {
    handleActionError(error);
  } finally {
    creating.value = false;
  }
}

async function promptForReason(
  message: string,
  title: string,
  confirmButtonText: string,
) {
  const result = await ElMessageBox.prompt(message, title, {
    cancelButtonText: $t('admin.common.cancel'),
    confirmButtonText,
    inputPlaceholder: $t('admin.authorization.reasonPlaceholder'),
    inputValidator: (value) =>
      (value ?? '').trim().length > 0 ||
      $t('admin.authorization.validation.reasonRequired'),
    type: 'warning',
  });
  return (result.value ?? '').trim();
}

async function revokeGrant(grant: AuthorizationGrant) {
  try {
    const reason = await promptForReason(
      $t('admin.authorization.revokePrompt'),
      $t('admin.authorization.revoke'),
      $t('admin.authorization.revoke'),
    );
    actionGrantID.value = grant.id;
    actionError.value = '';
    await revokeAuthorizationGrant(grant.id, reason);
    ElMessage.success($t('admin.authorization.revoked'));
    await fetchData();
  } catch (error) {
    if (!isDialogCancellation(error)) handleActionError(error);
  } finally {
    actionGrantID.value = undefined;
  }
}

async function reconcileGrant(grant: AuthorizationGrant) {
  try {
    const reason = await promptForReason(
      $t('admin.authorization.reconcilePrompt'),
      $t('admin.authorization.reconcile'),
      $t('admin.authorization.reconcile'),
    );
    actionGrantID.value = grant.id;
    actionError.value = '';
    await reconcileAuthorizationGrant(grant.id, reason);
    ElMessage.success($t('admin.authorization.reconciled'));
    await fetchData();
  } catch (error) {
    if (!isDialogCancellation(error)) handleActionError(error);
  } finally {
    actionGrantID.value = undefined;
  }
}

async function rebuildAll() {
  try {
    const reason = await promptForReason(
      $t('admin.authorization.rebuildPrompt'),
      $t('admin.authorization.rebuild'),
      $t('admin.authorization.rebuild'),
    );
    rebuilding.value = true;
    actionError.value = '';
    const result = await reconcileAllAuthorizationProjections(reason);
    ElMessage.success(
      $t('admin.authorization.rebuilt', { count: result.queued }),
    );
    await fetchData();
  } catch (error) {
    if (!isDialogCancellation(error)) handleActionError(error);
  } finally {
    rebuilding.value = false;
  }
}

function isDialogCancellation(error: unknown) {
  return error === 'cancel' || error === 'close';
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
    :title="$t('admin.routes.authorization.grants')"
    :total="total"
  >
    <template #actions>
      <ElButton :loading="rebuilding" plain @click="rebuildAll">
        {{ $t('admin.authorization.rebuild') }}
      </ElButton>
      <ElButton type="primary" @click="openCreate">
        {{ $t('admin.authorization.create') }}
      </ElButton>
    </template>

    <template #toolbar>
      <ElInputNumber
        v-model="query.subjectUserID"
        class="admin-toolbar-control--wide"
        :controls="false"
        :min="1"
        :placeholder="$t('admin.authorization.subjectUserId')"
        @keyup.enter="resetPageAndFetch"
      />
      <ElSelect
        v-model="query.role"
        clearable
        class="admin-toolbar-control"
        :placeholder="$t('admin.authorization.role')"
        :teleported="false"
      >
        <ElOption
          v-for="role in roleOptions"
          :key="role"
          :label="roleLabel(role)"
          :value="role"
        />
      </ElSelect>
      <ElSelect
        v-model="query.desiredState"
        clearable
        class="admin-toolbar-control"
        :placeholder="$t('admin.authorization.desiredState')"
        :teleported="false"
      >
        <ElOption :label="stateLabel('granted')" value="granted" />
        <ElOption :label="stateLabel('revoked')" value="revoked" />
      </ElSelect>
      <ElSelect
        v-model="query.projectionStatus"
        clearable
        class="admin-toolbar-control"
        :placeholder="$t('admin.authorization.projectionStatus')"
        :teleported="false"
      >
        <ElOption :label="stateLabel('pending')" value="pending" />
        <ElOption :label="stateLabel('applied')" value="applied" />
        <ElOption :label="stateLabel('failed')" value="failed" />
      </ElSelect>
      <ElButton type="primary" @click="resetPageAndFetch">
        {{ $t('admin.common.query') }}
      </ElButton>
      <ElButton @click="resetFilters">
        {{ $t('admin.common.reset') }}
      </ElButton>
    </template>

    <div class="authorization-flow" aria-label="Authorization control plane">
      <div class="authorization-flow__node">
        <strong>{{ $t('admin.authorization.controlPlane.database') }}</strong>
        <span>{{ $t('admin.authorization.controlPlane.databaseHint') }}</span>
      </div>
      <div class="authorization-flow__connector" aria-hidden="true">→</div>
      <div class="authorization-flow__node authorization-flow__node--fence">
        <strong>{{ $t('admin.authorization.controlPlane.fence') }}</strong>
        <span>{{ $t('admin.authorization.controlPlane.fenceHint') }}</span>
      </div>
      <div class="authorization-flow__connector" aria-hidden="true">→</div>
      <div class="authorization-flow__node">
        <strong>{{ $t('admin.authorization.controlPlane.projection') }}</strong>
        <span>{{ $t('admin.authorization.controlPlane.projectionHint') }}</span>
      </div>
    </div>

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

    <PersistentAdminTable
      table-key="authorization.grants"
      :loading="loading"
      :data="items"
      row-key="id"
      stripe
    >
      <PersistentAdminTableColumn
        column-key="subject"
        :label="$t('admin.authorization.subject')"
        :default-min-width="190"
      >
        <template #default="{ row }">
          <div class="admin-cell-title">
            {{ row.subjectDisplayName || row.subjectUsername }}
          </div>
          <span class="admin-sub-id">
            #{{ row.subjectUserID }} · {{ row.subjectUsername }}
          </span>
        </template>
      </PersistentAdminTableColumn>
      <PersistentAdminTableColumn
        column-key="role"
        :label="$t('admin.authorization.role')"
        :default-min-width="150"
      >
        <template #default="{ row }">
          <span class="authorization-role">{{ roleLabel(row.role) }}</span>
        </template>
      </PersistentAdminTableColumn>
      <PersistentAdminTableColumn
        column-key="scope"
        :label="$t('admin.authorization.scope')"
        :default-min-width="210"
        show-overflow-tooltip
      >
        <template #default="{ row }">
          <span class="admin-id-token">{{ scopeLabel(row) }}</span>
        </template>
      </PersistentAdminTableColumn>
      <PersistentAdminTableColumn
        column-key="source"
        :label="$t('admin.authorization.source')"
        :default-min-width="156"
      >
        <template #default="{ row }">
          <ElTag effect="plain">
            {{ sourceLabel(row.source) }}
          </ElTag>
        </template>
      </PersistentAdminTableColumn>
      <PersistentAdminTableColumn
        column-key="desiredState"
        :label="$t('admin.authorization.desiredState')"
        :default-width="108"
      >
        <template #default="{ row }">
          <ElTag :type="stateTagType(row.desiredState)" effect="light">
            {{ stateLabel(row.desiredState) }}
          </ElTag>
        </template>
      </PersistentAdminTableColumn>
      <PersistentAdminTableColumn
        column-key="projectionStatus"
        :label="$t('admin.authorization.projectionStatus')"
        :default-width="118"
      >
        <template #default="{ row }">
          <ElTag :type="stateTagType(row.projectionStatus)" effect="plain">
            {{ stateLabel(row.projectionStatus) }}
          </ElTag>
        </template>
      </PersistentAdminTableColumn>
      <PersistentAdminTableColumn
        column-key="activation"
        :label="$t('admin.authorization.active')"
        :default-width="126"
      >
        <template #default="{ row }">
          <span
            class="authorization-activation"
            :class="[{ 'authorization-activation--off': !row.activatedAt }]"
          >
            {{
              row.activatedAt
                ? $t('admin.authorization.active')
                : $t('admin.authorization.notActive')
            }}
          </span>
        </template>
      </PersistentAdminTableColumn>
      <PersistentAdminTableColumn
        column-key="revision"
        :label="$t('admin.authorization.revision')"
        :default-width="86"
      >
        <template #default="{ row }">
          <span class="admin-number">r{{ row.revision }}</span>
        </template>
      </PersistentAdminTableColumn>
      <PersistentAdminTableColumn
        column-key="reason"
        :label="$t('admin.authorization.reason')"
        :default-min-width="220"
        show-overflow-tooltip
      >
        <template #default="{ row }">
          <div class="admin-cell-body">{{ row.reason }}</div>
          <ElTooltip
            v-if="row.lastError"
            :content="row.lastError"
            placement="top"
          >
            <span class="authorization-error">
              {{ $t('admin.authorization.lastError') }}
            </span>
          </ElTooltip>
        </template>
      </PersistentAdminTableColumn>
      <PersistentAdminTableColumn
        column-key="updatedAt"
        :label="$t('admin.common.updatedAt')"
        :default-width="160"
      >
        <template #default="{ row }">
          <span class="admin-cell-muted">
            {{ formatAdminDateTime(row.updatedAt) }}
          </span>
        </template>
      </PersistentAdminTableColumn>
      <PersistentAdminTableColumn
        column-key="actions"
        fixed="right"
        :label="$t('admin.common.actions')"
        :default-width="190"
      >
        <template #default="{ row }">
          <div class="admin-action-group">
            <ElButton
              plain
              size="small"
              :loading="actionGrantID === row.id"
              @click="reconcileGrant(row)"
            >
              {{ $t('admin.authorization.reconcile') }}
            </ElButton>
            <ElButton
              v-if="
                row.desiredState === 'granted' &&
                row.source !== 'casdoor_org_admin'
              "
              plain
              size="small"
              type="danger"
              :loading="actionGrantID === row.id"
              @click="revokeGrant(row)"
            >
              {{ $t('admin.authorization.revoke') }}
            </ElButton>
          </div>
        </template>
      </PersistentAdminTableColumn>
    </PersistentAdminTable>

    <template #pagination>
      <ElPagination
        v-model:current-page="query.page"
        v-model:page-size="query.pageSize"
        background
        layout="total, sizes, prev, pager, next, jumper"
        :page-sizes="[10, 20, 50, 100]"
        :total="total"
        @current-change="fetchData"
        @size-change="resetPageAndFetch"
      />
    </template>

    <ElDialog
      v-model="createVisible"
      :title="$t('admin.authorization.createTitle')"
      width="min(560px, calc(100vw - 32px))"
    >
      <ElAlert
        class="authorization-create-hint"
        type="info"
        :closable="false"
        show-icon
        :title="$t('admin.authorization.createHint')"
      />
      <ElForm label-position="top" @submit.prevent="submitCreate">
        <ElFormItem :label="$t('admin.authorization.subjectUserId')" required>
          <ElInputNumber
            v-model="createForm.subjectUserID"
            :controls="false"
            :min="1"
            :placeholder="$t('admin.authorization.subjectUserIdPlaceholder')"
          />
        </ElFormItem>
        <ElFormItem :label="$t('admin.authorization.role')" required>
          <ElSelect v-model="createForm.role" :teleported="false">
            <ElOption
              v-for="role in assignableRoleOptions"
              :key="role"
              :label="roleLabel(role)"
              :value="role"
            />
          </ElSelect>
        </ElFormItem>
        <ElFormItem :label="$t('admin.authorization.schoolId')" required>
          <ElInputNumber
            v-model="createForm.schoolID"
            :controls="false"
            :min="1"
          />
        </ElFormItem>
        <ElFormItem :label="$t('admin.authorization.reason')" required>
          <ElInput
            v-model="createForm.reason"
            maxlength="500"
            :placeholder="$t('admin.authorization.reasonPlaceholder')"
            show-word-limit
            type="textarea"
            :rows="3"
          />
        </ElFormItem>
      </ElForm>
      <template #footer>
        <ElButton @click="createVisible = false">
          {{ $t('admin.common.cancel') }}
        </ElButton>
        <ElButton type="primary" :loading="creating" @click="submitCreate">
          {{ $t('admin.authorization.create') }}
        </ElButton>
      </template>
    </ElDialog>
  </AdminContentLayout>
</template>

<style scoped>
.authorization-flow {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto minmax(0, 1fr) auto minmax(0, 1fr);
  gap: 12px;
  align-items: center;
  padding: 16px 20px;
  background:
    linear-gradient(
      90deg,
      color-mix(in srgb, var(--el-color-primary) 5%, transparent),
      transparent 42%
    ),
    var(--el-fill-color-extra-light);
  border-bottom: 1px solid var(--el-border-color);
}

.authorization-flow__node {
  display: flex;
  flex-direction: column;
  gap: 3px;
  min-width: 0;
  padding: 12px 14px;
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-light);
  border-radius: 7px;
}

.authorization-flow__node--fence {
  border-color: color-mix(
    in srgb,
    var(--el-color-warning) 50%,
    var(--el-border-color)
  );
}

.authorization-flow__node strong {
  font-size: 13px;
  font-weight: 650;
}

.authorization-flow__node span {
  overflow: hidden;
  text-overflow: ellipsis;
  font-size: 12px;
  color: var(--el-text-color-secondary);
  white-space: nowrap;
}

.authorization-flow__connector {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  color: var(--el-color-primary);
}

.authorization-role {
  font-weight: 600;
}

.authorization-activation {
  display: inline-flex;
  gap: 6px;
  align-items: center;
  color: var(--el-color-success);
}

.authorization-activation::before {
  width: 7px;
  height: 7px;
  content: '';
  background: currentcolor;
  border-radius: 999px;
}

.authorization-activation--off {
  color: var(--el-text-color-secondary);
}

.authorization-error {
  display: block;
  margin-top: 5px;
  font-size: 12px;
  color: var(--el-color-danger);
  cursor: help;
}

.authorization-create-hint {
  margin-bottom: 16px;
}

:deep(.el-dialog .el-input-number),
:deep(.el-dialog .el-select) {
  width: 100%;
}

@media (max-width: 900px) {
  .authorization-flow {
    grid-template-columns: 1fr;
  }

  .authorization-flow__connector {
    height: 12px;
    text-align: center;
    transform: rotate(90deg);
  }

  .authorization-flow__node span {
    white-space: normal;
  }
}
</style>
