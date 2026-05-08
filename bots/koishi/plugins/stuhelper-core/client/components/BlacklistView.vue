<template>
  <div class="sh-view">
    <WorkspaceHead
      title="黑名单"
      description="按平台和作用域同步到后端 member blacklist。"
      :chips="headerChips"
    >
      <template #actions>
        <el-button class="sh-button sh-button--ghost" :disabled="loading" @click="refresh">
          {{ loading ? '刷新中…' : '刷新' }}
        </el-button>
        <el-button type="primary" class="sh-button sh-button--primary" @click="openAdd">
          添加用户
        </el-button>
      </template>
    </WorkspaceHead>

    <ConsolePageSkeleton v-if="loading && entries.length === 0" />

    <WorkspaceSection
      v-else
      title="黑名单成员"
      description="解除黑名单会立即影响对应作用域。"
      :meta="entries.length ? `${entries.length} 条` : ''"
      flush
    >
      <QueueTable
        v-if="entries.length > 0"
        :columns="COLUMNS"
        :rows="rows"
        empty-title="黑名单为空"
        empty-body="这里会列出后端 member blacklist 的当前生效条目。"
        actions-label="操作"
        @action="handleRowAction"
      >
        <template #cell-user="{ row }">
          <EntityChip kind="user" :id="String(row.cells.user.text)" />
        </template>
      </QueueTable>
      <EmptyState v-else title="黑名单为空" body="当前没有生效的黑名单成员。" />
    </WorkspaceSection>

    <Drawer :open="addOpen" title="添加黑名单用户" subtitle="member blacklist" @close="closeAdd">
      <section class="sh-drawer__section">
        <label class="sh-field">
          <span class="sh-field__label">平台</span>
          <el-input v-model.trim="draft.platform" class="sh-control sh-control--mono" placeholder="onebot" />
        </label>
        <label class="sh-field">
          <span class="sh-field__label">用户 ID</span>
          <el-input
            v-model.trim="draft.subjectID"
            class="sh-control sh-control--mono"
            placeholder="例如 1234567890"
            @keyup.enter="submitAdd"
          />
        </label>
        <label class="sh-field">
          <span class="sh-field__label">作用域</span>
          <el-select v-model="draft.scopeType" class="sh-control">
            <el-option label="单群" value="guild" />
            <el-option label="全局" value="global" />
          </el-select>
        </label>
        <label v-if="draft.scopeType === 'guild'" class="sh-field">
          <span class="sh-field__label">群 ID</span>
          <el-input v-model.trim="draft.guildID" class="sh-control sh-control--mono" placeholder="群号" />
        </label>
        <label class="sh-field">
          <span class="sh-field__label">原因</span>
          <el-input v-model.trim="draft.reasonText" class="sh-control" placeholder="手动加入黑名单" />
        </label>
      </section>
      <template #footer>
        <el-button class="sh-button sh-button--ghost" @click="closeAdd">取消</el-button>
        <el-button
          type="primary"
          class="sh-button sh-button--primary"
          :disabled="!canSubmit || adding"
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

import { blacklistApi, type BlacklistCreateInput, type MemberBlacklistEntry } from '../api'
import { useConfirm } from '../composables/use-confirm'
import type { ConsoleNavigationController } from '../composables/use-console-navigation'
import { formatTimestamp } from '../models/formatters'
import ConfirmDialog from './primitives/ConfirmDialog.vue'
import ConsolePageSkeleton from './primitives/ConsolePageSkeleton.vue'
import Drawer from './primitives/Drawer.vue'
import EmptyState from './primitives/EmptyState.vue'
import EntityChip from './primitives/EntityChip.vue'
import NoticeStack, { type NoticeItem } from './primitives/NoticeStack.vue'
import QueueTable, { type QueueTableColumn, type QueueTableRow } from './primitives/QueueTable.vue'
import WorkspaceHead, { type WorkspaceHeadChip } from './primitives/WorkspaceHead.vue'
import WorkspaceSection from './primitives/WorkspaceSection.vue'

const COLUMNS: QueueTableColumn[] = [
  { key: 'user', label: '用户 ID' },
  { key: 'platform', label: '平台', width: '110' },
  { key: 'scope', label: '作用域', width: '150' },
  { key: 'reason', label: '原因' },
  { key: 'time', label: '加入时间', width: '190' },
]

const props = defineProps<{ navigation?: ConsoleNavigationController }>()
const loading = ref(false)
const adding = ref(false)
const addOpen = ref(false)
const draft = ref(createDraft())
const entries = ref<readonly MemberBlacklistEntry[]>([])
const notices = ref<NoticeItem[]>([])
const lastSync = ref('')
const {
  state: confirmDialog,
  confirm,
  accept: acceptConfirm,
  cancel: cancelConfirm,
} = useConfirm()

