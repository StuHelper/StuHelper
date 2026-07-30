import type { Events as ConsoleEvents } from '@koishijs/plugin-console'
import type { Context, Universal } from 'koishi'

import type {
  AdmissionSession,
  AdmissionSessionCreateResult,
  AdmissionRuntimeSettingsInput,
  AdmissionRuntimeSettingsStore,
  GroupGuardBehaviorSettingsStore,
  GuardMemberRecord,
  GuardMemberStore,
  GuardPolicyStore,
  PlatformClient,
  PlatformAPIError,
  StuhelperGroupGuardPluginConfig,
} from '@stuhelper/koishi-shared'
import {
  PlatformAPIError as PlatformAPIErrorClass,
  DEFAULT_GROUP_GUARD_BEHAVIOR_SETTINGS,
} from '@stuhelper/koishi-shared'
import type { ModerationStore } from '@stuhelper/koishi-moderation-core'

import { formatAdmissionReminder } from './admission-format'
import { sendAdmissionReminderMessage } from './admission-reminder-delivery'
import { formatAdmissionActionError } from './admission-actions'
import { backendSyncUpdate } from './member-records'
import { requireAdmissionActionPlatform } from './admission-action-boundary'
import type { GuardBotRuntime } from './member-guard'
import type {
  AdmissionSubjectCoordinator,
  AdmissionSubjectRef,
} from './admission-subject-coordinator'
import {
  getGroupGuardMessages,
  groupGuardMessage,
  type GroupGuardMessageProvider,
  type GroupGuardMessages,
} from './group-guard-message-provider'

const ADMISSION_RUNTIME_PAGE_EVENT = 'stuhelperGroupGuard/page/admission-runtime'
const ADMISSION_RUNTIME_ACTION_EVENT = 'stuhelperGroupGuard/action/admission-member'
const ADMISSION_RUNTIME_SETTINGS_EVENT = 'stuhelperGroupGuard/action/save-admission-runtime-settings'
const CONSOLE_AUTHORITY = 4
const ACTIVE_MEMBER_LIMIT = 100

export type AdmissionRuntimeAction =
  | 'query'
  | 'resend'
  | 'regenerate'
  | 'skip'
  | 'reset-failures'
  | 'release-blacklist'

interface AdmissionRuntimeActionInput {
  readonly recordId: string
  readonly action: AdmissionRuntimeAction
}

interface ConsoleActionClient {
  auth?: {
    id: number
  }
}

interface AdmissionConsoleAPIDeps {
  readonly config: StuhelperGroupGuardPluginConfig
  readonly platform: PlatformClient
  readonly runtimeSettings: AdmissionRuntimeSettingsStore
  readonly behaviorSettings?: GroupGuardBehaviorSettingsStore
  readonly guardStore: GuardMemberStore
  readonly policyStore: GuardPolicyStore
  readonly moderationStore: ModerationStore
  readonly onRuntimeSettingsChanged?: () => void | Promise<void>
  readonly messageProvider?: GroupGuardMessageProvider
  readonly admissionSubjectCoordinator?: AdmissionSubjectCoordinator
}

declare module '@koishijs/console' {
  interface Events {
    [ADMISSION_RUNTIME_PAGE_EVENT](): ReturnType<typeof buildAdmissionRuntimePageData>
    [ADMISSION_RUNTIME_ACTION_EVENT](input: AdmissionRuntimeActionInput): Promise<string>
    [ADMISSION_RUNTIME_SETTINGS_EVENT](input: AdmissionRuntimeSettingsInput): Promise<string>
  }
}

