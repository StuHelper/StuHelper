<template>
  <div class="sh-view">
    <header class="sh-workspace-head">
      <div class="sh-workspace-head__copy">
        <h1 class="sh-workspace-head__title">处置中心</h1>
        <p class="sh-workspace-head__description">
          统一工作项列表 — 复核、准入、举报三类入口并列展示。高风险动作只在右侧处置面板执行。
        </p>
        <div class="sh-workspace-head__meta" v-if="data">
          <span class="sh-meta-chip sh-mono">
            更新 · {{ formatTimestamp(data.generatedAt) }}
          </span>
        </div>
      </div>
      <div class="sh-workspace-head__actions">
        <el-button
          class="sh-button sh-button--ghost"
          :disabled="loading"
          @click="loadData"
        >
          {{ loading ? '刷新中…' : '刷新' }}
        </el-button>
      </div>
    </header>

    <EmptyState
      v-if="error"
      title="加载失败"
      :body="error"
      tone="error"
    />
    <ConsolePageSkeleton v-else-if="loading && !data" />

    <template v-else-if="data">
      <section class="sh-dashboard-metrics">
        <article
          v-for="(metric, index) in metrics"
          :key="metric.label"
          class="sh-stat sh-dashboard-metric"
          :class="metricIntent(index, metric.value)"
        >
          <span class="sh-stat__label">{{ metric.label }}</span>
          <span class="sh-stat__value sh-num">{{ metric.value }}</span>
          <span class="sh-stat__note">{{ metric.note }}</span>
        </article>
      </section>

      <section class="sh-toolbar sh-review-toolbar">
        <label class="sh-field sh-review-toolbar__field">
          <span class="sh-field__label">类型</span>
          <el-select
            v-model="kindFilter"
            class="sh-control"
            clearable
            placeholder="全部类型"
            @change="syncSelection()"
          >
            <el-option label="全部类型" value="" />
            <el-option label="复核" value="review" />
            <el-option label="准入" value="admission" />
            <el-option label="举报" value="report" />
          </el-select>
        </label>
        <label class="sh-field sh-review-toolbar__field sh-review-toolbar__field--wide">
          <span class="sh-field__label">检索</span>
          <el-input
            v-model.trim="keyword"
            class="sh-control"
            placeholder="成员、群号、原因"
            @input="syncSelection()"
          />
        </label>
      </section>

      <div class="sh-split sh-split--7-5">
        <WorkspaceSection
          title="工作项列表"
          description="复核、准入、举报统一展示,按严重度排序。"
          :meta="`${filteredItems.length} 条`"
          flush
        >
          <EmptyState
            v-if="filteredItems.length === 0"
            title="没有匹配的工作项"
            body="清除过滤条件或等待新任务进入队列。"
          />
          <div v-else class="sh-lane">
            <button
              v-for="item in filteredItems"
              :key="item.id"
              type="button"
              class="sh-lane__row sh-lane__row--interactive"
              :class="{ 'sh-lane__row--active': selectedItem?.id === item.id }"
              @click="selectItem(item.id)"
            >
              <span class="sh-lane__dot" :class="kindDotClass(item.kind)"></span>
              <div class="sh-lane__body">
                <div class="sh-lane__title">{{ item.subjectLabel }}</div>
                <div class="sh-lane__subtitle">
                  <SeverityTag
                    :label="REVIEW_KIND_LABELS[item.kind]"
                    :intent="kindIntent(item.kind)"
                  />
                  <span class="sh-mono">{{ item.status }}</span>
                  <span>· {{ item.reason }}</span>
                </div>
              </div>
              <span class="sh-lane__time sh-mono">
                {{ formatTimestamp(item.createdAt) }}
              </span>
            </button>
          </div>
        </WorkspaceSection>

        <WorkspaceSection
          title="详情与处置"
          description="执行或驳回前,先确认上下文。"
        >
          <EmptyState
            v-if="!selectedItem"
            title="请选择左侧工作项"
            body="选中后这里会展示可用动作、原因与关联事件。"
          />
          <template v-else>
            <dl class="sh-keylist">
              <dt>对象</dt>
              <dd>{{ selectedItem.subjectLabel }}</dd>
              <dt>类型</dt>
              <dd>{{ REVIEW_KIND_LABELS[selectedItem.kind] }}</dd>
              <dt>优先级</dt>
              <dd>{{ selectedItem.priority }}</dd>
              <dt>创建时间</dt>
              <dd class="sh-mono">{{ formatTimestamp(selectedItem.createdAt) }}</dd>
              <dt>原因</dt>
              <dd>{{ selectedItem.reason || '—' }}</dd>
              <template v-if="selectedItem.guildId">
                <dt>来源群</dt>
                <dd>
                  <EntityChip kind="guild" :id="String(selectedItem.guildId)" />
                </dd>
              </template>
              <template v-if="selectedItem.memberId">
                <dt>对象成员</dt>
                <dd>
                  <EntityChip
                    kind="user"
                    :id="String(selectedItem.memberId)"
                    :guild-id="selectedItem.guildId || undefined"
                  />
                </dd>
              </template>
              <dt>关联事件</dt>
              <dd>
                {{ relatedEvents.map((item) => item.summary).join(' / ') || '暂无' }}
              </dd>
            </dl>

            <div class="sh-field">
              <span class="sh-field__label">处理备注(可选)</span>
              <el-input
                v-model.trim="reviewNote"
                class="sh-control"
                placeholder="记录这次决策依据"
              />
            </div>

            <div class="sh-btn-row sh-review__actions">
              <el-button
                v-for="(action, actionIndex) in selectedItem.availableActions"
                :key="action"
                class="sh-button"
                :class="actionButtonClass(actionIndex, action)"
                :type="actionIndex === 0 ? 'primary' : undefined"
                :disabled="actionLoading"
                @click="submitAction(action)"
              >
                {{ REVIEW_ACTION_LABELS[action] }}
              </el-button>
            </div>
          </template>
        </WorkspaceSection>
      </div>
    </template>

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
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'

