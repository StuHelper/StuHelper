<script setup lang="ts">
import type {
  OpenPlatformAppStatus,
  OpenPlatformAppWithScopes,
  OpenPlatformImportCasdoorAppRequest,
  OpenPlatformScope,
} from '#/api/admin';

import { computed, ref } from 'vue';

import { IconifyIcon } from '@vben/icons';

import {
  ElAlert,
  ElButton,
  ElLink,
  ElMessage,
  ElMessageBox,
  ElOption,
  ElPagination,
  ElPopconfirm,
  ElSelect,
  ElTag,
} from 'element-plus';

import {
  approveOpenPlatformApp,
  approveOpenPlatformRedirectURIRequest,
  approveOpenPlatformScope,
  getOpenPlatformAppList,
  importOpenPlatformCasdoorApp,
  rejectOpenPlatformRedirectURIRequest,
  rejectOpenPlatformScope,
  resumeOpenPlatformApp,
  revokeOpenPlatformApp,
  rotateOpenPlatformAppSecret,
  suspendOpenPlatformApp,
} from '#/api/admin';
import { adminErrorMessage, useAdminList } from '#/composables/use-admin-list';
import { $t } from '#/locales';
import { copyTextToClipboard } from '#/utils/clipboard';

import PersistentAdminTable from '../../shared/admin-table/PersistentAdminTable.vue';
import PersistentAdminTableColumn from '../../shared/admin-table/PersistentAdminTableColumn.vue';
import AdminContentLayout from '../../shared/AdminContentLayout.vue';
import {
  compactID,
  formatAdminDateTime,
  formatNullableText,
} from '../../shared/display';
import ImportCasdoorAppDialog from './ImportCasdoorAppDialog.vue';
import ResourceGrantsDialog from './ResourceGrantsDialog.vue';

const actionLoading = ref(false);
const actionError = ref('');
const importDialogVisible = ref(false);
const importLoading = ref(false);
const resourceGrantsDialogVisible = ref(false);
const resourceGrantsApp = ref<null | OpenPlatformAppWithScopes>(null);
const issuedSecret = ref<null | {
  appName: string;
  clientID: string;
  secret: string;
}>(null);

const {
  fetchData,
  items: apps,
  loadError,
  loading,
  query,
  resetPageAndFetch,
  total,
} = useAdminList<
  OpenPlatformAppWithScopes,
  { page: number; pageSize: number; status: OpenPlatformAppStatus }
>({
  fetcher: (listQuery) => getOpenPlatformAppList(listQuery),
  initialQuery: {
    page: 1,
    pageSize: 20,
    status: 'pending',
  },
});

const pendingApps = computed(
  () => apps.value.filter((item) => item.app.status === 'pending').length,
);

async function handleApproveScope(
  app: OpenPlatformAppWithScopes,
  scope: OpenPlatformScope,
) {
  if (actionLoading.value) {
    return;
  }
  actionLoading.value = true;
  actionError.value = '';
  try {
    await approveOpenPlatformScope(app.app.id, scope);
    await fetchData();
  } catch (error) {
    handleActionError(error);
  } finally {
    actionLoading.value = false;
  }
}

async function handleRejectScope(
  app: OpenPlatformAppWithScopes,
  scope: OpenPlatformScope,
) {
  if (actionLoading.value) {
    return;
  }
  const decisionNote = await promptDecisionNote('rejectScopePrompt');
  if (decisionNote === null) {
    return;
  }
  actionLoading.value = true;
  actionError.value = '';
  try {
    await rejectOpenPlatformScope(app.app.id, scope, decisionNote);
    await fetchData();
  } catch (error) {
    handleActionError(error);
  } finally {
    actionLoading.value = false;
  }
}

async function handleApproveApp(app: OpenPlatformAppWithScopes) {
  if (actionLoading.value) {
    return;
  }
  actionLoading.value = true;
  actionError.value = '';
  try {
    const approved = await approveOpenPlatformApp(app.app.id);
    issuedSecret.value = {
      appName: approved.app.displayName,
      clientID: approved.app.clientID,
      secret: approved.clientSecret,
    };
    await fetchData();
  } catch (error) {
    handleActionError(error);
  } finally {
    actionLoading.value = false;
  }
}

