<template>
  <div class="sh-view">
    <WorkspaceHead
      title="警告记录"
      description="跨群的累计警告次数。调整次数会立刻生效;清零后记录从本群表里移除。"
      :chips="headerChips"
    >
      <template #actions>
        <label class="sh-warns__toggle">
          <span class="sh-warns__toggle-label">解析名称</span>
          <el-switch v-model="fetchNames" size="small" @change="refresh" />
        </label>
        <el-button
          class="sh-button sh-button--ghost"
          :disabled="reloading"
          @click="reload"
        >
          {{ reloading ? '重载中…' : '重载' }}
        </el-button>
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
          :disabled="loading || initialLoadBlocked"
          @click="openAdd"
        >
          添加警告
        </el-button>
      </template>
    </WorkspaceHead>

    <ConsolePageSkeleton v-if="loading && sourceRecordCount === 0" />
    <EmptyState
      v-else-if="loadError && sourceRecordCount === 0"
      tone="error"
      title="加载警告记录失败"
      :body="loadError"
    >
      <template #action>
        <el-button class="sh-button sh-button--ghost" @click="refresh">重试</el-button>
      </template>
    </EmptyState>

    <template v-else>
      <div
        v-if="loadError && sourceRecordCount > 0"
        class="sh-warns-load-error"
        role="alert"
      >
        <div class="sh-warns-load-error__body">
          <strong>刷新警告记录失败</strong>
          <span>{{ loadError }}</span>
        </div>
        <el-button class="sh-button sh-button--ghost" @click="refresh">重试</el-button>
      </div>

      <div
        v-if="actionError"
        class="sh-warns-action-error"
        role="alert"
      >
        <div class="sh-warns-action-error__body">
          <strong>{{ actionErrorTitle }}</strong>
          <span>{{ actionError }}</span>
        </div>
        <el-button class="sh-button sh-button--ghost" @click="clearActionError">关闭</el-button>
      </div>

      <EmptyState
        v-if="totalRecords === 0"
        title="暂无警告记录"
        body="当前没有成员累计警告记录。添加后可在此按群组检索与调整次数。"
      />

      <div v-else class="sh-split sh-split--7-5">
        <WorkspaceSection
          title="群组"
          description="按群聚合,点击切换右侧详细列表。"
          :meta="`${guildIds.length} 群`"
          flush
        >
          <div class="sh-lane">
            <button
              v-for="guildId in guildIds"
              :key="guildId"
              type="button"
              class="sh-lane__row sh-lane__row--interactive"
              :class="{ 'sh-lane__row--active': selectedGuildId === guildId }"
              @click="selectedGuildId = guildId"
            >
              <span class="sh-lane__dot" :class="severityDot(groupCount(guildId))"></span>
              <div class="sh-lane__body">
                <div class="sh-lane__title">{{ guildName(guildId) }}</div>
                <div class="sh-lane__subtitle">
                  <EntityChip kind="guild" :id="guildId" inline />
                  · {{ groupCount(guildId) }} 条记录
                </div>
              </div>
              <span class="sh-lane__chevron" aria-hidden="true">›</span>
            </button>
          </div>
        </WorkspaceSection>

        <WorkspaceSection
          :title="selectedGroup ? guildName(selectedGroup[0].guildId) : '详情'"
          :description="selectedGroup ? '成员警告次数,可以直接在列上调整或一键清零。' : '请选择左侧群组查看警告列表。'"
          :meta="selectedGroup ? `${selectedGroup.length} 条` : ''"
          flush
        >
          <EmptyState
            v-if="!selectedGroup"
            title="请选择左侧群组"
            body="选中一个群后,这里会列出该群所有仍有记录的成员。"
          />
          <QueueTable
            v-else
            :columns="COLUMNS"
            :rows="rows"
            empty-title="该群已无警告"
            empty-body="这个群的警告记录已经全部清除。"
            actions-label="操作"
            @action="handleRowAction"
          >
            <template #cell-user="{ row }">
              <EntityChip
                kind="user"
                :id="warnUserCell(row.cells.user).secondary"
                :name="warnUserCell(row.cells.user).text"
                :guild-id="selectedGuildId ?? undefined"
              />
            </template>
            <template #cell-count="{ row }">
              <el-input-number
                :model-value="warnCountCell(row.cells.count).value"
                :min="0"
                :max="99"
                size="small"
                controls-position="right"
                @change="(val: number | undefined) => updateCount(row.id, val)"
              />
            </template>
          </QueueTable>
        </WorkspaceSection>
      </div>
    </template>

    <Drawer
      :open="addOpen"
      title="添加警告"
      subtitle="guildId · userId"
      @close="closeAdd"
    >
      <div
        v-if="actionError"
        class="sh-warns-action-error sh-warns-action-error--drawer"
        role="alert"
      >
        <div class="sh-warns-action-error__body">
          <strong>{{ actionErrorTitle }}</strong>
          <span>{{ actionError }}</span>
        </div>
        <el-button class="sh-button sh-button--ghost" @click="clearActionError">关闭</el-button>
      </div>

      <section class="sh-drawer__section">
        <h4 class="sh-drawer__section-title">目标</h4>
        <div class="sh-form-grid">
          <label class="sh-field">
            <span class="sh-field__label">群号</span>
            <el-input
              v-model.trim="draft.guildId"
              class="sh-control sh-control--mono"
              placeholder="例如 123456789"
            />
          </label>
          <label class="sh-field">
            <span class="sh-field__label">用户 ID</span>
            <el-input
              v-model.trim="draft.userId"
              class="sh-control sh-control--mono"
              placeholder="例如 1234567890"
              @keyup.enter="submitAdd"
            />
          </label>
        </div>
        <p class="sh-field__hint">
          添加后累计次数 +1。如果想一次加多个记录,请分别保存。
        </p>
      </section>
      <template #footer>
        <el-button class="sh-button sh-button--ghost" @click="closeAdd">取消</el-button>
        <el-button
          type="primary"
          class="sh-button sh-button--primary"
          :disabled="!canSubmitAdd || adding"
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
import { computed, onMounted, reactive, ref, watch } from 'vue'