export function registerAdmissionConsoleAPI(ctx: Context, deps: AdmissionConsoleAPIDeps) {
  const addConsoleListener = createAdmissionConsoleListenerRegistrar(ctx)

  addConsoleListener(ADMISSION_RUNTIME_PAGE_EVENT, async () => {
    return buildAdmissionRuntimePageData(ctx, deps)
  })

  addConsoleListener(ADMISSION_RUNTIME_ACTION_EVENT, async function (input) {
    return handleAdmissionRuntimeAction(ctx, deps, input, this as ConsoleActionClient)
  })

  addConsoleListener(ADMISSION_RUNTIME_SETTINGS_EVENT, async (input) => {
    await deps.runtimeSettings.saveSettings(parseRuntimeSettingsInput(input))
    await deps.onRuntimeSettingsChanged?.()
    return groupGuardMessage(await getGroupGuardMessages(deps.messageProvider), 'admissionConsoleSettingsSaved')
  })
}

function createAdmissionConsoleListenerRegistrar(ctx: Context) {
  const console = ctx.console
  return function addConsoleListener<K extends keyof ConsoleEvents>(
    event: K,
    callback: ConsoleEvents[K],
  ) {
    return ctx.effect(() => {
      console.addListener(event, callback, { authority: CONSOLE_AUTHORITY })
      const registration = console.listeners[event]
      return () => {
        if (console.listeners[event] === registration) {
          delete console.listeners[event]
        }
      }
    })
  }
}

export async function buildAdmissionRuntimePageData(ctx: Context, deps: AdmissionConsoleAPIDeps) {
  const [activeMembers, templates, bindings, settings, behaviorSettings, keywordRules] = await Promise.all([
    deps.guardStore.listActive(),
    deps.policyStore.listTemplates(),
    deps.policyStore.listBindings(),
    deps.runtimeSettings.getSettings(),
    deps.behaviorSettings?.getSettings() ?? DEFAULT_GROUP_GUARD_BEHAVIOR_SETTINGS,
    deps.moderationStore.listAllKeywordRules(),
  ])
  const sortedMembers = [...activeMembers]
    .sort((left, right) => left.deadlineAt.getTime() - right.deadlineAt.getTime())
    .slice(0, ACTIVE_MEMBER_LIMIT)

  return {
    generatedAt: new Date().toISOString(),
    platform: {
      baseUrl: redactURL(deps.config.platform.baseUrl),
      serviceTokenConfigured: Boolean(deps.config.platform.serviceToken),
    },
    scheduler: {
      fallbackScanEnabled: settings.fallbackScanEnabled,
      scanIntervalSeconds: deps.config.scheduler.scanIntervalSeconds,
    },
    actionStream: {
      enabled: settings.actionStreamEnabled,
      reconnectDelaySeconds: deps.config.actionStream?.reconnectDelaySeconds,
    },
    commands: {
      publicCommandsRegistered: true,
      publicCommandsEnabled: settings.publicCommandsEnabled,
      adminCommandsRegistered: true,
      adminCommandsEnabled: settings.adminCommandsEnabled,
      admissionCommandsRegistered: true,
      admissionCommandsEnabled: settings.admissionCommandsEnabled,
    },
    moderation: {
      enabled: settings.moderationEnabled,
      keywordRuleCount: keywordRules.length,
      repeatThreshold: behaviorSettings.moderation.repeatThreshold,
      repeatWindowSize: behaviorSettings.moderation.repeatWindowSize,
      antiRecallNotify: behaviorSettings.moderation.antiRecallNotify,
    },
    freshmanForward: {
      enabled: settings.freshmanForwardEnabled,
    },
    reminderDelivery: {
      groupEnabled: settings.reminderGroupEnabled,
      directEnabled: settings.reminderDirectEnabled,
    },
    timeCode: {
      reminderEnabled: settings.timeCodeReminderEnabled,
    },
    bots: ctx.bots.map((bot) => ({
      platform: bot.platform,
      selfId: bot.selfId,
      status: String((bot as { status?: unknown }).status ?? 'unknown'),
    })),
    stats: {
      templateCount: templates.length,
      bindingCount: bindings.length,
      enabledBindingCount: bindings.filter((binding) => binding.enabled).length,
      activeMemberCount: activeMembers.length,
      backendSyncPendingCount: activeMembers.filter((record) => record.backendSyncPending).length,
      membersWithAdmissionSessionCount: activeMembers.filter((record) => Boolean(record.admissionSessionID)).length,
      membersWithLastErrorCount: activeMembers.filter((record) => Boolean(record.lastError)).length,
    },
    templates: templates.map((template) => ({
      id: template.id,
      name: template.name,
      enabled: template.enabled,
      muteDurationSeconds: template.muteDurationSeconds,
      kickAfterMinutes: template.kickAfterMinutes,
      exemptUserCount: template.exemptUsers.length,
      updatedAt: template.updatedAt.toISOString(),
    })),
    bindings: bindings.map((binding) => ({
      id: binding.id,
      platform: binding.platform,
      guildId: binding.guildId,
      templateId: binding.templateId,
      kickAfterMinutes: binding.kickAfterMinutesOverride ?? templateKickAfterMinutes(templates, binding.templateId),
      kickAfterMinutesOverride: binding.kickAfterMinutesOverride ?? null,
      enabled: binding.enabled,
      note: binding.note,
      updatedAt: binding.updatedAt.toISOString(),
    })),
    activeMembers: sortedMembers.map(serializeGuardMember),
  }
}

