<script setup lang="ts">
import type {
  OpenPlatformAdminUserAuthorizedApp,
  OpenPlatformScope,
} from '#/api/admin';

import {
  ElAlert,
  ElButton,
  ElInputNumber,
  ElMessage,
  ElMessageBox,
  ElPagination,
  ElTag,
} from 'element-plus';

import {
  getOpenPlatformConsentList,
  revokeOpenPlatformConsent,
} from '#/api/admin';
import { useAdminAction } from '#/composables/use-admin-action';
import { useAdminList } from '#/composables/use-admin-list';
import { $t } from '#/locales';

import PersistentAdminTable from '../../shared/admin-table/PersistentAdminTable.vue';
import PersistentAdminTableColumn from '../../shared/admin-table/PersistentAdminTableColumn.vue';
import AdminContentLayout from '../../shared/AdminContentLayout.vue';
import {
  compactID,
  formatAdminDateTime,
  formatNullableText,
} from '../../shared/display';

const {
  fetchData,
  items: consents,
  loadError,
  loading,
  query,
  resetPageAndFetch,
  total,
} = useAdminList<
  OpenPlatformAdminUserAuthorizedApp,
  {
    appID: null | number;
    page: number;
    pageSize: number;
    userID: null | number;
  }
>({
  // 查询必须带 appID 或 userID 之一，挂载时不自动拉取
  fetchOnMount: false,
  fetcher: async (listQuery) => {
    const appID = normalizeOptionalID(listQuery.appID);
    const userID = normalizeOptionalID(listQuery.userID);
    if (!appID && !userID) {
      ElMessage.warning($t('admin.openPlatform.consents.queryRequired'));
      return { items: [], total: 0 };
    }
    return getOpenPlatformConsentList({
      appID,
      page: listQuery.page,
      pageSize: listQuery.pageSize,
      userID,
    });
  },
  initialQuery: {
    appID: null,
    page: 1,
    pageSize: 20,
    userID: null,
  },
});

const { actionError, actionPending, clearActionError, runAction } =
  useAdminAction();

function normalizeOptionalID(value?: null | number) {
  if (typeof value !== 'number' || !Number.isFinite(value) || value <= 0) {
    return undefined;
  }
  return Math.trunc(value);
}

function consentRowKey(row: OpenPlatformAdminUserAuthorizedApp) {
  return `${row.userID}:${row.app.id}`;
}

async function promptRevokeReason(message: string) {
  try {
    const result = await ElMessageBox.prompt(
      message,
      $t('admin.openPlatform.consents.reasonTitle'),
      {
        cancelButtonText: $t('admin.common.cancel'),
        confirmButtonText: $t('admin.common.confirm'),
        inputPlaceholder: $t('admin.openPlatform.consents.reasonPlaceholder'),
        inputValidator: (value) =>
          value.trim().length > 0 ||
          $t('admin.openPlatform.consents.reasonRequired'),
      },
    );
    return result.value.trim();
  } catch (_error) {
    void _error;
    return null;
  }
}

async function handleRevokeConsent(
  item: OpenPlatformAdminUserAuthorizedApp,
  scope?: OpenPlatformScope,
) {
  if (actionPending.value) {
    return;
  }
  const reason = await promptRevokeReason(
    scope
      ? $t('admin.openPlatform.consents.revokeScopePrompt', { scope })
      : $t('admin.openPlatform.consents.revokeAllPrompt', {
          app: item.app.displayName,
          user: item.userID,
        }),
  );
  if (reason === null) {
    return;
  }
  const succeeded = await runAction(
    () =>
      revokeOpenPlatformConsent({
        appID: item.app.id,
        reason,
        scopes: scope ? [scope] : undefined,
        userID: item.userID,
      }),
    { successMessage: $t('admin.openPlatform.consents.revoked') },
  );
  if (succeeded) {
    await fetchData();
  }
}
</script>

