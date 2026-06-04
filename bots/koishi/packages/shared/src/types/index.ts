export * from './member-blacklist'

export type PlatformVerificationState =
  | 'unbound'
  | 'bound_unverified'
  | 'verified'

export type GuardMemberActionState =
  | 'pending_verification'
  | 'released'
  | 'expired_pending_kick'

export type AdmissionSessionStatus =
  | 'joined_muted'
  | 'linked'
  | 'material_submitted'
  | 'verified'
  | 'expired_kicked'
  | 'cancelled'

export type FreshmanApplicationStatus =
  | 'pending'
  | 'approved'
  | 'rejected'

export type FreshmanMaterialType =
  | 'admission_notice'
  | 'admission_certificate'

export type AdmissionBotAction =
  | 'mute'
  | 'remind'
  | 'release'
  | 'kick'
  | 'blacklist'
  | 'forward'

export type FreshmanReviewAction = 'approve' | 'reject'

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
  fallbackScanEnabled?: boolean
}

export interface StuhelperAdmissionActionStreamConfig {
  enabled?: boolean
  reconnectDelaySeconds?: number
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
  enabled?: boolean
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

export interface StuhelperCommandConfig {
  enabled?: boolean
}

export interface StuhelperAdmissionCommandConfig {
  enabled?: boolean
  minAuthority?: number
  operatorQQIDs?: string[]
}

export interface StuhelperFreshmanForwardConfig {
  enabled?: boolean
}

export interface StuhelperCoreConfig {
  platform: StuhelperPlatformConfig
  guard: StuhelperGuardConfig
  console: StuhelperConsoleConfig
  runtimeModules?: {
    enabled?: boolean
  }
}

export interface StuhelperBindingPluginConfig {
  platform: StuhelperPlatformConfig
  binding: StuhelperBindingConfig
}

export interface StuhelperGroupGuardPluginConfig {
  platform: StuhelperPlatformConfig
  guard: StuhelperGuardConfig
  scheduler: StuhelperSchedulerConfig
  actionStream?: StuhelperAdmissionActionStreamConfig
  moderation: StuhelperModerationConfig
  fun: StuhelperFunConfig
  ai: StuhelperAIConfig
  commands?: StuhelperCommandConfig
  admissionCommands?: StuhelperAdmissionCommandConfig
  freshmanForward?: StuhelperFreshmanForwardConfig
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
}

export interface QQVerificationStatus {
  qqID: string
  userID: number | null
  boundAt: string | null
  verificationState: PlatformVerificationState
  profileVerificationStatus: 'unverified' | 'pending' | 'verified' | 'rejected'
  studentVerified: boolean
}

export interface AdmissionSession {
  readonly id: string
  readonly platform: string
  readonly guildID: string
  readonly channelID?: string
  readonly qqID: string
  readonly userID?: number | string | null
  readonly status: AdmissionSessionStatus
  readonly tokenExpiresAt: string
  readonly linkWaitDeadlineAt: string
  readonly submissionWaitDeadlineAt: string
  readonly manualReviewDeadlineAt?: string | null
  readonly initialMuteUntil: string
  readonly projectionPending: boolean
  readonly authURL?: string
  readonly maxMaterialBytes?: number
  readonly lastBotError?: string | null
  readonly failureCount?: number
  readonly remainingRetryCount?: number
  readonly willBlacklistOnTimeout?: boolean
}

export interface AdmissionSessionCreateRequest {
  readonly platform: string
  readonly guildID: string
  readonly channelID: string
  readonly qqID: string
  readonly botSelfID?: string
}

export interface AdmissionSessionSubjectRequest {
  readonly platform: string
  readonly guildID: string
  readonly qqID: string
}

export interface AdmissionSessionOperatorRequest extends AdmissionSessionSubjectRequest {
  readonly operatorQQID: string
}

export interface AdmissionSessionCreateResult {
  readonly session: AdmissionSession
  readonly token: string
  readonly authURL: string
}

export interface AdmissionFailureResetResult {
  readonly platform: string
  readonly guildID: string
  readonly qqID: string
  readonly previousFailureCount: number
}

export interface AdmissionJoinRequestEvent {
  readonly platform: string
  readonly guildID: string
  readonly qqID: string
  readonly requestID: string
  readonly decision?: AdmissionJoinRequestDecisionAction
  readonly success: boolean
  readonly error?: string
  readonly rawEvent?: Record<string, unknown>
}

export type AdmissionJoinRequestDecisionAction = 'approve' | 'reject'

export type AdmissionJoinRequestVerificationState = 'verified' | 'unverified'

export interface AdmissionJoinRequestDecisionRequest {
  readonly platform: string
  readonly guildID: string
  readonly qqID: string
  readonly requestID: string
  readonly rawEvent?: Record<string, unknown>
}

export interface AdmissionJoinRequestDecision {
  readonly decision: AdmissionJoinRequestDecisionAction
  readonly reason?: string
  readonly verificationState: AdmissionJoinRequestVerificationState
  readonly autoApproveVerifiedJoin: boolean
  readonly autoApproveUnverifiedJoin: boolean
  readonly policyID?: string
  readonly userID?: number | string | null
}

export interface AdmissionPendingActionsRequest {
  readonly platform: string
  readonly botSelfID: string
  readonly limit?: number
}

export interface AdmissionPendingAction {
  readonly actionID?: string
  readonly sessionID: string
  readonly action: Extract<AdmissionBotAction, 'remind' | 'release' | 'kick' | 'blacklist'>
  readonly platform?: string
  readonly botSelfID?: string
  readonly guildID?: string
  readonly channelID?: string
  readonly qqID?: string
  readonly authURL?: string
  readonly deadlineAt?: string
  readonly reason?: string
  readonly failureCount?: number
  readonly remainingRetryCount?: number
  readonly willBlacklistOnTimeout?: boolean
}

export interface AdmissionBotEventRequest {
  readonly action: AdmissionBotAction
  readonly success: boolean
  readonly messageID?: string
  readonly error?: string
}

export interface FreshmanApplication {
  readonly id: string
  readonly userID: number | string
  readonly schoolID: number
  readonly admissionSessionID?: string
  readonly applicantName?: string
  readonly applicantNameMasked: string
  readonly departmentOrMajor?: string | null
  readonly materialType: FreshmanMaterialType
  readonly status: FreshmanApplicationStatus
  readonly provisionalExpiresAt?: string | null
  readonly reviewedAt?: string | null
  readonly createdAt: string
}

export interface FreshmanForwardItem {
  readonly application: FreshmanApplication
  readonly materialURL: string
  readonly managementGuildIDs: readonly string[]
  readonly platform?: string
  readonly botSelfID?: string
  readonly schoolName?: string
  readonly qqID?: string
}

export interface FreshmanCommandContext {
  readonly operatorQQID: string
  readonly guildID: string
  readonly channelID?: string
  readonly rawCommand: string
}

export interface FreshmanReviewRequest extends FreshmanCommandContext {
  readonly action: FreshmanReviewAction
  readonly reason?: string
  readonly expiresInDays?: number
}
