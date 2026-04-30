<template>
  <Teleport to="body">
    <div class="stuhelperGroupCenter-portal">
      <div
        class="sh-overlay__scrim"
        :data-open="open ? 'true' : 'false'"
        @click="close"
      ></div>
      <aside
        class="sh-overlay"
        :data-open="open ? 'true' : 'false'"
        role="dialog"
        aria-modal="true"
        :aria-hidden="open ? 'false' : 'true'"
      >
        <template v-if="display">
        <header class="sh-overlay__head">
          <div class="sh-overlay__head-copy">
            <span class="sh-overlay__kind">
              {{ display.kind === 'user' ? 'USER' : 'GUILD' }}
            </span>
            <h2 class="sh-overlay__title">{{ profileTitle }}</h2>
            <span class="sh-overlay__id">{{ display.id }}</span>
          </div>
          <button
            type="button"
            class="sh-overlay__close"
            aria-label="关闭"
            @click="close"
          >
            ✕
          </button>
        </header>

        <div class="sh-overlay__body">
          <div v-if="state.loading" class="sh-overlay__loading">
            <span class="sh-overlay__spinner" aria-hidden="true"></span>
            <span>加载上下文…</span>
          </div>

          <div v-else-if="state.error" class="sh-overlay__error">
            <strong>加载失败</strong>
            <p>{{ state.error }}</p>
            <button type="button" class="sh-overlay__retry" @click="reload">重试</button>
          </div>

          <template v-else-if="state.profile">
            <section v-if="state.profile.kind === 'user'">
              <div class="sh-overlay__section-title">概览</div>
              <div class="sh-overlay__metrics">
                <div class="sh-overlay__metric">
                  <span class="sh-overlay__metric-value sh-num">{{ state.profile.summary.totalWarns }}</span>
                  <span class="sh-overlay__metric-label">累计警告</span>
                </div>
                <div class="sh-overlay__metric" :data-tone="state.profile.summary.blacklisted ? 'danger' : 'neutral'">
                  <span class="sh-overlay__metric-value">{{ state.profile.summary.blacklisted ? '是' : '否' }}</span>
                  <span class="sh-overlay__metric-label">在黑名单</span>
                </div>
                <div class="sh-overlay__metric" :data-tone="state.profile.summary.pendingReviews > 0 ? 'warning' : 'neutral'">
                  <span class="sh-overlay__metric-value sh-num">{{ state.profile.summary.pendingReviews }}</span>
                  <span class="sh-overlay__metric-label">待复核</span>
                </div>
                <div class="sh-overlay__metric" :data-tone="state.profile.summary.restrictedGuilds > 0 ? 'warning' : 'neutral'">
                  <span class="sh-overlay__metric-value sh-num">{{ state.profile.summary.restrictedGuilds }}</span>
                  <span class="sh-overlay__metric-label">限制中</span>
                </div>
              </div>
            </section>

            <section v-if="state.profile.kind === 'guild'">
              <div class="sh-overlay__section-title">概览</div>
              <div class="sh-overlay__metrics">
                <div class="sh-overlay__metric" :data-tone="state.profile.summary.configured ? 'primary' : 'neutral'">
                  <span class="sh-overlay__metric-value">{{ state.profile.summary.configured ? '已配置' : '未配置' }}</span>
                  <span class="sh-overlay__metric-label">群组配置</span>
                </div>
                <div class="sh-overlay__metric">
                  <span class="sh-overlay__metric-value sh-num">{{ state.profile.summary.warnedUsers }}</span>
                  <span class="sh-overlay__metric-label">被警告用户</span>
                </div>
                <div class="sh-overlay__metric" :data-tone="state.profile.summary.pendingMembers > 0 ? 'warning' : 'neutral'">
                  <span class="sh-overlay__metric-value sh-num">{{ state.profile.summary.pendingMembers }}</span>
                  <span class="sh-overlay__metric-label">限制中</span>
                </div>
                <div class="sh-overlay__metric" :data-tone="state.profile.summary.openReports > 0 ? 'warning' : 'neutral'">
                  <span class="sh-overlay__metric-value sh-num">{{ state.profile.summary.openReports }}</span>
                  <span class="sh-overlay__metric-label">未处理举报</span>
                </div>
              </div>
            </section>

            <section v-if="state.profile.warns.length > 0">
              <div class="sh-overlay__section-title">警告 · {{ state.profile.warns.length }}</div>
              <ul class="sh-overlay__list">
                <li
                  v-for="warn in state.profile.warns"
                  :key="`${warn.guildId}:${warn.userId}`"
                  class="sh-overlay__row"
                  @click="goWarns(warn.guildId, state.profile?.kind === 'user' ? warn.userId : undefined)"
                >
                  <span class="sh-overlay__row-primary">
                    {{ state.profile?.kind === 'user'
                      ? (warn.guildName || warn.guildId)
                      : warn.userId }}
                  </span>
                  <span class="sh-overlay__row-secondary"></span>
                  <span class="sh-overlay__row-meta sh-num">{{ warn.count }} 次</span>
                </li>
              </ul>
            </section>

            <section v-if="state.profile.kind === 'user' && state.profile.blacklist">
              <div class="sh-overlay__section-title">黑名单</div>
              <div class="sh-overlay__inline-fact">
                <span class="sh-overlay__row-primary">已加入黑名单</span>
                <span class="sh-overlay__row-meta sh-mono">{{ formatDate(state.profile.blacklist.addedAt) }}</span>
              </div>
            </section>

            <section v-if="state.profile.restricted.length > 0">
              <div class="sh-overlay__section-title">限制中 · {{ state.profile.restricted.length }}</div>
              <ul class="sh-overlay__list">
                <li
                  v-for="entry in state.profile.restricted"
                  :key="`${entry.guildId}:${entry.memberId}:${entry.joinedAt}`"
                  class="sh-overlay__row"
                  @click="goIdentity(entry.guildId, entry.memberId)"
                >
                  <span class="sh-overlay__row-primary">
                    {{ state.profile?.kind === 'user'
                      ? (entry.guildName || entry.guildId)
                      : entry.memberId }}
                  </span>
                  <span class="sh-overlay__row-secondary"></span>
                  <span class="sh-overlay__row-meta">
                    <span class="sh-overlay__pill" :data-tone="restrictedTone(entry.status)">{{ restrictedLabel(entry.status) }}</span>
                  </span>
                </li>
              </ul>
            </section>

            <section v-if="state.profile.reviews.length > 0">
              <div class="sh-overlay__section-title">复核 · {{ state.profile.reviews.length }}</div>
              <ul class="sh-overlay__list">
                <li
                  v-for="review in state.profile.reviews"
                  :key="review.id"
                  class="sh-overlay__row"
                  @click="goReview(review.id, review.guildId)"
                >
                  <span class="sh-overlay__row-primary">{{ review.actionType }}</span>
                  <span class="sh-overlay__row-secondary">{{ review.reason || review.guildName || review.guildId }}</span>
                  <span class="sh-overlay__row-meta sh-mono">{{ formatDate(review.createdAt) }}</span>
                </li>
              </ul>
            </section>

            <section v-if="state.profile.reports.length > 0">
              <div class="sh-overlay__section-title">举报 · {{ state.profile.reports.length }}</div>
              <ul class="sh-overlay__list">
                <li
                  v-for="report in state.profile.reports"
                  :key="report.id"
                  class="sh-overlay__row"
                  @click="goReports(report.guildId)"
                >
                  <span class="sh-overlay__row-primary">{{ report.reason || '(无原因)' }}</span>
                  <span class="sh-overlay__row-secondary">
                    {{ state.profile?.kind === 'user'
                      ? (report.guildName || report.guildId)
                      : report.targetMemberId }}
                  </span>
                  <span class="sh-overlay__row-meta sh-mono">{{ formatDate(report.createdAt) }}</span>
                </li>
              </ul>
            </section>

            <section v-if="state.profile.recentEvents.length > 0">
              <div class="sh-overlay__section-title">最近事件 · {{ state.profile.recentEvents.length }}</div>
              <ul class="sh-overlay__list">
                <li
                  v-for="event in state.profile.recentEvents"
                  :key="event.id"
                  class="sh-overlay__row"
                  @click="goLogs(event.guildId, state.profile?.kind === 'user' ? event.memberId ?? undefined : undefined)"
                >
                  <span class="sh-overlay__row-primary">
                    <span class="sh-overlay__pill" :data-tone="event.severity">{{ event.severity }}</span>
                    <span class="sh-overlay__row-kind">{{ event.kind }}</span>
                  </span>
                  <span class="sh-overlay__row-secondary">{{ event.message }}</span>
                  <span class="sh-overlay__row-meta sh-mono">{{ formatDate(event.createdAt) }}</span>
                </li>
              </ul>
            </section>

            <section v-if="!hasAnyData">
              <div class="sh-overlay__empty">
                {{ display.kind === 'user' ? '此用户当前无任何记录。' : '此群组当前无活跃记录。' }}
              </div>
            </section>

            <section>
              <div class="sh-overlay__section-title">在其他视图查看</div>
              <div class="sh-overlay__jump-grid">
                <button
                  v-for="jump in jumps"
                  :key="jump.view"
                  type="button"
                  class="sh-overlay__jump"
                  @click="goJump(jump)"
                >
                  <span class="sh-overlay__jump-label">{{ jump.label }}</span>
                  <span class="sh-overlay__jump-hint">{{ jump.hint }}</span>
                </button>
              </div>
            </section>
          </template>
        </div>
        </template>
      </aside>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'

