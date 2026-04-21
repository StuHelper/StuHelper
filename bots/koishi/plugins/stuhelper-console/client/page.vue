<template>
  <k-layout main="sh-console">
    <el-scrollbar class="sh-console__scroll" view-class="sh-console__view">
      <div class="sh-console__frame">
        <ConsoleShell>
          <template #sidebar>
            <ConsoleSidebar
              :items="sidebarItems"
              :active="activeSection"
              @select="selectSection"
            />
          </template>
          <template #header>
            <ConsoleWorkspaceHeader
              :title="activeSectionLabel"
              :description="activeSectionDescription"
              :meta="headerMeta"
            >
              <template #actions>
                <el-button
                  class="sh-button sh-button--ghost"
                  :disabled="loading"
                  @click="runTask(refresh)"
                >
                  {{ loading ? '刷新中…' : '刷新' }}
                </el-button>
              </template>
            </ConsoleWorkspaceHeader>
          </template>

          <ConsolePageSkeleton v-if="!data" />
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
            :selected-report-id="selectedReportId"
            :review-form="reviewForm"
            :run-task="runTask"
            :submit-review-and-focus="submitReviewAndFocus"
            :inspect-review="inspectReview"
            :inspect-report="inspectReport"
            :set-visible-review-ids="setVisibleReviewIds"
          />
          <AuditPage
            v-else-if="activeSection === 'audit'"
            v-model:query="auditQuery"
            v-model:kind="auditKind"
            :rows="auditRows"
            :event-count="recentEvents.length"
            :report-count="recentReports.length"
            :selected-id="selectedAuditId"
            @inspect="inspectAuditRow"
          />
          <PolicyCenterPage
            v-else-if="activeSection === 'policy'"
            :categories="policyCategoryItems"
            :active-category="activePolicyCategory"
            @select-category="selectPolicyCategory"
          >
            <PolicyKeywordRulesPanel
              v-if="activePolicyCategory === 'keyword-rules'"
              :keyword-rules="keywordRules"
              :rule-form="ruleForm"
              :run-task="runTask"
              :submit-rule="submitRule"
              :inspect-rule="(item) => openInspector('rule', item.id)"
              :load-rule="loadRule"
            />
            <PolicyCommandPoliciesPanel
              v-else-if="activePolicyCategory === 'command-policies'"
              :command-policies="commandPolicies"
              :supported-command-ids="supportedCommandIds"
              :policy-form="policyForm"
              :run-task="runTask"
              :submit-policy="submitPolicy"
              :load-policy="loadPolicy"
            />
            <PolicyMemberRolesPanel
              v-else-if="activePolicyCategory === 'member-roles'"
              :member-roles="memberRoles"
              :role-form="roleForm"
              :run-task="runTask"
              :submit-roles="submitRoles"
              :load-member-roles="loadMemberRoles"
            />
            <PolicyGuardTemplatesPanel
              v-else-if="activePolicyCategory === 'guard-templates'"
              :templates="guardTemplates"
              :template-form="templateForm"
              :run-task="runTask"
              :submit-template="submitTemplate"
              :load-template="loadTemplate"
              :inspect-template="(item) => openInspector('template', item.id)"
            />
            <PolicyGuardBindingsPanel
              v-else
              :templates="guardTemplates"
              :bindings="guardBindings"
              :binding-form="bindingForm"
              :run-task="runTask"
              :submit-binding="submitBinding"
              :load-binding="loadBinding"
              :inspect-binding="(item) => openInspector('binding', item.id)"
            />
          </PolicyCenterPage>
        </ConsoleShell>

        <NoticeStack :items="notices" @dismiss="dismissNotice" />
        <Drawer
          :open="inspector.open"
          :title="inspectorTitle"
          :subtitle="inspector.id"
          :sections="inspectorSections"
          @close="closeInspector"
        >
          <template #footer v-if="reviewPending">
            <el-button
              class="sh-button sh-button--ghost"
              @click="runTask(() => submitReviewAndFocus(inspector.id, 'reject'))"
            >
              驳回
            </el-button>
            <el-button
              type="primary"
              class="sh-button sh-button--primary"
              @click="runTask(() => submitReviewAndFocus(inspector.id, 'execute'))"
            >
              执行
            </el-button>
          </template>
        </Drawer>
      </div>
    </el-scrollbar>
  </k-layout>