function templateKickAfterMinutes(
  templates: Awaited<ReturnType<GuardPolicyStore['listTemplates']>>,
  templateId: string,
) {
  return templates.find((template) => template.id === templateId)?.kickAfterMinutes ?? null
}

function parseRuntimeSettingsInput(input: unknown): AdmissionRuntimeSettingsInput {
  if (!input || typeof input !== 'object' || Array.isArray(input)) {
    throw new Error('admission runtime settings input must be an object')
  }
  const record = input as Record<string, unknown>
  return {
    actionStreamEnabled: readOptionalBoolean(record.actionStreamEnabled),
    publicCommandsEnabled: readOptionalBoolean(record.publicCommandsEnabled),
    adminCommandsEnabled: readOptionalBoolean(record.adminCommandsEnabled),
    admissionCommandsEnabled: readOptionalBoolean(record.admissionCommandsEnabled),
    moderationEnabled: readOptionalBoolean(record.moderationEnabled),
    freshmanForwardEnabled: readOptionalBoolean(record.freshmanForwardEnabled),
    fallbackScanEnabled: readOptionalBoolean(record.fallbackScanEnabled),
    reminderGroupEnabled: readOptionalBoolean(record.reminderGroupEnabled),
    reminderDirectEnabled: readOptionalBoolean(record.reminderDirectEnabled),
    timeCodeReminderEnabled: readOptionalBoolean(record.timeCodeReminderEnabled),
  }
}

function readOptionalBoolean(value: unknown) {
  return typeof value === 'boolean' ? value : undefined
}

export async function handleAdmissionRuntimeAction(
  ctx: Context,
  deps: AdmissionConsoleAPIDeps,
  input: unknown,
  client?: ConsoleActionClient,
) {
  const parsed = parseAdmissionRuntimeActionInput(input)
  const messages = await getGroupGuardMessages(deps.messageProvider)
  const record = await deps.guardStore.getActiveByID(parsed.recordId)
  if (!record) {
    throw new Error(groupGuardMessage(messages, 'admissionConsoleRecordNotFound'))
  }

  try {
    switch (parsed.action) {
      case 'query':
        return await queryAdmissionSession(deps.platform, record, messages)
      case 'resend':
        return await resendAdmissionSession(ctx, deps, record, messages)
      case 'regenerate':
        return await regenerateAdmissionSession(ctx, deps, record, messages)
      case 'skip':
        return await skipAdmissionSession(ctx, deps, record, resolveConsoleOperatorID(client), messages)
      case 'reset-failures':
        return await resetAdmissionFailures(deps, record, resolveConsoleOperatorID(client), messages)
      case 'release-blacklist':
        return await releaseAdmissionBlacklist(deps, record, resolveConsoleOperatorID(client), messages)
    }
  } catch (error) {
    throw new Error(formatAdmissionConsoleActionError(error, messages, parsed.action, record))
  }
}

