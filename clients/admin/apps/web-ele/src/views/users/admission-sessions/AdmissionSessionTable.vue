<script setup lang="ts">
import type { AdmissionSession } from '#/api/admin';

import { ElButton, ElPagination, ElTag } from 'element-plus';

import PersistentAdminTable from '../../shared/admin-table/PersistentAdminTable.vue';
import PersistentAdminTableColumn from '../../shared/admin-table/PersistentAdminTableColumn.vue';
import {
  admissionReissueCommand,
  boolLabel,
  formatDateTime,
  formatText,
  statusLabel,
  statusTagType,
} from './options';

defineProps<{
  items: AdmissionSession[];
  loading: boolean;
  total: number;
}>();

const emit = defineEmits<{
  (e: 'copyAuthURL', url: string): void;
  (e: 'copyReissueCommand', command: string): void;
  (e: 'pageChange'): void;
  (e: 'pageSizeChange'): void;
}>();

const page = defineModel<number>('page', { required: true });
const pageSize = defineModel<number>('pageSize', { required: true });
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
        :default-width="120"
      >
        <template #default="{ row }">
          <ElTag :type="statusTagType(row.status)" data-field="status">
            {{ statusLabel(row.status) }}
          </ElTag>
        </template>
      </PersistentAdminTableColumn>
      <PersistentAdminTableColumn
        column-key="subject"
        label="成员"
        :default-min-width="180"
      >
        <template #default="{ row }">
          <div class="font-mono">{{ row.qqID }}</div>
          <div class="text-xs text-slate-500">用户 {{ formatText(row.userID) }}</div>
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
            Bot {{ formatText(row.botSelfID) }} · 频道 {{ formatText(row.channelID) }}
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
            已消费：{{ boolLabel(Boolean(row.tokenConsumedAt)) }}
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
            v-if="row.status !== 'verified'"
            data-action="copyReissueCommand"
            link
            size="small"
            type="warning"
            @click="emit('copyReissueCommand', admissionReissueCommand(row))"
          >
            复制重生命令
          </ElButton>
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
        label="Bot 错误"
        :default-min-width="220"
      >
        <template #default="{ row }">
          <span class="text-xs">{{ formatText(row.lastBotError) }}</span>
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
