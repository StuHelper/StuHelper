import type {} from '@koishijs/plugin-console'
import type { Context, Universal } from 'koishi'

import type {
  AdmissionSession,
  AdmissionSessionCreateResult,
  AdmissionRuntimeSettingsInput,
  AdmissionRuntimeSettingsStore,
  GuardMemberRecord,
  GuardMemberStore,
  GuardPolicyStore,
  PlatformClient,
  PlatformAPIError,
  StuhelperGroupGuardPluginConfig,
} from '@stuhelper/koishi-shared'
import {
  PlatformAPIError as PlatformAPIErrorClass,
} from '@stuhelper/koishi-shared'

import { formatAdmissionReminder } from './admission-format'
import { backendSyncUpdate } from './member-records'
import { requireAdmissionActionPlatform } from './admission-action-boundary'
import type { GuardBotRuntime } from './member-guard'

const ADMISSION_RUNTIME_PAGE_EVENT = 'stuhelperGroupGuard/page/admission-runtime'
const ADMISSION_RUNTIME_ACTION_EVENT = 'stuhelperGroupGuard/action/admission-member'
const ADMISSION_RUNTIME_SETTINGS_EVENT = 'stuhelperGroupGuard/action/save-admission-runtime-settings'
const CONSOLE_AUTHORITY = 4
const ACTIVE_MEMBER_LIMIT = 100
const STALE_ADMISSION_RECORD_MESSAGE = '入群认证记录已被其他任务处理，请刷新页面后确认当前状态。'

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
  readonly guardStore: GuardMemberStore
  readonly policyStore: GuardPolicyStore
  readonly onRuntimeSettingsChanged?: () => void | Promise<void>
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
  }, { authority: CONSOLE_AUTHORITY })

  ctx.console.addListener(ADMISSION_RUNTIME_ACTION_EVENT, async function (input) {
    return handleAdmissionRuntimeAction(ctx, deps, input, this as ConsoleActionClient)
  }, { authority: CONSOLE_AUTHORITY })

  ctx.console.addListener(ADMISSION_RUNTIME_SETTINGS_EVENT, async (input) => {
    await deps.runtimeSettings.saveSettings(parseRuntimeSettingsInput(input))
    await deps.onRuntimeSettingsChanged?.()
    return '已保存入群认证运行开关。'
  }, { authority: CONSOLE_AUTHORITY })
}

