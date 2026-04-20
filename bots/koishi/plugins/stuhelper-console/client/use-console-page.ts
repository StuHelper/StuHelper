import { computed, reactive, ref } from 'vue'
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
import {
  parseConsoleSearch,
  updateConsoleUrl,
  type ConsoleSearchState,
} from './navigation'
import type {
  StuhelperConsoleCommandPolicy,
  StuhelperConsoleData,
  StuhelperConsoleEvent,
  StuhelperConsoleGuardBinding,
  StuhelperConsoleGuardMember,
  StuhelperConsoleGuardTemplate,
  StuhelperConsoleKeywordRule,
  StuhelperConsoleMemberRole,
  StuhelperConsoleReport,
  StuhelperConsoleReview,
} from '../src/console-types'
import {
  buildDashboardModel,
  type DashboardTarget,
} from './dashboard/model'
import {
  DEFAULT_POLICY_CATEGORY_ID,
  resolvePolicyCategoryId,
  type PolicyCategoryId,
} from './policy/categories'
import { getNextFocusableId } from './queue/model'

export type InspectorKind =
  | 'member'
  | 'review'
  | 'event'
  | 'report'
  | 'template'
  | 'binding'
  | 'rule'

export interface InspectorState {
  open: boolean
  kind: InspectorKind | null
  id: string
  payload: unknown
}

export interface NoticeItem {
  id: string
  kind: 'success' | 'error'
  message: string
}

const NOTICE_TTL_MS = 4000
const REVIEW_QUEUE_ID = 'review'
const MEMBER_QUEUE_ID = 'member'

