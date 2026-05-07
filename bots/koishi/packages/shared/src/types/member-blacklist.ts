export type MemberBlacklistSubjectType = 'qq_user'
export type MemberBlacklistScopeType = 'global' | 'guild'
export type MemberBlacklistSource =
  | 'admission_failure'
  | 'manual_admin'
  | 'kick_blacklist'
  | 'moderation_action'
  | 'migration_legacy_koishi'
  | 'migration_admission_failure'
export type MemberBlacklistReasonCode =
  | 'admission_timeout_limit'
  | 'manual_blacklist'
  | 'manual_kick_blacklist'
  | 'violation_review_blacklist'
  | 'legacy_koishi_blacklist'
  | 'legacy_admission_blacklist'
export type MemberBlacklistActorType =
  | 'system'
  | 'admin_user'
  | 'qq_operator'
  | 'service_account'
export type MemberBlacklistCreatedFrom =
  | 'admission_worker'
  | 'qq_command'
  | 'koishi_console'
  | 'admin_console'
  | 'moderation_review'
  | 'migration_script'
export type MemberBlacklistStatus = 'active' | 'released' | 'expired' | 'all'
export type MemberBlacklistReleaseReasonCode =
  | 'manual_pardon'
  | 'release_only'
  | 'policy_expired_auto'
  | 'admission_appeal_passed'
  | 'migration_inverse'

export interface MemberBlacklistEntry {
  readonly id: string
  readonly platform: string
  readonly subjectType: MemberBlacklistSubjectType
  readonly subjectID: string
  readonly scopeType: MemberBlacklistScopeType
  readonly guildID?: string | null
  readonly source: MemberBlacklistSource
  readonly reasonCode: MemberBlacklistReasonCode
  readonly reasonText: string
  readonly metadata: Record<string, unknown>
  readonly createdByType: MemberBlacklistActorType
  readonly createdByID: string
  readonly createdFrom: MemberBlacklistCreatedFrom
  readonly expiresAt?: string | null
  readonly releasedAt?: string | null
  readonly releasedByType?: MemberBlacklistActorType | null
  readonly releasedByID?: string | null
  readonly releaseReasonCode?: MemberBlacklistReleaseReasonCode | null
  readonly releaseReason?: string | null
  readonly createdAt: string
  readonly updatedAt: string
}

export interface MemberBlacklistCreateRequest {
  readonly platform: string
  readonly subjectType: MemberBlacklistSubjectType
  readonly subjectID: string
  readonly scopeType: MemberBlacklistScopeType
  readonly guildID?: string
  readonly source: MemberBlacklistSource
  readonly reasonCode: MemberBlacklistReasonCode
  readonly reasonText: string
  readonly createdFrom?: MemberBlacklistCreatedFrom
  readonly expiresAt?: string | null
  readonly metadata: Record<string, unknown>
}

export interface MemberBlacklistReleaseRequest {
  readonly releaseReasonCode: MemberBlacklistReleaseReasonCode
  readonly releaseReason?: string
  readonly operatorQQID?: string
}

export interface MemberBlacklistReleaseBySubjectRequest extends MemberBlacklistReleaseRequest {
  readonly platform: string
  readonly subjectType: MemberBlacklistSubjectType
  readonly subjectID: string
  readonly scopeType: MemberBlacklistScopeType
  readonly guildID?: string
}

export interface MemberBlacklistAccessRequest {
  readonly platform: string
  readonly subjectType: MemberBlacklistSubjectType
  readonly subjectID: string
  readonly guildID?: string
}

export interface MemberBlacklistListRequest {
  readonly platform?: string
  readonly subjectType?: MemberBlacklistSubjectType
  readonly subjectID?: string
  readonly scopeType?: MemberBlacklistScopeType
  readonly source?: MemberBlacklistSource
  readonly guildID?: string
  readonly status?: MemberBlacklistStatus
  readonly page?: number
  readonly pageSize?: number
}

export interface MemberBlacklistListResult {
  readonly list: readonly MemberBlacklistEntry[]
  readonly total: number
}

export interface MemberBlacklistAccessDecision {
  readonly canJoin: boolean
  readonly decision: 'allowed' | 'blocked'
  readonly matchedBlacklist?: MemberBlacklistEntry
  readonly reason?: string
}

export interface PlatformRequestOptions {
  readonly timeoutMs?: number
}