import { consolePageApi } from '../../page-api'
import type { EntityProfile, EntityRestrictedFact } from '../../page-types'
import { useAppShell, type EntityRef } from '../../composables/use-app-shell'
import type {
  ConsoleNavigationController,
  ConsoleNavigationState,
} from '../../composables/use-console-navigation'
import type { ConsoleViewId } from '../../models/views'

const props = defineProps<{
  navigation: ConsoleNavigationController
}>()

const shell = useAppShell()

const open = computed(() => shell.entityTarget.value !== null)
const display = ref<EntityRef | null>(null)

interface ProfileState {
  loading: boolean
  error: string
  profile: EntityProfile | null
  loadedKey: string | null
}

const state = reactive<ProfileState>({
  loading: false,
  error: '',
  profile: null,
  loadedKey: null,
})

watch(
  () => shell.entityTarget.value,
  (next) => {
    if (next) {
      display.value = next
      void loadProfile(next)
    } else {
      window.setTimeout(() => {
        if (shell.entityTarget.value === null) {
          display.value = null
          state.profile = null
          state.error = ''
          state.loadedKey = null
        }
      }, 300)
    }
  },
  { immediate: true },
)

const profileTitle = computed(() => {
  if (state.profile) {
    return state.profile.kind === 'user'
      ? (state.profile.displayName || state.profile.id)
      : (state.profile.name || state.profile.id)
  }
  if (!display.value) return ''
  return display.value.name || display.value.id
})

