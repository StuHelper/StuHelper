import { getNextFocusableId } from './queue/model'
import { splitTokens } from './use-console-forms'
import type { InspectorKind, InspectorState } from './use-console-inspector'
import type { ConsoleSearchState } from './navigation'
import type {
  StuhelperCommandPolicyInput,
  StuhelperGuardBindingInput,
  StuhelperGuardTemplateInput,
  StuhelperConsoleGuardMember,
  StuhelperConsoleReport,
  StuhelperConsoleReview,
  StuhelperKeywordRuleInput,
} from '../src/console-types'

const REVIEW_QUEUE_ID = 'review'
const REPORT_QUEUE_ID = 'report'
const MEMBER_QUEUE_ID = 'member'

interface RefLike<T> {
  value: T
}

interface GuardFormState {
  action: 'mute' | 'unmute' | 'kick' | 'set-role' | 'unset-role'
  seconds: number
  reason: string
  roleId: string
  permanent: boolean
}

interface ReviewFormState {
  note: string
}

interface TemplateFormState {
  id: string
  name: string
  muteDurationSeconds: number
  kickAfterMinutes: number
  reminderTemplate: string
  exemptUsersText: string
  enabled: boolean
}

interface BindingFormState {
  platform: string
  guildId: string
  templateId: string
  enabled: boolean
  note: string
}

interface RoleFormState {
  guildId: string
  memberId: string
  rolesText: string
}

interface UseConsoleSubmitActionsOptions {
  selectedGuardIds: RefLike<string[]>
  guardForm: GuardFormState
  reviewForm: ReviewFormState
  ruleForm: StuhelperKeywordRuleInput
  templateForm: TemplateFormState
  bindingForm: BindingFormState
  roleForm: RoleFormState
  policyForm: StuhelperCommandPolicyInput & { rolesText: string }
  pendingReviews: RefLike<readonly StuhelperConsoleReview[]>
  visibleReviewIds: RefLike<readonly string[]>
  inspector: InspectorState
  openInspector: (kind: InspectorKind, id: string, reviewCandidateIds?: readonly string[]) => void
  closeInspector: () => void
  setRouteState: (
    next: Partial<ConsoleSearchState>,
    options?: { history?: 'push' | 'replace' },
  ) => void
  selectQueueItem: (section: ConsoleSearchState['section'], queue: string, id: string) => void
  runGuardBatchAction: (input: {
    action: GuardFormState['action']
    memberIds: string[]
    seconds?: number
    reason: string
    roleId?: string
    permanent?: boolean
  }) => Promise<unknown>
  runReviewAction: (input: { reviewId: string; action: 'execute' | 'reject'; note?: string }) => Promise<unknown>
  saveKeywordRule: (input: StuhelperKeywordRuleInput) => Promise<unknown>
  saveMemberRoles: (input: { guildId: string; memberId: string; roles: string[] }) => Promise<unknown>
  saveCommandPolicy: (input: StuhelperCommandPolicyInput) => Promise<unknown>
  saveGuardTemplate: (input: StuhelperGuardTemplateInput) => Promise<unknown>
  saveGuardBinding: (input: StuhelperGuardBindingInput) => Promise<unknown>
}

export function useConsoleSubmitActions(options: UseConsoleSubmitActionsOptions) {
  async function submitGuardAction() {
    const result = await options.runGuardBatchAction({
      action: options.guardForm.action,
      memberIds: options.selectedGuardIds.value,
      seconds: options.guardForm.seconds,
      reason: options.guardForm.reason,
      roleId: options.guardForm.roleId || undefined,
      permanent: options.guardForm.permanent,
    })
    options.selectedGuardIds.value = []
    return result
  }

  async function submitReviewAction(reviewId: string, action: 'execute' | 'reject') {
    const result = await options.runReviewAction({
      reviewId,
      action,
      note: options.reviewForm.note.trim() || undefined,
    })
    options.reviewForm.note = ''
    return result
  }

  async function submitReviewAndFocus(
    reviewId: string,
    action: 'execute' | 'reject',
    visibleIds?: readonly string[],
  ) {
    const candidateIds = resolveReviewCandidateIds(reviewId, visibleIds)
    const nextId = getNextFocusableId({
      ids: candidateIds,
      currentId: reviewId,
      removedId: reviewId,
    })
    const result = await submitReviewAction(reviewId, action)

    options.setRouteState({
      section: 'enforcement',
      queue: REVIEW_QUEUE_ID,
      id: nextId,
      source: 'direct',
    }, { history: 'replace' })

    if (!nextId) {
      options.closeInspector()
      return result
    }

    const nextReview = options.pendingReviews.value.find((review) => review.id === nextId)
    if (!nextReview) {
      options.closeInspector()
      return result
    }

    options.openInspector('review', nextReview.id, candidateIds)
    return result
  }

  async function submitRule() {
    return options.saveKeywordRule({
      ...options.ruleForm,
      note: options.ruleForm.note || null,
    })
  }

  async function submitRoles() {
    return options.saveMemberRoles({
      guildId: options.roleForm.guildId,
      memberId: options.roleForm.memberId,
      roles: splitTokens(options.roleForm.rolesText),
    })
  }

  async function submitPolicy() {
    return options.saveCommandPolicy({
      commandId: options.policyForm.commandId,
      minAuthority: options.policyForm.minAuthority,
      roles: splitTokens(options.policyForm.rolesText),
    })
  }

  async function submitTemplate() {
    return options.saveGuardTemplate({
      id: options.templateForm.id.trim(),
      name: options.templateForm.name.trim(),
      muteDurationSeconds: options.templateForm.muteDurationSeconds,
      kickAfterMinutes: options.templateForm.kickAfterMinutes,
      reminderTemplate: options.templateForm.reminderTemplate.trim(),
      exemptUsers: splitTokens(options.templateForm.exemptUsersText),
      enabled: options.templateForm.enabled,
    })
  }

  async function submitBinding() {
    return options.saveGuardBinding({
      platform: options.bindingForm.platform.trim(),
      guildId: options.bindingForm.guildId.trim(),
      templateId: options.bindingForm.templateId.trim(),
      enabled: options.bindingForm.enabled,
      note: options.bindingForm.note.trim() || null,
    })
  }

  function inspectMember(member: StuhelperConsoleGuardMember) {
    options.selectQueueItem('identity', MEMBER_QUEUE_ID, member.id)
    options.openInspector('member', member.id)
  }

  function inspectReview(
    review: StuhelperConsoleReview,
    reviewCandidateIds: readonly string[] = options.visibleReviewIds.value,
  ) {
    options.selectQueueItem('enforcement', REVIEW_QUEUE_ID, review.id)
    options.openInspector('review', review.id, reviewCandidateIds)
  }

  function inspectReport(report: StuhelperConsoleReport) {
    options.selectQueueItem('enforcement', REPORT_QUEUE_ID, report.id)
    options.openInspector('report', report.id)
  }

  function resolveReviewCandidateIds(
    reviewId: string,
    visibleIds?: readonly string[],
  ) {
    if (visibleIds?.length) return [...visibleIds]
    if (options.inspector.kind === 'review' && options.inspector.id === reviewId && options.inspector.reviewCandidateIds.length) {
      return [...options.inspector.reviewCandidateIds]
    }
    if (options.visibleReviewIds.value.length) return [...options.visibleReviewIds.value]
    return options.pendingReviews.value.map((review) => review.id)
  }

  return {
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
  }
}
