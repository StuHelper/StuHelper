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
          <DashboardPage
            v-else-if="activeSection === 'dashboard'"
            :model="dashboardModel"
            @open-target="openDashboardTarget"
          />
          <IdentityQueuePage
            v-else-if="activeSection === 'identity'"
            v-model:selected-guard-ids="selectedGuardIds"
            :pending-members="pendingMembers"
            :selected-id="selectedMemberId"
            :loading="loading"
            :guard-form="guardForm"
            :run-task="runTask"
            :submit-guard-action="submitGuardAction"
            :inspect-member="inspectMember"
          />
          <ReviewQueuePage
            v-else-if="activeSection === 'enforcement'"
            :pending-reviews="pendingReviews"
            :recent-reports="recentReports"
            :selected-id="selectedReviewId"
            :review-form="reviewForm"
            :run-task="runTask"
            :submit-review-and-focus="submitReviewAndFocus"
            :inspect-review="inspectReview"
            :inspect-report="inspectReport"
            :set-visible-review-ids="setVisibleReviewIds"
          />
          <EventsView
            v-else-if="activeSection === 'audit'"
            v-model:event-search="eventSearch"
            v-model:report-search="reportSearch"
            :filtered-events="filteredEvents"
            :filtered-reports="filteredReports"
            :inspect-event="inspectEvent"
            :inspect-report="inspectReport"
          />
          <PolicyCenterPage
            v-else-if="activeSection === 'policy'"
            :categories="policyCategoryItems"
            :active-category="activePolicyCategory"
            @select-category="selectPolicyCategory"
          >
            <RulesView
              v-if="activePolicyCategory === 'keyword-rules' || activePolicyCategory === 'command-policies'"
              :mode="activePolicyCategory"
              :show-header="false"
              :keyword-rules="keywordRules"
              :command-policies="commandPolicies"
              :supported-command-ids="supportedCommandIds"
              :rule-form="ruleForm"
              :policy-form="policyForm"
              :run-task="runTask"
              :submit-rule="submitRule"
              :submit-policy="submitPolicy"
              :inspect-rule="(item) => openInspector('rule', item.id, item)"
              :load-rule="loadRule"
              :load-policy="loadPolicy"
            />
            <MemberRolesPanel
              v-else-if="activePolicyCategory === 'member-roles'"
              :member-roles="memberRoles"
              :role-form="roleForm"
              :run-task="runTask"
              :submit-roles="submitRoles"
              :load-member-roles="loadMemberRoles"
            />
            <GuardPolicyPanel
              v-else
              :mode="activePolicyCategory"
              :templates="guardTemplates"
              :bindings="guardBindings"
              :template-form="templateForm"
              :binding-form="bindingForm"
              :run-task="runTask"
              :submit-template="submitTemplate"
              :submit-binding="submitBinding"
              :load-template="loadTemplate"
              :load-binding="loadBinding"
              :inspect-template="(item) => openInspector('template', item.id, item)"
              :inspect-binding="(item) => openInspector('binding', item.id, item)"
            />
          </PolicyCenterPage>
        </ConsoleShell>

        <NoticeStack :items="notices" @dismiss="dismissNotice" />
        <Drawer :open="inspector.open" :title="inspectorTitle" :subtitle="inspector.id" @close="closeInspector">
          <dl class="sh-keylist">
            <template v-for="item in inspectorDetails" :key="item.label">
              <dt>{{ item.label }}</dt>
              <dd :class="{ 'sh-mono': item.mono }">{{ item.value }}</dd>
            </template>
          </dl>
          <template #footer v-if="reviewPending">
            <button class="sh-btn sh-btn--ghost" @click="runTask(() => submitReviewAndFocus(inspector.id, 'reject'))">驳回</button>
            <button class="sh-btn sh-btn--primary" @click="runTask(() => submitReviewAndFocus(inspector.id, 'execute'))">执行</button>
          </template>
        </Drawer>
      </div>
    </el-scrollbar>
  </k-layout>
</template>

<script setup lang="ts">
import { computed } from 'vue'

import type {
  StuhelperConsoleEvent,
  StuhelperConsoleGuardBinding,
  StuhelperConsoleGuardMember,
  StuhelperConsoleGuardTemplate,
  StuhelperConsoleKeywordRule,
  StuhelperConsoleReport,
  StuhelperConsoleReview,
} from '../src/console-types'
import './styles/layout.css'
import DashboardPage from './components/dashboard/DashboardPage.vue'
import Drawer from './components/Drawer.vue'
import EmptyState from './components/EmptyState.vue'
import GuardPolicyPanel from './components/GuardPolicyPanel.vue'
import NoticeStack from './components/NoticeStack.vue'
import IdentityQueuePage from './components/queue/IdentityQueuePage.vue'
import PolicyCenterPage from './components/policy/PolicyCenterPage.vue'
import ReviewQueuePage from './components/queue/ReviewQueuePage.vue'
import ConsoleShell from './components/layout/ConsoleShell.vue'
import ConsoleSidebar from './components/layout/ConsoleSidebar.vue'
import ConsoleWorkspaceHeader from './components/layout/ConsoleWorkspaceHeader.vue'
import EventsView from './components/views/EventsView.vue'
import MemberRolesPanel from './components/views/MemberRolesPanel.vue'
import RulesView from './components/views/RulesView.vue'
import { POLICY_CATEGORIES } from './policy/categories'
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
  inspector,
  openInspector,
  closeInspector,
  inspectMember,
  inspectReview,
  inspectEvent,
  inspectReport,
  setVisibleReviewIds,
  selectPolicyCategory,
  notices,
  dismissNotice,
  pendingMembers,
  pendingReviews,
  selectedMemberId,
  selectedReviewId,
  activePolicyCategory,
  dashboardModel,
  keywordRules,
  commandPolicies,
  memberRoles,
  guardTemplates,
  guardBindings,
  recentReports,
  supportedCommandIds,
  filteredEvents,
  filteredReports,
  eventSearch,
  reportSearch,
  selectedGuardIds,
  guardForm,
  reviewForm,
  ruleForm,
  templateForm,
  bindingForm,
  roleForm,
  policyForm,
  runTask,
  refresh,
  submitGuardAction,
  submitReviewAndFocus,
  submitRule,
  submitTemplate,
  submitBinding,
  submitRoles,
  submitPolicy,
  loadRule,
  loadTemplate,
  loadBinding,
  loadMemberRoles,
  loadPolicy,
  openDashboardTarget,
} = useConsolePage()