function serializeGuardMember(record: GuardMemberRecord) {
  return {
    id: record.id,
    platform: record.platform,
    botSelfId: record.botSelfId,
    guildId: record.guildId,
    channelId: record.channelId,
    memberId: record.memberId,
    memberName: record.memberName,
    verificationState: record.verificationState,
    admissionSessionID: record.admissionSessionID,
    backendSyncPending: record.backendSyncPending,
    joinedAt: record.joinedAt.toISOString(),
    deadlineAt: record.deadlineAt.toISOString(),
    nextReminderAt: record.nextReminderAt?.toISOString() ?? null,
    manualReviewDeadlineAt: record.manualReviewDeadlineAt?.toISOString() ?? null,
    mutedAt: record.mutedAt?.toISOString() ?? null,
    reminderSentAt: record.reminderSentAt?.toISOString() ?? null,
    lastError: record.lastError,
    availableActions: availableActionsForMember(record),
  }
}

function availableActionsForMember(record: GuardMemberRecord): AdmissionRuntimeAction[] {
  const actions: AdmissionRuntimeAction[] = ['query', 'reset-failures', 'release-blacklist']
  if (!record.backendSyncPending) {
    actions.push('resend', 'regenerate', 'skip')
  } else {
    actions.push('regenerate', 'skip')
  }
  return actions
}

function parseAdmissionRuntimeActionInput(input: unknown): AdmissionRuntimeActionInput {
  if (!input || typeof input !== 'object' || Array.isArray(input)) {
    throw new Error('admission action input must be an object')
  }
  const record = input as Record<string, unknown>
  const recordId = typeof record.recordId === 'string' ? record.recordId.trim() : ''
  const action = typeof record.action === 'string' ? record.action.trim() : ''
  if (!recordId) {
    throw new Error('admission action recordId is required')
  }
  if (!isAdmissionRuntimeAction(action)) {
    throw new Error(`unsupported admission action: ${action || '(empty)'}`)
  }
  return { recordId, action }
}

function isAdmissionRuntimeAction(value: string): value is AdmissionRuntimeAction {
  return value === 'query' ||
    value === 'resend' ||
    value === 'regenerate' ||
    value === 'skip' ||
    value === 'reset-failures' ||
    value === 'release-blacklist'
}

async function queryAdmissionSession(
  platform: PlatformClient,
  record: GuardMemberRecord,
  messages: GroupGuardMessages,
) {
  const session = await platform.getAdmissionSessionByMember(admissionSubject(record))
  return formatAdmissionSessionSummary(session, messages)
}

async function resendAdmissionSession(
  ctx: Context,
  deps: AdmissionConsoleAPIDeps,
  record: GuardMemberRecord,
  messages: GroupGuardMessages,
) {
  const session = await deps.platform.resendAdmissionSessionLink(admissionSubject(record))
  const reminded = await sendReminderForRecord(ctx, deps, record, session, messages)
  if (reminded === false) {
    return groupGuardMessage(messages, 'admissionConsoleStaleRecord')
  }
  return groupGuardMessage(messages, 'admissionConsoleResendSuccess', { qqID: record.memberId })
}

