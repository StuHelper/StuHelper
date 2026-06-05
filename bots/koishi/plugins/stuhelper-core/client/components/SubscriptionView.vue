<template>
  <div class="sh-view">
    <WorkspaceHead
      title="订阅管理"
      description="管理将群管事件推送到的目标。每个订阅可独立选择推送的事件类型。"
      :chips="headerChips"
    >
      <template #actions>
        <label class="sh-warns__toggle">
          <span class="sh-warns__toggle-label">解析名称</span>
          <el-switch v-model="fetchNames" size="small" @change="refresh" />
        </label>
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
          @click="openCreate"
        >
          添加订阅
        </el-button>
      </template>
    </WorkspaceHead>

    <ConsolePageSkeleton v-if="loading && subscriptions.length === 0" />
    <EmptyState
      v-else-if="loadError && subscriptions.length === 0"
      tone="error"
      title="加载订阅失败"
      :body="loadError"
    >
      <template #action>
        <el-button class="sh-button sh-button--ghost" @click="refresh">重试</el-button>
      </template>
    </EmptyState>
    <template v-else>
      <div
        v-if="loadError && subscriptions.length > 0"
        class="sh-sub-load-error"
        role="alert"
      >
        <div class="sh-sub-load-error__body">
          <strong>刷新订阅失败</strong>
          <span>{{ loadError }}</span>
        </div>
        <el-button class="sh-button sh-button--ghost" @click="refresh">重试</el-button>
      </div>

      <div
        v-if="actionError"
        class="sh-sub-action-error"
        role="alert"
      >
        <div class="sh-sub-action-error__body">
          <strong>{{ actionErrorTitle }}</strong>
          <span>{{ actionError }}</span>
        </div>
        <el-button class="sh-button sh-button--ghost" @click="clearActionError">关闭</el-button>
      </div>

      <EmptyState
        v-if="subscriptions.length === 0"
        title="暂无订阅"
        body="创建订阅后,群管事件会推送到指定的群或私聊会话。"
      />
      <WorkspaceSection
        v-else
        title="订阅目标"
        description="点击任意卡片进入右侧编辑面板。"
        :meta="`${subscriptions.length} 条`"
      >
        <div class="sh-sub-grid">
          <button
            v-for="(sub, index) in subscriptions"
            :key="`${sub.type}-${sub.id}-${index}`"
            type="button"
            class="sh-sub-card"
            :class="{ 'sh-sub-card--active': editingIndex === index }"
            @click="openEdit(sub, index)"
          >
            <div class="sh-sub-card__head">
              <div class="sh-sub-card__identity">
                <span class="sh-sub-card__title">{{ sub.name || sub.id }}</span>
                <span v-if="sub.name" class="sh-sub-card__id sh-mono">#{{ sub.id }}</span>
              </div>
              <SeverityTag
                :label="sub.type === 'group' ? '群' : '私聊'"
                :intent="sub.type === 'group' ? 'primary' : 'info'"
              />
            </div>
            <ul class="sh-sub-card__features">
              <li
                v-for="feature in FEATURES"
                :key="feature.key"
                class="sh-sub-card__feature"
                :data-active="Boolean(sub.features[feature.key])"
              >
                <span class="sh-sub-card__dot" aria-hidden="true"></span>
                {{ feature.label }}
              </li>
            </ul>
          </button>
        </div>
      </WorkspaceSection>
    </template>

    <Drawer
      :open="formOpen"
      :title="isEdit ? '编辑订阅' : '添加订阅'"
      :subtitle="isEdit ? `#${draft.id}` : '一个订阅对应一个推送目标'"
      @close="closeForm"
      @closed="handleFormClosed"
    >
      <div
        v-if="actionError"
        class="sh-sub-action-error sh-sub-action-error--drawer"
        role="alert"
      >
        <div class="sh-sub-action-error__body">
          <strong>{{ actionErrorTitle }}</strong>
          <span>{{ actionError }}</span>
        </div>
        <el-button class="sh-button sh-button--ghost" @click="clearActionError">关闭</el-button>
      </div>

      <section class="sh-drawer__section">
        <h4 class="sh-drawer__section-title">基本信息</h4>
        <label class="sh-field">
          <span class="sh-field__label">类型</span>
          <el-radio-group v-model="draft.type" class="sh-sub-radio">
            <el-radio-button label="group">群聊</el-radio-button>
            <el-radio-button label="private">私聊</el-radio-button>
          </el-radio-group>
        </label>
        <label class="sh-field">
          <span class="sh-field__label">目标 ID</span>
          <el-input
            v-model.trim="draft.id"
            class="sh-control sh-control--mono"
            placeholder="例如 123456789"
            :disabled="isEdit"
          />
          <span v-if="isEdit" class="sh-field__hint">
            编辑模式下无法修改目标 ID,请删除后重新添加。
          </span>
        </label>
      </section>

      <section class="sh-drawer__section">
        <h4 class="sh-drawer__section-title">推送事件</h4>
        <div class="sh-sub-features">
          <label
            v-for="feature in FEATURES"
            :key="feature.key"
            class="sh-sub-feature"
          >
            <el-checkbox
              :model-value="Boolean(draft.features[feature.key])"
              class="sh-check"
              @update:model-value="(val: boolean) => toggleFeature(feature.key, val)"
            >
              {{ feature.label }}
            </el-checkbox>
            <span class="sh-sub-feature__hint">{{ feature.hint }}</span>
          </label>
        </div>
      </section>

      <template #footer>
        <el-popconfirm
          v-if="isEdit"
          :title="`确认删除订阅 #${draft.id}?`"
          confirm-button-text="删除"
          cancel-button-text="取消"
          @confirm="confirmRemove"
        >
          <template #reference>
            <el-button class="sh-button sh-button--danger">删除订阅</el-button>
          </template>
        </el-popconfirm>
        <span v-else class="sh-drawer__foot-spacer"></span>

        <el-button class="sh-button sh-button--ghost" @click="closeForm">取消</el-button>
        <el-button
          type="primary"
          class="sh-button sh-button--primary"
          :disabled="!canSave || saving"
          @click="save"
        >
          {{ saving ? '保存中…' : isEdit ? '保存' : '添加' }}
        </el-button>
      </template>
    </Drawer>

    <NoticeStack :items="notices" @dismiss="dismissNotice" />
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'

