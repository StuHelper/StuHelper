import type {
  CommandPolicyRecord,
  KeywordRuleRecord,
  MemberRoleRecord,
  ModerationEventRecord,
  ModerationOverview,
  ModerationReportRecord,
  ReviewQueueRecord,
} from '@stuhelper/koishi-moderation-core'
import type {
  GuardGroupBindingRecord,
  GuardMemberRecord,
  GuardTemplateRecord,
} from '@stuhelper/koishi-shared'

export interface StuhelperConsoleOverview extends Omit<ModerationOverview, 'recentEvents'> {}

export interface StuhelperConsoleGuardMember extends Omit<GuardMemberRecord, 'joinedAt' | 'deadlineAt' | 'mutedAt' | 'reminderSentAt' | 'releasedAt' | 'kickedAt' | 'createdAt' | 'updatedAt'> {
  joinedAt: string
  deadlineAt: string
  mutedAt: string | null
  reminderSentAt: string | null
  releasedAt: string | null
  kickedAt: string | null
  createdAt: string
  updatedAt: string
}

export interface StuhelperConsoleReview extends Omit<ReviewQueueRecord, 'createdAt' | 'updatedAt'> {
  createdAt: string
  updatedAt: string
}

export interface StuhelperConsoleKeywordRule extends Omit<KeywordRuleRecord, 'createdAt' | 'updatedAt'> {
  createdAt: string
  updatedAt: string
}

export interface StuhelperKeywordRuleInput {
  id: string
  guildId: string
  pattern: string
  matchMode: KeywordRuleRecord['matchMode']
  action: KeywordRuleRecord['action']
  enabled: boolean
  muteSeconds: number
  note?: string | null
}

export interface StuhelperConsoleCommandPolicy extends Omit<CommandPolicyRecord, 'createdAt' | 'updatedAt'> {
  createdAt: string
  updatedAt: string
}

export interface StuhelperConsoleMemberRole extends Omit<MemberRoleRecord, 'createdAt' | 'updatedAt'> {
  createdAt: string
  updatedAt: string
}

export interface StuhelperConsoleEvent extends Omit<ModerationEventRecord, 'createdAt' | 'updatedAt'> {
  createdAt: string
  updatedAt: string
}

export interface StuhelperConsoleReport extends Omit<ModerationReportRecord, 'createdAt' | 'updatedAt'> {
  createdAt: string
  updatedAt: string
}

export interface StuhelperConsoleGuardTemplate extends Omit<GuardTemplateRecord, 'createdAt' | 'updatedAt'> {
  createdAt: string
  updatedAt: string
}

export interface StuhelperConsoleGuardBinding extends Omit<GuardGroupBindingRecord, 'createdAt' | 'updatedAt'> {
  createdAt: string
  updatedAt: string
}

export interface StuhelperConsoleData {
  title: string
  generatedAt: string
  supportedCommandIds: string[]
  overview: StuhelperConsoleOverview
  pendingMembers: StuhelperConsoleGuardMember[]
  pendingReviews: StuhelperConsoleReview[]
  keywordRules: StuhelperConsoleKeywordRule[]
  commandPolicies: StuhelperConsoleCommandPolicy[]
  memberRoles: StuhelperConsoleMemberRole[]
  guardTemplates: StuhelperConsoleGuardTemplate[]
  guardBindings: StuhelperConsoleGuardBinding[]
  recentEvents: StuhelperConsoleEvent[]
  recentReports: StuhelperConsoleReport[]
}

export interface StuhelperConsoleServiceConfig {
  title: string
}

export interface StuhelperGuardBatchActionInput {
  action: 'mute' | 'unmute' | 'kick' | 'set-role' | 'unset-role'
  memberIds: string[]
  seconds?: number
  reason: string
  roleId?: string
  permanent?: boolean
}

export interface StuhelperReviewActionInput {
  reviewId: string
  action: 'execute' | 'reject'
  note?: string
}

export interface StuhelperMemberRoleInput {
  guildId: string
  memberId: string
  roles: string[]
}

export interface StuhelperCommandPolicyInput {
  commandId: string
  roles: string[]
  minAuthority: number
}

export interface StuhelperGuardTemplateInput {
  id: string
  name: string
  muteDurationSeconds: number
  kickAfterMinutes: number
  reminderTemplate: string
  exemptUsers: string[]
  enabled: boolean
}

export interface StuhelperGuardBindingInput {
  platform: string
  guildId: string
  templateId: string
  enabled: boolean
  note?: string | null
}
