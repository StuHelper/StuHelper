<template>
  <Teleport to="body">
    <transition name="sh-search-fade">
      <div
        v-if="shell.searchOpen.value"
        class="stuhelperGroupCenter-portal sh-search"
        role="presentation"
        @click.self="close"
      >
        <div
          ref="panelRef"
          class="sh-search__panel"
          role="dialog"
          aria-modal="true"
          aria-label="全站搜索"
          @keydown.tab="trapFocus"
          @keydown.escape.stop.prevent="close"
        >
          <div class="sh-search__head">
            <span class="sh-search__icon" aria-hidden="true">⌕</span>
            <input
              ref="inputRef"
              v-model="query"
              type="text"
              class="sh-search__input"
              aria-label="全站搜索"
              placeholder="输入用户 ID / 群号 / 视图名 / 命令…"
              @keydown.down.prevent="moveFocus(1)"
              @keydown.up.prevent="moveFocus(-1)"
              @keydown.enter.prevent="executeActive"
              @keydown.escape.stop.prevent="close"
            />
            <span class="sh-cmd__kbd">Esc</span>
          </div>

          <div class="sh-search__hint" v-if="!query.trim()">
            <span class="sh-search__hint-line">数字 = 查看用户 / 群组上下文</span>
            <span class="sh-search__hint-line">文字 = 跳转视图、命令或快捷动作</span>
            <span class="sh-search__hint-line">{{ shortcuts.search }} 打开 · {{ shortcuts.chat }} 实时聊天 · Esc 关闭</span>
          </div>

          <ul class="sh-search__list" v-if="results.length > 0">
            <li
              v-for="(result, index) in results"
              :key="result.id"
              class="sh-search__item"
            >
              <button
                type="button"
                class="sh-search__row"
                :aria-label="`${kindLabel(result.kind)}：${result.title}，${result.hint}`"
                :class="{ 'sh-search__row--active': index === activeIndex }"
                @mouseenter="activeIndex = index"
                @focus="activeIndex = index"
                @click="execute(result)"
              >
                <span class="sh-search__row-kind" :data-kind="result.kind">{{ kindLabel(result.kind) }}</span>
                <div class="sh-search__row-body">
                  <span class="sh-search__row-title">{{ result.title }}</span>
                  <span class="sh-search__row-hint">{{ result.hint }}</span>
                </div>
                <span class="sh-search__row-arrow" aria-hidden="true">↵</span>
              </button>
            </li>
          </ul>

          <div class="sh-search__empty" v-else-if="query.trim()">
            未找到匹配项。试试输入纯数字（用户 / 群号）、视图名或命令名。
          </div>
        </div>
      </div>
    </transition>
  </Teleport>
</template>

<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { useRouter } from 'vue-router'

import { useAppShell } from '../../composables/use-app-shell'
import type { ConsoleNavigationController } from '../../composables/use-console-navigation'
import type { ConsoleViewId } from '../../models/views'
import { searchCommandIntents, type ShellCommandIntent } from './command-intents'
import { getShellShortcutLabels } from './shortcut-labels'

const props = defineProps<{
  navigation: ConsoleNavigationController
}>()

const shell = useAppShell()
const shortcuts = getShellShortcutLabels()
const router = useRouter()
const panelRef = ref<HTMLElement | null>(null)
const inputRef = ref<HTMLInputElement | null>(null)
const query = ref('')
const activeIndex = ref(0)
const restoreFocusTarget = ref<HTMLElement | null>(null)

interface ViewIntent {
  readonly id: string
  readonly kind: 'view'
  readonly title: string
  readonly hint: string
  readonly view: ConsoleViewId
  readonly keywords: readonly string[]
}

interface EntityIntent {
  readonly id: string
  readonly kind: 'user' | 'guild'
  readonly title: string
  readonly hint: string
  readonly entityId: string
}

interface ActionIntent {
  readonly id: string
  readonly kind: 'action'
  readonly title: string
  readonly hint: string
  readonly run: () => void
}

