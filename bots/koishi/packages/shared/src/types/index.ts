export type PlatformVerificationState =
  | 'unbound'
  | 'bound_unverified'
  | 'verified'

export type GuardMemberActionState =
  | 'pending_verification'
  | 'released'
  | 'expired_pending_kick'

export const GUARD_MEMBER_TABLE = 'stuhelper_guard_member'

export interface StuhelperPlatformConfig {
  baseUrl: string
  serviceToken: string
}

export interface StuhelperBindingConfig {
  command: string
  codeTtlMinutes: number
}

export interface StuhelperGuardConfig {
  targetGroups: string[]
  muteDurationSeconds: number
  kickAfterMinutes: number
  reminderTemplate: string
  exemptUsers: string[]
}

export interface StuhelperAdminConfig {
  enableCommands: boolean
}

export interface StuhelperSchedulerConfig {
  scanIntervalSeconds: number
}

export type StuhelperKeywordRuleMatchMode = 'includes' | 'regex'
export type StuhelperKeywordRuleAction = 'warn' | 'delete' | 'mute' | 'review'

export interface StuhelperKeywordRuleConfig {
  id: string
  guildId: string
  pattern: string
  matchMode: StuhelperKeywordRuleMatchMode
  action: StuhelperKeywordRuleAction
  enabled?: boolean
  muteSeconds: number
  note: string
}

export interface StuhelperModerationConfig {
  repeatThreshold: number
  repeatWindowSize: number
  warningThresholdExpression: string
  defaultMuteSeconds: number
  antiRecallNotify: boolean
  keywordRules: StuhelperKeywordRuleConfig[]
}

export interface StuhelperFunConfig {
  diceSides: number
  muteLotteryBaseSeconds: number
  muteLotteryMaxSeconds: number
  muteLotteryPityThreshold: number
  muteLotteryPitySeconds: number
}

export interface StuhelperConsoleConfig {
  enabled: boolean
  title: string
}

export interface StuhelperAIConfig {
  enabled: boolean
  endpoint: string
  apiKey: string
  model: string
}

export interface StuhelperCoreConfig {
  platform: StuhelperPlatformConfig
  guard: StuhelperGuardConfig
  console: StuhelperConsoleConfig
}

export interface StuhelperBindingPluginConfig {
  platform: StuhelperPlatformConfig
  binding: StuhelperBindingConfig
}

export interface StuhelperGroupGuardPluginConfig {
  platform: StuhelperPlatformConfig
  guard: StuhelperGuardConfig
  scheduler: StuhelperSchedulerConfig
  moderation: StuhelperModerationConfig
  fun: StuhelperFunConfig
  ai: StuhelperAIConfig
}

export interface StuhelperAdminPluginConfig {
  platform: StuhelperPlatformConfig
  admin: StuhelperAdminConfig
  moderation: StuhelperModerationConfig
  fun: StuhelperFunConfig
}

export interface StuhelperConsolePluginConfig {
  console: StuhelperConsoleConfig
  moderation: StuhelperModerationConfig
}

export interface QQBinding {
  userID: number
  qqID: string
  qqNickname: string | null
  boundAt: string
  createdAt: string
  updatedAt: string
}

export interface QQBindingCode {
  code: string
  expiresAt: string
}

export interface ConsumeQQBindingRequest {
  code: string
  qqID: string
  qqNickname?: string
}

export interface QQVerificationStatus {
  qqID: string
  userID: number | null
  qqNickname: string | null
  boundAt: string | null
  verificationState: PlatformVerificationState
  profileVerificationStatus: 'unverified' | 'pending' | 'verified' | 'rejected'
  studentVerified: boolean
}