import { subscriptionApi } from '../api'
import type { Subscription } from '../types'
import { formatTimestamp } from '../models/formatters'
import ConsolePageSkeleton from './primitives/ConsolePageSkeleton.vue'
import Drawer from './primitives/Drawer.vue'
import EmptyState from './primitives/EmptyState.vue'
import NoticeStack, { type NoticeItem } from './primitives/NoticeStack.vue'
import SeverityTag from './primitives/SeverityTag.vue'
import WorkspaceHead, { type WorkspaceHeadChip } from './primitives/WorkspaceHead.vue'
import WorkspaceSection from './primitives/WorkspaceSection.vue'

type FeatureKey = keyof Required<Subscription['features']>

interface FeatureDef {
  key: FeatureKey
  label: string
  hint: string
}

const FEATURES: readonly FeatureDef[] = [
  { key: 'log', label: '日志推送', hint: '命令执行日志会推送到目标。' },
  { key: 'warning', label: '警告通知', hint: '新增 / 清除警告时通知目标。' },
  { key: 'blacklist', label: '黑名单', hint: '黑名单变动立刻同步。' },
  { key: 'muteExpire', label: '禁言解除', hint: '成员禁言到期时通知。' },
  { key: 'memberChange', label: '成员变动', hint: '入群 / 退群 / 踢出事件。' },
  { key: 'antiRecall', label: '防撤回', hint: '命中防撤回规则时通知。' },
]

const FEATURE_DEFAULTS: Required<Subscription['features']> = {
  log: true,
  memberChange: false,
  muteExpire: false,
  blacklist: true,
  warning: true,
  antiRecall: false,
}

const loading = ref(false)
const saving = ref(false)
const fetchNames = ref(true)
const formOpen = ref(false)
const subscriptions = ref<Subscription[]>([])
const notices = ref<NoticeItem[]>([])
const editingIndex = ref<number>(-1)
const lastSync = ref('')
const loadError = ref('')
const actionError = ref('')
const actionErrorTitle = ref('操作失败')
let refreshRequestSeq = 0
const draft = reactive<Subscription>({
  type: 'group',
  id: '',
  features: { ...FEATURE_DEFAULTS },
})

const isEdit = computed(() => editingIndex.value >= 0)
const canSave = computed(() => Boolean(draft.id.trim()))
const initialLoadBlocked = computed(() => Boolean(loadError.value && subscriptions.value.length === 0))