type SearchResult = ViewIntent | EntityIntent | ActionIntent | ShellCommandIntent

const VIEW_INTENTS: readonly ViewIntent[] = [
  { id: 'v:dashboard', kind: 'view', title: '总览 · Dashboard', hint: '工作台 / 总览', view: 'dashboard', keywords: ['总览', 'dashboard', '主页', 'overview'] },
  { id: 'v:admission', kind: 'view', title: '入群认证', hint: '工作台 / admission 运行态', view: 'admission', keywords: ['入群', '认证', '准入', '待准入', '新生', '新生审核', 'admission', 'join', 'verify'] },
  { id: 'v:review', kind: 'view', title: '处置中心', hint: '工作台 / 复核 + 举报队列', view: 'review', keywords: ['处置', 'review', '复核', '准入', '待准入', '举报', 'report'] },
  { id: 'v:identity', kind: 'view', title: '限制中', hint: '成员 / 待解除限制名单', view: 'identity', keywords: ['限制', '认证', 'identity', 'restricted', '成员'] },
  { id: 'v:warns', kind: 'view', title: '警告记录', hint: '成员 / 累计警告列表', view: 'warns', keywords: ['警告', 'warn', 'warns'] },
  { id: 'v:blacklist', kind: 'view', title: '黑名单', hint: '成员 / 全局封禁', view: 'blacklist', keywords: ['黑名单', 'blacklist', 'ban', '封禁'] },
  { id: 'v:config', kind: 'view', title: '群组配置', hint: '配置 / 各群功能开关', view: 'config', keywords: ['配置', 'config', '群组配置', 'guild'] },
  { id: 'v:roles', kind: 'view', title: '角色权限', hint: '配置 / 角色与权限', view: 'roles', keywords: ['角色', 'roles', '权限', 'permission'] },
  { id: 'v:settings', kind: 'view', title: '全局设置', hint: '配置 / 全局默认值', view: 'settings', keywords: ['设置', 'settings', '全局', 'global'] },
  { id: 'v:logs', kind: 'view', title: '日志检索', hint: '工具 / 命令日志', view: 'logs', keywords: ['日志', 'logs', 'log'] },
  { id: 'v:subscriptions', kind: 'view', title: '推送订阅', hint: '工具 / 通知订阅', view: 'subscriptions', keywords: ['订阅', 'subscriptions', '推送'] },
  { id: 'v:system', kind: 'view', title: '系统 / 缓存', hint: '工具 / 缓存与系统', view: 'system', keywords: ['系统', '缓存', 'cache', 'system'] },
]

const ACTION_INTENTS: readonly ActionIntent[] = [
  {
    id: 'a:chat',
    kind: 'action',
    title: '打开实时聊天',
    hint: `浮窗 · ${shortcuts.chat}`,
    run: () => shell.openChat(),
  },
  {
    id: 'a:rail',
    kind: 'action',
    title: '固定 / 取消固定侧栏',
    hint: '切换 NavRail 固定状态',
    run: () => shell.pinRail(!shell.railPinned.value),
  },
]

const ID_PATTERN = /^\d{5,}$/

const results = computed<readonly SearchResult[]>(() => {
  const raw = query.value.trim()
  if (!raw) {
    return VIEW_INTENTS.slice(0, 6)
  }

  if (ID_PATTERN.test(raw)) {
    return [
      {
        id: `e:user:${raw}`,
        kind: 'user',
        title: `查看用户 ${raw}`,
        hint: '打开实体上下文 · 显示该用户的警告 / 黑名单 / 复核',
        entityId: raw,
      },
      {
        id: `e:guild:${raw}`,
        kind: 'guild',
        title: `查看群组 ${raw}`,
        hint: '打开实体上下文 · 显示该群的警告 / 限制 / 复核',
        entityId: raw,
      },
    ]
  }

  const lower = raw.toLowerCase()
  const matched: SearchResult[] = []

  for (const intent of VIEW_INTENTS) {
    if (
      intent.title.toLowerCase().includes(lower) ||
      intent.hint.toLowerCase().includes(lower) ||
      intent.keywords.some((kw) => kw.toLowerCase().includes(lower))
    ) {
      matched.push(intent)
    }
  }

  for (const intent of ACTION_INTENTS) {
    if (
      intent.title.toLowerCase().includes(lower) ||
      intent.hint.toLowerCase().includes(lower)
    ) {
      matched.push(intent)
    }
  }

  matched.push(...searchCommandIntents(raw))

  return matched
})

