<script setup lang="ts">
import type { OpenPlatformDisclosureReport } from '#/api/admin';

import { computed, reactive } from 'vue';

import { ElAlert, ElButton, ElInputNumber, ElTag } from 'element-plus';

import { getOpenPlatformDisclosureReport } from '#/api/admin';
import { useAdminLoad } from '#/composables/use-admin-list';
import { $t } from '#/locales';

import PersistentAdminTable from '../../shared/admin-table/PersistentAdminTable.vue';
import PersistentAdminTableColumn from '../../shared/admin-table/PersistentAdminTableColumn.vue';
import AdminContentLayout from '../../shared/AdminContentLayout.vue';
import {
  compactID,
  formatAdminDateTime,
  formatNullableText,
} from '../../shared/display';

type ReportSummary = OpenPlatformDisclosureReport['summary'];

const emptySummary: ReportSummary = {
  denied: 0,
  granted: 0,
  rateLimited: 0,
  replayDetected: 0,
  total: 0,
  windowHours: 24,
};

const query = reactive({
  windowHours: 24,
});

const {
  data: report,
  load: fetchReport,
  loadError,
  loading,
} = useAdminLoad(() =>
  getOpenPlatformDisclosureReport({ windowHours: query.windowHours }),
);

const summary = computed(() => report.value?.summary ?? emptySummary);
const endpoints = computed(() => report.value?.endpoints ?? []);
const denialReasons = computed(() => report.value?.denialReasons ?? []);
const rateLimitDimensions = computed(
  () => report.value?.rateLimitDimensions ?? [],
);
const replayEvents = computed(() => report.value?.recentReplayEvents ?? []);

const summaryItems = computed(() => [
  {
    key: 'total',
    label: $t('admin.openPlatform.disclosure.summary.total'),
    value: summary.value.total,
  },
  {
    key: 'granted',
    label: $t('admin.openPlatform.disclosure.summary.granted'),
    value: summary.value.granted,
  },
  {
    key: 'denied',
    label: $t('admin.openPlatform.disclosure.summary.denied'),
    value: summary.value.denied,
  },
  {
    key: 'rateLimited',
    label: $t('admin.openPlatform.disclosure.summary.rateLimited'),
    value: summary.value.rateLimited,
  },
  {
    key: 'replayDetected',
    label: $t('admin.openPlatform.disclosure.summary.replayDetected'),
    value: summary.value.replayDetected,
  },
]);

function formatCount(value?: number) {
  return new Intl.NumberFormat().format(value ?? 0);
}

function formatRate(value?: number, total?: number) {
  if (!value || !total) return '0%';
  return `${((value / total) * 100).toFixed(1)}%`;
}

type TagType = 'danger' | 'info' | 'success' | 'warning';

function resultTagType(result: string): TagType {
  if (result === 'ok') return 'success';
  if (result === 'rate_limited') return 'warning';
  if (result.includes('unavailable') || result.includes('error')) {
    return 'danger';
  }
  return 'info';
}

function displayNumber(value?: null | number) {
  return typeof value === 'number' ? String(value) : '—';
}

function displayCompactNumber(value?: null | number) {
  return typeof value === 'number' ? compactID(String(value)) : '—';
}

function formatScopes(scopes: string[]) {
  return scopes.length > 0 ? scopes.join(', ') : '—';
}

function formatMetadata(metadata: Record<string, unknown>) {
  return JSON.stringify(metadata, null, 2);
}
</script>