const hasAnyData = computed(() => {
  if (!state.profile) return false
  if (state.profile.warns.length > 0) return true
  if (state.profile.kind === 'user' && state.profile.blacklist) return true
  if (state.profile.restricted.length > 0) return true
  if (state.profile.reviews.length > 0) return true
  if (state.profile.reports.length > 0) return true
  if (state.profile.recentEvents.length > 0) return true
  return false
})

interface JumpDef {
  readonly view: ConsoleViewId
  readonly label: string
  readonly hint: string
  apply(target: EntityRef): Partial<ConsoleNavigationState>
}

const userJumps: readonly JumpDef[] = [
  { view: 'review', label: '处置中心', hint: '与此人相关的复核 / 举报', apply: (t) => ({ guildId: t.guildId ?? null, memberId: t.id, keyword: t.id }) },
  { view: 'warns', label: '警告记录', hint: '此用户在各群的警告', apply: (t) => ({ keyword: t.id }) },
  { view: 'blacklist', label: '黑名单', hint: '检查是否在黑名单', apply: (t) => ({ keyword: t.id }) },
  { view: 'logs', label: '日志检索', hint: '此用户的命令历史', apply: (t) => ({ keyword: t.id }) },
  { view: 'identity', label: '身份认证', hint: '认证状态与限制上下文', apply: (t) => ({ guildId: t.guildId ?? null, memberId: t.id, keyword: t.id }) },
]