watch(
  () => shell.searchOpen.value,
  (open) => {
    if (open) {
      restoreFocusTarget.value = document.activeElement instanceof HTMLElement
        ? document.activeElement
        : null
      query.value = ''
      activeIndex.value = 0
      void nextTick(() => inputRef.value?.focus())
    }
  },
)

watch(
  () => results.value,
  () => {
    if (activeIndex.value >= results.value.length) {
      activeIndex.value = 0
    }
  },
)

function close(): void {
  shell.closeSearch()
  void nextTick(() => restoreFocus())
}

function moveFocus(delta: number): void {
  if (results.value.length === 0) return
  const next = (activeIndex.value + delta + results.value.length) % results.value.length
  activeIndex.value = next
}

function executeActive(): void {
  const result = results.value[activeIndex.value]
  if (result) execute(result)
}

function trapFocus(event: KeyboardEvent): void {
  const panel = panelRef.value
  if (!panel) return

  const focusable = Array.from(
    panel.querySelectorAll<HTMLElement>(
      'button:not([disabled]), input:not([disabled]), [tabindex]:not([tabindex="-1"])',
    ),
  ).filter((element) => element.offsetParent !== null)

  if (focusable.length === 0) return

  const first = focusable[0]
  const last = focusable[focusable.length - 1]
  const active = document.activeElement

  if (event.shiftKey && active === first) {
    event.preventDefault()
    last.focus()
  } else if (!event.shiftKey && active === last) {
    event.preventDefault()
    first.focus()
  }
}

function restoreFocus(): void {
  const target = restoreFocusTarget.value
  restoreFocusTarget.value = null
  if (target?.isConnected) {
    target.focus({ preventScroll: true })
  }
}

function execute(result: SearchResult): void {
  if (result.kind === 'view') {
    props.navigation.selectView(result.view)
  } else if (result.kind === 'user' || result.kind === 'guild') {
    shell.openEntity({ kind: result.kind, id: result.entityId })
  } else if (result.kind === 'command') {
    void router.push(result.path)
  } else {
    result.run()
  }
  close()
}

function kindLabel(kind: SearchResult['kind']): string {
  switch (kind) {
    case 'view':
      return '视图'
    case 'user':
      return '用户'
    case 'guild':
      return '群组'
    case 'command':
      return '命令'
    case 'action':
      return '动作'
  }
}
</script>

<style scoped>
.sh-search {
  position: fixed;
  inset: 0;
  z-index: var(--sh-z-search);
  display: flex;
  align-items: flex-start;
  justify-content: center;
  padding: 12vh var(--sh-s-4) var(--sh-s-4);
  background: rgba(0, 0, 0, 0.4);
  backdrop-filter: blur(4px);
}

.sh-search__panel {
  width: 100%;
  max-width: 640px;
  background: var(--sh-surface-0);
  border: 1px solid var(--sh-border);
  border-radius: var(--sh-r-3);
  box-shadow: var(--sh-shadow-overlay);
  overflow: hidden;
  display: flex;
  flex-direction: column;
  max-height: 70vh;
}

.sh-search__head {
  display: flex;
  align-items: center;
  gap: var(--sh-s-2);
  padding: 0 var(--sh-s-4);
  height: 52px;
  border-bottom: 1px solid var(--sh-border);
  background: var(--sh-surface-1);
}

