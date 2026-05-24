<script setup lang="ts">
import type { OpenPlatformAuditEvent, OpenPlatformScope } from '#/api/admin';

import { onMounted, reactive, ref } from 'vue';

import {
  ElButton,
  ElInputNumber,
  ElOption,
  ElPagination,
  ElSelect,
  ElTag,
} from 'element-plus';

import { getOpenPlatformAuditEventList } from '#/api/admin';
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

const loading = ref(false);
const events = ref<OpenPlatformAuditEvent[]>([]);
const total = ref(0);
const query = reactive<{
  appID?: number;
  eventType: string;
  page: number;
  pageSize: number;
  scope: '' | OpenPlatformScope;
  userID?: number;
}>({
  eventType: '',
  page: 1,
  pageSize: 20,
  scope: '',
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

async function fetchData() {
  loading.value = true;
  try {
    const data = await getOpenPlatformAuditEventList({
      appID: normalizeOptionalID(query.appID),
      eventType: query.eventType.trim() || undefined,
      page: query.page,
      pageSize: query.pageSize,
      scope: query.scope || undefined,
      userID: normalizeOptionalID(query.userID),
    });
    events.value = data.items;
    total.value = data.total;
  } finally {
    loading.value = false;
  }
}

function handleQuery() {
  query.page = 1;
  void fetchData();
}

function normalizeOptionalID(value?: number) {
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

onMounted(fetchData);
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
      <ElButton type="primary" @click="handleQuery">
        {{ $t('admin.common.query') }}
      </ElButton>
    </template>

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