async function regenerateAdmissionSession(
  ctx: Context,
  deps: AdmissionConsoleAPIDeps,
  record: GuardMemberRecord,
  messages: GroupGuardMessages,
) {
  const created = await deps.platform.regenerateAdmissionSessionLink({
    ...admissionSubject(record),
    channelID: record.channelId,
    botSelfID: record.botSelfId,
  })
  if (created.session.status === 'verified') {
    const bot = requireBotForRecord(ctx, record, messages)
    await bot.muteGuildMember(record.guildId, record.memberId, 0)
    const synced = await deps.guardStore.markBackendSynced(record.id, backendSyncUpdate(created))
    if (synced === false) {
      return groupGuardMessage(messages, 'admissionConsoleStaleRecord')
    }
    const released = await deps.guardStore.markReleased(record.id, new Date())
    if (released === false) {
      return groupGuardMessage(messages, 'admissionConsoleStaleRecord')
    }
    await deps.platform.recordAdmissionEvent(created.session.id, {
      action: 'release',
      success: true,
    })
    return groupGuardMessage(messages, 'admissionConsoleVerifiedReleaseSuccess', { qqID: record.memberId })
  }
  const bot = requireBotForRecord(ctx, record, messages)
  await resetMemberMute(bot, created.session, messages)
  const synced = await deps.guardStore.markBackendSynced(record.id, backendSyncUpdate(created))
  if (synced === false) {
    return groupGuardMessage(messages, 'admissionConsoleStaleRecord')
  }
  const reminded = await sendReminderForRecord(ctx, deps, record, created.session, messages)
  if (reminded === false) {
    return groupGuardMessage(messages, 'admissionConsoleStaleRecord')
  }
  return groupGuardMessage(messages, 'admissionConsoleRegenerateSuccess', { qqID: record.memberId })
}

async function skipAdmissionSession(
  ctx: Context,
  deps: AdmissionConsoleAPIDeps,
  record: GuardMemberRecord,
  operatorQQID: string,
  messages: GroupGuardMessages,
) {
  const ref = admissionRecordSubjectRef(record)
  deps.admissionSubjectCoordinator?.cancelSubject(ref)
  try {
    const session = await skipAdmissionSessionOrUseCancelled(deps.platform, record, operatorQQID)
    let unmuteError: unknown
    const released = await runAdmissionRecordSubjectExclusive(ref, deps, session.id, async () => {
      const synced = await deps.guardStore.markBackendSynced(record.id, {
        admissionSessionID: session.id,
        backendSyncPending: false,
        deadlineAt: new Date(session.linkWaitDeadlineAt),
        nextReminderAt: null,
        manualReviewDeadlineAt: session.manualReviewDeadlineAt ? new Date(session.manualReviewDeadlineAt) : null,
      })
      if (synced === false) return false
      const released = await deps.guardStore.markReleased(record.id, new Date())
      if (released === false) return false
      try {
        const bot = requireBotForRecord(ctx, record, messages)
        await bot.muteGuildMember(record.guildId, record.memberId, 0)
      } catch (error) {
        unmuteError = error
      }
      return true
    })
    if (released === false) {
      return groupGuardMessage(messages, 'admissionConsoleStaleRecord')
    }
    if (unmuteError) {
      return groupGuardMessage(messages, 'admissionConsoleSkipUnmuteFailed', {
        qqID: record.memberId,
        error: formatAdmissionActionError(unmuteError),
      })
    }
    return groupGuardMessage(messages, 'admissionConsoleSkipSuccess', { qqID: record.memberId })
  } catch (error) {
    deps.admissionSubjectCoordinator?.clearSubjectCancellation(ref)
    throw error
  }
}

async function skipAdmissionSessionOrUseCancelled(
  platform: PlatformClient,
  record: GuardMemberRecord,
  operatorQQID: string,
) {
  try {
    return await platform.skipAdmissionSessionForMember({
      ...admissionSubject(record),
      operatorQQID,
    })
  } catch (error) {
    if (!isAdmissionInvalidStateError(error)) {
      throw error
    }
    const session = await platform.getAdmissionSessionByMember(admissionSubject(record))
    if (session.status === 'cancelled') {
      return session
    }
    throw error
  }
}

