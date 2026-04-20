<template>
  <k-layout main="sh-console">
    <el-scrollbar class="sh-console__scroll" view-class="sh-console__view">
      <div class="sh-console__frame">
        <ConsoleShell>
          <template #sidebar>
            <ConsoleSidebar :items="sidebarItems" :active="activeSection" @select="selectSection" />
          </template>
          <template #header>
            <ConsoleWorkspaceHeader
              :title="sectionMeta.title"
              :description="sectionMeta.description"
              :meta="headerMeta"
            >
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
          <template v-else>
            <section class="sh-shell-grid">
              <ConsolePanel title="当前计数" description="侧栏 badge 与工作区摘要都从这里取值。">
                <ul class="sh-summary-list">
                  <li
                    v-for="item in summaryItems"
                    :key="item.label"
                    class="sh-summary-list__item"
                  >
                    <span class="sh-summary-list__label">{{ item.label }}</span>
                    <strong
                      class="sh-summary-list__value"
                      :class="{ 'sh-num': typeof item.value === 'number' }"
                    >
                      {{ item.value }}
                    </strong>
                  </li>
                </ul>
              </ConsolePanel>
              <ConsolePanel title="工作区状态" description="统一头部、URL section 状态和刷新入口已接入。">
                <dl class="sh-status-list">
                  <div
                    v-for="item in statusItems"
                    :key="item.label"
                    class="sh-status-list__row"
                  >
                    <dt class="sh-status-list__label">{{ item.label }}</dt>
                    <dd class="sh-status-list__value">{{ item.value }}</dd>
                  </div>
                </dl>
              </ConsolePanel>
            </section>
            <ConsolePanel
              :title="sectionMeta.placeholderTitle"
              :description="sectionMeta.placeholderDescription"
            >
              <EmptyState
                :title="sectionMeta.emptyTitle"
                :body="sectionMeta.emptyBody"
              >
                <template #action>
                  <button
                    type="button"
                    class="sh-btn sh-btn--ghost"
                    :disabled="loading"
                    @click="runTask(refresh)"
                  >
                    {{ loading ? '刷新中…' : '重新拉取数据' }}
                  </button>
                </template>
              </EmptyState>
            </ConsolePanel>
          </template>
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
import type { ConsoleSectionId } from './sections'
import { formatTimestamp, useConsolePage } from './use-console-page'
interface SummaryItem {
  label: string
  value: number | string
}

