<script setup lang="ts">
import type { OpenPlatformAuditEvent, OpenPlatformScope } from '#/api/admin';

import {
  ElAlert,
  ElButton,
  ElInputNumber,
  ElOption,
  ElPagination,
  ElSelect,
  ElTag,
} from 'element-plus';

import { getOpenPlatformAuditEventList } from '#/api/admin';
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
import {
  eventTagType,
  knownOpenPlatformAuditEventTypes,
  openPlatformAuditEventTypeLabelKeys,
} from './auditEvents';

const {
  fetchData,
  items: events,
  loadError,
  loading,
  query,
  resetPageAndFetch,
  total,
} = useAdminList<
  OpenPlatformAuditEvent,
  {
    appID: null | number;
    eventType: string;
    page: number;
    pageSize: number;
    scope: '' | OpenPlatformScope;
    userID: null | number;
  }
>({
  fetcher: (listQuery) =>
    getOpenPlatformAuditEventList({
      appID: normalizeOptionalID(listQuery.appID),
      eventType: listQuery.eventType.trim() || undefined,
      page: listQuery.page,
      pageSize: listQuery.pageSize,
      scope: listQuery.scope || undefined,
      userID: normalizeOptionalID(listQuery.userID),
    }),
  initialQuery: {
    appID: null,
    eventType: '',
    page: 1,
    pageSize: 20,
    scope: '',
    userID: null,
  },
});

const scopeOptions: OpenPlatformScope[] = [
  'profile.basic.read',
  'email.read',
  'phone.read',
  'stu.identity.status.read',
  'stu.identity.type.read',
  'stu.student.status.read',
  'stu.student.school.read',
  'resource.read',
  'resource.write',
  'offline_access',
];

const knownEventTypes = knownOpenPlatformAuditEventTypes;

function normalizeOptionalID(value?: null | number) {
  if (typeof value !== 'number' || !Number.isFinite(value) || value <= 0) {
    return undefined;
  }
  return Math.trunc(value);
}

function eventTypeLabel(eventType: string) {
  const key =
    openPlatformAuditEventTypeLabelKeys[
      eventType as keyof typeof openPlatformAuditEventTypeLabelKeys
    ];
  return key ? $t(key) : eventType;
}

function displayNumber(value?: null | number) {
  return typeof value === 'number' ? String(value) : '—';
}

function displayCompactNumber(value?: null | number) {
  return typeof value === 'number' ? compactID(String(value)) : '—';
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
    :title="$t('admin.routes.openPlatform.auditEvents')"
    :total="total"
  >
    <template #toolbar>
      <ElInputNumber
        v-model="query.appID"
        class="admin-toolbar-control"
        :controls="false"
        :min="1"
        :placeholder="$t('admin.openPlatform.audit.appIdPlaceholder')"
      />
      <ElInputNumber
        v-model="query.userID"
        class="admin-toolbar-control"
        :controls="false"
        :min="1"
        :placeholder="$t('admin.openPlatform.audit.userIdPlaceholder')"
      />
      <ElSelect
        v-model="query.eventType"
        allow-create
        class="admin-toolbar-control--wide"
        clearable
        default-first-option
        filterable
        :placeholder="$t('admin.openPlatform.audit.eventTypePlaceholder')"
        :teleported="false"
      >
        <ElOption
          v-for="eventType in knownEventTypes"
          :key="eventType"
          :label="eventTypeLabel(eventType)"
          :value="eventType"
        />
      </ElSelect>
      <ElSelect
        v-model="query.scope"
        class="admin-toolbar-control--wide"
        clearable
        filterable
        :placeholder="$t('admin.openPlatform.audit.scopePlaceholder')"
        :teleported="false"
      >
        <ElOption
          v-for="scope in scopeOptions"
          :key="scope"
          :label="scope"
          :value="scope"
        />
      </ElSelect>
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
      table-key="open-platform.audit-events"
      :loading="loading"
      :data="events"
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
        column-key="eventType"
        :label="$t('admin.openPlatform.audit.eventType')"
        :default-min-width="260"
      >
        <template #default="{ row }">
          <div class="open-platform-audit-event">
            <ElTag :type="eventTagType(row.eventType)" size="small">
              {{ eventTypeLabel(row.eventType) }}
            </ElTag>
            <div class="admin-cell-muted" :title="row.eventType">
              {{ row.eventType }}
            </div>
          </div>
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
        column-key="userID"
        :label="$t('admin.openPlatform.audit.userId')"
        :default-width="112"
      >
        <template #default="{ row }">
          <span class="admin-sub-id" :title="displayNumber(row.userID)">
            {{ displayCompactNumber(row.userID) }}
          </span>
        </template>
      </PersistentAdminTableColumn>

      <PersistentAdminTableColumn
        column-key="scope"
        :label="$t('admin.openPlatform.audit.scope')"
        :default-width="188"
        show-overflow-tooltip
      >
        <template #default="{ row }">
          <span class="admin-cell-muted">
            {{ formatNullableText(row.scope) }}
          </span>
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
        :default-min-width="320"
      >
        <template #default="{ row }">
          <pre
            v-if="hasMetadata(row.metadata)"
            class="open-platform-audit-metadata"
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
.open-platform-audit-event {
  display: grid;
  gap: 6px;
  min-width: 0;
}

.open-platform-audit-event .admin-cell-muted {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.open-platform-audit-metadata {
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