import { warnsApi, type WarnListItem } from '../api'
import { useActionFeedback } from '../composables/use-action-feedback'
import { useConfirm } from '../composables/use-confirm'
import type { ConsoleNavigationController } from '../composables/use-console-navigation'
import { formatTimestamp } from '../models/formatters'
import ConfirmDialog from './primitives/ConfirmDialog.vue'
import ConsolePageSkeleton from './primitives/ConsolePageSkeleton.vue'
import Drawer from './primitives/Drawer.vue'
import EmptyState from './primitives/EmptyState.vue'
import EntityChip from './primitives/EntityChip.vue'
import NoticeStack from './primitives/NoticeStack.vue'
import QueueTable, {
  type QueueTableCell,
  type QueueTableCellObject,
  type QueueTableColumn,
  type QueueTableRow,
} from './primitives/QueueTable.vue'
import WorkspaceHead, { type WorkspaceHeadChip } from './primitives/WorkspaceHead.vue'
import WorkspaceSection from './primitives/WorkspaceSection.vue'

type ProcessedWarn = WarnListItem

interface WarnUserCell extends QueueTableCellObject {
  secondary: string
}

interface WarnCountCell extends QueueTableCellObject {
  value: number
}

const props = defineProps<{
  navigation?: ConsoleNavigationController
}>()

const COLUMNS: QueueTableColumn[] = [
  { key: 'user', label: '用户' },
  { key: 'time', label: '时间', width: '180' },
  { key: 'count', label: '次数', width: '160', align: 'right' },
]

const loading = ref(false)
const reloading = ref(false)
const adding = ref(false)
const fetchNames = ref(true)
const addOpen = ref(false)
const selectedGuildId = ref('')
const guildFilter = ref('')
const keyword = ref('')
const lastSync = ref('')
const loadError = ref('')
const groups = ref<Record<string, ProcessedWarn[]>>({})
let refreshRequestSeq = 0
const draft = reactive({ guildId: '', userId: '' })
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

