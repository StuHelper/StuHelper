import { computed, ref, watch } from 'vue'
import { store } from '@koishijs/client'

import {
  refreshConsoleData,
  runGuardBatchAction,
  runReviewAction,
  saveCommandPolicy,
  saveGuardBinding,
  saveGuardTemplate,
  saveKeywordRule,
  saveMemberRoles,
} from './api'
import { useConsoleAudit } from './use-console-audit'
import { buildDashboardModel, type DashboardTarget } from './dashboard/model'
import {
  DEFAULT_POLICY_CATEGORY_ID,
  resolvePolicyCategoryId,
  type PolicyCategoryId,
} from './policy/categories'
import { useConsoleForms } from './use-console-forms'
import {
  closeInspector as closeInspectorState,
  createInspectorState,
  openInspector as openInspectorState,
  useInspectorPayload,
  type InspectorKind,
} from './use-console-inspector'
import { useConsoleNavigation } from './use-console-navigation'
import { useConsoleNotices } from './use-console-notices'
import { useConsoleSubmitActions } from './use-console-submit-actions'
import type { StuhelperConsoleData } from '../src/console-types'

const REVIEW_QUEUE_ID = 'review'
const REPORT_QUEUE_ID = 'report'
const MEMBER_QUEUE_ID = 'member'

export function useConsolePage() {
  const data = computed(
    () => (store as Record<string, unknown>).stuhelperConsole as StuhelperConsoleData | undefined,
  )
  const title = computed(() => data.value?.title || 'StuHelper 群管中心')
  const generatedAt = computed(() => data.value?.generatedAt || '')
  const visibleReviewIds = ref<string[]>([])

  const {
    routeState,
    setRouteState,
    getSelectedQueueId,
    selectQueueItem,
  } = useConsoleNavigation()

  const inspector = createInspectorState()
  const inspectorPayload = useInspectorPayload(data, inspector)
  const { loading, notices, pushNotice, dismissNotice, runTask } = useConsoleNotices()
  const {
    selectedGuardIds,
    guardForm,
    reviewForm,
    ruleForm,
    templateForm,
    bindingForm,
    roleForm,
    policyForm,
    loadRule,
    loadMemberRoles,
    loadPolicy,
    loadTemplate,
    loadBinding,
  } = useConsoleForms()

  const pendingMembers = computed(() => data.value?.pendingMembers || [])
  const pendingReviews = computed(() => data.value?.pendingReviews || [])
  const keywordRules = computed(() => data.value?.keywordRules || [])
  const commandPolicies = computed(() => data.value?.commandPolicies || [])
  const memberRoles = computed(() => data.value?.memberRoles || [])
  const guardTemplates = computed(() => data.value?.guardTemplates || [])
  const guardBindings = computed(() => data.value?.guardBindings || [])
  const recentEvents = computed(() => data.value?.recentEvents || [])
  const recentReports = computed(() => data.value?.recentReports || [])
  const dashboardModel = computed(() => buildDashboardModel(data.value))
  const supportedCommandIds = computed(() => data.value?.supportedCommandIds || [])
  const selectedMemberId = computed(() => getSelectedQueueId('identity', MEMBER_QUEUE_ID))
  const selectedReviewId = computed(() => getSelectedQueueId('enforcement', REVIEW_QUEUE_ID))
  const selectedReportId = computed(() => getSelectedQueueId('enforcement', REPORT_QUEUE_ID))
  const activePolicyCategory = computed(() =>
    routeState.value.section === 'policy'
      ? resolvePolicyCategoryId(routeState.value.queue)
      : DEFAULT_POLICY_CATEGORY_ID,
  )

  function openInspector(kind: InspectorKind, id: string, reviewCandidateIds: readonly string[] = []) {
    openInspectorState(inspector, kind, id, reviewCandidateIds)
  }

  function closeInspector() {
    closeInspectorState(inspector)
  }

  const {
    selectedAuditId,
    auditKind,
    auditQuery,
    auditRows,
    inspectAuditRow,
  } = useConsoleAudit({
    routeState,
    setRouteState,
    recentEvents,
    recentReports,
    inspector,
    closeInspector,
    openInspector,
    pushNotice,
  })

  const {
    submitGuardAction,
    submitReviewAction,
    submitReviewAndFocus,
    submitRule,
    submitRoles,
    submitPolicy,
    submitTemplate,
    submitBinding,
    inspectMember,
    inspectReview,
    inspectReport,
  } = useConsoleSubmitActions({
    selectedGuardIds,
    guardForm,
    reviewForm,
    ruleForm,
    templateForm,
    bindingForm,
    roleForm,
    policyForm,
    pendingReviews,
    visibleReviewIds,
    inspector,
    openInspector,
    closeInspector,
    setRouteState,
    selectQueueItem,
    runGuardBatchAction,
    runReviewAction,
    saveKeywordRule,
    saveMemberRoles,
    saveCommandPolicy,
    saveGuardTemplate,
    saveGuardBinding,
  })

  watch(inspectorPayload, (payload) => {
    if (!inspector.open || !data.value) return
    if (payload) return
    closeInspector()
  })

  async function refresh() {
    return refreshConsoleData()
  }

  function setVisibleReviewIds(ids: readonly string[]) {
    visibleReviewIds.value = [...ids]
  }

  function selectPolicyCategory(category: PolicyCategoryId) {
    if (
      routeState.value.section === 'policy'
      && activePolicyCategory.value === category
    ) {
      return
    }

    closeInspector()
    setRouteState({
      section: 'policy',
      queue: category,
      id: '',
      source: 'nav',
    })
  }

  function openDashboardTarget(target: DashboardTarget) {
    closeInspector()
    setRouteState(target)
  }

  return {
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
    keywordRules,
    commandPolicies,
    memberRoles,
    guardTemplates,
    guardBindings,
    recentEvents,
    recentReports,
    dashboardModel,
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
    submitReviewAction,
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
  }
}
