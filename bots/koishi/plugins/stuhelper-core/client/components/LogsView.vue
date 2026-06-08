<template>
  <div class="sh-view">
    <WorkspaceHead
      title="日志检索"
      description="全局命令执行日志。本页只用于检索与原始日志浏览;处置动作请在处置中心完成。"
      :chips="headerChips"
    >
      <template #actions>
        <el-button class="sh-button sh-button--ghost" @click="resetFilters">重置</el-button>
        <el-button
          type="primary"
          class="sh-button sh-button--primary"
          :disabled="loading"
          @click="runSearch"
        >
          {{ loading ? '检索中…' : '检索' }}
        </el-button>
      </template>
    </WorkspaceHead>

    <WorkspaceSection title="过滤条件" description="时间范围与若干关键字段的交集匹配。">
      <label class="sh-field sh-logs__field--time">
        <span class="sh-field__label">时间范围</span>
        <el-date-picker
          v-model="dateRange"
          class="sh-control"
          type="datetimerange"
          range-separator="→"
          start-placeholder="开始时间"
          end-placeholder="结束时间"
          value-format="x"
        />
      </label>
      <div class="sh-form-grid">
        <label class="sh-field">
          <span class="sh-field__label">命令</span>
          <el-input
            v-model="searchParams.command"
            class="sh-control sh-control--mono"
            placeholder="命令名"
            clearable
          />
        </label>
        <label class="sh-field">
          <span class="sh-field__label">用户 ID</span>
          <el-input
            v-model="searchParams.userId"
            class="sh-control sh-control--mono"
            placeholder="1234567890"
            clearable
          />
        </label>
        <label class="sh-field">
          <span class="sh-field__label">用户名</span>
          <el-input
            v-model="searchParams.username"
            class="sh-control"
            placeholder="昵称"
            clearable
          />
        </label>
        <label class="sh-field">
          <span class="sh-field__label">群组 ID</span>
          <el-input
            v-model="searchParams.guildId"
            class="sh-control sh-control--mono"
            placeholder="12345678"
            clearable
          />
        </label>
        <label class="sh-field sh-logs__field--wide">
          <span class="sh-field__label">详情关键字</span>
          <el-input
            v-model="searchParams.details"
            class="sh-control"
            placeholder="错误信息、参数或结果"
            clearable
            @keyup.enter="runSearch"
          />
        </label>
      </div>
    </WorkspaceSection>

    <ConsolePageSkeleton v-if="loading && logs.length === 0" />

    <WorkspaceSection
      v-else
      title="检索结果"
      :description="logs.length ? '点击日志正文区域在右侧 Drawer 查看原始执行详情；用户与群组标识可打开实体概览。' : '调整上方过滤条件后点击检索。'"
      :meta="total > 0 ? `${total.toLocaleString()} 条` : ''"
      flush
    >
      <div v-if="searchError" class="sh-logs__error" role="alert">
        <div class="sh-logs__error-body">
          <strong>加载日志失败</strong>
          <span>{{ searchError }}</span>
        </div>
        <el-button
          class="sh-button sh-button--ghost"
          size="small"
          :disabled="loading"
          @click="runSearch"
        >
          重试
        </el-button>
      </div>
      <EmptyState
        v-if="logs.length === 0 && !loading && !searchError"
        title="暂无匹配记录"
        body="这段时间没有满足过滤条件的命令执行记录。"
      />
      <template v-else>
        <div class="sh-table-shell">
          <el-table
            :data="logs"
            row-key="id"
            class="sh-grid-table"
            @row-click="openDetail"
          >
            <el-table-column label="时间" prop="timestamp" width="160">
              <template #default="{ row }">
                <span class="sh-mono">{{ formatTime(row.timestamp) }}</span>
              </template>
            </el-table-column>
            <el-table-column label="命令" prop="command" width="140">
              <template #default="{ row }">
                <code class="sh-logs__cmd">{{ row.command }}</code>
              </template>
            </el-table-column>
            <el-table-column label="用户" width="170">
              <template #default="{ row }">
                <EntityChip
                  kind="user"
                  :id="String(row.userId ?? '')"
                  :name="row.username || undefined"
                  :guild-id="row.guildId || undefined"
                />
              </template>
            </el-table-column>
            <el-table-column label="群组" width="140">
              <template #default="{ row }">
                <EntityChip
                  v-if="row.guildId"
                  kind="guild"
                  :id="String(row.guildId)"
                  :name="row.guildName || undefined"
                />
                <span v-else class="sh-logs__private">私聊</span>
              </template>
            </el-table-column>
            <el-table-column label="状态" width="92">
              <template #default="{ row }">
                <SeverityTag
                  :label="row.success ? '成功' : '失败'"
                  :intent="row.success ? 'success' : 'danger'"
                />
              </template>
            </el-table-column>
            <el-table-column label="详情" min-width="240">
              <template #default="{ row }">
                <span class="sh-logs__detail" :title="getDetail(row)">
                  {{ getDetail(row) }}
                </span>
              </template>
            </el-table-column>
          </el-table>
        </div>

        <div class="sh-logs__pagination">
          <el-pagination
            v-model:current-page="searchParams.page"
            v-model:page-size="searchParams.pageSize"
            :total="total"
            :page-sizes="[20, 50, 100]"
            layout="total, sizes, prev, pager, next"
            size="small"
            @size-change="runSearch"
            @current-change="runSearch"
          />
        </div>
      </template>
    </WorkspaceSection>

    <Drawer
      :open="Boolean(detailLog)"
      :title="detailLog?.command ?? ''"
      :subtitle="detailLog ? formatTime(detailLog.timestamp) : ''"
      @close="closeDetail"
    >
      <template v-if="detailLog">
        <section class="sh-drawer__section">
          <h4 class="sh-drawer__section-title">基本信息</h4>
          <dl class="sh-keylist">
            <dt>状态</dt>
            <dd>
              <SeverityTag
                :label="detailLog.success ? '成功' : '失败'"
                :intent="detailLog.success ? 'success' : 'danger'"
              />
            </dd>
            <dt>时间</dt>
            <dd class="sh-mono">{{ formatTime(detailLog.timestamp) }}</dd>
            <dt>用户</dt>
            <dd>{{ detailLog.username || '—' }}</dd>
            <dt>用户 ID</dt>
            <dd class="sh-mono">{{ detailLog.userId }}</dd>
            <template v-if="detailLog.guildId">
              <dt>群组</dt>
              <dd>{{ detailLog.guildName || detailLog.guildId }}</dd>
              <dt>群组 ID</dt>
              <dd class="sh-mono">{{ detailLog.guildId }}</dd>
            </template>
            <dt>平台</dt>
            <dd class="sh-mono">{{ detailLog.platform }}</dd>
            <dt>执行耗时</dt>
            <dd class="sh-mono">{{ detailLog.executionTime ?? 0 }} ms</dd>
          </dl>
        </section>

        <section v-if="detailLog.args && detailLog.args.length > 0" class="sh-drawer__section">
          <h4 class="sh-drawer__section-title">命令参数</h4>
          <pre class="sh-logs__code">{{ detailLog.args.join(' ') }}</pre>
        </section>

        <section
          v-if="detailLog.options && Object.keys(detailLog.options).length > 0"
          class="sh-drawer__section"
        >
          <h4 class="sh-drawer__section-title">命令选项</h4>
          <pre class="sh-logs__code">{{ JSON.stringify(detailLog.options, null, 2) }}</pre>
        </section>

        <section v-if="detailLog.result" class="sh-drawer__section">
          <h4 class="sh-drawer__section-title">执行结果</h4>
          <pre class="sh-logs__code">{{ detailLog.result }}</pre>
        </section>

        <section v-if="!detailLog.success && detailLog.error" class="sh-drawer__section">
          <h4 class="sh-drawer__section-title">错误信息</h4>
          <pre class="sh-logs__code sh-logs__code--error">{{ detailLog.error }}</pre>
        </section>
      </template>
    </Drawer>

    <NoticeStack :items="notices" @dismiss="dismissNotice" />
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'