async function copyIssuedSecret() {
  if (!issuedSecret.value) {
    return;
  }
  await copyToClipboard(
    issuedSecret.value.secret,
    $t('admin.openPlatform.apps.secretCopied'),
  );
}

async function copyToClipboard(text: string, successMessage: string) {
  const copied = await copyTextToClipboard(text);
  if (copied) {
    ElMessage.success(successMessage);
    return true;
  }
  ElMessage.error($t('admin.openPlatform.apps.secretCopyFailed'));
  return false;
}

async function handleImportCasdoorApp(
  payload: OpenPlatformImportCasdoorAppRequest,
) {
  if (importLoading.value) {
    return;
  }
  importLoading.value = true;
  actionError.value = '';
  try {
    const imported = await importOpenPlatformCasdoorApp(payload);
    if (imported.clientSecret) {
      issuedSecret.value = {
        appName: imported.app.displayName,
        clientID: imported.app.clientID,
        secret: imported.clientSecret,
      };
    } else {
      ElMessage.success(
        $t('admin.openPlatform.apps.importedWithoutSecret', {
          app: imported.app.displayName,
        }),
      );
    }
    importDialogVisible.value = false;
    query.status = 'approved';
    await fetchData();
  } catch (error) {
    handleActionError(error);
  } finally {
    importLoading.value = false;
  }
}

async function handleApproveRedirectURIRequest(
  app: OpenPlatformAppWithScopes,
  requestID: number,
) {
  if (actionLoading.value) {
    return;
  }
  actionLoading.value = true;
  actionError.value = '';
  try {
    await approveOpenPlatformRedirectURIRequest(app.app.id, requestID);
    await fetchData();
  } catch (error) {
    handleActionError(error);
  } finally {
    actionLoading.value = false;
  }
}

async function handleRejectRedirectURIRequest(
  app: OpenPlatformAppWithScopes,
  requestID: number,
) {
  if (actionLoading.value) {
    return;
  }
  const decisionNote = await promptDecisionNote('rejectRedirectPrompt');
  if (decisionNote === null) {
    return;
  }
  actionLoading.value = true;
  actionError.value = '';
  try {
    await rejectOpenPlatformRedirectURIRequest(
      app.app.id,
      requestID,
      decisionNote,
    );
    await fetchData();
  } catch (error) {
    handleActionError(error);
  } finally {
    actionLoading.value = false;
  }
}

async function handleRotateSecret(app: OpenPlatformAppWithScopes) {
  if (actionLoading.value) {
    return;
  }
  const reason = await promptLifecycleReason('rotateSecretPrompt');
  if (reason === null) {
    return;
  }
  actionLoading.value = true;
  actionError.value = '';
  try {
    const rotated = await rotateOpenPlatformAppSecret(app.app.id, reason);
    issuedSecret.value = {
      appName: rotated.app.displayName,
      clientID: rotated.app.clientID,
      secret: rotated.clientSecret,
    };
    await fetchData();
  } catch (error) {
    handleActionError(error);
  } finally {
    actionLoading.value = false;
  }
}

async function handleSuspendApp(app: OpenPlatformAppWithScopes) {
  if (actionLoading.value) {
    return;
  }
  const reason = await promptLifecycleReason('suspendPrompt');
  if (reason === null) {
    return;
  }
  actionLoading.value = true;
  actionError.value = '';
  try {
    await suspendOpenPlatformApp(app.app.id, reason);
    await fetchData();
  } catch (error) {
    handleActionError(error);
  } finally {
    actionLoading.value = false;
  }
}

async function handleResumeApp(app: OpenPlatformAppWithScopes) {
  if (actionLoading.value) {
    return;
  }
  const reason = await promptLifecycleReason('resumePrompt');
  if (reason === null) {
    return;
  }
  actionLoading.value = true;
  actionError.value = '';
  try {
    await resumeOpenPlatformApp(app.app.id, reason);
    await fetchData();
  } catch (error) {
    handleActionError(error);
  } finally {
    actionLoading.value = false;
  }
}

async function handleRevokeApp(app: OpenPlatformAppWithScopes) {
  if (actionLoading.value) {
    return;
  }
  const reason = await promptLifecycleReason('revokePrompt');
  if (reason === null) {
    return;
  }
  actionLoading.value = true;
  actionError.value = '';
  try {
    await revokeOpenPlatformApp(app.app.id, reason);
    await fetchData();
  } catch (error) {
    handleActionError(error);
  } finally {
    actionLoading.value = false;
  }
}