async function resetAdmissionFailures(
  deps: AdmissionConsoleAPIDeps,
  record: GuardMemberRecord,
  operatorQQID: string,
  messages: GroupGuardMessages,
) {
  const result = await deps.platform.resetAdmissionFailureCount({
    ...admissionSubject(record),
    operatorQQID,
  })
  return groupGuardMessage(messages, 'admissionConsoleResetFailureCountSuccess', {
    qqID: result.qqID,
    previousFailureCount: result.previousFailureCount,
  })
}

async function releaseAdmissionBlacklist(
  deps: AdmissionConsoleAPIDeps,
  record: GuardMemberRecord,
  operatorQQID: string,
  messages: GroupGuardMessages,
) {
  await deps.platform.releaseMemberBlacklistBySubject({
    platform: record.platform,
    subjectType: 'qq_user',
    subjectID: record.memberId,
    scopeType: 'guild',
    guildID: record.guildId,
    releaseReasonCode: 'release_only',
    releaseReason: 'released by Koishi WebUI admission console',
    operatorQQID,
  })
  return groupGuardMessage(messages, 'admissionConsoleReleaseBlacklistSuccess', { qqID: record.memberId })
}

async function sendReminderForRecord(
  ctx: Context,
  deps: AdmissionConsoleAPIDeps,
  record: GuardMemberRecord,
  session: AdmissionSession,
  messages: GroupGuardMessages,
) {
  if (!session.authURL) {
    throw new Error(groupGuardMessage(messages, 'admissionConsoleMissingResendURL'))
  }
  const bot = requireBotForRecord(ctx, record, messages)
  const delivery = await deps.runtimeSettings.getAdmissionReminderDeliveryConfig()
  const result = await sendAdmissionReminderMessage({
    bot,
    guildId: record.guildId,
    channelId: record.channelId,
    memberId: record.memberId,
    content: formatAdmissionReminder({
      memberId: record.memberId,
      authURL: session.authURL,
      deadlineAt: reminderDeadline(session),
      failureCount: session.failureCount,
      remainingRetryCount: session.remainingRetryCount,
      willBlacklistOnTimeout: session.willBlacklistOnTimeout,
      messages,
    }),
    delivery,
    messages,
    shouldSend: () => !deps.admissionSubjectCoordinator?.isCancelled(record, session.id),
  })
  if (result.cancelled) {
    return false
  }
  const reminded = await deps.guardStore.markReminderSent(record.id, new Date())
  if (reminded === false) {
    return false
  }
  await deps.platform.recordAdmissionEvent(session.id, {
    action: 'remind',
    success: true,
    ...(result.messageID ? { messageID: result.messageID } : {}),
  })
  return true
}

async function resetMemberMute(
  bot: Universal.Methods,
  session: AdmissionSession,
  messages: GroupGuardMessages,
) {
  const muteDurationMs = new Date(session.initialMuteUntil).getTime() - Date.now()
  if (!Number.isFinite(muteDurationMs) || muteDurationMs <= 0) {
    throw new Error(groupGuardMessage(messages, 'admissionConsoleInvalidMuteDeadline'))
  }
  await bot.muteGuildMember(session.guildID, session.qqID, muteDurationMs)
}

function requireBotForRecord(
  ctx: Context,
  record: GuardMemberRecord,
  messages: GroupGuardMessages,
): GuardBotRuntime {
  for (const bot of ctx.bots as GuardBotRuntime[]) {
    if (bot.selfId !== record.botSelfId) {
      continue
    }
    const platform = requireAdmissionActionPlatform(bot)
    if (platform === record.platform) {
      return bot
    }
  }
  throw new Error(groupGuardMessage(messages, 'admissionConsoleBotNotFound', {
    platform: record.platform,
    botSelfID: record.botSelfId,
  }))
}


function admissionSubject(record: GuardMemberRecord) {
  return {
    platform: record.platform,
    guildID: record.guildId,
    qqID: record.memberId,
  }
}

