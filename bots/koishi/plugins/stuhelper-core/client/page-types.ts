export interface ConsoleModuleState {
  name: string
  description: string
  state: 'unloaded' | 'loading' | 'loaded' | 'error'
  error?: string
}

export interface ConsoleGuardMember {
  id: string
  platform: string
  botSelfId: string
  guildId: string
  channelId: string
  memberId: string
  memberName: string
  verificationState: 'unbound' | 'bound_unverified' | 'verified'
  joinedAt: string
  deadlineAt: string
  mutedAt: string | null
  reminderSentAt: string | null
  releasedAt: string | null
  kickedAt: string | null
  lastError: string | null
  createdAt: string
  updatedAt: string
}

export interface ConsoleReviewRecord {
  id: string
  platform: string
  botSelfId: string
  guildId: string
  channelId: string
  memberId: string
  actionType: 'kick' | 'kick_and_block'
  status: 'pending' | 'approved' | 'rejected' | 'executed'
  reason: string
  operatorMemberId: string | null
  resolutionNote: string | null
  payload: Record<string, unknown> | null
  createdAt: string
  updatedAt: string
}

export interface ConsoleReportRecord {
  id: string
  platform: string
  botSelfId: string
  guildId: string
  channelId: string
  reporterMemberId: string
  targetMemberId: string
  reason: string
  aiStatus: 'pending' | 'completed' | 'disabled' | 'failed'
  aiSeverity: 'none' | 'low' | 'medium' | 'high'
  aiSummary: string | null
  createdAt: string
  updatedAt: string
}

export interface ConsoleEventRecord {
  id: string
  platform: string
  botSelfId: string
  guildId: string
  channelId: string
  memberId: string
  type: string
  level: string
  summary: string
  payload: Record<string, unknown> | null
  createdAt: string
  updatedAt: string
}

export interface ConsoleCommandPolicy {
  commandId: string
  roles: string[]
  minAuthority: number
  createdAt: string
  updatedAt: string
}

export interface ConsoleGuardTemplate {
  id: string
  name: string
  muteDurationSeconds: number
  kickAfterMinutes: number
  reminderTemplate: string
  exemptUsers: string[]
  enabled: boolean
  createdAt: string
  updatedAt: string
}

export interface ConsoleGuardBinding {
  id: string
  platform: string
  guildId: string
  templateId: string
  enabled: boolean
  note: string | null
  createdAt: string
  updatedAt: string
}

export interface DashboardPageData {
  generatedAt: string
  overview: {
    pendingReviews: number
    pendingAdmissions: number
    openReports: number
    highRiskEvents: number
    policyItems: number
  }
  pendingMembers: ConsoleGuardMember[]
  pendingReviews: ConsoleReviewRecord[]
  recentEvents: ConsoleEventRecord[]
  recentReports: ConsoleReportRecord[]
  commandPolicies: ConsoleCommandPolicy[]
  guardTemplates: ConsoleGuardTemplate[]
  guardBindings: ConsoleGuardBinding[]
  systemStatus: ConsoleModuleState[]
}

export interface IdentityLookupError {
  memberId: string
  message: string
}

export interface IdentityMemberSnapshot extends ConsoleGuardMember {
  profile: {
    qqID: string
    userID: number | null
    qqNickname: string | null
    boundAt: string | null
    verificationState: 'unbound' | 'bound_unverified' | 'verified'
    profileVerificationStatus: 'unverified' | 'pending' | 'verified' | 'rejected'
    studentVerified: boolean
  } | null
}

export interface IdentityPageData {
  generatedAt: string
  summary: {
    pendingMembers: number
    verifiedMembers: number
    boundUnverifiedMembers: number
    unboundMembers: number
    releasedMembers: number
  }
  groups: Array<{
    guildId: string
    memberCount: number
    pendingCount: number
    releasedCount: number
  }>
  members: IdentityMemberSnapshot[]
  recentReleases: IdentityMemberSnapshot[]
  lookupErrors: IdentityLookupError[]
}

export interface ReviewWorkItem {
  id: string
  kind: 'review' | 'admission' | 'report'
  guildId: string
  memberId: string | null
  subjectLabel: string
  status: string
  priority: 'low' | 'medium' | 'high' | 'critical'
  createdAt: string
  availableActions: Array<
    'execute'
    | 'reject'
    | 'approve'
    | 'deny'
    | 'defer'
    | 'dismiss'
    | 'escalate'
    | 'create-review'
  >
  relatedEventIds: string[]
  reason: string
  secondaryLabel: string
}

export interface ReviewPageData {
  generatedAt: string
  items: ReviewWorkItem[]
  events: ConsoleEventRecord[]
}

export interface ConfigGovernancePageData {
  generatedAt: string
  workspaces: Array<{
    id: 'guild-config' | 'templates' | 'bindings' | 'command-policies'
    label: string
  }>
  groupConfigs: Array<{
    guildId: string
    guildName: string
    guildAvatar: string
    config: Record<string, unknown>
  }>
  templates: Array<ConsoleGuardTemplate & {
    source: { kind: string; label: string }
  }>
  bindings: Array<ConsoleGuardBinding & {
    effectiveTemplateName: string
  }>
  commandPolicies: ConsoleCommandPolicy[]
  supportedCommandIds: string[]
}