import { logsApi } from '../api'
import { useActionFeedback } from '../composables/use-action-feedback'
import type { ConsoleNavigationController } from '../composables/use-console-navigation'
import type { LogRecord, LogSearchParams } from '../types'
import { formatTimestamp } from '../models/formatters'
import ConsolePageSkeleton from './primitives/ConsolePageSkeleton.vue'
import Drawer from './primitives/Drawer.vue'
import EmptyState from './primitives/EmptyState.vue'
import EntityChip from './primitives/EntityChip.vue'
import NoticeStack from './primitives/NoticeStack.vue'
import SeverityTag from './primitives/SeverityTag.vue'
import WorkspaceHead, { type WorkspaceHeadChip } from './primitives/WorkspaceHead.vue'
import WorkspaceSection from './primitives/WorkspaceSection.vue'

const props = defineProps<{
  navigation?: ConsoleNavigationController
}>()

const loading = ref(false)
const logs = ref<LogRecord[]>([])
const total = ref(0)
const searchError = ref('')
const dateRange = ref<[number, number] | null>(null)
const detailLog = ref<LogRecord | null>(null)
const {
  notices,
  pushError,
  dismissNotice,
  errorMessage,
} = useActionFeedback()

const searchParams = reactive<LogSearchParams>({
  page: 1,
  pageSize: 20,
})
let searchRequestSeq = 0

const headerChips = computed<WorkspaceHeadChip[]>(() => {
  const chips: WorkspaceHeadChip[] = []
  if (total.value > 0) {
    chips.push({ text: `${total.value.toLocaleString()} 条`, numeric: true })
  }
  if (searchParams.page && searchParams.pageSize) {
    chips.push({
      text: `第 ${searchParams.page} 页 · ${searchParams.pageSize} 条/页`,
      mono: true,
    })
  }
  return chips
})