const guildJumps: readonly JumpDef[] = [
  { view: 'config', label: '群组配置', hint: '此群的功能开关', apply: (t) => ({ workspace: 'guild-config', guildId: t.id, keyword: t.id }) },
  { view: 'warns', label: '警告记录', hint: '此群的累计警告', apply: (t) => ({ guildId: t.id }) },
  { view: 'identity', label: '受限成员', hint: '此群仍在限制中的成员', apply: (t) => ({ guildId: t.id }) },
  { view: 'review', label: '处置中心', hint: '此群的复核 / 举报', apply: (t) => ({ guildId: t.id }) },
  { view: 'logs', label: '日志检索', hint: '此群的命令历史', apply: (t) => ({ guildId: t.id }) },
]

const jumps = computed<readonly JumpDef[]>(() => {
  if (!display.value) return []
  return display.value.kind === 'user' ? userJumps : guildJumps
})

async function loadProfile(target: EntityRef) {
  const key = entityKey(target)
  state.loading = true
  state.error = ''
  if (state.loadedKey !== key) {
    state.profile = null
  }
  try {
    const profile = await consolePageApi.entityProfile({
      kind: target.kind,
      id: target.id,
      guildId: target.guildId,
    })
    if (entityKey(shell.entityTarget.value) === key) {
      state.profile = profile
      state.loadedKey = key
    }
  } catch (cause) {
    if (entityKey(shell.entityTarget.value) === key) {
      state.error = cause instanceof Error ? cause.message : '加载失败'
    }
  } finally {
    if (entityKey(shell.entityTarget.value) === key) {
      state.loading = false
    }
  }
}

function entityKey(target: EntityRef | null): string | null {
  if (!target) return null
  return `${target.kind}:${target.id}:${target.guildId ?? ''}`
}

function reload() {
  if (display.value) void loadProfile(display.value)
}

function close(): void {
  shell.closeEntity()
}

function goJump(jump: JumpDef): void {
  const target = shell.entityTarget.value
  if (!target) return
  props.navigation.jumpTo({ view: jump.view, ...jump.apply(target) })
  close()
}

function goWarns(guildId: string, userId?: string): void {
  props.navigation.jumpTo({ view: 'warns', guildId, keyword: userId ?? '' })
  close()
}

function goIdentity(guildId: string, memberId: string): void {
  props.navigation.jumpTo({ view: 'identity', guildId, memberId, keyword: memberId })
  close()
}

function goReview(itemId: string, guildId: string): void {
  props.navigation.jumpTo({ view: 'review', guildId, itemId })
  close()
}

function goReports(guildId: string): void {
  props.navigation.jumpTo({ view: 'review', guildId })
  close()
}

function goLogs(guildId: string, memberId?: string): void {
  props.navigation.jumpTo({ view: 'logs', guildId, keyword: memberId ?? '' })
  close()
}

function restrictedTone(status: EntityRestrictedFact['status']): string {
  if (status === 'pending') return 'warning'
  if (status === 'kicked') return 'danger'
  return 'success'
}

function restrictedLabel(status: EntityRestrictedFact['status']): string {
  if (status === 'pending') return '待解除'
  if (status === 'kicked') return '已踢出'
  return '已通过'
}

function formatDate(value: string | null): string {
  if (!value) return '—'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  const now = new Date()
  const sameYear = date.getFullYear() === now.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  const hour = String(date.getHours()).padStart(2, '0')
  const minute = String(date.getMinutes()).padStart(2, '0')
  return sameYear ? `${month}-${day} ${hour}:${minute}` : `${date.getFullYear()}-${month}-${day}`
}
</script>

<style scoped>
.sh-overlay__loading,
.sh-overlay__error,
.sh-overlay__empty {
  padding: var(--sh-s-4);
  font-size: var(--sh-t-meta);
  color: var(--sh-fg-2);
  text-align: center;
}

.sh-overlay__loading {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: var(--sh-s-2);
}

.sh-overlay__spinner {
  width: 12px;
  height: 12px;
  border-radius: var(--sh-r-full);
  border: 1.5px solid var(--sh-border-strong);
  border-top-color: var(--sh-primary);
  animation: sh-overlay-spin 0.8s linear infinite;
}

@keyframes sh-overlay-spin {
  to { transform: rotate(360deg); }
}