<template>
  <AdminContentLayout
    :title="$t('admin.routes.openPlatform.disclosureReport')"
    :total="summary.total"
  >
    <template #toolbar>
      <ElInputNumber
        v-model="query.windowHours"
        class="admin-toolbar-control"
        :controls="false"
        :max="168"
        :min="1"
        :placeholder="$t('admin.openPlatform.disclosure.windowHours')"
      />
      <ElButton type="primary" @click="fetchReport">
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
      <ElButton size="small" :loading="loading" @click="fetchReport">
        {{ $t('admin.common.retry') }}
      </ElButton>
    </ElAlert>

    <div v-loading="loading" class="open-platform-disclosure-report">
      <section class="open-platform-disclosure-report__summary">
        <div
          v-for="item in summaryItems"
          :key="item.key"
          class="open-platform-disclosure-report__metric"
        >
          <span>{{ item.label }}</span>
          <strong>{{ formatCount(item.value) }}</strong>
        </div>
      </section>

      <section class="open-platform-disclosure-report__section">
        <header class="open-platform-disclosure-report__section-header">
          {{ $t('admin.openPlatform.disclosure.endpoints') }}
        </header>
        <PersistentAdminTable
          table-key="open-platform.disclosure.endpoints"
          :data="endpoints"
          row-key="endpoint"
          stripe
        >
          <PersistentAdminTableColumn
            column-key="endpoint"
            :label="$t('admin.openPlatform.disclosure.endpoint')"
            :default-min-width="180"
          >
            <template #default="{ row }">
              <span class="admin-sub-id">{{ row.endpoint }}</span>
            </template>
          </PersistentAdminTableColumn>
          <PersistentAdminTableColumn
            column-key="total"
            :label="$t('admin.openPlatform.disclosure.total')"
            :default-width="120"
          >
            <template #default="{ row }">
              {{ formatCount(row.total) }}
            </template>
          </PersistentAdminTableColumn>
          <PersistentAdminTableColumn
            column-key="granted"
            :label="$t('admin.openPlatform.disclosure.granted')"
            :default-width="136"
          >
            <template #default="{ row }">
              {{ formatCount(row.granted) }}
              <span class="admin-cell-muted">
                {{ formatRate(row.granted, row.total) }}
              </span>
            </template>
          </PersistentAdminTableColumn>
          <PersistentAdminTableColumn
            column-key="denied"
            :label="$t('admin.openPlatform.disclosure.denied')"
            :default-width="136"
          >
            <template #default="{ row }">
              {{ formatCount(row.denied) }}
              <span class="admin-cell-muted">
                {{ formatRate(row.denied, row.total) }}
              </span>
            </template>
          </PersistentAdminTableColumn>
          <PersistentAdminTableColumn
            column-key="rateLimited"
            :label="$t('admin.openPlatform.disclosure.rateLimited')"
            :default-width="136"
          >
            <template #default="{ row }">
              {{ formatCount(row.rateLimited) }}
            </template>
          </PersistentAdminTableColumn>
          <PersistentAdminTableColumn
            column-key="replayDetected"
            :label="$t('admin.openPlatform.disclosure.replayDetected')"
            :default-width="148"
          >
            <template #default="{ row }">
              {{ formatCount(row.replayDetected) }}
            </template>
          </PersistentAdminTableColumn>
        </PersistentAdminTable>
      </section>

      <section class="open-platform-disclosure-report__split">
        <div class="open-platform-disclosure-report__panel">
          <header class="open-platform-disclosure-report__section-header">
            {{ $t('admin.openPlatform.disclosure.denialReasons') }}
          </header>
          <PersistentAdminTable
            table-key="open-platform.disclosure.denial-reasons"
            :data="denialReasons"
            row-key="reason"
            stripe
          >
            <PersistentAdminTableColumn
              column-key="reason"
              :label="$t('admin.openPlatform.disclosure.reason')"
              :default-min-width="160"
            >
              <template #default="{ row }">
                <ElTag :type="resultTagType(row.reason)" size="small">
                  {{ row.reason }}
                </ElTag>
              </template>
            </PersistentAdminTableColumn>
            <PersistentAdminTableColumn
              column-key="total"
              :label="$t('admin.openPlatform.disclosure.total')"
              :default-width="112"
            >
              <template #default="{ row }">
                {{ formatCount(row.total) }}
              </template>
            </PersistentAdminTableColumn>
          </PersistentAdminTable>
        </div>

        <div class="open-platform-disclosure-report__panel">
          <header class="open-platform-disclosure-report__section-header">
            {{ $t('admin.openPlatform.disclosure.rateLimitDimensions') }}
          </header>
          <PersistentAdminTable
            table-key="open-platform.disclosure.rate-limit-dimensions"
            :data="rateLimitDimensions"
            row-key="dimension"
            stripe
          >
            <PersistentAdminTableColumn
              column-key="dimension"
              :label="$t('admin.openPlatform.disclosure.dimension')"
              :default-min-width="160"
            >
              <template #default="{ row }">
                <span class="admin-sub-id">{{ row.dimension }}</span>
              </template>
            </PersistentAdminTableColumn>
            <PersistentAdminTableColumn
              column-key="total"
              :label="$t('admin.openPlatform.disclosure.total')"
              :default-width="112"
            >
              <template #default="{ row }">
                {{ formatCount(row.total) }}
              </template>
            </PersistentAdminTableColumn>
          </PersistentAdminTable>
        </div>
      </section>

      <section class="open-platform-disclosure-report__section">
        <header class="open-platform-disclosure-report__section-header">
          {{ $t('admin.openPlatform.disclosure.replayEvents') }}
        </header>
        <PersistentAdminTable
          table-key="open-platform.disclosure.replay-events"
          :data="replayEvents"
          row-key="id"
          stripe
        >
          <PersistentAdminTableColumn
            column-key="detectedAt"
            :label="$t('admin.common.time')"
            :default-width="156"
          >
            <template #default="{ row }">
              <span class="admin-cell-muted">
                {{ formatAdminDateTime(row.detectedAt) }}
              </span>
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
            column-key="endpoint"
            :label="$t('admin.openPlatform.disclosure.endpoint')"
            :default-width="144"
          >
            <template #default="{ row }">
              <span class="admin-sub-id">{{ row.endpoint }}</span>
            </template>
          </PersistentAdminTableColumn>
          <PersistentAdminTableColumn
            column-key="result"
            :label="$t('admin.openPlatform.disclosure.reason')"
            :default-width="160"
          >
            <template #default="{ row }">
              <ElTag :type="resultTagType(row.result)" size="small">
                {{ row.result }}
              </ElTag>
            </template>
          </PersistentAdminTableColumn>
          <PersistentAdminTableColumn
            column-key="count"
            :label="$t('admin.openPlatform.disclosure.count')"
            :default-width="104"
          >
            <template #default="{ row }">
              {{ formatCount(row.count) }}
            </template>
          </PersistentAdminTableColumn>
          <PersistentAdminTableColumn
            column-key="scopes"
            :label="$t('admin.openPlatform.audit.scope')"
            :default-min-width="220"
            show-overflow-tooltip
          >
            <template #default="{ row }">
              <span class="admin-cell-muted">
                {{ formatScopes(row.scopes) }}
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
              <span class="admin-sub-id">
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
              <pre class="open-platform-disclosure-report__metadata">{{
                formatMetadata(row.metadata)
              }}</pre>
            </template>
          </PersistentAdminTableColumn>
        </PersistentAdminTable>
      </section>
    </div>
  </AdminContentLayout>
