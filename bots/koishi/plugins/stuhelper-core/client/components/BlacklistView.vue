<template>
  <div class="sh-view">
    <WorkspaceHead
      title="黑名单"
      description="永久拉黑的成员,任意群组都拒绝命令与入群请求。新增条目请尽量搭配处置中心的举报记录。"
      :chips="headerChips"
    >
      <template #actions>
        <el-button
          class="sh-button sh-button--ghost"
          :disabled="loading"
          @click="refresh"
        >
          {{ loading ? '刷新中…' : '刷新' }}
        </el-button>
        <el-button
          type="primary"
          class="sh-button sh-button--primary"
          @click="openAdd"
        >
          添加用户
        </el-button>
      </template>
    </WorkspaceHead>

    <ConsolePageSkeleton v-if="loading && entries.length === 0" />

    <WorkspaceSection
      v-else
      title="黑名单成员"
      description="移除操作会立即同步到所有群,需要人工审慎判断。"
      :meta="entries.length ? `${entries.length} 条` : ''"
      flush
    >
      <QueueTable
        v-if="entries.length > 0"
        :columns="COLUMNS"
        :rows="rows"
        empty-title="黑名单为空"
        empty-body="这里会列出所有被加入黑名单的用户。"
        actions-label="操作"
        @action="handleRowAction"
      >
        <template #cell-user="{ row }">
          <EntityChip kind="user" :id="String(row.id)" />
        </template>
      </QueueTable>
      <EmptyState
        v-else
        title="黑名单为空"
        body="当前没有被永久拉黑的成员,一切正常。"
      />
    </WorkspaceSection>

    <Drawer
      :open="addOpen"
      title="添加黑名单用户"
      subtitle="userId · 立即在所有群生效"
      @close="closeAdd"
    >
      <section class="sh-drawer__section">
        <h4 class="sh-drawer__section-title">用户标识</h4>
        <label class="sh-field">
          <span class="sh-field__label">用户 ID</span>
          <el-input
            v-model.trim="draftUserId"
            class="sh-control sh-control--mono"
            placeholder="例如 1234567890"
            @keyup.enter="submitAdd"
          />
          <span class="sh-field__hint">
            支持原始 ID 或 <code>&lt;at&gt;</code> 片段;保存时自动提取数字 ID。
          </span>
        </label>
      </section>
      <template #footer>
        <el-button class="sh-button sh-button--ghost" @click="closeAdd">取消</el-button>
        <el-button
          type="primary"
          class="sh-button sh-button--primary"
          :disabled="!draftUserId.trim() || adding"
          @click="submitAdd"
        >
          {{ adding ? '添加中…' : '添加' }}
        </el-button>
      </template>
    </Drawer>

    <ConfirmDialog
      :open="confirmDialog.open"
      :title="confirmDialog.title"
      :message="confirmDialog.message"
      :tone="confirmDialog.tone"
      :confirm-text="confirmDialog.confirmText"
      :cancel-text="confirmDialog.cancelText"
      @confirm="acceptConfirm"
      @cancel="cancelConfirm"
    />

    <NoticeStack :items="notices" @dismiss="dismissNotice" />
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'

import { blacklistApi } from '../api'
import { useConfirm } from '../composables/use-confirm'
import type { BlacklistRecord } from '../types'
import { formatTimestamp } from '../models/formatters'
import ConfirmDialog from './primitives/ConfirmDialog.vue'
import ConsolePageSkeleton from './primitives/ConsolePageSkeleton.vue'
import Drawer from './primitives/Drawer.vue'
import EmptyState from './primitives/EmptyState.vue'
import EntityChip from './primitives/EntityChip.vue'
import NoticeStack, { type NoticeItem } from './primitives/NoticeStack.vue'
import QueueTable, {
  type QueueTableColumn,
  type QueueTableRow,
} from './primitives/QueueTable.vue'
import WorkspaceHead, { type WorkspaceHeadChip } from './primitives/WorkspaceHead.vue'
import WorkspaceSection from './primitives/WorkspaceSection.vue'