export async function buildAdmissionRuntimePageData(ctx: Context, deps: AdmissionConsoleAPIDeps) {
  const [activeMembers, templates, bindings, settings] = await Promise.all([
    deps.guardStore.listActive(),
    deps.policyStore.listTemplates(),
    deps.policyStore.listBindings(),
    deps.runtimeSettings.getSettings(),
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
    guard: {
      targetGroups: [...deps.config.guard.targetGroups],
      muteDurationSeconds: deps.config.guard.muteDurationSeconds,
      kickAfterMinutes: deps.config.guard.kickAfterMinutes,
      reminderTemplate: deps.config.guard.reminderTemplate,
      exemptUserCount: deps.config.guard.exemptUsers.length,
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
      publicCommandsRegistered: deps.config.commands?.enabled !== false,
      publicCommandsEnabled: settings.publicCommandsEnabled,
      admissionCommandsRegistered: deps.config.admissionCommands?.enabled !== false,
      admissionCommandsEnabled: settings.admissionCommandsEnabled,
      admissionCommandMinAuthority: deps.config.admissionCommands?.minAuthority,
      admissionCommandOperatorQQIDCount: deps.config.admissionCommands?.operatorQQIDs?.length ?? 0,
    },
    moderation: {
      enabled: settings.moderationEnabled,
      keywordRuleCount: deps.config.moderation.keywordRules.length,
      repeatThreshold: deps.config.moderation.repeatThreshold,
      repeatWindowSize: deps.config.moderation.repeatWindowSize,
      antiRecallNotify: deps.config.moderation.antiRecallNotify,
    },
    freshmanForward: {
      enabled: settings.freshmanForwardEnabled,
    },
    bots: ctx.bots.map((bot) => ({
      platform: bot.platform,
      selfId: bot.selfId,
      status: String((bot as { status?: unknown }).status ?? 'unknown'),
    })),
    stats: {
      targetGroupCount: deps.config.guard.targetGroups.length,
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
    admissionCommandsEnabled: readOptionalBoolean(record.admissionCommandsEnabled),
    moderationEnabled: readOptionalBoolean(record.moderationEnabled),
    freshmanForwardEnabled: readOptionalBoolean(record.freshmanForwardEnabled),
    fallbackScanEnabled: readOptionalBoolean(record.fallbackScanEnabled),
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
  const record = await deps.guardStore.getActiveByID(parsed.recordId)
  if (!record) {
    throw new Error('入群认证记录不存在或已结束。')
  }

  try {
    switch (parsed.action) {
      case 'query':
        return await queryAdmissionSession(deps.platform, record)
      case 'resend':
        return await resendAdmissionSession(ctx, deps, record)
      case 'regenerate':
        return await regenerateAdmissionSession(ctx, deps, record)
      case 'skip':
        return await skipAdmissionSession(ctx, deps, record, resolveConsoleOperatorID(client))
      case 'reset-failures':
        return await resetAdmissionFailures(deps.platform, record, resolveConsoleOperatorID(client))
      case 'release-blacklist':
        return await releaseAdmissionBlacklist(deps.platform, record, resolveConsoleOperatorID(client))
    }
  } catch (error) {
    throw new Error(formatAdmissionConsoleActionError(error))
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

async function queryAdmissionSession(platform: PlatformClient, record: GuardMemberRecord) {
  const session = await platform.getAdmissionSessionByMember(admissionSubject(record))
  return formatAdmissionSessionSummary(session)
}

async function resendAdmissionSession(
  ctx: Context,
  deps: AdmissionConsoleAPIDeps,
  record: GuardMemberRecord,
) {
  const session = await deps.platform.resendAdmissionSessionLink(admissionSubject(record))
  const reminded = await sendReminderForRecord(ctx, deps, record, session)
  if (reminded === false) {
    return STALE_ADMISSION_RECORD_MESSAGE
  }
  return `已重发 QQ ${record.memberId} 的入群认证链接。`
}

async function regenerateAdmissionSession(
  ctx: Context,
  deps: AdmissionConsoleAPIDeps,
  record: GuardMemberRecord,
) {
  const created = await deps.platform.regenerateAdmissionSessionLink({
    ...admissionSubject(record),
    channelID: record.channelId,
    botSelfID: record.botSelfId,
  })
  if (created.session.status === 'verified') {
    const bot = requireBotForRecord(ctx, record)
    await bot.muteGuildMember(record.guildId, record.memberId, 0)
    const synced = await deps.guardStore.markBackendSynced(record.id, backendSyncUpdate(created))
    if (synced === false) {
      return STALE_ADMISSION_RECORD_MESSAGE
    }
    const released = await deps.guardStore.markReleased(record.id, new Date())
    if (released === false) {
      return STALE_ADMISSION_RECORD_MESSAGE
    }
    await deps.platform.recordAdmissionEvent(created.session.id, {
      action: 'release',
      success: true,
    })
    return `QQ ${record.memberId} 已完成学生认证，已解除禁言。`
  }
  const bot = requireBotForRecord(ctx, record)
  await resetMemberMute(bot, created.session)
  const synced = await deps.guardStore.markBackendSynced(record.id, backendSyncUpdate(created))
  if (synced === false) {
    return STALE_ADMISSION_RECORD_MESSAGE
  }
  const reminded = await sendReminderForRecord(ctx, deps, record, created.session)
  if (reminded === false) {
    return STALE_ADMISSION_RECORD_MESSAGE
  }
  return `已重新生成 QQ ${record.memberId} 的入群认证链接并重置禁言。`
}

async function skipAdmissionSession(
  ctx: Context,
  deps: AdmissionConsoleAPIDeps,
  record: GuardMemberRecord,
  operatorQQID: string,
) {
  const session = await deps.platform.skipAdmissionSessionForMember({
    ...admissionSubject(record),
    operatorQQID,
  })
  const bot = requireBotForRecord(ctx, record)
  await bot.muteGuildMember(record.guildId, record.memberId, 0)
  const synced = await deps.guardStore.markBackendSynced(record.id, {
    admissionSessionID: session.id,
    backendSyncPending: false,
    deadlineAt: new Date(session.linkWaitDeadlineAt),
    nextReminderAt: null,
    manualReviewDeadlineAt: session.manualReviewDeadlineAt ? new Date(session.manualReviewDeadlineAt) : null,
  })
  if (synced === false) {
    return STALE_ADMISSION_RECORD_MESSAGE
  }
  const released = await deps.guardStore.markReleased(record.id, new Date())
  if (released === false) {
    return STALE_ADMISSION_RECORD_MESSAGE
  }
  return `已跳过 QQ ${record.memberId} 在本群的入群认证并解除禁言。`
}

async function resetAdmissionFailures(
  platform: PlatformClient,
  record: GuardMemberRecord,
  operatorQQID: string,
) {
  const result = await platform.resetAdmissionFailureCount({
    ...admissionSubject(record),
    operatorQQID,
  })
  return `已清空 QQ ${result.qqID} 在本群的入群未认证次数（原次数：${result.previousFailureCount}）。`
}

async function releaseAdmissionBlacklist(
  platform: PlatformClient,
  record: GuardMemberRecord,
  operatorQQID: string,
) {
  await platform.releaseMemberBlacklistBySubject({
    platform: record.platform,
    subjectType: 'qq_user',
    subjectID: record.memberId,
    scopeType: 'guild',
    guildID: record.guildId,
    releaseReasonCode: 'release_only',
    releaseReason: 'released by Koishi WebUI admission console',
    operatorQQID,
  })
  return `已解除 QQ ${record.memberId} 在本群的入群拉黑状态。`
}

async function sendReminderForRecord(
  ctx: Context,
  deps: AdmissionConsoleAPIDeps,
  record: GuardMemberRecord,
  session: AdmissionSession,
) {
  if (!session.authURL) {
    throw new Error('后端没有返回可发送的认证链接。')
  }
  const bot = requireBotForRecord(ctx, record)
  const messageID = await sendBotMessage(bot, record.channelId, formatAdmissionReminder({
    memberId: record.memberId,
    authURL: session.authURL,
    deadlineAt: reminderDeadline(session),
    failureCount: session.failureCount,
    remainingRetryCount: session.remainingRetryCount,
    willBlacklistOnTimeout: session.willBlacklistOnTimeout,
    messages: deps.config.messages,
  }))
  const reminded = await deps.guardStore.markReminderSent(record.id, new Date())
  if (reminded === false) {
    return false
  }
  await deps.platform.recordAdmissionEvent(session.id, {
    action: 'remind',
    success: true,
    ...(messageID ? { messageID } : {}),
  })
  return true
}

async function resetMemberMute(bot: Universal.Methods, session: AdmissionSession) {
  const muteDurationMs = new Date(session.initialMuteUntil).getTime() - Date.now()
  if (!Number.isFinite(muteDurationMs) || muteDurationMs <= 0) {
    throw new Error('入群认证禁言期限无效，无法重置禁言。')
  }
  await bot.muteGuildMember(session.guildID, session.qqID, muteDurationMs)
}

function requireBotForRecord(ctx: Context, record: GuardMemberRecord): GuardBotRuntime {
  for (const bot of ctx.bots as GuardBotRuntime[]) {
    if (bot.selfId !== record.botSelfId) {
      continue
    }
    const platform = requireAdmissionActionPlatform(bot)
    if (platform === record.platform) {
      return bot
    }
  }
  throw new Error(`未找到可操作该记录的 Bot：${record.platform}/${record.botSelfId}`)
}

async function sendBotMessage(bot: Universal.Methods, channelId: string, message: string) {
  if (!message) return undefined
  const result = await bot.sendMessage(channelId, message)
  if (Array.isArray(result)) {
    return typeof result[0] === 'string' ? result[0] : undefined
  }
  return typeof result === 'string' ? result : undefined
}

function admissionSubject(record: GuardMemberRecord) {
  return {
    platform: record.platform,
    guildID: record.guildId,
    qqID: record.memberId,
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

function formatAdmissionSessionSummary(session: AdmissionSession) {
  const deadline = reminderDeadline(session)
  return [
    `QQ ${session.qqID} 的入群认证状态：${session.status}`,
    `会话：${session.id}`,
    `截止：${deadline.toISOString()}`,
    `学生认证：${session.status === 'verified' ? '已通过' : '未完成'}`,
  ].join('\n')
}

function resolveConsoleOperatorID(client: ConsoleActionClient | undefined) {
  return `console:${client?.auth?.id ?? 'unknown'}`
}

function formatAdmissionConsoleActionError(error: unknown) {
  if (error instanceof PlatformAPIErrorClass) {
    if (error.status === 404) {
      return '未找到该 QQ 的入群认证记录或拉黑记录。'
    }
    if (error.status === 409) {
      return '当前入群认证状态不允许该操作。'
    }
    if (error.status === 401 || error.status === 403) {
      return '机器人服务凭据无权访问入群认证接口。'
    }
    return `StuHelper 平台接口异常：${error.status} ${error.message}`
  }
  const platformError = error as PlatformAPIError | undefined
  if (platformError?.name === 'PlatformAPIError' && typeof platformError.status === 'number') {
    return `StuHelper 平台接口异常：${platformError.status} ${platformError.message}`
  }
  return error instanceof Error ? error.message : '入群认证操作失败。'
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