</template>

<style scoped>
.open-platform-disclosure-report {
  min-height: 360px;
}

.open-platform-disclosure-report__summary {
  display: grid;
  grid-template-columns: repeat(5, minmax(120px, 1fr));
  border-bottom: 1px solid var(--el-border-color);
}

.open-platform-disclosure-report__metric {
  display: grid;
  gap: 6px;
  min-width: 0;
  padding: 16px 18px;
  border-right: 1px solid var(--el-border-color);
}

.open-platform-disclosure-report__metric:last-child {
  border-right: 0;
}

.open-platform-disclosure-report__metric span {
  overflow: hidden;
  text-overflow: ellipsis;
  font-size: 12px;
  line-height: 18px;
  color: var(--el-text-color-secondary);
  white-space: nowrap;
}

.open-platform-disclosure-report__metric strong {
  overflow: hidden;
  text-overflow: ellipsis;
  font-size: 24px;
  font-weight: 650;
  font-variant-numeric: tabular-nums;
  line-height: 1.2;
  white-space: nowrap;
}

.open-platform-disclosure-report__section {
  border-bottom: 1px solid var(--el-border-color);
}

.open-platform-disclosure-report__section:last-child {
  border-bottom: 0;
}

.open-platform-disclosure-report__section-header {
  padding: 14px 18px;
  font-size: 14px;
  font-weight: 650;
  line-height: 20px;
  border-bottom: 1px solid var(--el-border-color);
}

.open-platform-disclosure-report__split {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
  border-bottom: 1px solid var(--el-border-color);
}

.open-platform-disclosure-report__panel:first-child {
  border-right: 1px solid var(--el-border-color);
}

.open-platform-disclosure-report__metadata {
  min-width: 260px;
  max-width: 520px;
  max-height: 120px;
  margin: 0;
  overflow: auto;
  font-family: var(--vben-font-family-mono);
  font-size: 12px;
  line-height: 1.5;
  color: var(--el-text-color-regular);
  white-space: pre-wrap;
}

@media (max-width: 960px) {
  .open-platform-disclosure-report__summary,
  .open-platform-disclosure-report__split {
    grid-template-columns: 1fr;
  }

  .open-platform-disclosure-report__metric,
  .open-platform-disclosure-report__panel:first-child {
    border-right: 0;
  }

  .open-platform-disclosure-report__metric {
    border-bottom: 1px solid var(--el-border-color);
  }
}
</style>