const activeSection = computed(() => routeState.value.section)
const activeSectionLabel = computed(
  () =>
    CONSOLE_SECTIONS.find((section) => section.id === activeSection.value)?.label ??
    CONSOLE_SECTIONS[0].label,
)
const sidebarItems = computed(() => buildSidebarItems(data.value))
const policyCategoryItems = computed(() => {
  const counts = {
    'keyword-rules': keywordRules.value.length,
    'command-policies': commandPolicies.value.length,
    'member-roles': memberRoles.value.length,
    'guard-templates': guardTemplates.value.length,
    'guard-bindings': guardBindings.value.length,
  } as const

  return POLICY_CATEGORIES.map((item) => ({
    ...item,
    count: counts[item.id],
  }))
})
const headerMeta = computed(() => {
  const updatedAt = generatedAt.value ? `最近同步 ${formatTimestamp(generatedAt.value)}` : '尚未获取同步时间'
  return `${title.value} · ${updatedAt}`
})
const reviewPending = computed(
  () => inspector.kind === 'review' && (inspector.payload as StuhelperConsoleReview | null)?.status === 'pending',
)
const inspectorTitle = computed(
  () => ({ member: '成员详情', review: '复核详情', event: '事件详情', report: '举报详情', template: '模板详情', binding: '绑定详情', rule: '规则详情' }[inspector.kind || 'event']),
)
const inspectorDetails = computed(() => {
  const payload = inspector.payload as Record<string, unknown> | null
  if (!payload) return []
  if (inspector.kind === 'member') return detailList(payload as unknown as StuhelperConsoleGuardMember, [['成员', 'memberName'], ['成员 ID', 'memberId', true], ['群号', 'guildId', true], ['状态', 'verificationState'], ['截止', 'deadlineAt', true], ['最后错误', 'lastError']])
  if (inspector.kind === 'review') return detailList(payload as unknown as StuhelperConsoleReview, [['成员', 'memberId', true], ['动作', 'actionType'], ['状态', 'status'], ['原因', 'reason'], ['提交时间', 'createdAt', true], ['备注', 'resolutionNote']])
  if (inspector.kind === 'event') return detailList(payload as unknown as StuhelperConsoleEvent, [['类型', 'type'], ['级别', 'level'], ['成员', 'memberId', true], ['群号', 'guildId', true], ['摘要', 'summary'], ['时间', 'createdAt', true]])
  if (inspector.kind === 'report') return detailList(payload as unknown as StuhelperConsoleReport, [['举报人', 'reporterMemberId', true], ['目标', 'targetMemberId', true], ['AI 状态', 'aiStatus'], ['AI 等级', 'aiSeverity'], ['AI 摘要', 'aiSummary'], ['原因', 'reason'], ['时间', 'createdAt', true]])
  if (inspector.kind === 'template') return detailList(payload as unknown as StuhelperConsoleGuardTemplate, [['模板', 'name'], ['模板 ID', 'id', true], ['禁言秒数', 'muteDurationSeconds', true], ['踢出分钟数', 'kickAfterMinutes', true], ['提醒文案', 'reminderTemplate'], ['启用', 'enabled']])
  if (inspector.kind === 'binding') return detailList(payload as unknown as StuhelperConsoleGuardBinding, [['平台', 'platform'], ['群号', 'guildId', true], ['模板', 'templateId', true], ['启用', 'enabled'], ['备注', 'note']])
  return detailList(payload as unknown as StuhelperConsoleKeywordRule, [['规则', 'id', true], ['群号', 'guildId', true], ['模式', 'matchMode'], ['动作', 'action'], ['表达式', 'pattern'], ['备注', 'note']])
})

function selectSection(section: ConsoleSectionId) {
  if (section === activeSection.value) return
  closeInspector()
  setRouteState({
    section,
    queue: section === 'policy' ? activePolicyCategory.value : null,
    id: '',
    source: 'nav',
  })
}

function detailList(record: Record<string, unknown>, fields: Array<[string, string, boolean?]>) {
  return fields.map(([label, key, mono]) => ({ label, value: normalizeValue(record[key]), mono: Boolean(mono) }))
}

function normalizeValue(value: unknown) {
  if (value === null || value === undefined || value === '') return '—'
  if (Array.isArray(value)) return value.join(', ') || '—'
  if (typeof value === 'boolean') return value ? '是' : '否'
  return String(value)
}
</script>
