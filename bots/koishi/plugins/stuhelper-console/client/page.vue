<template>
  <k-layout main="sh-console">
    <el-scrollbar class="sh-console__scroll" view-class="sh-console__view">
      <div class="sh-console__frame">
        <ConsoleShell>
          <template #sidebar>
            <ConsoleSidebar :items="sidebarItems" :active="activeSection" @select="selectSection" />
          </template>
          <template #header>
            <ConsoleWorkspaceHeader :title="activeSectionLabel" :meta="headerMeta">
              <template #actions>
                <button
                  type="button"
                  class="sh-btn sh-btn--ghost"
                  :disabled="loading"
                  @click="runTask(refresh)"
                >
                  {{ loading ? '刷新中…' : '刷新' }}
                </button>
              </template>
            </ConsoleWorkspaceHeader>
          </template>
          <EmptyState
            v-if="!data"
            title="控制台数据尚未就绪"
            body="点击刷新，或确认 stuhelper-console 数据服务已经正确加载。"
          />
          <ConsolePanel
            v-else
            title="工作区待接入"
            description="当前页面只保留管理壳层、侧栏计数和统一头部。"
          >
            <EmptyState
              :title="`${activeSectionLabel} 内容将在后续任务接入`"
              body="本任务不预加载 dashboard 卡片、分区摘要或具体业务视图。"
            />
          </ConsolePanel>
        </ConsoleShell>
      </div>
    </el-scrollbar>
  </k-layout>
</template>

<script setup lang="ts">
import { computed } from 'vue'

import './styles/layout.css'
import EmptyState from './components/EmptyState.vue'
import ConsolePanel from './components/ConsolePanel.vue'
import ConsoleShell from './components/layout/ConsoleShell.vue'
import ConsoleSidebar from './components/layout/ConsoleSidebar.vue'
import ConsoleWorkspaceHeader from './components/layout/ConsoleWorkspaceHeader.vue'
import { buildSidebarItems } from './sidebar-items'
import { CONSOLE_SECTIONS, type ConsoleSectionId } from './sections'
import { formatTimestamp, useConsolePage } from './use-console-page'

const {
  data,
  title,
  generatedAt,
  loading,
  routeState,
  setRouteState,
  pendingMembers,
  pendingReviews,
  keywordRules,
  commandPolicies,
  memberRoles,
  guardTemplates,
  guardBindings,
  recentEvents,
  recentReports,
  runTask,
  refresh,
} = useConsolePage()

const policyCount = computed(
  () =>
    keywordRules.value.length +
    commandPolicies.value.length +
    memberRoles.value.length +
    guardTemplates.value.length +
    guardBindings.value.length,
)

const auditCount = computed(
  () => recentEvents.value.length + recentReports.value.length,
)

const activeSection = computed(() => routeState.value.section)

const activeSectionLabel = computed(
  () =>
    CONSOLE_SECTIONS.find((section) => section.id === activeSection.value)?.label ??
    CONSOLE_SECTIONS[0].label,
)

const sidebarItems = computed(() =>
  buildSidebarItems({
    pendingMembers: pendingMembers.value.length,
    pendingReviews: pendingReviews.value.length,
    policyCount: policyCount.value,
    auditCount: auditCount.value,
  }),
)

const headerMeta = computed(() => {
  const updatedAt = generatedAt.value
    ? `最近同步 ${formatTimestamp(generatedAt.value)}`
    : '尚未获取同步时间'
  return `${title.value} · ${updatedAt}`
})

function selectSection(section: ConsoleSectionId) {
  if (section === activeSection.value) return

  setRouteState({
    section,
    queue: null,
    id: '',
    source: 'nav',
  })
}
</script>