const headerChips = computed<WorkspaceHeadChip[]>(() => {
  const chips: WorkspaceHeadChip[] = [
    { text: `${subscriptions.value.length} 条订阅`, numeric: true },
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
    const next = await subscriptionApi.list(fetchNames.value)
    if (requestSeq !== refreshRequestSeq) return
    subscriptions.value = next
    lastSync.value = formatTimestamp(Date.now())
  } catch (cause) {
    if (requestSeq !== refreshRequestSeq) return
    const details = errorMessage(cause, '加载订阅失败')
    loadError.value = details
    pushError('加载订阅失败', details)
  } finally {
    if (requestSeq === refreshRequestSeq) {
      loading.value = false
    }
  }
}

function openCreate() {
  if (initialLoadBlocked.value) {
    pushError('订阅尚未加载', '订阅尚未加载，无法添加订阅')
    return
  }
  resetDraft()
  formOpen.value = true
}

function openEdit(sub: Subscription, index: number) {
  editingIndex.value = index
  draft.type = sub.type
  draft.id = sub.id
  draft.features = { ...FEATURE_DEFAULTS, ...sub.features }
  formOpen.value = true
}

function closeForm() {
  formOpen.value = false
}

function handleFormClosed() {
  resetDraft()
}

function resetDraft() {
  editingIndex.value = -1
  draft.type = 'group'
  draft.id = ''
  draft.features = { ...FEATURE_DEFAULTS }
}

function toggleFeature(key: FeatureKey, value: boolean) {
  draft.features = { ...draft.features, [key]: value }
}

async function save() {
  if (!canSave.value || saving.value) return
  saving.value = true
  clearActionError()
  try {
    const payload: Subscription = {
      type: draft.type,
      id: draft.id.trim(),
      features: { ...draft.features },
    }
    if (isEdit.value) {
      await subscriptionApi.update(editingIndex.value, payload)
      pushSuccess('订阅已更新')
    } else {
      await subscriptionApi.add(payload)
      pushSuccess('订阅已添加')
    }
    formOpen.value = false
    await refresh()
  } catch (cause) {
    setActionError(isEdit.value ? '更新失败' : '添加失败', cause, isEdit.value ? '更新失败' : '添加失败')
  } finally {
    saving.value = false
  }
}

async function confirmRemove() {
  if (editingIndex.value < 0) return
  clearActionError()
  try {
    await subscriptionApi.remove(editingIndex.value)
    pushSuccess('订阅已删除')
    formOpen.value = false
    await refresh()
  } catch (cause) {
    setActionError('删除失败', cause, '删除失败')
  }
}

function pushSuccess(message: string) {
  notices.value.push({ id: noticeId(), kind: 'success', message })
  scheduleDismiss()
}

function pushError(title: string, message: string) {
  notices.value.push({ id: noticeId(), kind: 'error', title, message })
  scheduleDismiss()
}

function setActionError(title: string, cause: unknown, fallback: string) {
  const message = errorMessage(cause, fallback)
  actionErrorTitle.value = title
  actionError.value = message
  notices.value.push({ id: noticeId(), kind: 'error', title, message })
  scheduleDismiss()
}

function clearActionError() {
  actionErrorTitle.value = '操作失败'
  actionError.value = ''
}

function errorMessage(cause: unknown, fallback: string): string {
  if (cause instanceof Error && cause.message) return cause.message
  if (typeof cause === 'string' && cause.trim()) return cause
  return fallback
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
.sh-sub-load-error,
.sh-sub-action-error {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--sh-s-3);
  padding: var(--sh-s-3) var(--sh-s-4);
  border: 1px solid rgba(248, 81, 73, 0.28);
  border-radius: var(--sh-r-2);
  background: rgba(248, 81, 73, 0.08);
}

.sh-sub-action-error--drawer {
  margin-bottom: var(--sh-s-4);
}

.sh-sub-load-error__body,
.sh-sub-action-error__body {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.sh-sub-load-error__body strong,
.sh-sub-action-error__body strong {
  color: #ff8a80;
  font-size: var(--sh-t-body);
}

.sh-sub-load-error__body span,
.sh-sub-action-error__body span {
  color: var(--sh-fg-2);
  font-size: var(--sh-t-meta);
  overflow-wrap: anywhere;
}

.sh-sub-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: var(--sh-s-3);
}

.sh-sub-card {
  display: flex;
  flex-direction: column;
  gap: var(--sh-s-3);
  padding: var(--sh-s-4);
  border: 1px solid var(--sh-border);
  border-radius: var(--sh-r-3);
  background: var(--sh-surface-0);
  color: var(--sh-fg);
  cursor: pointer;
  text-align: left;
  transition:
    background var(--sh-dur-fast) var(--sh-ease),
    border-color var(--sh-dur-fast) var(--sh-ease),
    box-shadow var(--sh-dur-fast) var(--sh-ease);
}

.sh-sub-card:hover {
  border-color: var(--sh-border-strong);
  background: var(--sh-surface-hover);
}

.sh-sub-card--active {
  border-color: var(--sh-primary);
  box-shadow: 0 0 0 3px var(--sh-primary-soft);
}

.sh-sub-card__head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: var(--sh-s-2);
}

.sh-sub-card__identity {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.sh-sub-card__title {
  font-size: var(--sh-t-body-lg);
  font-weight: var(--sh-w-semibold);
  color: var(--sh-fg);
  line-height: 1.2;
}

.sh-sub-card__id {
  font-size: var(--sh-t-meta);
  color: var(--sh-fg-2);
}

.sh-sub-card__features {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 6px var(--sh-s-3);
  margin: 0;
  padding: 0;
  list-style: none;
}

.sh-sub-card__feature {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: var(--sh-t-meta);
  color: var(--sh-fg-3);
}

.sh-sub-card__feature[data-active='true'] {
  color: var(--sh-fg);
}

.sh-sub-card__dot {
  width: 6px;
  height: 6px;
  border-radius: var(--sh-r-full);
  background: var(--sh-fg-3);
  opacity: 0.45;
  flex-shrink: 0;
}

.sh-sub-card__feature[data-active='true'] .sh-sub-card__dot {
  background: var(--sh-success);
  opacity: 1;
}

.sh-sub-radio {
  display: inline-flex;
}

.sh-sub-features {
  display: grid;
  gap: var(--sh-s-3);
}

.sh-sub-feature {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.sh-sub-feature__hint {
  font-size: var(--sh-t-meta);
  color: var(--sh-fg-3);
  padding-left: 26px;
}

.sh-drawer__foot-spacer {
  flex: 1;
}
</style>
