export type ModerationEventType =
  | 'join_guarded'
  | 'join_released'
  | 'join_expired'
  | 'keyword_hit'
  | 'repeat_hit'
  | 'message_deleted'
  | 'warning_threshold'
  | 'review_created'
  | 'review_resolved'
  | 'report_created'
  | 'report_ai_reviewed'
  | 'action_executed'

export type ModerationRiskLevel = 'info' | 'low' | 'medium' | 'high' | 'critical'

export type ReviewActionType = 'kick' | 'kick_and_block'
export type ReviewStatus = 'pending' | 'approved' | 'rejected' | 'executed'

export type KeywordMatchMode = 'includes' | 'regex'
export type KeywordActionType = 'warn' | 'delete' | 'mute' | 'review'

export type AIReviewStatus = 'pending' | 'completed' | 'disabled' | 'failed'
export type AIReviewSeverity = 'none' | 'low' | 'medium' | 'high'

export interface ModerationRuntimeRef {
  platform: string
  botSelfId: string
}

export interface ThresholdMetrics {
  warnings: number
  repeats: number
  reports: number
}

export interface KeywordRuleRecord {
  id: string
  guildId: string
  pattern: string
  matchMode: KeywordMatchMode
  action: KeywordActionType
  enabled: boolean
  muteSeconds: number
  note: string | null
  createdAt: Date
  updatedAt: Date
}

export interface KeywordMatchContext {
  guildId: string
  content: string
  normalizedContent: string
}

export interface KeywordHit {
  ruleId: string
  action: KeywordActionType
  muteSeconds: number
  note: string | null
}

export interface MessageLedgerRecord {
  messageId: string
  platform: string
  botSelfId: string
  guildId: string
  channelId: string
  memberId: string
  content: string
  normalizedContent: string
  quoteMessageId: string | null
  createdAt: Date
  deletedAt: Date | null
}

export interface WarningCounterRecord {
  id: string
  guildId: string
  memberId: string
  total: number
  lastReason: string | null
  lastAt: Date | null
  createdAt: Date
  updatedAt: Date
}

export interface ModerationEventRecord {
  id: string
  platform: string
  botSelfId: string
  guildId: string
  channelId: string
  memberId: string
  type: ModerationEventType
  level: ModerationRiskLevel
  summary: string
  payload: Record<string, unknown> | null
  createdAt: Date
  updatedAt: Date
}

export interface ReviewQueueRecord {
  id: string
  platform: string
  botSelfId: string
  guildId: string
  channelId: string
  memberId: string
  actionType: ReviewActionType
  status: ReviewStatus
  reason: string
  operatorMemberId: string | null
  resolutionNote: string | null
  payload: Record<string, unknown> | null
  createdAt: Date
  updatedAt: Date
}

export interface MemberRoleRecord {
  id: string
  guildId: string
  memberId: string
  roles: string[]
  createdAt: Date
  updatedAt: Date
}

export interface CommandPolicyRecord {
  commandId: string
  roles: string[]
  minAuthority: number
  createdAt: Date
  updatedAt: Date
}

export interface FunProfileRecord {
  memberId: string
  muteDrawCount: number
  pityBonus: number
  lastDrawAt: Date | null
  createdAt: Date
  updatedAt: Date
}

export interface ModerationReportRecord {
  id: string
  platform: string
  botSelfId: string
  guildId: string
  channelId: string
  reporterMemberId: string
  targetMemberId: string
  reason: string
  aiStatus: AIReviewStatus
  aiSeverity: AIReviewSeverity
  aiSummary: string | null
  createdAt: Date
  updatedAt: Date
}

export interface ModerationOverview {
  pendingReviews: number
  openReports: number
  warningMembers: number
  highRiskEvents: number
  recentEvents: ModerationEventRecord[]
}
