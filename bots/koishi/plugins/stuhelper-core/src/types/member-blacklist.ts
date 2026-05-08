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

export type MemberBlacklistReleaseReasonCode =
  | 'manual_pardon'
  | 'release_only'
  | 'admission_appeal_passed'

export interface MemberBlacklistEntry {
  id: string
  platform: string
  subjectType: 'qq_user'
  subjectID: string
  scopeType: MemberBlacklistScopeType
  guildID?: string | null
  source: MemberBlacklistSource
  reasonCode: MemberBlacklistReasonCode
  reasonText: string
  createdAt: string
  expiresAt?: string | null
  releasedAt?: string | null
}

export interface MemberBlacklistListResult {
  list: readonly MemberBlacklistEntry[]
  total: number
}

export interface MemberBlacklistCreateParams {
  subjectID: string
  scopeType: MemberBlacklistScopeType
  guildID?: string
  reasonText?: string
}

export interface MemberBlacklistReleaseParams {
  id: string
  releaseReasonCode: MemberBlacklistReleaseReasonCode
  releaseReason?: string
}
