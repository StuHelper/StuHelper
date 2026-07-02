import type {} from '@koishijs/plugin-console'
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
  StuhelperGroupGuardPluginConfig,
} from '@stuhelper/koishi-shared'
import {
  CONSOLE_MIN_AUTHORITY,
  DEFAULT_GROUP_GUARD_BEHAVIOR_SETTINGS,
} from '@stuhelper/koishi-shared'
import type { ModerationStore } from '@stuhelper/koishi-moderation-core'

import { formatAdmissionReminder } from './admission-format'
import {
  sendAdmissionReminderMessage,
  type AdmissionReminderDeliveryLogger,
} from './admission-reminder-delivery'
import { formatAdmissionActionError } from './admission-actions'
import {
  asPlatformAPIError,
  formatAdmissionSessionSummary,
  reminderDeadline,
  skipAdmissionSessionOrUseCancelled,
} from './admission-summary'
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
  readonly logger?: AdmissionReminderDeliveryLogger
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
  if (!ctx.console) {
    return
  }

  ctx.console.addListener(ADMISSION_RUNTIME_PAGE_EVENT, async () => {
    return buildAdmissionRuntimePageData(ctx, deps)
  }, { authority: CONSOLE_MIN_AUTHORITY })

  ctx.console.addListener(ADMISSION_RUNTIME_ACTION_EVENT, async function (input) {
    return handleAdmissionRuntimeAction(ctx, deps, input, this as ConsoleActionClient)
  }, { authority: CONSOLE_MIN_AUTHORITY })

  ctx.console.addListener(ADMISSION_RUNTIME_SETTINGS_EVENT, async (input) => {
    await deps.runtimeSettings.saveSettings(parseRuntimeSettingsInput(input))
    await deps.onRuntimeSettingsChanged?.()
    return groupGuardMessage(await getGroupGuardMessages(deps.messageProvider), 'admissionConsoleSettingsSaved')
  }, { authority: CONSOLE_MIN_AUTHORITY })
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
      enabled: binding.enabled,
      note: binding.note,
      updatedAt: binding.updatedAt.toISOString(),
    })),
    activeMembers: sortedMembers.map(serializeGuardMember),
  }
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
    throw new Error(formatAdmissionConsoleActionError(error, messages))
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
  // F051：后端 session 已重新生成，先持久化本地同步状态，再执行 bot 动作；
  // unmute/重新禁言失败只提示操作者，不能让本地记录停留在旧 session。
  if (created.session.status === 'verified') {
    const synced = await deps.guardStore.markBackendSynced(record.id, backendSyncUpdate(created))
    if (synced === false) {
      return groupGuardMessage(messages, 'admissionConsoleStaleRecord')
    }
    const released = await deps.guardStore.markReleased(record.id, new Date())
    if (released === false) {
      return groupGuardMessage(messages, 'admissionConsoleStaleRecord')
    }
    let unmuteError: unknown
    try {
      const bot = requireBotForRecord(ctx, record, messages)
      await bot.muteGuildMember(record.guildId, record.memberId, 0)
    } catch (error) {
      unmuteError = error
    }
    await deps.platform.recordAdmissionEvent(created.session.id, {
      action: 'release',
      success: true,
    })
    if (unmuteError) {
      return groupGuardMessage(messages, 'admissionConsoleVerifiedReleaseUnmuteFailed', {
        qqID: record.memberId,
        error: formatAdmissionActionError(unmuteError),
      })
    }
    return groupGuardMessage(messages, 'admissionConsoleVerifiedReleaseSuccess', { qqID: record.memberId })
  }
  const synced = await deps.guardStore.markBackendSynced(record.id, backendSyncUpdate(created))
  if (synced === false) {
    return groupGuardMessage(messages, 'admissionConsoleStaleRecord')
  }
  try {
    const bot = requireBotForRecord(ctx, record, messages)
    await resetMemberMute(bot, created.session, messages)
  } catch (error) {
    return groupGuardMessage(messages, 'admissionConsoleRegenerateMuteFailed', {
      qqID: record.memberId,
      error: formatAdmissionActionError(error),
    })
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
    const session = await skipAdmissionSessionOrUseCancelled(deps.platform, admissionSubject(record), operatorQQID)
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
    logger: deps.logger,
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

function resolveConsoleOperatorID(client: ConsoleActionClient | undefined) {
  return `console:${client?.auth?.id ?? 'unknown'}`
}

function formatAdmissionConsoleActionError(
  error: unknown,
  messages: GroupGuardMessages,
) {
  const platformError = asPlatformAPIError(error)
  if (platformError) {
    if (platformError.status === 404) {
      return groupGuardMessage(messages, 'admissionConsoleErrorNotFound')
    }
    if (platformError.status === 409) {
      return groupGuardMessage(messages, 'admissionConsoleErrorInvalidState')
    }
    if (platformError.status === 401 || platformError.status === 403) {
      return groupGuardMessage(messages, 'admissionConsoleErrorUnauthorized')
    }
    return groupGuardMessage(messages, 'admissionConsoleErrorPlatform', {
      status: platformError.status,
      message: platformError.message,
    })
  }
  return error instanceof Error ? error.message : groupGuardMessage(messages, 'admissionConsoleErrorFallback')
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
