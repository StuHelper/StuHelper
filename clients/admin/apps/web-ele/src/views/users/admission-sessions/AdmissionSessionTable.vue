<script setup lang="ts">
import type { AdmissionSessionAction } from './options';

import type { AdmissionSession } from '#/api/admin';

import { ElButton, ElPagination, ElPopconfirm, ElTag } from 'element-plus';

import { $t } from '#/locales';

import PersistentAdminTable from '../../shared/admin-table/PersistentAdminTable.vue';
import PersistentAdminTableColumn from '../../shared/admin-table/PersistentAdminTableColumn.vue';
import {
  admissionReissueCommand,
  boolLabel,
  botErrorLabel,
  canManageAdmissionSession,
  formatDateTime,
  formatText,
  statusLabel,
  statusOperationHint,
  statusTagType,
} from './options';

const props = withDefaults(
  defineProps<{
    actionLoadingById?: Readonly<Record<string, string | undefined>>;
    canManage: boolean;
    items: AdmissionSession[];
    loading: boolean;
    total: number;
  }>(),
  { actionLoadingById: () => ({}) },
);

const emit = defineEmits<{
  (e: 'copyAuthURL', url: string): void;
  (e: 'copyReissueCommand', command: string): void;
  (e: 'requestResend', id: string): void;
  (e: 'requestRegenerate', id: string): void;
  (e: 'requestCancel', id: string): void;
  (e: 'pageChange'): void;
  (e: 'pageSizeChange'): void;
}>();

const page = defineModel<number>('page', { required: true });
const pageSize = defineModel<number>('pageSize', { required: true });

function sessionActionLoading(
  row: AdmissionSession,
  action: AdmissionSessionAction,
) {
  return props.actionLoadingById[row.id] === action;
}

function sessionActionRunning(row: AdmissionSession) {
  return Boolean(props.actionLoadingById[row.id]);
}

function sessionActionDisabled(row: AdmissionSession) {
  return props.loading || sessionActionRunning(row);
}
</script>

