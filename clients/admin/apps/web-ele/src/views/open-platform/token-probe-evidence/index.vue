<script setup lang="ts">
import type {
  OpenPlatformTokenProbeEvidence,
  OpenPlatformTokenProbeResult,
} from '#/api/admin';

import {
  ElAlert,
  ElButton,
  ElInput,
  ElInputNumber,
  ElOption,
  ElPagination,
  ElSelect,
  ElTag,
} from 'element-plus';

import { getOpenPlatformTokenProbeEvidenceList } from '#/api/admin';
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
  items: records,
  loadError,
  loading,
  query,
  resetPageAndFetch,
  total,
} = useAdminList<
  OpenPlatformTokenProbeEvidence,
  {
    appID: null | number;
    clientID: string;
    page: number;
    pageSize: number;
    result: '' | OpenPlatformTokenProbeResult;
    reviewerUserID: null | number;
  }
>({
  fetcher: (listQuery) =>
    getOpenPlatformTokenProbeEvidenceList({
      appID: normalizeOptionalID(listQuery.appID),
      clientID: listQuery.clientID.trim() || undefined,
      page: listQuery.page,
      pageSize: listQuery.pageSize,
      result: listQuery.result || undefined,
      reviewerUserID: normalizeOptionalID(listQuery.reviewerUserID),
    }),
  initialQuery: {
    appID: null,
    clientID: '',
    page: 1,
    pageSize: 20,
    result: '',
    reviewerUserID: null,
  },
});

const resultOptions: OpenPlatformTokenProbeResult[] = ['passed', 'failed'];

function normalizeOptionalID(value?: null | number) {
  if (typeof value !== 'number' || !Number.isFinite(value) || value <= 0) {
    return undefined;
  }
  return Math.trunc(value);
}

function displayNumber(value?: null | number) {
  return typeof value === 'number' ? String(value) : '—';
}

function displayCompactNumber(value?: null | number) {
  return typeof value === 'number' ? compactID(String(value)) : '—';
}

function resultLabel(result: OpenPlatformTokenProbeResult) {
  return $t(`admin.openPlatform.tokenProbe.resultStatus.${result}`);
}

function resultTagType(result: OpenPlatformTokenProbeResult) {
  return result === 'passed' ? 'success' : 'danger';
}

function joinClaims(claims: string[]) {
  return claims.length > 0 ? claims.join(', ') : '—';
}

function formatTokenClaims(tokenClaims: Record<string, string[]>) {
  return JSON.stringify(tokenClaims, null, 2);
}

function hasMetadata(metadata: Record<string, unknown>) {
  return Object.keys(metadata).length > 0;
}

function formatMetadata(metadata: Record<string, unknown>) {
  return JSON.stringify(metadata, null, 2);
}
</script>