async function runAdmissionRecordSubjectExclusive<T>(
  ref: AdmissionSubjectRef,
  deps: AdmissionConsoleAPIDeps,
  admissionSessionID: string,
  run: () => Promise<T>,
) {
  if (!deps.admissionSubjectCoordinator) {
    return run()
  }
  deps.admissionSubjectCoordinator.cancel(ref, admissionSessionID)
  try {
    return await deps.admissionSubjectCoordinator.runExclusive(ref, run)
  } finally {
    deps.admissionSubjectCoordinator.clearSubjectCancellation(ref)
  }
}

function admissionRecordSubjectRef(record: GuardMemberRecord): AdmissionSubjectRef {
  return {
    platform: record.platform,
    botSelfId: record.botSelfId,
    guildId: record.guildId,
    memberId: record.memberId,
  }
}

function reminderDeadline(session: AdmissionSession) {
  switch (session.status) {
    case 'linked':
      return new Date(session.submissionWaitDeadlineAt)
    case 'material_submitted':
      return new Date(session.manualReviewDeadlineAt || session.submissionWaitDeadlineAt)
    default:
      return new Date(session.linkWaitDeadlineAt)
  }
}

function formatAdmissionSessionSummary(
  session: AdmissionSession,
  messages: GroupGuardMessages,
) {
  return compactRenderedMessage(groupGuardMessage(messages, 'admissionQuerySummary', {
    qqID: session.qqID,
    statusLabel: statusLabel(session.status, messages),
    sessionID: session.id,
    qqLinkedLabel: isQQLinked(session)
      ? groupGuardMessage(messages, 'admissionQueryQQLinked')
      : groupGuardMessage(messages, 'admissionQueryQQUnlinked'),
    studentVerificationLabel: studentVerificationLabel(session, messages),
    deadlineLine: describeDeadline(session, messages) ?? '',
    nextStep: nextAdmissionStep(session, messages),
    lastBotErrorLine: session.lastBotError
      ? groupGuardMessage(messages, 'admissionQueryLastBotError', { lastBotError: session.lastBotError })
      : '',
  }))
}

function resolveConsoleOperatorID(client: ConsoleActionClient | undefined) {
  return `console:${client?.auth?.id ?? 'unknown'}`
}

function formatAdmissionConsoleActionError(
  error: unknown,
  messages: GroupGuardMessages,
  action: AdmissionRuntimeAction,
  record: GuardMemberRecord,
) {
  if (error instanceof PlatformAPIErrorClass) {
    if (error.status === 404) {
      return formatAdmissionConsoleNotFoundError(messages, action, record)
    }
    if (error.status === 409) {
      return groupGuardMessage(messages, 'admissionConsoleErrorInvalidState')
    }
    if (error.status === 401 || error.status === 403) {
      return groupGuardMessage(messages, 'admissionConsoleErrorUnauthorized')
    }
    return groupGuardMessage(messages, 'admissionConsoleErrorPlatform', {
      status: error.status,
      message: error.message,
    })
  }
  const platformError = error as PlatformAPIError | undefined
  if (platformError?.name === 'PlatformAPIError' && typeof platformError.status === 'number') {
    if (platformError.status === 404) {
      return formatAdmissionConsoleNotFoundError(messages, action, record)
    }
    return groupGuardMessage(messages, 'admissionConsoleErrorPlatform', {
      status: platformError.status,
      message: platformError.message,
    })
  }
  return error instanceof Error ? error.message : groupGuardMessage(messages, 'admissionConsoleErrorFallback')
}

function formatAdmissionConsoleNotFoundError(
  messages: GroupGuardMessages,
  action: AdmissionRuntimeAction,
  record: GuardMemberRecord,
) {
  if (action === 'release-blacklist') {
    return groupGuardMessage(messages, 'admissionConsoleReleaseBlacklistNotFound', { qqID: record.memberId })
  }
  return groupGuardMessage(messages, 'admissionConsoleErrorNotFound', { qqID: record.memberId })
}