.sh-overlay__error strong {
  display: block;
  margin-bottom: var(--sh-s-2);
  color: var(--sh-danger);
}

.sh-overlay__error p {
  margin: 0 0 var(--sh-s-3);
  color: var(--sh-fg-2);
}

.sh-overlay__retry {
  border: 1px solid var(--sh-border-strong);
  background: var(--sh-surface-1);
  color: var(--sh-fg);
  border-radius: var(--sh-r-2);
  padding: 4px 12px;
  cursor: pointer;
  font-size: var(--sh-t-meta);
  transition: all var(--sh-dur-fast) var(--sh-ease);
}

.sh-overlay__retry:hover {
  border-color: var(--sh-primary);
  color: var(--sh-primary);
}

.sh-overlay__metrics {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: var(--sh-s-2);
}

.sh-overlay__metric {
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: var(--sh-s-3);
  border: 1px solid var(--sh-border);
  border-radius: var(--sh-r-2);
  background: var(--sh-surface-0);
}

.sh-overlay__metric-value {
  font-size: var(--sh-t-stat);
  font-weight: var(--sh-w-semibold);
  color: var(--sh-fg);
  letter-spacing: -0.02em;
  font-variant-numeric: tabular-nums;
}

.sh-overlay__metric-label {
  font-size: var(--sh-t-meta);
  color: var(--sh-fg-2);
}

.sh-overlay__metric[data-tone='warning'] .sh-overlay__metric-value { color: var(--sh-warning); }
.sh-overlay__metric[data-tone='danger']  .sh-overlay__metric-value { color: var(--sh-danger); }
.sh-overlay__metric[data-tone='primary'] .sh-overlay__metric-value { color: var(--sh-primary); }

.sh-overlay__list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
}

.sh-overlay__row {
  display: grid;
  grid-template-columns: minmax(0, 1.4fr) minmax(0, 1fr) auto;
  align-items: center;
  gap: var(--sh-s-2);
  padding: 8px var(--sh-s-3);
  border-bottom: 1px solid var(--sh-border);
  cursor: pointer;
  transition: background var(--sh-dur-fast) var(--sh-ease);
}

.sh-overlay__row:last-child {
  border-bottom-color: transparent;
}

.sh-overlay__row:hover {
  background: var(--sh-surface-hover);
}

.sh-overlay__row-primary {
  font-size: var(--sh-t-body);
  font-weight: var(--sh-w-medium);
  color: var(--sh-fg);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.sh-overlay__row-secondary {
  font-size: var(--sh-t-meta);
  color: var(--sh-fg-2);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.sh-overlay__row-meta {
  font-size: var(--sh-t-meta);
  color: var(--sh-fg-3);
  text-align: right;
  white-space: nowrap;
}

.sh-overlay__row-kind {
  font-family: var(--sh-font-mono);
  color: var(--sh-fg-1);
}

.sh-overlay__inline-fact {
  display: flex;
  justify-content: space-between;
  padding: 8px var(--sh-s-3);
  background: var(--sh-danger-soft);
  border: 1px solid var(--sh-danger);
  color: var(--sh-fg);
  border-radius: var(--sh-r-2);
}

.sh-overlay__pill {
  display: inline-block;
  font: var(--sh-w-medium) var(--sh-t-caption) / 1 var(--sh-font-mono);
  padding: 2px 6px;
  border-radius: var(--sh-r-1);
  background: var(--sh-surface-1);
  color: var(--sh-fg-2);
  text-transform: uppercase;
  letter-spacing: 0.04em;
}

.sh-overlay__pill[data-tone='warning']  { background: var(--sh-warning-soft); color: var(--sh-warning); }
.sh-overlay__pill[data-tone='danger']   { background: var(--sh-danger-soft);  color: var(--sh-danger); }
.sh-overlay__pill[data-tone='success']  { background: var(--sh-success-soft); color: var(--sh-success); }
.sh-overlay__pill[data-tone='high']     { background: var(--sh-warning-soft); color: var(--sh-warning); }
.sh-overlay__pill[data-tone='critical'] { background: var(--sh-danger-soft);  color: var(--sh-danger); }
.sh-overlay__pill[data-tone='info']     { background: var(--sh-info-soft);    color: var(--sh-info); }
</style>