.sh-search__icon {
  font-size: 18px;
  color: var(--sh-fg-3);
}

.sh-search__input {
  flex: 1;
  background: transparent;
  border: 0;
  outline: 0;
  font-family: var(--sh-font-ui);
  font-size: var(--sh-t-body-lg);
  color: var(--sh-fg);
  letter-spacing: 0;
}

.sh-search__input::placeholder {
  color: var(--sh-fg-3);
}

.sh-search__hint {
  padding: var(--sh-s-3) var(--sh-s-4);
  font-size: var(--sh-t-meta);
  color: var(--sh-fg-3);
  display: grid;
  gap: 4px;
  border-bottom: 1px solid var(--sh-border);
}

.sh-search__hint-line {
  font-family: var(--sh-font-ui);
}

.sh-search__list {
  list-style: none;
  margin: 0;
  padding: 6px 0;
  overflow-y: auto;
  flex: 1 1 auto;
}

.sh-search__item {
  margin: 0;
  padding: 0;
}

.sh-search__row {
  display: grid;
  grid-template-columns: auto 1fr auto;
  width: 100%;
  gap: var(--sh-s-3);
  align-items: center;
  padding: 8px var(--sh-s-4);
  border: 0;
  background: transparent;
  color: inherit;
  font: inherit;
  text-align: left;
  cursor: pointer;
  transition: background var(--sh-dur-fast) var(--sh-ease);
}

.sh-search__row--active,
.sh-search__row:hover {
  background: var(--sh-primary-soft);
}

.sh-search__row:focus-visible {
  outline: 2px solid var(--sh-primary);
  outline-offset: -2px;
}

.sh-search__row-kind {
  font: var(--sh-w-medium) var(--sh-t-caption) / 1 var(--sh-font-mono);
  text-transform: uppercase;
  letter-spacing: 0.14em;
  padding: 3px 6px;
  border-radius: var(--sh-r-1);
  background: var(--sh-surface-1);
  color: var(--sh-fg-2);
}

.sh-search__row-kind[data-kind='view']   { background: var(--sh-primary-soft); color: var(--sh-primary); }
.sh-search__row-kind[data-kind='user']   { background: var(--sh-info-soft);    color: var(--sh-info); }
.sh-search__row-kind[data-kind='guild']  { background: var(--sh-success-soft); color: var(--sh-success); }
.sh-search__row-kind[data-kind='action'] { background: var(--sh-warning-soft); color: var(--sh-warning); }
.sh-search__row-kind[data-kind='command'] { background: var(--sh-danger-soft); color: var(--sh-danger); }

.sh-search__row-body {
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.sh-search__row-title {
  font-size: var(--sh-t-body);
  font-weight: var(--sh-w-medium);
  color: var(--sh-fg);
  letter-spacing: 0;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.sh-search__row-hint {
  font-size: var(--sh-t-meta);
  color: var(--sh-fg-3);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.sh-search__row-arrow {
  font-family: var(--sh-font-mono);
  color: var(--sh-fg-3);
}

.sh-search__row--active .sh-search__row-arrow {
  color: var(--sh-primary);
}

.sh-search__empty {
  padding: var(--sh-s-5);
  text-align: center;
  color: var(--sh-fg-3);
  font-size: var(--sh-t-meta);
}

.sh-search-fade-enter-active,
.sh-search-fade-leave-active {
  transition: opacity var(--sh-dur-base) var(--sh-ease);
}

.sh-search-fade-enter-active .sh-search__panel,
.sh-search-fade-leave-active .sh-search__panel {
  transition: transform var(--sh-dur-base) var(--sh-ease);
}

.sh-search-fade-enter-from,
.sh-search-fade-leave-to {
  opacity: 0;
}

.sh-search-fade-enter-from .sh-search__panel,
.sh-search-fade-leave-to .sh-search__panel {
  transform: translateY(-12px);
}
</style>
