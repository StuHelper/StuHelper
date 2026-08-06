<script setup lang="ts">
import type { AdmissionSessionAction } from './options';

import type { AdmissionSession } from '#/api/admin';

import { ElButton, ElPagination, ElPopconfirm, ElTag } from 'element-plus';

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
    actionLoadingById?: Record<string, AdmissionSessionAction | undefined>;
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
        label="状态"
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
        label="成员"
        :default-min-width="180"
      >
        <template #default="{ row }">
          <div class="font-mono">{{ row.qqID }}</div>
          <div class="text-xs text-slate-500">
            用户 {{ formatText(row.userID) }}
          </div>
        </template>
      </PersistentAdminTableColumn>
      <PersistentAdminTableColumn
        column-key="runtime"
        label="运行时"
        :default-min-width="210"
      >
        <template #default="{ row }">
          <div>{{ row.platform }} / 群 {{ row.guildID }}</div>
          <div class="text-xs text-slate-500">
            Bot {{ formatText(row.botSelfID) }} · 频道
            {{ formatText(row.channelID) }}
          </div>
          <div class="text-xs text-slate-500" data-field="runtimeBoundary">
            后端 admission session；Koishi WebUI 显示现场 guard record
          </div>
        </template>
      </PersistentAdminTableColumn>
      <PersistentAdminTableColumn
        column-key="token"
        label="链接"
        :default-min-width="260"
      >
        <template #default="{ row }">
          <div class="font-mono text-xs">{{ row.id }}</div>
          <div class="text-xs text-slate-500">
            JOIN 链接：{{ row.authURL ? '可复制' : '未返回' }} · 已消费：{{
              boolLabel(Boolean(row.tokenConsumedAt))
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
            复制认证链接
          </ElButton>
          <ElButton
            v-if="canManageAdmissionSession(row.status)"
            data-action="copyReissueCommand"
            link
            size="small"
            type="warning"
            @click="emit('copyReissueCommand', admissionReissueCommand(row))"
          >
            复制重生命令
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
            请求重发
          </ElButton>
          <ElPopconfirm
            v-if="canManage && canManageAdmissionSession(row.status)"
            title="重新生成会取消当前未完成会话，确认继续？"
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
                重新生成
              </ElButton>
            </template>
          </ElPopconfirm>
          <ElPopconfirm
            v-if="canManage && canManageAdmissionSession(row.status)"
            title="确认取消该入群认证会话？"
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
                取消会话
              </ElButton>
            </template>
          </ElPopconfirm>
        </template>
      </PersistentAdminTableColumn>
      <PersistentAdminTableColumn
        column-key="deadlines"
        label="期限"
        :default-min-width="240"
      >
        <template #default="{ row }">
          <div>链接：{{ formatDateTime(row.linkWaitDeadlineAt) }}</div>
          <div>提交：{{ formatDateTime(row.submissionWaitDeadlineAt) }}</div>
          <div>审核：{{ formatDateTime(row.manualReviewDeadlineAt) }}</div>
        </template>
      </PersistentAdminTableColumn>
      <PersistentAdminTableColumn
        column-key="events"
        label="事件"
        :default-min-width="220"
      >
        <template #default="{ row }">
          <div>消费：{{ formatDateTime(row.tokenConsumedAt) }}</div>
          <div>通过：{{ formatDateTime(row.verifiedAt) }}</div>
          <div>取消：{{ formatDateTime(row.cancelledAt) }}</div>
        </template>
      </PersistentAdminTableColumn>
      <PersistentAdminTableColumn
        column-key="botError"
        label="Bot 执行诊断"
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
        label="禁言到期"
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
