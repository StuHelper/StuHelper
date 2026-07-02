import type { components, operations } from './api.gen'

export type { components, operations, paths } from './api.gen'
export * from './member-blacklist'

type Schemas = components['schemas']

// API 契约枚举一律取自 openapi 生成类型（api.gen.ts），server 端改动会先体现为这里的编译错误。
export type PlatformVerificationState = Schemas['QQVerificationStatus']['verificationState']

export type GuardMemberActionState =
  | 'pending_verification'
  | 'released'
  | 'expired_pending_kick'

export type AdmissionSessionStatus = Schemas['AdmissionStatus']

export type FreshmanApplicationStatus = Schemas['FreshmanApplicationStatus']

export type FreshmanMaterialType = Schemas['FreshmanApplication']['materialType']

export type AdmissionBotAction = Schemas['BotAdmissionEventRequest']['action']

export type AdmissionJoinHandlingStrategy = Schemas['BotAdmissionPolicyTarget']['joinHandlingStrategy']

export type FreshmanReviewAction = Schemas['FreshmanReviewRequest']['action']

export const GUARD_MEMBER_TABLE = 'stuhelper_guard_member'

/** Koishi console 管理面（WebUI）操作所需的最低 authority 等级。 */
export const CONSOLE_MIN_AUTHORITY = 4

export interface StuhelperPlatformConfig {
  baseUrl: string
  serviceToken: string
}

export interface StuhelperBindingMessageConfig {
  directOnly: string
  missingCode: string
  successVerified: string
  successUnverified: string
  unavailable: string
  invalidCode: string
  unauthorized: string
  conflict: string
  notConfigured: string
  rateLimited: string
  tooManyFailures: string
}

export interface StuhelperAdminMessageConfig {
  guardStatusCommandDescription: string
  guardWarningCommandDescription: string
  guardReviewListCommandDescription: string
  guardBatchMuteCommandDescription: string
  guardKickReviewCommandDescription: string
  guardBlockReviewCommandDescription: string
  freshmanViewCommandDescription: string
  freshmanApproveCommandDescription: string
  freshmanRejectCommandDescription: string
  freshmanBlacklistReleaseCommandDescription: string
  commandAccessDenied: string
  adminCommandsDisabled: string
  guardWarningMissingContext: string
  guardReviewListMissingContext: string
  guardBatchMuteGroupOnly: string
  guardBatchMuteInvalidPayload: string
  guardBatchMuteNoTargets: string
  guardBatchMuteSuccess: string
  guardBatchMuteEventSummary: string
  guardBatchMuteEventReason: string
  guardReviewRequestMissingArgs: string
  guardReviewRequestSuccess: string
  guardReviewRequestEventSummary: string
  guardReviewKickActionLabel: string
  guardReviewKickAndBlockActionLabel: string
  guardPendingMembersEmpty: string
  guardPendingMembersHeader: string
  guardPendingMemberLine: string
  guardWarningCounterEmpty: string
  guardWarningCounterNoReason: string
  guardWarningCounterSummary: string
  guardPendingReviewsEmpty: string
  guardPendingReviewsHeader: string
  guardPendingReviewLine: string
  freshmanManagementGroupOnly: string
  freshmanMissingApplicationID: string
  freshmanApproveInvalidFormat: string
  freshmanApproveInvalidExtension: string
  freshmanRejectInvalidFormat: string
  freshmanBlacklistReleaseInvalidFormat: string
  freshmanApproveSuccess: string
  freshmanApproveSuccessWithExtension: string
  freshmanRejectSuccess: string
  freshmanBlacklistScopeGlobal: string
  freshmanBlacklistScopeGuild: string
  freshmanBlacklistReleaseSuccess: string
  freshmanApplicationSummary: string
  freshmanApplicationDepartmentFallback: string
  freshmanApplicationExpiryFallback: string
  freshmanCommandFailed: string
  freshmanOperatorQQUnbound: string
  freshmanOperatorForbidden: string
  freshmanManagementGuildForbidden: string
  freshmanBackendForbidden: string
}

export interface StuhelperSchedulerConfig {
  scanIntervalSeconds: number
}

export interface StuhelperAdmissionActionStreamConfig {
  reconnectDelaySeconds?: number
}

export interface StuhelperAdmissionReminderDeliveryConfig {
  groupEnabled?: boolean
  directEnabled?: boolean
}

