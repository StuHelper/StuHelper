<template>
  <header class="sh-cmd">
    <div class="sh-cmd__crumb">
      <button
        type="button"
        class="sh-cmd__menu"
        :title="shell.railExpanded.value ? '收起导航' : '展开导航'"
        @click="shell.toggleRail()"
      >
        <k-icon name="stuhelperGroupCenter:octicons.three-bars" />
      </button>
      <span class="sh-cmd__crumb-section">{{ crumb.section }}</span>
      <span class="sh-cmd__crumb-sep">/</span>
      <span class="sh-cmd__crumb-page">{{ crumb.page }}</span>
    </div>

    <button
      type="button"
      class="sh-cmd__search"
      aria-label="打开全站搜索"
      :title="`${shortcuts.search} 全站搜索`"
      @click="onSearchClick"
    >
      <span class="sh-cmd__search-icon" aria-hidden="true">⌕</span>
      <span class="sh-cmd__search-text">搜索成员 / 群 / 命令…</span>
      <span class="sh-cmd__kbd">{{ shortcuts.search }}</span>
    </button>

    <div class="sh-cmd__right">
      <button
        type="button"
        class="sh-cmd__pulse"
        :data-tone="reviewTone"
        :aria-label="`待处理复核 ${pulse.pendingReviews} 条，进入处置中心`"
        :title="`待处理复核 ${pulse.pendingReviews} 条 — 进入处置中心`"
        @click="goPulse('review')"
      >
        <span class="sh-cmd__pulse-dot"></span>
        <span class="sh-cmd__pulse-num">{{ pulse.pendingReviews }}</span>
        <span class="sh-cmd__pulse-label">复核</span>
      </button>
      <button
        type="button"
        class="sh-cmd__pulse"
        :data-tone="admissionTone"
        :aria-label="`待认证成员 ${pulse.pendingAdmissions} 条，进入限制中`"
        :title="`待认证成员 ${pulse.pendingAdmissions} 条 — 进入限制中`"
        @click="goPulse('identity')"
      >
        <span class="sh-cmd__pulse-dot"></span>
        <span class="sh-cmd__pulse-num">{{ pulse.pendingAdmissions }}</span>
        <span class="sh-cmd__pulse-label">限制</span>
      </button>
      <button
        type="button"
        class="sh-cmd__pulse"
        :data-tone="reportTone"
        :aria-label="`未处理举报 ${pulse.openReports} 条，进入处置中心`"
        :title="`未处理举报 ${pulse.openReports} 条 — 进入处置中心`"
        @click="goPulse('review')"
      >
        <span class="sh-cmd__pulse-dot"></span>
        <span class="sh-cmd__pulse-num">{{ pulse.openReports }}</span>
        <span class="sh-cmd__pulse-label">举报</span>
      </button>

      <button
        type="button"
        class="sh-cmd__chat"
        :data-active="shell.chatOpen.value && !shell.chatMinimized.value ? 'true' : 'false'"
        :aria-label="shell.chatOpen.value ? '关闭实时聊天' : '打开实时聊天'"
        :title="shell.chatOpen.value ? `关闭实时聊天 (${shortcuts.chat})` : `打开实时聊天 (${shortcuts.chat})`"
        @click="shell.toggleChat()"
      >
        <k-icon name="stuhelperGroupCenter:octicons.discussion" />
        <span class="sh-cmd__chat-label">聊天</span>
        <span class="sh-cmd__kbd">{{ shortcuts.chat }}</span>
      </button>
    </div>
  </header>
</template>

<script setup lang="ts">
import { computed } from 'vue'

import { useAppShell } from '../../composables/use-app-shell'
import type { ConsoleNavigationController } from '../../composables/use-console-navigation'
import type { ConsoleViewId } from '../../models/views'
import type { PulseState } from '../../composables/use-pulse'
import { getShellShortcutLabels } from './shortcut-labels'

const props = defineProps<{
  navigation: ConsoleNavigationController
  pulse: PulseState
}>()

const shell = useAppShell()
const shortcuts = getShellShortcutLabels()

interface CrumbDef {
  readonly section: string
  readonly page: string
}

const VIEW_CRUMBS: Readonly<Record<ConsoleViewId, CrumbDef>> = {
  dashboard: { section: '工作台', page: '总览' },
  admission: { section: '工作台', page: '入群认证' },
  review: { section: '工作台', page: '处置中心' },
  identity: { section: '成员', page: '限制中' },
  warns: { section: '成员', page: '警告记录' },
  blacklist: { section: '成员', page: '黑名单' },
  config: { section: '配置', page: '群组配置' },
  roles: { section: '配置', page: '角色权限' },
  settings: { section: '配置', page: '全局设置' },
  logs: { section: '工具', page: '日志检索' },
  subscriptions: { section: '工具', page: '推送订阅' },
  system: { section: '工具', page: '系统 / 缓存' },
  chat: { section: '工具', page: '实时聊天' },
}

const crumb = computed<CrumbDef>(() => {
  const view = props.navigation.state.value.view
  return VIEW_CRUMBS[view] ?? { section: '群管中心', page: '总览' }
})

const reviewTone = computed<'primary' | 'warning' | 'danger'>(() => {
  const n = props.pulse.pendingReviews
  if (n > 10) return 'danger'
  if (n > 0) return 'warning'
  return 'primary'
})

const admissionTone = computed<'primary' | 'warning' | 'danger'>(() => {
  const n = props.pulse.pendingAdmissions
  if (n > 20) return 'danger'
  if (n > 0) return 'warning'
  return 'primary'
})

const reportTone = computed<'primary' | 'warning' | 'danger'>(() => {
  const n = props.pulse.openReports
  if (n > 5) return 'danger'
  if (n > 0) return 'warning'
  return 'primary'
})

function goPulse(view: ConsoleViewId): void {
  props.navigation.selectView(view)
}

function onSearchClick(): void {
  shell.openSearch()
}
</script>