function handleOpenResourceGrants(app: OpenPlatformAppWithScopes) {
  resourceGrantsApp.value = app;
  resourceGrantsDialogVisible.value = true;
}

function handleActionError(error: unknown) {
  actionError.value = adminErrorMessage(error);
  ElMessage.error(actionError.value);
}

async function promptLifecycleReason(messageKey: string) {
  try {
    const result = await ElMessageBox.prompt(
      $t(`admin.openPlatform.apps.${messageKey}`),
      $t('admin.openPlatform.apps.reasonTitle'),
      {
        cancelButtonText: $t('admin.common.cancel'),
        confirmButtonText: $t('admin.common.confirm'),
        inputPlaceholder: $t('admin.openPlatform.apps.reasonPlaceholder'),
        inputValidator: (value) =>
          value.trim().length > 0 ||
          $t('admin.openPlatform.apps.reasonRequired'),
      },
    );
    return result.value.trim();
  } catch (_error) {
    void _error;
    return null;
  }
}

async function promptDecisionNote(messageKey: string) {
  try {
    const result = await ElMessageBox.prompt(
      $t(`admin.openPlatform.apps.${messageKey}`),
      $t('admin.openPlatform.apps.decisionNoteTitle'),
      {
        cancelButtonText: $t('admin.common.cancel'),
        confirmButtonText: $t('admin.common.confirm'),
        inputPlaceholder: $t('admin.openPlatform.apps.decisionNotePlaceholder'),
      },
    );
    return result.value.trim();
  } catch (_error) {
    void _error;
    return null;
  }
}

type TagType = 'danger' | 'info' | 'success' | 'warning';

function appStatusTag(status: string): TagType {
  const map: Record<string, TagType> = {
    approved: 'success',
    pending: 'warning',
    revoked: 'danger',
    suspended: 'info',
  };
  return map[status] || 'info';
}

function scopeStatusTag(status: string): TagType {
  const map: Record<string, TagType> = {
    approved: 'success',
    pending: 'warning',
    rejected: 'danger',
    withdrawn: 'info',
  };
  return map[status] || 'info';
}

function statusLabel(status: string) {
  return $t(`admin.openPlatform.apps.status.${status}`);
}

function scopeStatusLabel(status: string) {
  return $t(`admin.openPlatform.apps.scopeStatus.${status}`);
}

function approvedScopeCount(app: OpenPlatformAppWithScopes) {
  return app.scopes.filter((scope) => scope.status === 'approved').length;
}

function pendingScopes(app: OpenPlatformAppWithScopes) {
  return app.scopes.filter((scope) => scope.status === 'pending');
}

function pendingRedirectURIRequests(app: OpenPlatformAppWithScopes) {
  return app.redirectURIRequests.filter(
    (request) => request.status === 'pending',
  );
}

function canApproveApp(app: OpenPlatformAppWithScopes) {
  return app.app.status === 'pending' && approvedScopeCount(app) > 0;
}

function canRotateSecret(app: OpenPlatformAppWithScopes) {
  return app.app.status === 'approved' || app.app.status === 'suspended';
}

function canSuspendApp(app: OpenPlatformAppWithScopes) {
  return app.app.status === 'approved';
}

function canResumeApp(app: OpenPlatformAppWithScopes) {
  return app.app.status === 'suspended';
}

function canRevokeApp(app: OpenPlatformAppWithScopes) {
  return app.app.status !== 'revoked';
}

function canManageResourceGrants(app: OpenPlatformAppWithScopes) {
  return app.app.status !== 'pending';
}

function hasAvailableAction(app: OpenPlatformAppWithScopes) {
  return (
    pendingScopes(app).length > 0 ||
    pendingRedirectURIRequests(app).length > 0 ||
    canApproveApp(app) ||
    canManageResourceGrants(app) ||
    canRotateSecret(app) ||
    canSuspendApp(app) ||
    canResumeApp(app) ||
    canRevokeApp(app)
  );
}
</script>

