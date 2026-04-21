import type {
  AIReviewSeverity,
  KeywordActionType,
  ModerationRiskLevel,
  ReviewActionType,
  ReviewStatus,
} from '@stuhelper/koishi-moderation-core'
import type { PlatformVerificationState } from '@stuhelper/koishi-shared'

export type ActionIntent =
  | 'neutral'
  | 'primary'
  | 'warning'
  | 'danger'
  | 'success'
  | 'info'
  | 'muted'

type ConsoleAction =
  | KeywordActionType
  | ReviewActionType
  | 'mute'
  | 'set-role'
  | 'unset-role'
  | 'unmute'

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

export function describeReviewAction(action: ReviewActionType) {
  switch (action) {
    case 'kick':
      return '踢出成员'
    case 'kick_and_block':
      return '踢出并拉黑'
  }
}

export function describeReviewStatus(status: ReviewStatus) {
  switch (status) {
    case 'pending':
      return '待复核'
    case 'approved':
      return '已批准'
    case 'rejected':
      return '已驳回'
    case 'executed':
      return '已执行'
  }
}

export function describeVerificationState(state: PlatformVerificationState) {
  switch (state) {
    case 'unbound':
      return '未绑定'
    case 'bound_unverified':
      return '待认证'
    case 'verified':
      return '已认证'
  }
}

export function describeAction(action: ConsoleAction): { label: string; intent: ActionIntent } {
  switch (action) {
    case 'mute':
      return { label: '禁言', intent: 'warning' }
    case 'unmute':
      return { label: '解除禁言', intent: 'success' }
    case 'kick':
      return { label: '踢出成员', intent: 'danger' }
    case 'kick_and_block':
      return { label: '踢出并拉黑', intent: 'danger' }
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
  }
}

export function describeLevel(level: ModerationRiskLevel | AIReviewSeverity | string): ActionIntent {
  switch (level) {
    case 'critical':
    case 'high':
      return 'danger'
    case 'medium':
    case 'warn':
    case 'warning':
      return 'warning'
    case 'info':
    case 'low':
      return 'info'
    case 'success':
    case 'ok':
      return 'success'
    case 'none':
      return 'muted'
    default:
      return 'neutral'
  }
}