export function useConsolePage() {
  const data = computed(
    () => (store as Record<string, unknown>).stuhelperConsole as StuhelperConsoleData | undefined,
  )
  const title = computed(() => data.value?.title || 'StuHelper 群管中心')
  const generatedAt = computed(() => data.value?.generatedAt || '')
  const loading = ref(false)

  const routeState = ref<ConsoleSearchState>(
    parseConsoleSearch(new URLSearchParams(window.location.search)),
  )
  function setRouteState(next: Partial<ConsoleSearchState>) {
    routeState.value = { ...routeState.value, ...next }
    const url = updateConsoleUrl(new URL(window.location.href), routeState.value)
    window.history.replaceState(window.history.state, '', url)
  }

  function getSelectedQueueId(section: ConsoleSearchState['section'], queue: string) {
    if (routeState.value.section !== section) return ''
    if (routeState.value.queue && routeState.value.queue !== queue) return ''
    return routeState.value.id
  }

  function selectQueueItem(
    section: ConsoleSearchState['section'],
    queue: string,
    id: string,
  ) {
    setRouteState({ section, queue, id, source: 'direct' })
  }

  const inspector = reactive<InspectorState>({
    open: false,
    kind: null,
    id: '',
    payload: null,
  })

  function openInspector(kind: InspectorKind, id: string, payload: unknown) {
    inspector.open = true
    inspector.kind = kind
    inspector.id = id
    inspector.payload = payload
  }

  function closeInspector() {
    inspector.open = false
    inspector.kind = null
    inspector.id = ''
    inspector.payload = null
  }

  const notices = ref<NoticeItem[]>([])

  function pushNotice(kind: NoticeItem['kind'], message: string) {
    const id = `${Date.now()}-${Math.random().toString(36).slice(2, 7)}`
    notices.value = [...notices.value, { id, kind, message }]
    window.setTimeout(() => dismissNotice(id), NOTICE_TTL_MS)
  }

  function dismissNotice(id: string) {
    notices.value = notices.value.filter((item) => item.id !== id)
  }

  const selectedGuardIds = ref<string[]>([])
  const guardForm = reactive({
    action: 'mute' as 'mute' | 'unmute' | 'kick' | 'set-role' | 'unset-role',
    seconds: 600,
    reason: '控制台批量操作',
    roleId: '',
    permanent: false,
  })
  const reviewForm = reactive({ note: '' })
  const ruleForm = reactive({
    id: '',
    guildId: '*',
    pattern: '',
    matchMode: 'includes' as 'includes' | 'regex',
    action: 'warn' as 'warn' | 'delete' | 'mute' | 'review',
    enabled: true,
    muteSeconds: 0,
    note: '',
  })
  const templateForm = reactive({
    id: '',
    name: '',
    muteDurationSeconds: 600,
    kickAfterMinutes: 30,
    reminderTemplate: '请先完成 StuHelper 注册、QQ 绑定与学生认证。',
    exemptUsersText: '',
    enabled: true,
  })
  const bindingForm = reactive({
    platform: '',
    guildId: '',
    templateId: '',
    enabled: true,
    note: '',
  })
  const roleForm = reactive({ guildId: '', memberId: '', rolesText: '' })
  const policyForm = reactive({ commandId: 'report', minAuthority: 0, rolesText: '' })

  const eventSearch = ref('')
  const reportSearch = ref('')
  const visibleReviewIds = ref<string[]>([])

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
  const activePolicyCategory = computed(() =>
    routeState.value.section === 'policy'
      ? resolvePolicyCategoryId(routeState.value.queue)
      : DEFAULT_POLICY_CATEGORY_ID,
  )

  const filteredEvents = computed(() => {
    const q = eventSearch.value.trim().toLowerCase()
    if (!q) return recentEvents.value
    return recentEvents.value.filter((event) =>
      [event.memberId, event.summary, event.level, event.guildId]
        .filter(Boolean)
        .some((field) => String(field).toLowerCase().includes(q)),
    )
  })

  const filteredReports = computed(() => {
    const q = reportSearch.value.trim().toLowerCase()
    if (!q) return recentReports.value
    return recentReports.value.filter((report) =>
      [report.reporterMemberId, report.targetMemberId, report.reason, report.aiSummary]
        .filter(Boolean)
        .some((field) => String(field).toLowerCase().includes(q)),
    )
  })

  async function runTask(task: () => Promise<unknown>) {
    loading.value = true
    try {
      const result = await task()
      const message = typeof result === 'string' && result.trim() ? result : '操作已提交并刷新。'
      pushNotice('success', message)
      return result
    } catch (error) {
      const message = error instanceof Error ? error.message : String(error)
      pushNotice('error', message)
      throw error
    } finally {
      loading.value = false
    }
  }

  async function refresh() {
    return refreshConsoleData()
  }

  async function submitGuardAction() {
    const result = await runGuardBatchAction({
      action: guardForm.action,
      memberIds: selectedGuardIds.value,
      seconds: guardForm.seconds,
      reason: guardForm.reason,
      roleId: guardForm.roleId || undefined,
      permanent: guardForm.permanent,
    })
    selectedGuardIds.value = []
    return result
  }

  async function submitReviewAction(reviewId: string, action: 'execute' | 'reject') {
    const result = await runReviewAction({
      reviewId,
      action,
      note: reviewForm.note.trim() || undefined,
    })
    reviewForm.note = ''
    return result
  }

  async function submitReviewAndFocus(
    reviewId: string,
    action: 'execute' | 'reject',
    visibleIds = visibleReviewIds.value.length
      ? visibleReviewIds.value
      : pendingReviews.value.map((review) => review.id),
  ) {
    const nextId = getNextFocusableId({
      ids: visibleIds,
      currentId: reviewId,
      removedId: reviewId,
    })
    const result = await submitReviewAction(reviewId, action)

    setRouteState({
      section: 'enforcement',
      queue: REVIEW_QUEUE_ID,
      id: nextId,
      source: 'direct',
    })

    if (!nextId) {
      closeInspector()
      return result
    }

    const nextReview = pendingReviews.value.find((review) => review.id === nextId)
    if (!nextReview) {
      closeInspector()
      return result
    }

    openInspector('review', nextReview.id, nextReview)
    return result
  }

  async function submitRule() {
    return saveKeywordRule({
      ...ruleForm,
      note: ruleForm.note || null,
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString(),
    })
  }

  async function submitRoles() {
    return saveMemberRoles({
      guildId: roleForm.guildId,
      memberId: roleForm.memberId,
      roles: splitTokens(roleForm.rolesText),
    })
  }

  async function submitPolicy() {
    return saveCommandPolicy({
      commandId: policyForm.commandId,
      minAuthority: policyForm.minAuthority,
      roles: splitTokens(policyForm.rolesText),
    })
  }

  async function submitTemplate() {
    return saveGuardTemplate({
      id: templateForm.id.trim(),
      name: templateForm.name.trim(),
      muteDurationSeconds: templateForm.muteDurationSeconds,
      kickAfterMinutes: templateForm.kickAfterMinutes,
      reminderTemplate: templateForm.reminderTemplate.trim(),
      exemptUsers: splitTokens(templateForm.exemptUsersText),
      enabled: templateForm.enabled,
    })
  }

  async function submitBinding() {
    return saveGuardBinding({
      platform: bindingForm.platform.trim(),
      guildId: bindingForm.guildId.trim(),
      templateId: bindingForm.templateId.trim(),
      enabled: bindingForm.enabled,
      note: bindingForm.note.trim() || null,
    })
  }

  function loadRule(rule: StuhelperConsoleKeywordRule) {
    Object.assign(ruleForm, { ...rule, note: rule.note || '' })
  }

  function loadMemberRoles(entry: StuhelperConsoleMemberRole) {
    roleForm.guildId = entry.guildId
    roleForm.memberId = entry.memberId
    roleForm.rolesText = entry.roles.join(', ')
  }

  function loadPolicy(policy: StuhelperConsoleCommandPolicy) {
    policyForm.commandId = policy.commandId
    policyForm.minAuthority = policy.minAuthority
    policyForm.rolesText = policy.roles.join(', ')
  }

  function loadTemplate(template: StuhelperConsoleGuardTemplate) {
    templateForm.id = template.id
    templateForm.name = template.name
    templateForm.muteDurationSeconds = template.muteDurationSeconds
    templateForm.kickAfterMinutes = template.kickAfterMinutes
    templateForm.reminderTemplate = template.reminderTemplate
    templateForm.exemptUsersText = template.exemptUsers.join(', ')
    templateForm.enabled = template.enabled
  }

  function loadBinding(binding: StuhelperConsoleGuardBinding) {
    bindingForm.platform = binding.platform
    bindingForm.guildId = binding.guildId
    bindingForm.templateId = binding.templateId
    bindingForm.enabled = binding.enabled
    bindingForm.note = binding.note || ''
  }

  function inspectMember(member: StuhelperConsoleGuardMember) {
    selectQueueItem('identity', MEMBER_QUEUE_ID, member.id)
    openInspector('member', member.id, member)
  }

  function inspectReview(review: StuhelperConsoleReview) {
    selectQueueItem('enforcement', REVIEW_QUEUE_ID, review.id)
    openInspector('review', review.id, review)
  }

  function inspectEvent(event: StuhelperConsoleEvent) {
    openInspector('event', event.id, event)
  }

  function inspectReport(report: StuhelperConsoleReport) {
    openInspector('report', report.id, report)
  }

  function setVisibleReviewIds(ids: readonly string[]) {
    visibleReviewIds.value = [...ids]
  }

  function selectPolicyCategory(category: PolicyCategoryId) {
    if (
      routeState.value.section === 'policy' &&
      activePolicyCategory.value === category
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
    setRouteState({
      section: target.section,
      queue: target.section === 'policy' ? activePolicyCategory.value : null,
      id: '',
      source: 'dashboard',
    })
  }

  return {
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
    openDashboardTarget,
    notices,
    pushNotice,
    dismissNotice,
    pendingMembers,
    pendingReviews,
    selectedMemberId,
    selectedReviewId,
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
  }
}

function splitTokens(input: string) {
  return input
    .split(',')
    .map((item) => item.trim())
    .filter(Boolean)
}

export function formatTimestamp(value?: string | null) {
  if (!value) return '—'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  const mm = String(date.getMonth() + 1).padStart(2, '0')
  const dd = String(date.getDate()).padStart(2, '0')
  const hh = String(date.getHours()).padStart(2, '0')
  const min = String(date.getMinutes()).padStart(2, '0')
  return `${mm}-${dd} ${hh}:${min}`
}

export type ActionIntent =
  | 'neutral'
  | 'primary'
  | 'warning'
  | 'danger'
  | 'success'
  | 'info'
  | 'muted'

export function describeAction(action: string): { label: string; intent: ActionIntent } {
  switch (action) {
    case 'mute':
      return { label: '禁言', intent: 'warning' }
    case 'unmute':
      return { label: '解除禁言', intent: 'success' }
    case 'kick':
      return { label: '踢人（复核）', intent: 'danger' }
    case 'kick-permanent':
      return { label: '踢人拉黑（复核）', intent: 'danger' }
    case 'set-role':
      return { label: '设置角色', intent: 'primary' }
    case 'unset-role':
      return { label: '移除角色', intent: 'muted' }
    case 'warn':
      return { label: '警告', intent: 'warning' }
    case 'delete':
      return { label: '撤回', intent: 'primary' }
    case 'review':
      return { label: '转复核', intent: 'info' }
    default:
      return { label: action, intent: 'neutral' }
  }
}

export function describeLevel(level: string): ActionIntent {
  switch (level) {
    case 'critical':
    case 'high':
      return 'danger'
    case 'medium':
    case 'warn':
    case 'warning':
      return 'warning'
    case 'info':
      return 'info'
    case 'success':
    case 'ok':
      return 'success'
    default:
      return 'neutral'
  }
}