import { useConfirm, type ConfirmTone } from '../composables/use-confirm'
import type { ConsoleNavigationController } from '../composables/use-console-navigation'
import { consolePageApi } from '../page-api'
import type { ReviewPageData, ReviewWorkItem } from '../page-types'
import { formatTimestamp } from '../models/formatters'
import { buildReviewModel, REVIEW_ACTION_LABELS, REVIEW_KIND_LABELS } from '../models/review'
import ConfirmDialog from './primitives/ConfirmDialog.vue'
import ConsolePageSkeleton from './primitives/ConsolePageSkeleton.vue'
import EmptyState from './primitives/EmptyState.vue'
import EntityChip from './primitives/EntityChip.vue'
import SeverityTag, { type TagIntent } from './primitives/SeverityTag.vue'
import WorkspaceSection from './primitives/WorkspaceSection.vue'

const props = defineProps<{
  navigation?: ConsoleNavigationController
}>()

const loading = ref(false)
const error = ref('')
const data = ref<ReviewPageData | null>(null)
const kindFilter = ref('')
const keyword = ref('')
const selectedItemId = ref('')
const reviewNote = ref('')
const actionLoading = ref(false)
const {
  state: confirmDialog,
  confirm,
  accept: acceptConfirm,
  cancel: cancelConfirm,
} = useConfirm()

const model = computed(() => {
  if (!data.value) return null
  return buildReviewModel(data.value, {
    workspace: kindFilter.value,
    keyword: keyword.value,
    itemId: selectedItemId.value,
  })
})
const metrics = computed(() => model.value?.metrics ?? [])
const filteredItems = computed(() => model.value?.filteredItems ?? [])
const selectedItem = computed(() => model.value?.selectedItem ?? null)
const relatedEvents = computed(() => model.value?.relatedEvents ?? [])