export interface StuhelperGroupGuardMessageConfig {
  publicReportCommandDescription: string
  diceCommandDescription: string
  muteLotteryCommandDescription: string
  admissionQueryCommandDescription: string
  admissionResendCommandDescription: string
  admissionRegenerateCommandDescription: string
  admissionSkipCommandDescription: string
  admissionResetFailureCountCommandDescription: string
  admissionReleaseBlacklistCommandDescription: string
  admissionReminder: string
  admissionTimeoutNormal: string
  admissionTimeoutWithFailures: string
  admissionTimeoutBlacklist: string
  backendPendingReminder: string
  admissionReleaseCompleted: string
  admissionKickTimeout: string
  admissionTimeCodeReminder: string
  admissionTimeCodeVerified: string
  admissionTimeCodeInvalid: string
  admissionTimeCodeKickTimeout: string
  admissionBlacklistKick: string
  antiRecallNotify: string
  moderationMuteNotice: string
  moderationUnmuteNotice: string
  moderationKickNotice: string
  freshmanForwardSummary: string
  freshmanForwardUnknownField: string
  freshmanMaterialTypeAdmissionNotice: string
  freshmanMaterialTypeAdmissionCertificate: string
  publicReportMissingArgs: string
  publicCommandsDisabled: string
  muteLotteryGroupOnly: string
  commandAccessDenied: string
  diceResult: string
  muteLotteryResult: string
  muteLotteryPityResult: string
  reportGroupOnly: string
  reportRecordedAIUnavailable: string
  reportAIReviewFailed: string
  reportHighRisk: string
  reportMediumRisk: string
  reportLowRisk: string
  reportNoAction: string
  admissionCommandGroupOnly: string
  admissionCommandsDisabled: string
  admissionCommandMissingQQ: string
  admissionCommandMissingOperator: string
  admissionCommandUnsupportedPlatform: string
  admissionCommandPolicyDisabled: string
  admissionCommandNotFound: string
  admissionCommandInvalidState: string
  admissionCommandUnauthorized: string
  admissionCommandFailed: string
  admissionCommandPlatformError: string
  admissionCommandMissingResendURL: string
  admissionCommandStaleRecord: string
  admissionReminderDeliveryDisabled: string
  admissionReminderDeliveryFailure: string
  admissionReminderDeliveryGroupChannelLabel: string
  admissionReminderDeliveryDirectChannelLabel: string
  admissionSkipSuccess: string
  admissionSkipUnmuteFailed: string
  admissionAlreadyVerifiedRegenerate: string
  admissionResetFailureCountSuccess: string
  admissionReleaseBlacklistNotFound: string
  admissionReleaseBlacklistSuccess: string
  admissionQuerySummary: string
  admissionQueryDeadlineLink: string
  admissionQueryDeadlineSubmission: string
  admissionQueryDeadlineManualReview: string
  admissionQueryDeadlineUnset: string
  admissionQueryLastBotError: string
  admissionQueryQQLinked: string
  admissionQueryQQUnlinked: string
  admissionQueryStudentVerified: string
  admissionQueryStudentFreshmanPending: string
  admissionQueryStudentUnverified: string
  admissionStatusJoinedMuted: string
  admissionStatusLinked: string
  admissionStatusMaterialSubmitted: string
  admissionStatusVerified: string
  admissionStatusExpiredKicked: string
  admissionStatusCancelled: string
  admissionNextStepJoinedMuted: string
  admissionNextStepLinked: string
  admissionNextStepMaterialSubmitted: string
  admissionNextStepVerifiedWithBotError: string
  admissionNextStepVerified: string
  admissionNextStepExpiredKickedLinked: string
  admissionNextStepExpiredKicked: string
  admissionNextStepCancelled: string
  admissionNextStepDefault: string
  admissionConsoleSettingsSaved: string
  admissionConsoleRecordNotFound: string
  admissionConsoleStaleRecord: string
  admissionConsoleResendSuccess: string
  admissionConsoleVerifiedReleaseSuccess: string
  admissionConsoleVerifiedReleaseUnmuteFailed: string
  admissionConsoleRegenerateSuccess: string
  admissionConsoleRegenerateMuteFailed: string
  admissionConsoleSkipSuccess: string
  admissionConsoleSkipUnmuteFailed: string
  admissionConsoleResetFailureCountSuccess: string
  admissionConsoleReleaseBlacklistSuccess: string
  admissionConsoleMissingResendURL: string
  admissionConsoleInvalidMuteDeadline: string
  admissionConsoleBotNotFound: string
  admissionConsoleErrorNotFound: string
  admissionConsoleErrorInvalidState: string
  admissionConsoleErrorUnauthorized: string
  admissionConsoleErrorPlatform: string
  admissionConsoleErrorFallback: string
  moderationWarnEventSummary: string
  moderationMuteEventSummary: string
  moderationUnmuteEventSummary: string
  moderationKickEventSummary: string
  moderationAntiRecallEventSummary: string
  moderationKeywordHitEventSummary: string
  moderationKeywordHitReason: string
  moderationKeywordRuleHitReason: string
  moderationRepeatHitEventSummary: string
  moderationRepeatHitReason: string
  moderationRepeatAutoMuteReason: string
  moderationMuteLotteryEventSummary: string
  reportCreatedEventSummary: string
  reportAIReviewedEventSummary: string
  reportAIWarnReason: string
  reportAIMuteReason: string
  reportAISummaryFallback: string
  admissionBlacklistEventSummary: string
  admissionJoinMutedEventSummary: string
  admissionJoinAlreadyVerifiedEventSummary: string
  admissionJoinBackendUnavailableEventSummary: string
  admissionJoinBackendVerifiedEventSummary: string
  admissionTimeCodeJoinGuardedEventSummary: string
  admissionTimeCodeVerifiedEventSummary: string
  admissionTimeCodeExpiredEventSummary: string
  admissionCommandInvalidMuteDeadline: string
}

