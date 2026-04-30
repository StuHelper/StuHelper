import type {
  CommandPolicyRecord,
  ModerationEventRecord,
  ModerationReportRecord,
  ReviewQueueRecord,
} from '@stuhelper/koishi-moderation-core'
import type {
  GuardGroupBindingRecord,
  GuardMemberRecord,
  GuardTemplateRecord,
  QQVerificationStatus,
} from '@stuhelper/koishi-shared'

export interface ModuleStateSnapshot {
  name: string
  description: string
  state: 'unloaded' | 'loading' | 'loaded' | 'error'
  error?: string
}

export interface SerializedGuardMember extends Omit<GuardMemberRecord, 'joinedAt' | 'deadlineAt' | 'mutedAt' | 'reminderSentAt' | 'releasedAt' | 'kickedAt' | 'createdAt' | 'updatedAt'> {
  joinedAt: string
  deadlineAt: string
  mutedAt: string | null
  reminderSentAt: string | null
  releasedAt: string | null
  kickedAt: string | null
  createdAt: string
  updatedAt: string
}

export interface SerializedReview extends Omit<ReviewQueueRecord, 'createdAt' | 'updatedAt'> {
  createdAt: string
  updatedAt: string
}

export interface SerializedReport extends Omit<ModerationReportRecord, 'createdAt' | 'updatedAt'> {
  createdAt: string
  updatedAt: string
}

export interface SerializedEvent extends Omit<ModerationEventRecord, 'createdAt' | 'updatedAt'> {
  createdAt: string
  updatedAt: string
}

export interface SerializedGuardTemplate extends Omit<GuardTemplateRecord, 'createdAt' | 'updatedAt'> {
  createdAt: string
  updatedAt: string
}

export interface SerializedGuardBinding extends Omit<GuardGroupBindingRecord, 'createdAt' | 'updatedAt'> {
  createdAt: string
  updatedAt: string
}

export interface SerializedCommandPolicy extends Omit<CommandPolicyRecord, 'createdAt' | 'updatedAt'> {
  createdAt: string
  updatedAt: string
}

export interface DashboardOverview {
  pendingReviews: number
  pendingAdmissions: number
  openReports: number
  highRiskEvents: number
  policyItems: number
}

export interface DashboardPageData {
  generatedAt: string
  overview: DashboardOverview
  pendingMembers: SerializedGuardMember[]
  pendingReviews: SerializedReview[]
  recentEvents: SerializedEvent[]
  recentReports: SerializedReport[]
  commandPolicies: SerializedCommandPolicy[]
  guardTemplates: SerializedGuardTemplate[]
  guardBindings: SerializedGuardBinding[]
  systemStatus: ModuleStateSnapshot[]
}

export interface IdentityLookupError {
  memberId: string
  message: string
}

export interface IdentityMemberSnapshot extends SerializedGuardMember {
  profile: QQVerificationStatus | null
}

export interface IdentityGroupSummary {
  guildId: string
  memberCount: number
  pendingCount: number
  releasedCount: number
}

export interface IdentitySummary {
  pendingMembers: number
  verifiedMembers: number
  boundUnverifiedMembers: number
  unboundMembers: number
  releasedMembers: number
}

export interface IdentityPageData {
  generatedAt: string
  summary: IdentitySummary
  groups: IdentityGroupSummary[]
  members: IdentityMemberSnapshot[]
  recentReleases: IdentityMemberSnapshot[]
  lookupErrors: IdentityLookupError[]
}

export type ReviewWorkItemKind = 'review' | 'admission' | 'report'
export type ReviewWorkItemAction =
  | 'execute'
  | 'reject'
  | 'approve'
  | 'deny'
  | 'defer'
  | 'dismiss'
  | 'escalate'
  | 'create-review'

export interface ReviewWorkItem {
  id: string
  kind: ReviewWorkItemKind
  guildId: string
  memberId: string | null
  subjectLabel: string
  status: string
  priority: 'low' | 'medium' | 'high' | 'critical'
  createdAt: string
  availableActions: ReviewWorkItemAction[]
  relatedEventIds: string[]
  reason: string
  secondaryLabel: string
}

export interface ReviewPageData {
  generatedAt: string
  items: ReviewWorkItem[]
  events: SerializedEvent[]
}

export interface GovernanceSource {
  kind: 'template-library' | 'binding' | 'command-policy'
  label: string
}

export interface ConfigWorkspace {
  id: 'guild-config' | 'templates' | 'bindings' | 'command-policies'
  label: string
}

export interface ConfigGroupSnapshot {
  guildId: string
  guildName: string
  guildAvatar: string
  config: Record<string, unknown>
}

export interface GovernanceTemplateSnapshot extends SerializedGuardTemplate {
  source: GovernanceSource
}

export interface GovernanceBindingSnapshot extends SerializedGuardBinding {
  effectiveTemplateName: string
}

export interface ConfigGovernancePageData {
  generatedAt: string
  workspaces: ConfigWorkspace[]
  groupConfigs: ConfigGroupSnapshot[]
  templates: GovernanceTemplateSnapshot[]
  bindings: GovernanceBindingSnapshot[]
  commandPolicies: SerializedCommandPolicy[]
  supportedCommandIds: string[]
}

export interface EntityProfileQuery {
  kind: 'user' | 'guild'
  id: string
  /** Optional guild context when kind='user', used to bias which guild's data is foregrounded. */
  guildId?: string
}

export interface EntityWarnFact {
  guildId: string
  guildName: string | null
  userId: string
  count: number
}

export interface EntityBlacklistFact {
  userId: string
  reason: string | null
  addedAt: string | null
}

export interface EntityRestrictedFact {
  guildId: string
  guildName: string | null
  memberId: string
  status: 'pending' | 'released' | 'kicked'
  joinedAt: string
  deadlineAt: string
}

export interface EntityReviewFact {
  id: string
  guildId: string
  guildName: string | null
  memberId: string | null
  status: string
  actionType: string
  reason: string
  createdAt: string
}

export interface EntityReportFact {
  id: string
  guildId: string
  guildName: string | null
  targetMemberId: string
  reporterMemberId: string
  reason: string
  status: string
  createdAt: string
}

export interface EntityEventFact {
  id: string
  guildId: string
  guildName: string | null
  memberId: string | null
  kind: string
  severity: 'info' | 'warning' | 'high' | 'critical'
  message: string
  createdAt: string
}

export interface UserEntityProfileSummary {
  activeWarnGuilds: number
  totalWarns: number
  blacklisted: boolean
  pendingReviews: number
  openReports: number
  restrictedGuilds: number
}

export interface UserEntityProfile {
  kind: 'user'
  generatedAt: string
  id: string
  displayName: string | null
  avatar: string | null
  summary: UserEntityProfileSummary
  warns: EntityWarnFact[]
  blacklist: EntityBlacklistFact | null
  restricted: EntityRestrictedFact[]
  reviews: EntityReviewFact[]
  reports: EntityReportFact[]
  recentEvents: EntityEventFact[]
}

export interface GuildEntityProfileSummary {
  configured: boolean
  pendingMembers: number
  warnedUsers: number
  pendingReviews: number
  openReports: number
}

export interface GuildEntityProfile {
  kind: 'guild'
  generatedAt: string
  id: string
  name: string | null
  avatar: string | null
  summary: GuildEntityProfileSummary
  warns: EntityWarnFact[]
  restricted: EntityRestrictedFact[]
  reviews: EntityReviewFact[]
  reports: EntityReportFact[]
  recentEvents: EntityEventFact[]
}

export type EntityProfile = UserEntityProfile | GuildEntityProfile