<template>
  <div class="open-platform-apps-page">
    <AdminContentLayout
      :title="$t('admin.routes.openPlatform.apps')"
      :total="total"
    >
      <template #actions>
        <ElButton type="primary" @click="importDialogVisible = true">
          {{ $t('admin.openPlatform.apps.importCasdoor') }}
        </ElButton>
      </template>

      <template #toolbar>
        <ElSelect
          v-model="query.status"
          class="admin-toolbar-control"
          :placeholder="$t('admin.common.status')"
          :teleported="false"
          @change="resetPageAndFetch"
        >
          <ElOption :label="$t('admin.common.all')" value="all" />
          <ElOption
            :label="$t('admin.openPlatform.apps.status.pending')"
            value="pending"
          />
          <ElOption
            :label="$t('admin.openPlatform.apps.status.approved')"
            value="approved"
          />
          <ElOption
            :label="$t('admin.openPlatform.apps.status.suspended')"
            value="suspended"
          />
          <ElOption
            :label="$t('admin.openPlatform.apps.status.revoked')"
            value="revoked"
          />
        </ElSelect>
        <ElButton type="primary" @click="resetPageAndFetch">
          {{ $t('admin.common.query') }}
        </ElButton>
        <ElTag v-if="pendingApps > 0" type="warning">
          {{
            $t('admin.openPlatform.apps.pendingCount', { count: pendingApps })
          }}
        </ElTag>
      </template>

      <ElAlert
        v-if="issuedSecret"
        class="open-platform-secret-alert"
        show-icon
        type="success"
        :closable="true"
        @close="issuedSecret = null"
      >
        <template #title>
          {{
            $t('admin.openPlatform.apps.secretIssued', {
              app: issuedSecret.appName,
            })
          }}
        </template>
        <div class="open-platform-secret-alert__body">
          <p class="open-platform-secret-alert__notice">
            {{ $t('admin.openPlatform.apps.secretOneTimeNotice') }}
          </p>
          <span>{{ issuedSecret.clientID }}</span>
          <div class="open-platform-secret-alert__secret">
            <code>{{ issuedSecret.secret }}</code>
            <ElButton
              plain
              size="small"
              type="primary"
              @click="copyIssuedSecret"
            >
              <IconifyIcon icon="lucide:copy" />
              {{ $t('admin.openPlatform.apps.copySecret') }}
            </ElButton>
          </div>
        </div>
      </ElAlert>

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
        table-key="open-platform.apps"
        :loading="loading"
        :data="apps"
        row-key="app.id"
        stripe
      >
        <PersistentAdminTableColumn
          column-key="id"
          :label="$t('admin.common.id')"
          :default-width="104"
        >
          <template #default="{ row }">
            <span class="admin-id-token" :title="String(row.app.id)">
              {{ compactID(String(row.app.id)) }}
            </span>
          </template>
        </PersistentAdminTableColumn>

        <PersistentAdminTableColumn
          column-key="app"
          :label="$t('admin.openPlatform.apps.app')"
          :default-min-width="240"
          show-overflow-tooltip
        >
          <template #default="{ row }">
            <div class="admin-cell-title" :title="row.app.displayName">
              {{ row.app.displayName }}
            </div>
            <div class="admin-cell-muted" :title="row.app.clientID">
              {{ row.app.clientID }}
            </div>
          </template>
        </PersistentAdminTableColumn>

        <PersistentAdminTableColumn
          column-key="links"
          :label="$t('admin.openPlatform.apps.links')"
          :default-width="136"
        >
          <template #default="{ row }">
            <div class="open-platform-link-group">
              <ElLink
                :href="row.app.homepageURL"
                target="_blank"
                type="primary"
              >
                {{ $t('admin.openPlatform.apps.homepage') }}
              </ElLink>
              <ElLink
                :href="row.app.privacyPolicyURL"
                target="_blank"
                type="primary"
              >
                {{ $t('admin.openPlatform.apps.privacy') }}
              </ElLink>
            </div>
          </template>
        </PersistentAdminTableColumn>

        <PersistentAdminTableColumn
          column-key="redirects"
          :label="$t('admin.openPlatform.apps.redirects')"
          :default-min-width="220"
          show-overflow-tooltip
        >
          <template #default="{ row }">
            <div
              v-for="redirect in row.app.redirectURIs"
              :key="redirect"
              class="admin-sub-id"
              :title="redirect"
            >
              {{ redirect }}
            </div>
            <div
              v-for="request in pendingRedirectURIRequests(row)"
              :key="request.id"
              class="open-platform-redirect-request"
            >
              <div class="open-platform-redirect-request__title">
                <ElTag size="small" type="warning">
                  {{ $t('admin.openPlatform.apps.redirectPending') }}
                </ElTag>
                <span>{{ formatAdminDateTime(request.updatedAt) }}</span>
              </div>
              <div
                v-for="redirect in request.redirectURIs"
                :key="`${request.id}-${redirect}`"
                class="admin-sub-id"
                :title="redirect"
              >
                {{ redirect }}
              </div>
              <div class="admin-cell-muted">
                {{ formatNullableText(request.reason) }}
              </div>
            </div>
          </template>
        </PersistentAdminTableColumn>

        <PersistentAdminTableColumn
          column-key="scopes"
          :label="$t('admin.openPlatform.apps.scopes')"
          :default-min-width="300"
        >
          <template #default="{ row }">
            <div class="open-platform-scope-list">
              <div
                v-for="scope in row.scopes"
                :key="scope.scope"
                class="open-platform-scope"
              >
                <div class="open-platform-scope__title">
                  <span>{{ scope.displayName }}</span>
                  <ElTag :type="scopeStatusTag(scope.status)" size="small">
                    {{ scopeStatusLabel(scope.status) }}
                  </ElTag>
                </div>
                <div class="admin-cell-muted">
                  {{ scope.scope }}
                </div>
                <div class="admin-cell-muted">
                  {{ formatNullableText(scope.reason) }}
                </div>
              </div>
            </div>
          </template>
        </PersistentAdminTableColumn>

        <PersistentAdminTableColumn
          column-key="status"
          :label="$t('admin.common.status')"
          :default-width="112"
        >
          <template #default="{ row }">
            <ElTag :type="appStatusTag(row.app.status)" size="small">
              {{ statusLabel(row.app.status) }}
            </ElTag>
          </template>
        </PersistentAdminTableColumn>

        <PersistentAdminTableColumn
          column-key="createdAt"
          :label="$t('admin.common.createdAt')"
          :default-width="148"
        >
          <template #default="{ row }">
            <span class="admin-cell-muted">
              {{ formatAdminDateTime(row.app.createdAt) }}
            </span>
          </template>
        </PersistentAdminTableColumn>

        <PersistentAdminTableColumn
          column-key="actions"
          fixed="right"
          :label="$t('admin.common.actions')"
          :default-width="460"
        >
          <template #default="{ row }">
            <div class="admin-action-group">
              <ElPopconfirm
                v-for="scope in pendingScopes(row)"
                :key="scope.scope"
                :title="
                  $t('admin.openPlatform.apps.confirmApproveScope', {
                    scope: scope.displayName,
                  })
                "
                @confirm="handleApproveScope(row, scope.scope)"
              >
                <template #reference>
                  <ElButton
                    plain
                    size="small"
                    type="warning"
                    :disabled="actionLoading"
                  >
                    {{ $t('admin.openPlatform.apps.approveScope') }}
                  </ElButton>
                </template>
              </ElPopconfirm>
              <ElButton
                v-for="scope in pendingScopes(row)"
                :key="`${scope.scope}-reject`"
                plain
                size="small"
                type="danger"
                :disabled="actionLoading"
                @click="handleRejectScope(row, scope.scope)"
              >
                {{ $t('admin.openPlatform.apps.rejectScope') }}
              </ElButton>
              <template
                v-for="request in pendingRedirectURIRequests(row)"
                :key="request.id"
              >
                <ElPopconfirm
                  :title="$t('admin.openPlatform.apps.confirmApproveRedirect')"
                  @confirm="handleApproveRedirectURIRequest(row, request.id)"
                >
                  <template #reference>
                    <ElButton
                      plain
                      size="small"
                      type="warning"
                      :disabled="actionLoading"
                    >
                      {{ $t('admin.openPlatform.apps.approveRedirect') }}
                    </ElButton>
                  </template>
                </ElPopconfirm>
                <ElButton
                  plain
                  size="small"
                  type="danger"
                  :disabled="actionLoading"
                  @click="handleRejectRedirectURIRequest(row, request.id)"
                >
                  {{ $t('admin.openPlatform.apps.rejectRedirect') }}
                </ElButton>
              </template>
              <ElPopconfirm
                v-if="canApproveApp(row)"
                :title="$t('admin.openPlatform.apps.confirmApproveApp')"
                @confirm="handleApproveApp(row)"
              >
                <template #reference>
                  <ElButton
                    plain
                    size="small"
                    type="success"
                    :disabled="actionLoading"
                  >
                    {{ $t('admin.openPlatform.apps.approveApp') }}
                  </ElButton>
                </template>
              </ElPopconfirm>
              <span v-if="!hasAvailableAction(row)" class="admin-cell-muted">
                {{ $t('admin.common.unavailable') }}
              </span>
              <ElButton
                v-if="canManageResourceGrants(row)"
                plain
                size="small"
                type="primary"
                :disabled="actionLoading"
                @click="handleOpenResourceGrants(row)"
              >
                {{ $t('admin.openPlatform.apps.resourceGrants') }}
              </ElButton>
              <ElButton
                v-if="canRotateSecret(row)"
                plain
                size="small"
                type="primary"
                :disabled="actionLoading"
                @click="handleRotateSecret(row)"
              >
                {{ $t('admin.openPlatform.apps.rotateSecret') }}
              </ElButton>
              <ElButton
                v-if="canSuspendApp(row)"
                plain
                size="small"
                type="info"
                :disabled="actionLoading"
                @click="handleSuspendApp(row)"
              >
                {{ $t('admin.openPlatform.apps.suspendApp') }}
              </ElButton>
              <ElButton
                v-if="canResumeApp(row)"
                plain
                size="small"
                type="success"
                :disabled="actionLoading"
                @click="handleResumeApp(row)"
              >
                {{ $t('admin.openPlatform.apps.resumeApp') }}
              </ElButton>
              <ElButton
                v-if="canRevokeApp(row)"
                plain
                size="small"
                type="danger"
                :disabled="actionLoading"
                @click="handleRevokeApp(row)"
              >
                {{ $t('admin.openPlatform.apps.revokeApp') }}
              </ElButton>
            </div>
          </template>
        </PersistentAdminTableColumn>
      </PersistentAdminTable>

      <template #pagination>
        <ElPagination
          v-model:current-page="query.page"
          v-model:page-size="query.pageSize"
          :total="total"
          layout="total, prev, pager, next"
          @current-change="fetchData"
        />
      </template>
    </AdminContentLayout>

    <ImportCasdoorAppDialog
      v-model:visible="importDialogVisible"
      :submitting="importLoading"
      @submit="handleImportCasdoorApp"
    />
    <ResourceGrantsDialog
      v-model:visible="resourceGrantsDialogVisible"
      :app="resourceGrantsApp"
    />
  </div>