<template>
  <div>
    <PersistentAdminTable
      table-key="users.admissionSessions"
      :loading="loading"
      :data="items"
      row-key="id"
      stripe
    >
      <PersistentAdminTableColumn
        column-key="status"
        :label="$t('admin.users.admissionSessions.statusColumn')"
        :default-min-width="180"
      >
        <template #default="{ row }">
          <div class="grid gap-1">
            <ElTag :type="statusTagType(row.status)" data-field="status">
              {{ statusLabel(row.status) }}
            </ElTag>
            <span
              class="text-xs leading-5 text-slate-500"
              data-field="statusHint"
            >
              {{ statusOperationHint(row.status) }}
            </span>
          </div>
        </template>
      </PersistentAdminTableColumn>
      <PersistentAdminTableColumn
        column-key="subject"
        :label="$t('admin.users.admissionSessions.memberColumn')"
        :default-min-width="180"
      >
        <template #default="{ row }">
          <div class="font-mono">{{ row.qqID }}</div>
          <div class="text-xs text-slate-500">
            {{
              $t('admin.users.admissionSessions.userLine', {
                id: formatText(row.userID),
              })
            }}
          </div>
        </template>
      </PersistentAdminTableColumn>
      <PersistentAdminTableColumn
        column-key="runtime"
        :label="$t('admin.users.admissionSessions.runtimeColumn')"
        :default-min-width="210"
      >
        <template #default="{ row }">
          <div>
            {{
              $t('admin.users.admissionSessions.guildLine', {
                guild: row.guildID,
                platform: row.platform,
              })
            }}
          </div>
          <div class="text-xs text-slate-500">
            {{
              $t('admin.users.admissionSessions.botLine', {
                bot: formatText(row.botSelfID),
                channel: formatText(row.channelID),
              })
            }}
          </div>
          <div class="text-xs text-slate-500" data-field="runtimeBoundary">
            {{ $t('admin.users.admissionSessions.boundaryNote') }}
          </div>
        </template>
      </PersistentAdminTableColumn>
      <PersistentAdminTableColumn
        column-key="token"
        :label="$t('admin.users.admissionSessions.linkColumn')"
        :default-min-width="260"
      >
        <template #default="{ row }">
          <div class="font-mono text-xs">{{ row.id }}</div>
          <div class="text-xs text-slate-500">
            {{
              $t('admin.users.admissionSessions.joinLinkLine', {
                availability: row.authURL
                  ? $t('admin.users.admissionSessions.linkCopyable')
                  : $t('admin.users.admissionSessions.linkNotReturned'),
                consumed: boolLabel(Boolean(row.tokenConsumedAt)),
              })
            }}
          </div>
          <ElButton
            v-if="row.authURL"
            data-action="copyAuthURL"
            link
            size="small"
            type="primary"
            @click="emit('copyAuthURL', row.authURL)"
          >
            {{ $t('admin.users.admissionSessions.copyAuthURL') }}
          </ElButton>
          <ElButton
            v-if="row.status !== 'verified'"
            data-action="copyReissueCommand"
            link
            size="small"
            type="warning"
            @click="emit('copyReissueCommand', admissionReissueCommand(row))"
          >
            {{ $t('admin.users.admissionSessions.copyReissue') }}
          </ElButton>
          <ElButton
            v-if="canManage && canManageAdmissionSession(row.status)"
            data-action="requestResend"
            link
            size="small"
            type="primary"
            :disabled="sessionActionDisabled(row)"
            :loading="sessionActionLoading(row, 'resend')"
            @click="emit('requestResend', row.id)"
          >
            {{ $t('admin.users.admissionSessions.requestResend') }}
          </ElButton>
          <ElPopconfirm
            v-if="canManage && canManageAdmissionSession(row.status)"
            :title="$t('admin.users.admissionSessions.regenerateConfirm')"
            @confirm="emit('requestRegenerate', row.id)"
          >
            <template #reference>
              <ElButton
                data-action="requestRegenerate"
                link
                size="small"
                type="warning"
                :disabled="sessionActionDisabled(row)"
                :loading="sessionActionLoading(row, 'regenerate')"
              >
                {{ $t('admin.users.admissionSessions.regenerate') }}
              </ElButton>
            </template>
          </ElPopconfirm>
          <ElPopconfirm
            v-if="canManage && canManageAdmissionSession(row.status)"
            :title="$t('admin.users.admissionSessions.cancelConfirm')"
            @confirm="emit('requestCancel', row.id)"
          >
            <template #reference>
              <ElButton
                data-action="requestCancel"
                link
                size="small"
                type="danger"
                :disabled="sessionActionDisabled(row)"
                :loading="sessionActionLoading(row, 'cancel')"
              >
                {{ $t('admin.users.admissionSessions.cancelSession') }}
              </ElButton>
            </template>
          </ElPopconfirm>
        </template>
      </PersistentAdminTableColumn>
      <PersistentAdminTableColumn
        column-key="deadlines"
        :label="$t('admin.users.admissionSessions.deadlinesColumn')"
        :default-min-width="240"
      >
        <template #default="{ row }">
          <div>
            {{
              $t('admin.users.admissionSessions.linkDeadline', {
                time: formatDateTime(row.linkWaitDeadlineAt),
              })
            }}
          </div>
          <div>
            {{
              $t('admin.users.admissionSessions.submitDeadline', {
                time: formatDateTime(row.submissionWaitDeadlineAt),
              })
            }}
          </div>
          <div>
            {{
              $t('admin.users.admissionSessions.reviewDeadline', {
                time: formatDateTime(row.manualReviewDeadlineAt),
              })
            }}
          </div>
        </template>
      </PersistentAdminTableColumn>
      <PersistentAdminTableColumn
        column-key="events"
        :label="$t('admin.users.admissionSessions.eventsColumn')"
        :default-min-width="220"
      >
        <template #default="{ row }">
          <div>
            {{
              $t('admin.users.admissionSessions.consumedAt', {
                time: formatDateTime(row.tokenConsumedAt),
              })
            }}
          </div>
          <div>
            {{
              $t('admin.users.admissionSessions.verifiedAt', {
                time: formatDateTime(row.verifiedAt),
              })
            }}
          </div>
          <div>
            {{
              $t('admin.users.admissionSessions.cancelledAt', {
                time: formatDateTime(row.cancelledAt),
              })
            }}
          </div>
        </template>
      </PersistentAdminTableColumn>
      <PersistentAdminTableColumn
        column-key="botError"
        :label="$t('admin.users.admissionSessions.botDiagnosticsColumn')"
        :default-min-width="260"
      >
        <template #default="{ row }">
          <div class="grid gap-1" data-field="botDiagnostics">
            <ElTag size="small" :type="row.lastBotError ? 'danger' : 'info'">
              {{ botErrorLabel(row) }}
            </ElTag>
            <span class="text-xs leading-5 text-slate-500">
              {{ formatText(row.lastBotError) }}
            </span>
          </div>
        </template>
      </PersistentAdminTableColumn>
      <PersistentAdminTableColumn
        column-key="initialMuteUntil"
        :label="$t('admin.users.admissionSessions.muteUntilColumn')"
        :default-width="180"
      >
        <template #default="{ row }">
          {{ formatDateTime(row.initialMuteUntil) }}
        </template>
      </PersistentAdminTableColumn>
    </PersistentAdminTable>

    <div class="admin-embedded-pagination">
      <ElPagination
        v-model:current-page="page"
        v-model:page-size="pageSize"
        :total="total"
        layout="total, prev, pager, next, sizes"
        @current-change="emit('pageChange')"
        @size-change="emit('pageSizeChange')"
      />
    </div>
  </div>
</template>