const COLUMNS: QueueTableColumn[] = [
  { key: 'user', label: '用户 ID' },
  { key: 'time', label: '加入时间', width: '220' },
]

const loading = ref(false)
const adding = ref(false)
const addOpen = ref(false)
const draftUserId = ref('')
const blacklist = ref<Record<string, BlacklistRecord>>({})
const notices = ref<NoticeItem[]>([])
const lastSync = ref('')
const {
  state: confirmDialog,
  confirm,
  accept: acceptConfirm,
  cancel: cancelConfirm,
} = useConfirm()

const entries = computed(() =>
  Object.entries(blacklist.value).map(([userId, record]) => ({ userId, record })),
)

const rows = computed<QueueTableRow[]>(() =>
  entries.value.map(({ userId, record }) => ({
    id: userId,
    cells: {
      user: { text: formatUserId(userId) },
      time: {
        text: record.timestamp ? formatTimestamp(record.timestamp) : '未知',
        mono: true,
      },
    },
    actions: [{ key: 'remove', label: '移除', tone: 'danger' }],
  })),
)

const headerChips = computed<WorkspaceHeadChip[]>(() => {
  const chips: WorkspaceHeadChip[] = [
    { text: `${entries.value.length} 条记录`, numeric: true },
  ]
  if (lastSync.value) {
    chips.push({ text: `更新 · ${lastSync.value}`, mono: true })
  }
  return chips
})

onMounted(refresh)

async function refresh() {
  loading.value = true
  try {
    blacklist.value = await blacklistApi.list()
    lastSync.value = formatTimestamp(Date.now())
  } catch (cause) {
    pushError(cause, '加载黑名单失败')
  } finally {
    loading.value = false
  }
}

function openAdd() {
  draftUserId.value = ''
  addOpen.value = true
}

function closeAdd() {
  addOpen.value = false
}

async function submitAdd() {
  const userId = draftUserId.value.trim()
  if (!userId) return
  adding.value = true
  try {
    await blacklistApi.add(userId, { userId, timestamp: Date.now() })
    pushSuccess(`已将 ${formatUserId(userId)} 加入黑名单`)
    addOpen.value = false
    draftUserId.value = ''
    await refresh()
  } catch (cause) {
    pushError(cause, '添加失败')
  } finally {
    adding.value = false
  }
}

function handleRowAction(payload: { rowId: string; action: string }) {
  if (payload.action !== 'remove') return
  void removeUser(payload.rowId)
}

async function removeUser(userId: string) {
  const confirmed = await confirm({
    title: '移除黑名单成员',
    message: `确定要从黑名单移除 ${formatUserId(userId)} 吗？此操作会立即影响所有群组。`,
    tone: 'danger',
    confirmText: '移除',
  })
  if (!confirmed) return

  try {
    await blacklistApi.remove(userId)
    pushSuccess(`已从黑名单移除 ${formatUserId(userId)}`)
    await refresh()
  } catch (cause) {
    pushError(cause, '移除失败')
  }
}

function formatUserId(id: string): string {
  if (id.startsWith('<at')) {
    const match = id.match(/id="(\d+)"/)
    if (match) return match[1]
  }
  return id
}

function pushSuccess(message: string) {
  notices.value.push({ id: noticeId(), kind: 'success', message })
  scheduleDismiss()
}

function pushError(cause: unknown, fallback: string) {
  const message = cause instanceof Error ? cause.message : fallback
  notices.value.push({ id: noticeId(), kind: 'error', message })
  scheduleDismiss()
}

function dismissNotice(id: string) {
  notices.value = notices.value.filter((item) => item.id !== id)
}

function scheduleDismiss() {
  const id = notices.value[notices.value.length - 1]?.id
  if (!id) return
  window.setTimeout(() => dismissNotice(id), 4000)
}

function noticeId(): string {
  return `notice-${Math.random().toString(36).slice(2, 8)}-${Date.now()}`
}
</script>