watch(
  () => props.navigation?.state.value,
  (state) => {
    if (state?.view !== 'logs') return
    if (applyNavigationState()) void runSearch()
  },
)

onMounted(() => {
  applyNavigationState()
  void runSearch()
})

function applyNavigationState(): boolean {
  const state = props.navigation?.state.value
  if (!state || state.view !== 'logs') return false
  const nextGuildId = state.guildId || undefined
  const nextUserId = state.keyword || state.memberId || undefined
  const guildChanged = updateSearchParam('guildId', nextGuildId)
  const userChanged = updateSearchParam('userId', nextUserId)
  return guildChanged || userChanged
}

function updateSearchParam<K extends keyof LogSearchParams>(
  key: K,
  value: LogSearchParams[K] | undefined,
): boolean {
  if (searchParams[key] === value) return false
  searchParams[key] = value as LogSearchParams[K]
  searchParams.page = 1
  return true
}

async function runSearch() {
  const requestSeq = ++searchRequestSeq
  loading.value = true
  searchError.value = ''
  try {
    const params: LogSearchParams = { ...searchParams }
    if (dateRange.value) {
      params.startTime = dateRange.value[0]
      params.endTime = dateRange.value[1]
    }
    const result = await logsApi.search(params)
    if (requestSeq !== searchRequestSeq) return
    logs.value = result.list
    total.value = result.total
    searchError.value = ''
  } catch (cause) {
    if (requestSeq !== searchRequestSeq) return
    const message = errorMessage(cause, '加载日志失败')
    searchError.value = message
    pushError('加载日志失败', message)
  } finally {
    if (requestSeq === searchRequestSeq) {
      loading.value = false
    }
  }
}

function resetFilters() {
  dateRange.value = null
  searchParams.command = undefined
  searchParams.userId = undefined
  searchParams.username = undefined
  searchParams.guildId = undefined
  searchParams.details = undefined
  searchParams.page = 1
  props.navigation?.replaceState({
    guildId: null,
    memberId: null,
    itemId: null,
    keyword: '',
  })
  void runSearch()
}

function getDetail(log: LogRecord): string {
  if (!log.success && log.error) return log.error
  if (log.result) return log.result
  if (log.options && Object.keys(log.options).length > 0) {
    return JSON.stringify(log.options)
  }
  if (log.args && log.args.length > 0) return log.args.join(' ')
  return '—'
}

function formatTime(timestamp: string | number | undefined): string {
  return formatTimestamp(timestamp)
}

function openDetail(row: LogRecord) {
  detailLog.value = row
}

function closeDetail() {
  detailLog.value = null
}
</script>

<style scoped>
.sh-logs__field--time {
  min-width: min(100%, 420px);
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.sh-logs__field--wide {
  grid-column: 1 / -1;
}

.sh-logs__cmd {
  display: inline-flex;
  align-items: center;
  min-height: 22px;
  padding: 0 8px;
  border-radius: var(--sh-r-1);
  background: var(--sh-primary-soft);
  color: var(--sh-primary);
  font-family: var(--sh-font-mono);
  font-size: var(--sh-t-meta);
  font-weight: var(--sh-w-medium);
}

.sh-logs__private {
  color: var(--sh-fg-3);
  font-size: var(--sh-t-meta);
}

.sh-logs__error {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: var(--sh-s-3);
  margin: 0 0 var(--sh-s-3);
  padding: var(--sh-s-3);
  border: 1px solid color-mix(in srgb, var(--sh-danger) 38%, transparent);
  border-radius: var(--sh-radius-md);
  background: color-mix(in srgb, var(--sh-danger) 10%, transparent);
  color: var(--sh-danger);
}

.sh-logs__error-body {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.sh-logs__detail {
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
  color: var(--sh-fg-2);
  font-size: var(--sh-t-meta);
  line-height: var(--sh-l-normal);
}

.sh-logs__user {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.sh-logs__pagination {
  display: flex;
  justify-content: flex-end;
  padding: var(--sh-s-3) var(--sh-s-5);
  border-top: 1px solid var(--sh-border);
}

.sh-logs__code {
  margin: 0;
  padding: var(--sh-s-3);
  border: 1px solid var(--sh-border);
  border-radius: var(--sh-r-2);
  background: var(--sh-surface-1);
  color: var(--sh-fg-1);
  font-family: var(--sh-font-mono);
  font-size: var(--sh-t-meta);
  line-height: var(--sh-l-normal);
  white-space: pre-wrap;
  word-break: break-word;
  overflow-x: auto;
}

.sh-logs__code--error {
  background: var(--sh-danger-soft);
  border-color: color-mix(in oklch, var(--sh-danger) 28%, var(--sh-border));
  color: var(--sh-danger);
}

:deep(.sh-grid-table .el-table__row) {
  cursor: pointer;
}
</style>