</template>

<script setup lang="ts">
import { computed } from 'vue'

import './styles/layout.css'
import AuditPage from './components/audit/AuditPage.vue'
import ConsolePageSkeleton from './components/ConsolePageSkeleton.vue'
import DashboardPage from './components/dashboard/DashboardPage.vue'
import Drawer from './components/Drawer.vue'
import ConsoleShell from './components/layout/ConsoleShell.vue'
import ConsoleSidebar from './components/layout/ConsoleSidebar.vue'
import ConsoleWorkspaceHeader from './components/layout/ConsoleWorkspaceHeader.vue'
import NoticeStack from './components/NoticeStack.vue'
import PolicyCenterPage from './components/policy/PolicyCenterPage.vue'
import PolicyCommandPoliciesPanel from './components/policy/PolicyCommandPoliciesPanel.vue'
import PolicyGuardBindingsPanel from './components/policy/PolicyGuardBindingsPanel.vue'
import PolicyGuardTemplatesPanel from './components/policy/PolicyGuardTemplatesPanel.vue'
import PolicyKeywordRulesPanel from './components/policy/PolicyKeywordRulesPanel.vue'
import PolicyMemberRolesPanel from './components/policy/PolicyMemberRolesPanel.vue'
import IdentityQueuePage from './components/queue/IdentityQueuePage.vue'
import ReviewQueuePage from './components/queue/ReviewQueuePage.vue'
import { POLICY_CATEGORIES } from './policy/categories'
import { SECTION_DESCRIPTIONS } from './section-descriptions'
import { buildSidebarItems } from './sidebar-items'
import { CONSOLE_SECTIONS, type ConsoleSectionId } from './sections'
import { buildInspectorSections } from './ui-state'
import { formatTimestamp } from './formatters'
import { useConsolePage } from './use-console-page'

const {
  data,
  title,
  generatedAt,
  loading,
  routeState,
  setRouteState,
  inspector,
  inspectorPayload,
  openInspector,
  closeInspector,
  notices,
  dismissNotice,
  pendingMembers,
  pendingReviews,
  selectedMemberId,
  selectedReviewId,
  selectedReportId,
  selectedAuditId,
  activePolicyCategory,
  dashboardModel,
  keywordRules,
  commandPolicies,
  memberRoles,
  guardTemplates,
  guardBindings,
  recentEvents,
  recentReports,
  supportedCommandIds,
  auditRows,
  auditQuery,
  auditKind,
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
  inspectMember,
  inspectReview,
  inspectReport,
  inspectAuditRow,
  setVisibleReviewIds,
  selectPolicyCategory,
  openDashboardTarget,
} = useConsolePage()

const activeSection = computed(() => routeState.value.section)
const activeSectionLabel = computed(
  () =>
    CONSOLE_SECTIONS.find((section) => section.id === activeSection.value)?.label ??
    CONSOLE_SECTIONS[0].label,
)
const activeSectionDescription = computed(
  () => SECTION_DESCRIPTIONS[activeSection.value],
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
const reviewPending = computed(() => {
  const payload = inspectorPayload.value as { status?: string } | null
  return inspector.kind === 'review' && payload?.status === 'pending'
})
const inspectorTitle = computed(() => {
  const titleMap = {
    member: '成员详情',
    review: '复核详情',
    event: '事件详情',
    report: '举报详情',
    template: '模板详情',
    binding: '绑定详情',
    rule: '规则详情',
  } as const

  return inspector.kind ? titleMap[inspector.kind] : ''
})
const inspectorSections = computed(() =>
  buildInspectorSections(inspector.kind, inspectorPayload.value),
)

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
</script>