function isAdmissionInvalidStateError(error: unknown) {
  if (error instanceof PlatformAPIErrorClass) {
    return error.status === 409
  }
  const platformError = error as PlatformAPIError | undefined
  return platformError?.name === 'PlatformAPIError' && platformError.status === 409
}

function describeDeadline(
  session: AdmissionSession,
  messages: GroupGuardMessages,
) {
  switch (session.status) {
    case 'joined_muted':
      return groupGuardMessage(messages, 'admissionQueryDeadlineLink', { deadlineAt: session.linkWaitDeadlineAt })
    case 'linked':
      return groupGuardMessage(messages, 'admissionQueryDeadlineSubmission', { deadlineAt: session.submissionWaitDeadlineAt })
    case 'material_submitted':
      return groupGuardMessage(messages, 'admissionQueryDeadlineManualReview', {
        deadlineAt: session.manualReviewDeadlineAt || groupGuardMessage(messages, 'admissionQueryDeadlineUnset'),
      })
    default:
      return undefined
  }
}

function isQQLinked(session: AdmissionSession) {
  return hasLinkedUser(session) ||
    session.status === 'linked' ||
    session.status === 'material_submitted' ||
    session.status === 'verified'
}

function hasLinkedUser(session: AdmissionSession) {
  return session.userID !== undefined && session.userID !== null && String(session.userID).trim() !== ''
}

function studentVerificationLabel(
  session: AdmissionSession,
  messages: GroupGuardMessages,
) {
  switch (session.status) {
    case 'verified':
      return groupGuardMessage(messages, 'admissionQueryStudentVerified')
    case 'material_submitted':
      return groupGuardMessage(messages, 'admissionQueryStudentFreshmanPending')
    default:
      return groupGuardMessage(messages, 'admissionQueryStudentUnverified')
  }
}

function nextAdmissionStep(
  session: AdmissionSession,
  messages: GroupGuardMessages,
) {
  switch (session.status) {
    case 'joined_muted':
      return groupGuardMessage(messages, 'admissionNextStepJoinedMuted')
    case 'linked':
      return groupGuardMessage(messages, 'admissionNextStepLinked')
    case 'material_submitted':
      return groupGuardMessage(messages, 'admissionNextStepMaterialSubmitted')
    case 'verified':
      return session.lastBotError
        ? groupGuardMessage(messages, 'admissionNextStepVerifiedWithBotError')
        : groupGuardMessage(messages, 'admissionNextStepVerified')
    case 'expired_kicked':
      return hasLinkedUser(session)
        ? groupGuardMessage(messages, 'admissionNextStepExpiredKickedLinked')
        : groupGuardMessage(messages, 'admissionNextStepExpiredKicked')
    case 'cancelled':
      return groupGuardMessage(messages, 'admissionNextStepCancelled')
    default:
      return groupGuardMessage(messages, 'admissionNextStepDefault')
  }
}

function statusLabel(
  status: AdmissionSession['status'],
  messages: GroupGuardMessages,
) {
  switch (status) {
    case 'joined_muted':
      return groupGuardMessage(messages, 'admissionStatusJoinedMuted')
    case 'linked':
      return groupGuardMessage(messages, 'admissionStatusLinked')
    case 'material_submitted':
      return groupGuardMessage(messages, 'admissionStatusMaterialSubmitted')
    case 'verified':
      return groupGuardMessage(messages, 'admissionStatusVerified')
    case 'expired_kicked':
      return groupGuardMessage(messages, 'admissionStatusExpiredKicked')
    case 'cancelled':
      return groupGuardMessage(messages, 'admissionStatusCancelled')
    default:
      return status
  }
}

function compactRenderedMessage(message: string) {
  return message
    .split('\n')
    .map((line) => line.trimEnd())
    .filter((line) => line.trim())
    .join('\n')
}

function redactURL(value: string) {
  try {
    const url = new URL(value)
    url.username = ''
    url.password = ''
    return url.toString().replace(/\/$/, '')
  } catch {
    return value
  }
}
