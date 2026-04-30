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
          @click="openAdd"
        >
          添加警告
        </el-button>
      </template>
    </WorkspaceHead>

    <ConsolePageSkeleton v-if="loading && totalRecords === 0" />
    <EmptyState
      v-else-if="totalRecords === 0"
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
                <EntityChip kind="guild" :id="guildId" inline @click.stop />
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
              :id="(row.cells.user as any).secondary"
              :name="(row.cells.user as any).text"
              :guild-id="selectedGuildId ?? undefined"
            />
          </template>
          <template #cell-count="{ row }">
            <el-input-number
              :model-value="(row.cells.count as any).value"
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

    <Drawer
      :open="addOpen"
      title="添加警告"
      subtitle="guildId · userId"
      @close="closeAdd"
    >
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

import { warnsApi } from '../api'
import { useConfirm } from '../composables/use-confirm'
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

interface ProcessedWarn {
  key: string
  userId: string
  userName: string
  userAvatar: string
  guildId: string
  guildName: string
  guildAvatar: string
  count: number
  timestamp: number
}

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
const notices = ref<NoticeItem[]>([])
const lastSync = ref('')
const groups = ref<Record<string, ProcessedWarn[]>>({})
const draft = reactive({ guildId: '', userId: '' })
const {
  state: confirmDialog,
  confirm,
  accept: acceptConfirm,
  cancel: cancelConfirm,
} = useConfirm()

const guildIds = computed(() => Object.keys(groups.value))
const selectedGroup = computed(() =>
  selectedGuildId.value ? groups.value[selectedGuildId.value] ?? null : null,
)
const totalRecords = computed(() =>
  Object.values(groups.value).reduce((acc, list) => acc + list.length, 0),
)

const rows = computed<QueueTableRow[]>(() => {
  const list = selectedGroup.value ?? []
  return list.map((item) => ({
    id: item.key,
    cells: {
      user: {
        text: item.userName && item.userName !== 'Unknown' ? item.userName : '未知用户',
        secondary: item.userId,
      },
      time: {
        text: item.timestamp ? formatTimestamp(item.timestamp) : '未知',
        mono: true,
      },
      count: { text: String(item.count), value: item.count } as unknown as {
        text: string
        value: number
      },
    },
    actions: [{ key: 'clear', label: '清除', tone: 'danger' }],
  }))
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

const canSubmitAdd = computed(() => Boolean(draft.guildId.trim() && draft.userId.trim()))

watch(guildIds, (ids) => {
  if (ids.length === 0) {
    selectedGuildId.value = ''
    return
  }
  if (!selectedGuildId.value || !groups.value[selectedGuildId.value]) {
    selectedGuildId.value = ids[0]
  }
})

onMounted(refresh)

async function refresh() {
  loading.value = true
  try {
    const list = (await warnsApi.list(fetchNames.value)) as ProcessedWarn[]
    const next: Record<string, ProcessedWarn[]> = {}
    for (const item of list) {
      if (!next[item.guildId]) next[item.guildId] = []
      next[item.guildId].push(item)
    }
    groups.value = next
    lastSync.value = formatTimestamp(Date.now())
  } catch (cause) {
    pushError(cause, '加载警告记录失败')
  } finally {
    loading.value = false
  }
}

async function reload() {
  reloading.value = true
  try {
    await warnsApi.reload()
    pushSuccess('警告数据已重新加载')
    await refresh()
  } catch (cause) {
    pushError(cause, '重新加载失败')
  } finally {
    reloading.value = false
  }
}

function openAdd() {
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
    pushError(cause, '添加失败')
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

  try {
    await warnsApi.update(key, next)
    pushSuccess(next <= 0 ? '警告已清除' : '警告次数已更新')
    await refresh()
  } catch (cause) {
    pushError(cause, '更新警告失败')
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
  return groups.value[guildId]?.length ?? 0
}

function severityDot(count: number): string {
  if (count >= 10) return 'sh-lane__dot--danger'
  if (count >= 3) return 'sh-lane__dot--warning'
  return 'sh-lane__dot--primary'
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

<style scoped>
.sh-warns__user {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}
</style>
