<template>
  <div class="sh-view">
    <WorkspaceHead
      title="黑名单"
      description="按本群或全局范围生效的成员黑名单。新增全局条目需要单独确认。"
      :chips="headerChips"
    >
      <template #actions>
        <el-button class="sh-button sh-button--ghost" :disabled="loading" @click="refresh">
          {{ loading ? '刷新中…' : '刷新' }}
        </el-button>
        <el-button
          type="primary"
          class="sh-button sh-button--primary"
          :disabled="loading || initialLoadBlocked"
          @click="openAdd"
        >
          添加用户
        </el-button>
      </template>
    </WorkspaceHead>

    <ConsolePageSkeleton v-if="loading && blacklist.length === 0" />
    <EmptyState
      v-else-if="loadError && blacklist.length === 0"
      tone="error"
      title="加载黑名单失败"
      :body="loadError"
    >
      <template #action>
        <el-button class="sh-button sh-button--ghost" @click="refresh">重试</el-button>
      </template>
    </EmptyState>

    <template v-else>
      <div
        v-if="loadError && blacklist.length > 0"
        class="sh-blacklist-load-error"
        role="alert"
      >
        <div class="sh-blacklist-load-error__body">
          <strong>刷新黑名单失败</strong>
          <span>{{ loadError }}</span>
        </div>
        <el-button class="sh-button sh-button--ghost" @click="refresh">重试</el-button>
      </div>

      <div
        v-if="actionError"
        class="sh-blacklist-action-error"
        role="alert"
      >
        <div class="sh-blacklist-action-error__body">
          <strong>{{ actionErrorTitle }}</strong>
          <span>{{ actionError }}</span>
        </div>
        <el-button class="sh-button sh-button--ghost" @click="clearActionError">关闭</el-button>
      </div>

      <WorkspaceSection
        title="黑名单成员"
        description="移除操作会立即同步到所有群,需要人工审慎判断。"
        :meta="entries.length ? `${entries.length} 条` : ''"
        flush
      >
        <QueueTable
          v-if="entries.length > 0"
          :columns="COLUMNS" :rows="rows"
          empty-title="黑名单为空"
          empty-body="这里会列出所有被加入黑名单的用户。"
          actions-label="操作" @action="handleRowAction"
        >
          <template #cell-user="{ value }">
            <EntityChip kind="user" :id="String(value.text)" />
          </template>
        </QueueTable>
        <EmptyState
          v-else
          title="黑名单为空"
          body="当前没有被永久拉黑的成员,一切正常。"
        />
      </WorkspaceSection>
    </template>

    <Drawer :open="addOpen" title="添加黑名单用户" subtitle="用户 · 生效范围" @close="closeAdd">
      <div
        v-if="actionError"
        class="sh-blacklist-action-error sh-blacklist-action-error--drawer"
        role="alert"
      >
        <div class="sh-blacklist-action-error__body">
          <strong>{{ actionErrorTitle }}</strong>
          <span>{{ actionError }}</span>
        </div>
        <el-button class="sh-button sh-button--ghost" @click="clearActionError">关闭</el-button>
      </div>

      <section class="sh-drawer__section">
        <h4 class="sh-drawer__section-title">用户标识</h4>
        <label class="sh-field">
          <span class="sh-field__label">用户 ID</span>
          <el-input v-model.trim="draftUserId" class="sh-control sh-control--mono" placeholder="例如 1234567890" @keyup.enter="submitAdd" />
        </label>
        <label class="sh-field">
          <span class="sh-field__label">范围</span>
          <el-select v-model="draftScope" class="sh-control">
            <el-option label="本群" value="guild" />
            <el-option label="全局" value="global" />
          </el-select>
        </label>
        <label v-if="draftScope === 'guild'" class="sh-field">
          <span class="sh-field__label">群 ID</span>
          <el-input v-model.trim="draftGuildId" class="sh-control sh-control--mono" placeholder="例如 100000000" />
        </label>
        <label class="sh-field">
          <span class="sh-field__label">原因</span>
          <el-input v-model.trim="draftReason" class="sh-control" placeholder="手动加入黑名单" />
        </label>
      </section>
      <template #footer>
        <el-button class="sh-button sh-button--ghost" @click="closeAdd">取消</el-button>
        <el-button type="primary" class="sh-button sh-button--primary" :disabled="!canSubmitAdd || adding" @click="submitAdd">
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
import { useActionFeedback } from '../composables/use-action-feedback'
import { useConfirm } from '../composables/use-confirm'
import type { ConsoleNavigationController } from '../composables/use-console-navigation'
import type { MemberBlacklistEntry, MemberBlacklistScopeType } from '../types'
import {
  canSubmitBlacklistDraft,
  filterBlacklistEntries,
  formatBlacklistScope,
  MEMBER_BLACKLIST_COLUMNS,
  normalizeBlacklistUserID,
  toBlacklistRows,
} from '../models/member-blacklist'
import { formatTimestamp } from '../models/formatters'
import ConfirmDialog from './primitives/ConfirmDialog.vue'
import ConsolePageSkeleton from './primitives/ConsolePageSkeleton.vue'
import Drawer from './primitives/Drawer.vue'
import EmptyState from './primitives/EmptyState.vue'
import EntityChip from './primitives/EntityChip.vue'
import NoticeStack from './primitives/NoticeStack.vue'
import QueueTable from './primitives/QueueTable.vue'
import WorkspaceHead, { type WorkspaceHeadChip } from './primitives/WorkspaceHead.vue'
import WorkspaceSection from './primitives/WorkspaceSection.vue'