const filteredGroups = computed(() => filterWarnGroups(groups.value, {
  guildId: guildFilter.value,
  keyword: keyword.value,
}))
const guildIds = computed(() => Object.keys(filteredGroups.value))
const selectedGroup = computed(() =>
  selectedGuildId.value ? filteredGroups.value[selectedGuildId.value] ?? null : null,
)
const totalRecords = computed(() =>
  Object.values(filteredGroups.value).reduce((acc, list) => acc + list.length, 0),
)
const sourceRecordCount = computed(() =>
  Object.values(groups.value).reduce((acc, list) => acc + list.length, 0),
)
const initialLoadBlocked = computed(() => Boolean(loadError.value && sourceRecordCount.value === 0))

const rows = computed<QueueTableRow[]>(() => {
  const list = selectedGroup.value ?? []
  return list.map((item) => {
    const userCell: WarnUserCell = {
      text: item.userName && item.userName !== 'Unknown' ? item.userName : '未知用户',
      secondary: item.userId,
    }
    const countCell: WarnCountCell = { text: String(item.count), value: item.count }

    return {
      id: item.key,
      cells: {
        user: userCell,
        time: {
          text: item.timestamp ? formatTimestamp(item.timestamp) : '未知',
          mono: true,
        },
        count: countCell,
      },
      actions: [{ key: 'clear', label: '清除', tone: 'danger' }],
    }
  })
})

const headerChips = computed<WorkspaceHeadChip[]>(() => {
  const chips: WorkspaceHeadChip[] = [
    { text: `${totalRecords.value} 条记录 · ${guildIds.value.length} 群`, numeric: true },
  ]
  if (lastSync.value) {
    chips.push({ text: `更新 · ${lastSync.value}`, mono: true })
  }
  return chips
})

function warnUserCell(cell: QueueTableCell | undefined): WarnUserCell {
  if (!isCellObject(cell) || typeof cell.secondary !== 'string') {
    throw new Error('Invalid warn user cell')
  }
  return {
    text: cell.text,
    secondary: cell.secondary,
    mono: cell.mono,
    tone: cell.tone,
  }
}

function warnCountCell(cell: QueueTableCell | undefined): WarnCountCell {
  if (!isCellObject(cell) || !('value' in cell) || typeof cell.value !== 'number') {
    throw new Error('Invalid warn count cell')
  }
  return {
    text: cell.text,
    value: cell.value,
    mono: cell.mono,
    tone: cell.tone,
  }
}

function isCellObject(cell: QueueTableCell | undefined): cell is QueueTableCellObject {
  return typeof cell === 'object' && cell !== null && typeof cell.text === 'string'
}

const canSubmitAdd = computed(() => Boolean(draft.guildId.trim() && draft.userId.trim()))

watch(guildIds, (ids) => {
  if (ids.length === 0) {
    selectedGuildId.value = ''
    return
  }
  applyNavigationState()
  if (!selectedGuildId.value || !filteredGroups.value[selectedGuildId.value]) {
    selectedGuildId.value = ids[0]
  }
})

watch(
  () => props.navigation?.state.value,
  (state) => {
    if (state?.view !== 'warns') return
    applyNavigationState()
  },
)

onMounted(refresh)

async function refresh() {
  const requestSeq = ++refreshRequestSeq
  loading.value = true
  loadError.value = ''
  try {
    const list = await warnsApi.list(fetchNames.value)
    const next: Record<string, ProcessedWarn[]> = {}
    for (const item of list) {
      if (!next[item.guildId]) next[item.guildId] = []
      next[item.guildId].push(item)
    }
    if (requestSeq !== refreshRequestSeq) return
    groups.value = next
    lastSync.value = formatTimestamp(Date.now())
    applyNavigationState()
  } catch (cause) {
    if (requestSeq !== refreshRequestSeq) return
    const details = errorMessage(cause, '加载警告记录失败')
    loadError.value = details
    pushError('加载警告记录失败', details)
  } finally {
    if (requestSeq === refreshRequestSeq) {
      loading.value = false
    }
  }
}

async function reload() {
  reloading.value = true
  clearActionError()
  try {
    await warnsApi.reload()
    pushSuccess('警告数据已重新加载')
    await refresh()
  } catch (cause) {
    setActionError('重新加载失败', cause, '重新加载失败')
  } finally {
    reloading.value = false
  }
}

function openAdd() {
  if (initialLoadBlocked.value) {
    pushError('警告记录尚未加载', '警告记录尚未加载，无法添加警告')
    return
  }
  draft.guildId = ''
  draft.userId = ''
  addOpen.value = true
}