<template>
  <AdminContentLayout
    :title="$t('admin.routes.openPlatform.tokenProbeEvidence')"
    :total="total"
  >
    <template #toolbar>
      <ElInputNumber
        v-model="query.appID"
        class="admin-toolbar-control"
        :controls="false"
        :min="1"
        :placeholder="$t('admin.openPlatform.tokenProbe.appIdPlaceholder')"
      />
      <ElInputNumber
        v-model="query.reviewerUserID"
        class="admin-toolbar-control"
        :controls="false"
        :min="1"
        :placeholder="
          $t('admin.openPlatform.tokenProbe.reviewerUserIdPlaceholder')
        "
      />
      <ElSelect
        v-model="query.result"
        class="admin-toolbar-control"
        clearable
        :placeholder="$t('admin.openPlatform.tokenProbe.resultPlaceholder')"
        :teleported="false"
      >
        <ElOption
          v-for="result in resultOptions"
          :key="result"
          :label="resultLabel(result)"
          :value="result"
        />
      </ElSelect>
      <ElInput
        v-model="query.clientID"
        class="admin-toolbar-control--wide"
        clearable
        :placeholder="$t('admin.openPlatform.tokenProbe.clientIdPlaceholder')"
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

    <PersistentAdminTable
      table-key="open-platform.token-probe-evidence"
      :loading="loading"
      :data="records"
      row-key="id"
      stripe
    >
      <PersistentAdminTableColumn
        column-key="id"
        :label="$t('admin.common.id')"
        :default-width="104"
      >
        <template #default="{ row }">
          <span class="admin-id-token" :title="String(row.id)">
            {{ compactID(String(row.id)) }}
          </span>
        </template>
      </PersistentAdminTableColumn>

      <PersistentAdminTableColumn
        column-key="createdAt"
        :label="$t('admin.common.time')"
        :default-width="156"
      >
        <template #default="{ row }">
          <span class="admin-cell-muted">
            {{ formatAdminDateTime(row.createdAt) }}
          </span>
        </template>
      </PersistentAdminTableColumn>

      <PersistentAdminTableColumn
        column-key="result"
        :label="$t('admin.openPlatform.tokenProbe.result')"
        :default-width="112"
      >
        <template #default="{ row }">
          <ElTag :type="resultTagType(row.result)" size="small">
            {{ resultLabel(row.result) }}
          </ElTag>
        </template>
      </PersistentAdminTableColumn>

      <PersistentAdminTableColumn
        column-key="appID"
        :label="$t('admin.openPlatform.audit.appId')"
        :default-width="112"
      >
        <template #default="{ row }">
          <span class="admin-sub-id" :title="displayNumber(row.appID)">
            {{ displayCompactNumber(row.appID) }}
          </span>
        </template>
      </PersistentAdminTableColumn>

      <PersistentAdminTableColumn
        column-key="reviewerUserID"
        :label="$t('admin.openPlatform.tokenProbe.reviewerUserId')"
        :default-width="128"
      >
        <template #default="{ row }">
          <span class="admin-sub-id" :title="displayNumber(row.reviewerUserID)">
            {{ displayCompactNumber(row.reviewerUserID) }}
          </span>
        </template>
      </PersistentAdminTableColumn>

      <PersistentAdminTableColumn
        column-key="clientID"
        :label="$t('admin.openPlatform.tokenProbe.clientId')"
        :default-min-width="180"
        show-overflow-tooltip
      >
        <template #default="{ row }">
          <span class="admin-sub-id" :title="row.clientID">
            {{ row.clientID }}
          </span>
        </template>
      </PersistentAdminTableColumn>

      <PersistentAdminTableColumn
        column-key="redirectURI"
        :label="$t('admin.openPlatform.tokenProbe.redirectUri')"
        :default-min-width="240"
        show-overflow-tooltip
      >
        <template #default="{ row }">
          <span class="admin-cell-muted" :title="row.redirectURI">
            {{ row.redirectURI }}
          </span>
        </template>
      </PersistentAdminTableColumn>

      <PersistentAdminTableColumn
        column-key="probeMethod"
        :label="$t('admin.openPlatform.tokenProbe.probeMethod')"
        :default-width="148"
        show-overflow-tooltip
      >
        <template #default="{ row }">
          <span class="admin-cell-muted">{{ row.probeMethod }}</span>
        </template>
      </PersistentAdminTableColumn>

      <PersistentAdminTableColumn
        column-key="inspectedClaims"
        :label="$t('admin.openPlatform.tokenProbe.inspectedClaims')"
        :default-min-width="220"
        show-overflow-tooltip
      >
        <template #default="{ row }">
          <span
            class="admin-cell-muted"
            :title="joinClaims(row.inspectedClaims)"
          >
            {{ joinClaims(row.inspectedClaims) }}
          </span>
        </template>
      </PersistentAdminTableColumn>

      <PersistentAdminTableColumn
        column-key="businessClaims"
        :label="$t('admin.openPlatform.tokenProbe.businessClaims')"
        :default-min-width="220"
        show-overflow-tooltip
      >
        <template #default="{ row }">
          <span
            :class="[
              row.businessClaims.length > 0
                ? 'open-platform-token-probe-danger'
                : 'admin-cell-muted',
            ]"
            :title="joinClaims(row.businessClaims)"
          >
            {{ joinClaims(row.businessClaims) }}
          </span>
        </template>
      </PersistentAdminTableColumn>

      <PersistentAdminTableColumn
        column-key="error"
        :label="$t('admin.openPlatform.tokenProbe.error')"
        :default-min-width="240"
        show-overflow-tooltip
      >
        <template #default="{ row }">
          <span class="admin-cell-muted" :title="formatNullableText(row.error)">
            {{ formatNullableText(row.error) }}
          </span>
        </template>
      </PersistentAdminTableColumn>

      <PersistentAdminTableColumn
        column-key="tokenClaims"
        :label="$t('admin.openPlatform.tokenProbe.tokenClaims')"
        :default-min-width="260"
      >
        <template #default="{ row }">
          <pre
            class="open-platform-token-probe-json"
            v-text="formatTokenClaims(row.tokenClaims)"
          ></pre>
        </template>
      </PersistentAdminTableColumn>

      <PersistentAdminTableColumn
        column-key="requestID"
        :label="$t('admin.openPlatform.audit.requestId')"
        :default-min-width="180"
        show-overflow-tooltip
      >
        <template #default="{ row }">
          <span class="admin-sub-id" :title="formatNullableText(row.requestID)">
            {{ formatNullableText(row.requestID) }}
          </span>
        </template>
      </PersistentAdminTableColumn>

      <PersistentAdminTableColumn
        column-key="metadata"
        :label="$t('admin.openPlatform.audit.metadata')"
        :default-min-width="300"
      >
        <template #default="{ row }">
          <pre
            v-if="hasMetadata(row.metadata)"
            class="open-platform-token-probe-json"
            v-text="formatMetadata(row.metadata)"
          ></pre>
          <span v-else class="admin-cell-muted">
            {{ $t('admin.common.unavailable') }}
          </span>
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
.open-platform-token-probe-danger {
  font-weight: 600;
  color: var(--el-color-danger);
}

.open-platform-token-probe-json {
  max-height: 128px;
  padding: 8px;
  margin: 0;
  overflow: auto;
  font-family: var(--el-font-family);
  font-size: 12px;
  line-height: 1.45;
  color: var(--el-text-color-regular);
  overflow-wrap: anywhere;
  white-space: pre-wrap;
  background: var(--el-fill-color-light);
  border-radius: 6px;
}
</style>