<template>
  <AdminContentLayout
    :title="$t('admin.routes.openPlatform.consents')"
    :total="total"
  >
    <template #toolbar>
      <ElInputNumber
        v-model="query.appID"
        class="admin-toolbar-control"
        :controls="false"
        :min="1"
        :placeholder="$t('admin.openPlatform.consents.appIdPlaceholder')"
      />
      <ElInputNumber
        v-model="query.userID"
        class="admin-toolbar-control"
        :controls="false"
        :min="1"
        :placeholder="$t('admin.openPlatform.consents.userIdPlaceholder')"
      />
      <ElButton type="primary" @click="resetPageAndFetch">
        {{ $t('admin.common.query') }}
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

    <PersistentAdminTable
      v-loading="loading"
      table-key="open-platform.consents"
      :data="consents"
      :row-key="consentRowKey"
    >
      <PersistentAdminTableColumn
        column-key="userID"
        :label="$t('admin.openPlatform.consents.userId')"
        min-width="120"
      >
        <template #default="{ row }">
          <span :title="String(row.userID)">
            {{ compactID(String(row.userID)) }}
          </span>
        </template>
      </PersistentAdminTableColumn>
      <PersistentAdminTableColumn
        column-key="app"
        :label="$t('admin.openPlatform.consents.app')"
        min-width="220"
      >
        <template #default="{ row }">
          <div>{{ row.app.displayName }}</div>
          <div class="admin-cell-muted" :title="row.app.clientID">
            {{ row.app.clientID }}
          </div>
        </template>
      </PersistentAdminTableColumn>
      <PersistentAdminTableColumn
        column-key="appID"
        :label="$t('admin.openPlatform.consents.appId')"
        min-width="110"
      >
        <template #default="{ row }">
          <span :title="String(row.app.id)">
            {{ compactID(String(row.app.id)) }}
          </span>
        </template>
      </PersistentAdminTableColumn>
      <PersistentAdminTableColumn
        column-key="scopes"
        :label="$t('admin.openPlatform.consents.scopes')"
        min-width="280"
      >
        <template #default="{ row }">
          <div class="open-platform-consent-scopes">
            <ElTag
              v-for="scope in row.scopes"
              :key="scope.scope"
              size="small"
              type="info"
            >
              {{ scope.scope }}
            </ElTag>
          </div>
        </template>
      </PersistentAdminTableColumn>
      <PersistentAdminTableColumn
        column-key="grantedAt"
        :label="$t('admin.openPlatform.consents.grantedAt')"
        min-width="170"
      >
        <template #default="{ row }">
          {{ formatAdminDateTime(row.scopes[0]?.grantedAt) }}
        </template>
      </PersistentAdminTableColumn>
      <PersistentAdminTableColumn
        column-key="lastUsedAt"
        :label="$t('admin.openPlatform.consents.lastUsedAt')"
        min-width="170"
      >
        <template #default="{ row }">
          {{
            formatNullableText(formatAdminDateTime(row.scopes[0]?.lastUsedAt))
          }}
        </template>
      </PersistentAdminTableColumn>
      <PersistentAdminTableColumn
        column-key="actions"
        fixed="right"
        :label="$t('admin.common.actions')"
        min-width="260"
      >
        <template #default="{ row }">
          <div class="admin-table-actions">
            <ElButton
              plain
              size="small"
              type="danger"
              :disabled="actionPending"
              @click="handleRevokeConsent(row)"
            >
              {{ $t('admin.openPlatform.consents.revokeAll') }}
            </ElButton>
            <ElButton
              v-for="scope in row.scopes"
              :key="scope.scope"
              plain
              size="small"
              type="warning"
              :disabled="actionPending"
              @click="handleRevokeConsent(row, scope.scope)"
            >
              {{ $t('admin.openPlatform.consents.revokeScope') }}
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
</template>

<style scoped>
.open-platform-consent-scopes {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}
</style>