const COLUMNS = MEMBER_BLACKLIST_COLUMNS

const props = defineProps<{
  navigation?: ConsoleNavigationController
}>()

const loading = ref(false)
const adding = ref(false)
const addOpen = ref(false)
const draftUserId = ref('')
const draftScope = ref<MemberBlacklistScopeType>('guild')
const draftGuildId = ref('')
const draftReason = ref('')
const blacklist = ref<readonly MemberBlacklistEntry[]>([])
const lastSync = ref('')
const loadError = ref('')
const removingIds = ref(new Set<string>())
let refreshRequestSeq = 0
const {
  state: confirmDialog,
  confirm,
  accept: acceptConfirm,
  cancel: cancelConfirm,
} = useConfirm()
const {
  actionError,
  actionErrorTitle,
  notices,
  pushSuccess,
  pushError,
  setActionError,
  clearActionError,
  dismissNotice,
  errorMessage,
} = useActionFeedback()

const keyword = computed(() => props.navigation?.state.value.keyword.trim().toLowerCase() ?? '')
const entries = computed(() => filterBlacklistEntries(blacklist.value, keyword.value))
const canSubmitAdd = computed(() => canSubmitBlacklistDraft({
  userId: draftUserId.value,
  scope: draftScope.value,
  guildId: draftGuildId.value,
}))

const rows = computed(() => toBlacklistRows(entries.value).map((row) => {
  if (!removingIds.value.has(row.id) || !row.actions?.length) return row
  return {
    ...row,
    actions: row.actions.map((action) => ({ ...action, disabled: true })),
  }
}))
const initialLoadBlocked = computed(() => Boolean(loadError.value && blacklist.value.length === 0))

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
  const requestSeq = ++refreshRequestSeq
  loading.value = true
  loadError.value = ''
  try {
    const next = await blacklistApi.list()
    if (requestSeq !== refreshRequestSeq) return
    blacklist.value = next.list
    lastSync.value = formatTimestamp(Date.now())
  } catch (cause) {
    if (requestSeq !== refreshRequestSeq) return
    const details = errorMessage(cause, '加载黑名单失败')
    loadError.value = details
    pushError('加载黑名单失败', details)
  } finally {
    if (requestSeq === refreshRequestSeq) {
      loading.value = false
    }
  }
}

function openAdd() {
  if (initialLoadBlocked.value) {
    pushError('黑名单尚未加载', '黑名单尚未加载，无法添加用户')
    return
  }
  draftUserId.value = ''
  draftScope.value = 'guild'
  draftGuildId.value = ''
  draftReason.value = ''
  addOpen.value = true
}