function closeAdd() {
  addOpen.value = false
}

async function submitAdd() {
  if (!canSubmitAdd.value) return
  adding.value = true
  clearActionError()
  const guildIdSnapshot = draft.guildId.trim()
  try {
    await warnsApi.add(guildIdSnapshot, draft.userId.trim())
    pushSuccess(`已在 ${guildIdSnapshot} 添加警告`)
    addOpen.value = false
    draft.guildId = ''
    draft.userId = ''
    await refresh()
    if (groups.value[guildIdSnapshot]) {
      selectedGuildId.value = guildIdSnapshot
    }
  } catch (cause) {
    setActionError('添加失败', cause, '添加失败')
  } finally {
    adding.value = false
  }
}

async function updateCount(key: string, next: number | undefined) {
  if (next === undefined) return
  if (next <= 0) {
    const confirmed = await confirm({
      title: '清除警告记录',
      message: '确定要清除这条警告记录吗？清零后该成员会从当前群组列表中移除。',
      tone: 'danger',
      confirmText: '清除',
    })
    if (!confirmed) {
      await refresh()
      return
    }
  }

  clearActionError()
  try {
    await warnsApi.update(key, next)
    pushSuccess(next <= 0 ? '警告已清除' : '警告次数已更新')
    await refresh()
  } catch (cause) {
    setActionError('更新警告失败', cause, '更新警告失败')
    await refresh()
  }
}

function handleRowAction(payload: { rowId: string; action: string }) {
  if (payload.action !== 'clear') return
  void updateCount(payload.rowId, 0)
}

function guildName(guildId: string): string {
  const first = groups.value[guildId]?.[0]
  if (!first) return guildId
  return first.guildName && first.guildName !== 'Unknown' ? first.guildName : guildId
}

function groupCount(guildId: string): number {
  return filteredGroups.value[guildId]?.length ?? 0
}

function applyNavigationState(): void {
  const state = props.navigation?.state.value
  if (state?.view !== 'warns') return
  guildFilter.value = state.guildId || ''
  keyword.value = state.keyword || ''
  if (state.guildId) {
    selectedGuildId.value = state.guildId
  }
}

interface WarnGroupFilter {
  readonly guildId: string
  readonly keyword: string
}

function filterWarnGroups(
  source: Record<string, ProcessedWarn[]>,
  filter: WarnGroupFilter,
): Record<string, ProcessedWarn[]> {
  const query = filter.keyword.trim().toLowerCase()
  return Object.fromEntries(
    Object.entries(source)
      .filter(([guildId]) => !filter.guildId || guildId === filter.guildId)
      .map(([guildId, list]) => [guildId, list.filter((item) => matchesWarn(item, query))] as const)
      .filter(([, list]) => list.length > 0),
  )
}

function matchesWarn(item: ProcessedWarn, query: string): boolean {
  if (!query) return true
  return [item.userId, item.userName, item.guildId, item.guildName]
    .filter(Boolean)
    .some((value) => String(value).toLowerCase().includes(query))
}

function severityDot(count: number): string {
  if (count >= 10) return 'sh-lane__dot--danger'
  if (count >= 3) return 'sh-lane__dot--warning'
  return 'sh-lane__dot--primary'
}
</script>

<style scoped>
.sh-warns-load-error,
.sh-warns-action-error {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--sh-s-3);
  padding: var(--sh-s-3) var(--sh-s-4);
  border: 1px solid rgba(248, 81, 73, 0.28);
  border-radius: var(--sh-r-2);
  background: rgba(248, 81, 73, 0.08);
}

.sh-warns-action-error--drawer {
  margin-bottom: var(--sh-s-4);
}

.sh-warns-load-error__body,
.sh-warns-action-error__body {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.sh-warns-load-error__body strong,
.sh-warns-action-error__body strong {
  color: #ff8a80;
  font-size: var(--sh-t-body);
}

.sh-warns-load-error__body span,
.sh-warns-action-error__body span {
  color: var(--sh-fg-2);
  font-size: var(--sh-t-meta);
  overflow-wrap: anywhere;
}

.sh-warns__user {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}
</style>
