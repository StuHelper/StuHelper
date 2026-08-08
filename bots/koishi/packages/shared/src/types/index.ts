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
  | 'created'
  | 'awaiting_account_link'
  | 'awaiting_requirements'
  | 'pending_manual_review'
  | 'eligible'
  | 'action_pending'
  | 'admitted'
  | 'released'
  | 'expired'
  | 'rejected'
  | 'cancelled'

export type AdmissionBotAction =
  | 'mute'
  | 'remind'
  | 'release'
  | 'kick'
  | 'blacklist'

export type AdmissionJoinHandlingStrategy =
  | 'post_join_guard'
  | 'join_request_review'
  | 'post_join_time_code'

export const GUARD_MEMBER_TABLE = 'stuhelper_guard_member'

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
}

export interface StuhelperAdminMessageConfig {
  guardStatusCommandDescription: string
  guardWarningCommandDescription: string
  guardReviewListCommandDescription: string
  guardBatchMuteCommandDescription: string
  guardKickReviewCommandDescription: string
  guardBlockReviewCommandDescription: string
  commandAccessDenied: string
  adminCommandsDisabled: string
  guardGuildContextRequired: string
  guardWarningMissingContext: string
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
  admissionQueryStudentManualReviewPending: string
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
  admissionConsoleRegenerateSuccess: string
  admissionConsoleSkipSuccess: string
  admissionConsoleSkipUnmuteFailed: string
  admissionConsoleResetFailureCountSuccess: string
  admissionConsoleReleaseBlacklistNotFound: string
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

export interface StuhelperGroupGuardPluginConfig {
  platform: StuhelperPlatformConfig
  scheduler: StuhelperSchedulerConfig
  actionStream?: StuhelperAdmissionActionStreamConfig
  alerting?: StuhelperAlertmanagerWebhookConfig
}

export interface StuhelperAlertmanagerWebhookConfig {
  enabled: boolean
  bearerToken: string
  botSelfID: string
}

export interface StuhelperAdminPluginConfig {
  platform: StuhelperPlatformConfig
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

export interface AdmissionPolicyTarget {
  readonly policyID: string
  readonly platform: string
  readonly guildID: string
  readonly guardEnabled?: boolean
  readonly joinHandlingStrategy?: AdmissionJoinHandlingStrategy
  readonly linkWaitSeconds?: number
  readonly managementGuildIDs: readonly string[]
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
  readonly joinHandlingStrategy?: AdmissionJoinHandlingStrategy
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
  readonly dispatchAttempt?: number
  readonly eligibilityRevision?: number
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

export interface AdmissionBotActionEventRequest extends AdmissionBotEventRequest {
  readonly dispatchAttempt: number
}