function closeAdd() {
  addOpen.value = false
}

async function submitAdd() {
  if (!canSubmitAdd.value || adding.value) return
  const userId = normalizeBlacklistUserID(draftUserId.value.trim())
  adding.value = true
  clearActionError()
  try {
    if (draftScope.value === 'global' && !await confirmGlobalAdd(userId)) return
    await blacklistApi.add({
      subjectID: userId,
      scopeType: draftScope.value,
      guildID: draftScope.value === 'guild' ? draftGuildId.value.trim() : undefined,
      reasonText: draftReason.value.trim() || undefined,
    })
    pushSuccess(`已将 ${userId} 加入黑名单`)
    addOpen.value = false
    resetDraft()
    await refresh()
  } catch (cause) {
    setActionError('添加失败', cause, '添加失败')
  } finally {
    adding.value = false
  }
}

function handleRowAction(payload: { rowId: string; action: string }) {
  if (payload.action === 'release_only') {
    void removeUser(payload.rowId, 'release_only')
  } else if (payload.action === 'forgive') {
    void removeUser(payload.rowId, 'manual_pardon')
  }
}

async function removeUser(
  userId: string,
  releaseReasonCode: 'manual_pardon' | 'release_only',
) {
  const entry = blacklist.value.find((item) => item.id === userId)
  if (!entry || removingIds.value.has(userId)) return
  const isForgive = releaseReasonCode === 'manual_pardon'
  removingIds.value = new Set([...removingIds.value, userId])

  try {
    const confirmed = await confirm({
      title: isForgive ? '宽恕黑名单成员（重置失败计数）' : '解除黑名单成员',
      message: isForgive
        ? `确定要宽恕 ${entry.subjectID} 的${formatBlacklistScope(entry)}黑名单吗？这会重置认证失败计数。`
        : `确定要解除 ${entry.subjectID} 的${formatBlacklistScope(entry)}黑名单吗？认证失败计数会保留。`,
      tone: 'danger',
      confirmText: isForgive ? '宽恕' : '解除',
    })
    if (!confirmed) return

    clearActionError()
    await blacklistApi.remove({
      id: entry.id,
      releaseReasonCode,
    })
    pushSuccess(isForgive
      ? `已宽恕并解除 ${entry.subjectID}`
      : `已从黑名单解除 ${entry.subjectID}`)
    await refresh()
  } catch (cause) {
    setActionError('解除失败', cause, '解除失败')
  } finally {
    const nextRemoving = new Set(removingIds.value)
    nextRemoving.delete(userId)
    removingIds.value = nextRemoving
  }
}

function resetDraft() {
  draftUserId.value = ''
  draftGuildId.value = ''
  draftReason.value = ''
}

function confirmGlobalAdd(userId: string) {
  return confirm({
    title: '添加全局黑名单',
    message: `确定要将 ${userId} 加入全局黑名单吗？该成员会被所有群拒绝。`,
    tone: 'danger',
    confirmText: '添加全局黑名单',
  })
}
</script>

<style scoped>
.sh-blacklist-load-error,
.sh-blacklist-action-error {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--sh-s-3);
  padding: var(--sh-s-3) var(--sh-s-4);
  border: 1px solid rgba(248, 81, 73, 0.28);
  border-radius: var(--sh-r-2);
  background: rgba(248, 81, 73, 0.08);
}

.sh-blacklist-action-error--drawer {
  margin-bottom: var(--sh-s-4);
}

.sh-blacklist-load-error__body,
.sh-blacklist-action-error__body {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.sh-blacklist-load-error__body strong,
.sh-blacklist-action-error__body strong {
  color: #ff8a80;
  font-size: var(--sh-t-body);
}

.sh-blacklist-load-error__body span,
.sh-blacklist-action-error__body span {
  color: var(--sh-fg-2);
  font-size: var(--sh-t-meta);
  overflow-wrap: anywhere;
}
</style>
