export type MemberBlacklistSubjectType = 'qq_user'
export type MemberBlacklistScopeType = 'guild' | 'global'
export type MemberBlacklistSource =
  | 'admission_failure'
  | 'manual_admin'
  | 'kick_blacklist'
  | 'moderation_action'
  | 'migration_legacy_koishi'
  | 'migration_admission_failure'
export type MemberBlacklistCreatedFrom =
  | 'admission_worker'
  | 'qq_command'
  | 'koishi_console'
  | 'admin_console'
  | 'moderation_review'

export interface MemberBlacklistEntry {
  readonly id: string
  readonly platform: string
  readonly subjectType: MemberBlacklistSubjectType
  readonly subjectID: string
  readonly scopeType: MemberBlacklistScopeType
  readonly guildID?: string
  readonly source: MemberBlacklistSource
  readonly reasonCode: string
  readonly reasonText: string
  readonly createdByType: string
  readonly createdByID: string
  readonly createdFrom: MemberBlacklistCreatedFrom
  readonly expiresAt?: string
  readonly releasedAt?: string
  readonly releaseReasonCode?: string
  readonly releaseReason?: string
  readonly metadata: Record<string, unknown>
  readonly createdAt: string
  readonly updatedAt: string
}

export interface MemberBlacklistAccessQuery {
  readonly platform: string
  readonly subjectType: MemberBlacklistSubjectType
  readonly subjectID: string
  readonly guildID: string
}

export interface MemberBlacklistAccessDecision {
  readonly canJoin: boolean
  readonly decision: 'allowed' | 'blocked'
  readonly matchedBlacklist?: MemberBlacklistEntry
}

export interface MemberBlacklistListQuery {
  readonly platform?: string
  readonly subjectType?: MemberBlacklistSubjectType
  readonly subjectID?: string
  readonly scopeType?: MemberBlacklistScopeType
  readonly guildID?: string
  readonly page?: number
  readonly pageSize?: number
  readonly state?: 'active' | 'all'
}

export interface MemberBlacklistListResult {
  readonly items: readonly MemberBlacklistEntry[]
  readonly total: number
}

export interface MemberBlacklistCreateRequest {
  readonly platform: string
  readonly subjectType: MemberBlacklistSubjectType
  readonly subjectID: string
  readonly scopeType: MemberBlacklistScopeType
  readonly guildID?: string
  readonly source: MemberBlacklistSource
  readonly reasonCode: string
  readonly reasonText: string
  readonly createdFrom: MemberBlacklistCreatedFrom
  readonly operatorID?: string
  readonly expiresAt?: string
  readonly metadata?: Record<string, unknown>
}

export interface MemberBlacklistReleaseRequest {
  readonly releaseReasonCode: string
  readonly releaseReason?: string
  readonly operatorID?: string
}

export interface MemberBlacklistReleaseBySubjectRequest extends MemberBlacklistReleaseRequest {
  readonly platform: string
  readonly subjectType: MemberBlacklistSubjectType
  readonly subjectID: string
  readonly scopeType: MemberBlacklistScopeType
  readonly guildID?: string
}