</template>

<style scoped>
.open-platform-apps-page {
  min-width: 0;
}

.open-platform-secret-alert {
  margin: 12px 18px 0;
}

.open-platform-secret-alert__body {
  display: grid;
  gap: 6px;
  margin-top: 6px;
  font-size: 12px;
}

.open-platform-secret-alert__notice {
  margin: 0;
  color: var(--el-text-color-secondary);
}

.open-platform-secret-alert__secret {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: flex-start;
  min-width: 0;
}

.open-platform-secret-alert__secret code {
  flex: 1 1 260px;
  min-width: 0;
  padding: 6px 8px;
  color: var(--el-color-success-dark-2);
  overflow-wrap: anywhere;
  background: var(--el-fill-color-light);
  border-radius: 6px;
}

.open-platform-secret-alert__secret .el-button {
  display: inline-flex;
  gap: 4px;
  align-items: center;
}

.open-platform-link-group,
.open-platform-scope-list {
  display: grid;
  gap: 6px;
}

.open-platform-redirect-request {
  display: grid;
  gap: 6px;
  padding: 8px 0;
  margin-top: 8px;
  border-top: 1px solid var(--el-border-color-lighter);
}

.open-platform-redirect-request__title {
  display: flex;
  gap: 8px;
  align-items: center;
  justify-content: space-between;
  min-width: 0;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.open-platform-scope {
  display: grid;
  gap: 4px;
  padding: 8px 0;
}

.open-platform-scope + .open-platform-scope {
  border-top: 1px solid var(--el-border-color-lighter);
}

.open-platform-scope__title {
  display: flex;
  gap: 8px;
  align-items: center;
  justify-content: space-between;
  min-width: 0;
  font-weight: 600;
}

.open-platform-scope__title span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