export interface StuhelperCoreConfig {
  platform: StuhelperPlatformConfig
}

export interface StuhelperBindingPluginConfig {
  platform: StuhelperPlatformConfig
}

export interface StuhelperRetentionConfig {
  messageRetentionDays: number
  eventRetentionDays: number
}

export interface StuhelperGroupGuardPluginConfig {
  platform: StuhelperPlatformConfig
  scheduler: StuhelperSchedulerConfig
  actionStream?: StuhelperAdmissionActionStreamConfig
  retention: StuhelperRetentionConfig
}

export interface StuhelperAdminPluginConfig {
  platform: StuhelperPlatformConfig
}

// ---- API 契约类型：全部别名自 api.gen.ts，禁止在此手写字段 ----

export type QQBinding = Schemas['QQBinding']

export type QQBindingCode = Schemas['QQBindingCode']

export type ConsumeQQBindingRequest = Schemas['ConsumeQQBindingRequest']

export type QQVerificationStatus = Schemas['QQVerificationStatus']

export type AdmissionSession = Schemas['AdmissionSession']

export type AdmissionSessionCreateRequest = Schemas['BotAdmissionSessionCreateRequest']

export type AdmissionSessionSubjectRequest = Schemas['BotAdmissionSessionSubjectRequest']

export type AdmissionSessionOperatorRequest = Schemas['BotAdmissionSessionOperatorRequest']

export type AdmissionSessionCreateResult = Schemas['CreatedAdmissionSession']

export type AdmissionPolicyTarget = Schemas['BotAdmissionPolicyTarget']

export type AdmissionFailureResetResult = Schemas['BotAdmissionFailureResetResult']

export type AdmissionJoinRequestEvent = Schemas['BotAdmissionJoinRequestEvent']

export type AdmissionJoinRequestDecisionAction = Schemas['BotAdmissionJoinRequestDecision']['decision']

export type AdmissionJoinRequestVerificationState = Schemas['BotAdmissionJoinRequestDecision']['verificationState']

export type AdmissionJoinRequestDecisionRequest = Schemas['BotAdmissionJoinRequestDecisionRequest']

export type AdmissionJoinRequestDecision = Schemas['BotAdmissionJoinRequestDecision']

export type AdmissionPendingActionsRequest =
  operations['listBotPendingAdmissionActions']['parameters']['query']

export type AdmissionPendingAction = Schemas['BotAdmissionPendingAction']

export type AdmissionBotEventRequest = Schemas['BotAdmissionEventRequest']

export type FreshmanApplication = Schemas['FreshmanApplication']

export type FreshmanForwardItem =
  operations['listBotPendingFreshmanForwards']['responses'][200]['content']['application/json']['data'][number]

export type FreshmanCommandContext = Schemas['BotFreshmanCommandContext']

export type FreshmanReviewRequest = Schemas['BotFreshmanReviewRequest']