interface SectionMeta {
  title: string
  description: string
  placeholderTitle: string
  placeholderDescription: string
  emptyTitle: string
  emptyBody: string
}
interface SectionCounts {
  pendingMembers: number
  pendingReviews: number
  policyCount: number
  auditCount: number
  keywordRules: number
  commandPolicies: number
  memberRoles: number
  guardTemplates: number
  guardBindings: number
  recentEvents: number
  recentReports: number
}
const SECTION_META: Record<ConsoleSectionId, SectionMeta> = {
  dashboard: {
    title: '首页驾驶舱',
    description: '统一查看五个主分区的积压情况与当前同步状态。',
    placeholderTitle: '驾驶舱内容区',
    placeholderDescription: '本任务先稳定壳层、侧栏和统一头部，不再继续复用旧 tab 页面。',
    emptyTitle: '驾驶舱细节尚未挂载',
    emptyBody: '当前内容区只保留计数摘要和统一入口，具体工作区内容会在独立页面里接入。',
  },
  enforcement: {
    title: '处置中心',
    description: '承载人工复核和高风险动作入口，侧栏计数对应待复核数量。',
    placeholderTitle: '处置中心内容区',
    placeholderDescription: '处置列表本次不再直接渲染，先把布局和 badge 计数固定下来。',
    emptyTitle: '处置队列尚未挂载',
    emptyBody: '当前只保留统一头部和计数摘要，旧 tab 内容没有继续塞回这个区域。',
  },
  identity: {
    title: '身份认证',
    description: '聚焦待认证成员与相关身份配置，侧栏计数对应待认证人数。',
    placeholderTitle: '身份认证内容区',
    placeholderDescription: '本区域先作为稳定的主工作区容器，具体准入列表后续独立接入。',
    emptyTitle: '认证工作区尚未挂载',
    emptyBody: '当前页只处理管理壳层、侧栏顺序和 URL section 状态，不承载旧的批量操作表单。',
  },
  policy: {
    title: '策略中心',
    description: '聚合规则、模板、绑定和命令权限，侧栏计数展示策略总量。',
    placeholderTitle: '策略中心内容区',
    placeholderDescription: '策略相关明细会继续拆分为独立内容，不再依赖旧 tabs。',
    emptyTitle: '策略明细尚未挂载',
    emptyBody: '当前只展示聚合计数，方便确认侧栏 badge 和统一头部已经接上真实数据。',
  },
  audit: {
    title: '审计检索',
    description: '统一承接事件与举报检索，侧栏计数展示最近可检索记录量。',
    placeholderTitle: '审计检索内容区',
    placeholderDescription: '本区域保留为统一内容容器，搜索表格后续单独迁入。',
    emptyTitle: '审计视图尚未挂载',
    emptyBody: '当前页只负责路由 section、计数和布局骨架，详细日志表格暂不挂载。',
  },
}

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
const policyCount = computed(() => keywordRules.value.length + commandPolicies.value.length + memberRoles.value.length + guardTemplates.value.length + guardBindings.value.length)
const auditCount = computed(() => recentEvents.value.length + recentReports.value.length)
const activeSection = computed(() => routeState.value.section)
const sectionMeta = computed(() => SECTION_META[activeSection.value])
const sidebarItems = computed(() =>
  buildSidebarItems({
    pendingMembers: pendingMembers.value.length,
    pendingReviews: pendingReviews.value.length,
    policyCount: policyCount.value,
    auditCount: auditCount.value,
  }),
)
const sectionCounts = computed<SectionCounts>(() => ({
  pendingMembers: pendingMembers.value.length,
  pendingReviews: pendingReviews.value.length,
  policyCount: policyCount.value,
  auditCount: auditCount.value,
  keywordRules: keywordRules.value.length,
  commandPolicies: commandPolicies.value.length,
  memberRoles: memberRoles.value.length,
  guardTemplates: guardTemplates.value.length,
  guardBindings: guardBindings.value.length,
  recentEvents: recentEvents.value.length,
  recentReports: recentReports.value.length,
}))
const summaryItems = computed(() =>
  buildSummaryItems(activeSection.value, sectionCounts.value, routeState.value.source),
)
const statusItems = computed(() => [
  { label: '工作区', value: title.value },
  { label: '当前 section', value: activeSection.value },
  { label: '导航来源', value: routeState.value.source },
  { label: '队列上下文', value: routeState.value.queue ?? '—' },
  { label: '定位 ID', value: routeState.value.id || '—' },
])
const headerMeta = computed(() => {
  if (!generatedAt.value) return '尚未获取同步时间'
  return `最近同步 ${formatTimestamp(generatedAt.value)}`
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

function buildSummaryItems(
  section: ConsoleSectionId,
  counts: SectionCounts,
  source: string,
): SummaryItem[] {
  switch (section) {
    case 'dashboard':
      return [
        { label: '待复核', value: counts.pendingReviews },
        { label: '待认证成员', value: counts.pendingMembers },
        { label: '策略总量', value: counts.policyCount },
        { label: '审计记录', value: counts.auditCount },
      ]
    case 'enforcement':
      return [
        { label: '人工复核', value: counts.pendingReviews },
        { label: '最近举报', value: counts.recentReports },
        { label: '最近事件', value: counts.recentEvents },
      ]
    case 'identity':
      return [
        { label: '待认证成员', value: counts.pendingMembers },
        { label: '成员角色', value: counts.memberRoles },
        { label: '群绑定', value: counts.guardBindings },
      ]
    case 'policy':
      return [
        { label: '关键词规则', value: counts.keywordRules },
        { label: '群模板', value: counts.guardTemplates },
        { label: '群绑定', value: counts.guardBindings },
        { label: '命令权限', value: counts.commandPolicies },
      ]
    case 'audit':
      return [
        { label: '事件日志', value: counts.recentEvents },
        { label: '举报日志', value: counts.recentReports },
        { label: '当前来源', value: source },
      ]
  }
}
</script>