const keyword = computed(() => props.navigation?.state.value.keyword.trim().toLowerCase() ?? '')
const visibleEntries = computed(() => filterEntries(entries.value, keyword.value))
const canSubmit = computed(() => {
  return Boolean(draft.value.subjectID.trim()) &&
    (draft.value.scopeType === 'global' || Boolean(draft.value.guildID?.trim()))
})
const rows = computed<QueueTableRow[]>(() => visibleEntries.value.map(toRow))
const headerChips = computed<WorkspaceHeadChip[]>(() => {
  const chips: WorkspaceHeadChip[] = [{ text: `${visibleEntries.value.length} 条记录`, numeric: true }]
  if (lastSync.value) chips.push({ text: `更新 · ${lastSync.value}`, mono: true })
  return chips
})

onMounted(refresh)

async function refresh() {
  loading.value = true
  try {
    entries.value = (await blacklistApi.list()).items
    lastSync.value = formatTimestamp(Date.now())
  } catch (cause) {
    pushError(cause, '加载黑名单失败')
  } finally {
    loading.value = false
  }
}

function openAdd() {
  draft.value = createDraft()
  addOpen.value = true
}

function closeAdd() {
  addOpen.value = false
}

async function submitAdd() {
  if (!canSubmit.value) return
  adding.value = true
  try {
    const input = toCreateInput(draft.value)
    await blacklistApi.add(input)
    pushSuccess(`已将 ${input.subjectID} 加入黑名单`)
    addOpen.value = false
    await refresh()
  } catch (cause) {
    pushError(cause, '添加失败')
  } finally {
    adding.value = false
  }
}

function handleRowAction(payload: { rowId: string; action: string }) {
  if (payload.action !== 'remove') return
  void removeEntry(payload.rowId)
}

async function removeEntry(id: string) {
  const entry = entries.value.find((item) => item.id === id)
  const confirmed = await confirm({
    title: '移除黑名单成员',
    message: `确定要移除 ${entry?.subjectID || id} 的黑名单记录吗？`,
    tone: 'danger',
    confirmText: '移除',
  })
  if (!confirmed) return

  try {
    await blacklistApi.remove(id, 'Koishi console release')
    pushSuccess(`已移除 ${entry?.subjectID || id}`)
    await refresh()
  } catch (cause) {
    pushError(cause, '移除失败')
  }
}

function createDraft(): BlacklistCreateInput {
  return { platform: 'onebot', subjectID: '', scopeType: 'guild', guildID: '', reasonText: '' }
}

function toCreateInput(input: BlacklistCreateInput): BlacklistCreateInput {
  return {
    platform: input.platform?.trim(),
    subjectID: normalizeSubjectID(input.subjectID),
    scopeType: input.scopeType,
    guildID: input.scopeType === 'guild' ? input.guildID?.trim() : undefined,
    reasonText: input.reasonText?.trim(),
  }
}

function toRow(entry: MemberBlacklistEntry): QueueTableRow {
  return {
    id: entry.id,
    cells: {
      user: { text: entry.subjectID },
      platform: { text: entry.platform, mono: true },
      scope: { text: formatScope(entry), mono: true },
      reason: { text: entry.reasonText || entry.reasonCode },
      time: { text: formatTimestamp(Date.parse(entry.createdAt)), mono: true },
    },
    actions: [{ key: 'remove', label: '移除', tone: 'danger' }],
  }
}

function formatScope(entry: MemberBlacklistEntry): string {
  return entry.scopeType === 'global' ? '全局' : `群 ${entry.guildID || ''}`.trim()
}

function filterEntries(records: readonly MemberBlacklistEntry[], query: string) {
  if (!query) return [...records]
  return records.filter((entry) => {
    return [entry.subjectID, entry.platform, entry.guildID || '', entry.reasonText]
      .some((value) => value.toLowerCase().includes(query))
  })
}

function normalizeSubjectID(value: string): string {
  const match = value.match(/id="(\d+)"/)
  return match ? match[1] : value.trim()
}

function pushSuccess(message: string) {
  notices.value.push({ id: noticeId(), kind: 'success', message })
  scheduleDismiss()
}

function pushError(cause: unknown, fallback: string) {
  notices.value.push({ id: noticeId(), kind: 'error', message: cause instanceof Error ? cause.message : fallback })
  scheduleDismiss()
}

function dismissNotice(id: string) {
  notices.value = notices.value.filter((item) => item.id !== id)
}

function scheduleDismiss() {
  const id = notices.value[notices.value.length - 1]?.id
  if (id) window.setTimeout(() => dismissNotice(id), 4000)
}

function noticeId(): string {
  return `notice-${Math.random().toString(36).slice(2, 8)}-${Date.now()}`
}
</script>