loadData()

async function loadData() {
  loading.value = true
  error.value = ''
  try {
    data.value = await consolePageApi.review()
    kindFilter.value = props.navigation?.state.value.workspace || ''
    keyword.value = props.navigation?.state.value.keyword || ''
    selectedItemId.value = props.navigation?.state.value.itemId || data.value.items[0]?.id || ''
    syncSelection()
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : String(cause)
  } finally {
    loading.value = false
  }
}

function selectItem(itemId: string) {
  selectedItemId.value = itemId
  syncSelection()
}

function syncSelection() {
  const item = filteredItems.value.find((entry) => entry.id === selectedItemId.value) ?? filteredItems.value[0] ?? null
  if (!item) {
    props.navigation?.replaceState({
      workspace: kindFilter.value || null,
      guildId: null,
      memberId: null,
      itemId: null,
      keyword: keyword.value,
    })
    return
  }
  selectedItemId.value = item.id
  props.navigation?.replaceState({
    workspace: item.kind,
    guildId: item.guildId,
    memberId: item.memberId,
    itemId: item.id,
    keyword: keyword.value,
  })
}

async function submitAction(action: ReviewWorkItem['availableActions'][number]) {
  const item = selectedItem.value
  if (!item) return
  const label = REVIEW_ACTION_LABELS[action]
  const confirmed = await confirm({
    title: '确认处置',
    message: `确定要对 ${item.subjectLabel} 执行「${label}」吗？`,
    tone: actionConfirmTone(action),
    confirmText: label,
  })
  if (!confirmed) return

  actionLoading.value = true
  error.value = ''
  try {
    await consolePageApi.workItemAction({
      kind: item.kind,
      itemId: item.id,
      action,
      note: reviewNote.value || undefined,
    })
    reviewNote.value = ''
    await loadData()
    syncSelection()
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : String(cause)
  } finally {
    actionLoading.value = false
  }
}

function metricIntent(index: number, value: number): string {
  if (index === 0 && value > 0) return 'sh-stat--primary'
  if (index === 1 && value > 5) return 'sh-stat--warning'
  if (index === 2 && value > 10) return 'sh-stat--warning'
  if (index === 3 && value > 0) return 'sh-stat--danger'
  return ''
}

function kindDotClass(kind: ReviewWorkItem['kind']): string {
  if (kind === 'review') return 'sh-lane__dot--primary'
  if (kind === 'admission') return 'sh-lane__dot--warning'
  return 'sh-lane__dot--danger'
}

function kindIntent(kind: ReviewWorkItem['kind']): TagIntent {
  if (kind === 'review') return 'primary'
  if (kind === 'admission') return 'warning'
  return 'danger'
}

function actionButtonClass(index: number, action: string): string {
  if (action === 'reject' || action === 'deny' || action === 'dismiss') {
    return 'sh-button--danger'
  }
  if (index === 0) return 'sh-button--primary'
  return 'sh-button--ghost'
}

function actionConfirmTone(action: string): ConfirmTone {
  if (action === 'reject' || action === 'deny' || action === 'dismiss') {
    return 'danger'
  }
  return 'normal'
}
</script>

<style scoped>
.sh-review-toolbar {
  gap: var(--sh-s-3);
  padding: 0;
}

.sh-review-toolbar__field {
  min-width: 180px;
  flex: 0 1 220px;
}

.sh-review-toolbar__field--wide {
  flex-basis: 320px;
}

.sh-review__actions {
  justify-content: flex-end;
}

.sh-lane__body {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
  text-align: left;
}

.sh-lane__subtitle {
  display: inline-flex;
  flex-wrap: wrap;
  gap: 6px;
  align-items: center;
  font-size: var(--sh-t-meta);
  color: var(--sh-fg-2);
}
</style>
