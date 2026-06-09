import { h, type Context, type Session } from 'koishi'

import {
  COMMAND_POLICY_IDS,
  canExecuteCommand,
  type CommandPolicyRecord,
  type ModerationStore,
} from '@stuhelper/koishi-moderation-core'
import {
  PlatformAPIError,
  renderMessageTemplate,
  resolveGroupGuardMessages,
  type AdmissionSession,
  type AdmissionSessionCreateResult,
  type AdmissionRuntimeSettingsStore,
  type GuardMemberStore,
  type GuardPolicyStore,
  type PlatformClient,
  type StuhelperGroupGuardPluginConfig,
} from '@stuhelper/koishi-shared'

import { formatAdmissionReminder } from './admission-format'
import { sendAdmissionReminderMessage } from './admission-reminder-delivery'
import { resolveAdmissionSubjectPlatform } from './admission-subject-platform'
import type { AdmissionSubjectCoordinator } from './admission-subject-coordinator'
import type { AdmissionReminderDeduper } from './admission-reminder-deduper'
import { backendSyncUpdate } from './member-records'
import {
  getGroupGuardMessages,
  groupGuardMessage,
  type GroupGuardMessageProvider,
  type GroupGuardMessages,
} from './group-guard-message-provider'

const DEFAULT_ADMISSION_COMMAND_AUTHORITY = 4
const DUPLICATE_COMMAND_SUPPRESS_MS = 30_000
const DEFAULT_ADMISSION_COMMAND_POLICY: CommandPolicyRecord = {
  commandId: COMMAND_POLICY_IDS.admissionAdmin,
  roles: [],
  minAuthority: DEFAULT_ADMISSION_COMMAND_AUTHORITY,
  createdAt: new Date(0),
  updatedAt: new Date(0),
}

interface AdmissionAdminCommandDeps {
  readonly platform: PlatformClient
  readonly guardStore: GuardMemberStore
  readonly policyStore: GuardPolicyStore
  readonly moderationStore: ModerationStore
  readonly config: StuhelperGroupGuardPluginConfig
  readonly runtimeSettings?: AdmissionRuntimeSettingsStore
  readonly admissionSubjectCoordinator?: AdmissionSubjectCoordinator
  readonly reminderDeduper?: AdmissionReminderDeduper
  readonly messageProvider?: GroupGuardMessageProvider
}

interface AdmissionCommandContext {
  readonly session: Session
  readonly platform: string
  readonly guildID: string
  readonly channelID: string
  readonly qqID: string
  readonly botSelfID: string
}

export function registerAdmissionAdminCommands(ctx: Context, deps: AdmissionAdminCommandDeps) {
  const commandDeduper = new AdmissionAdminCommandDeduper()
  const messages = resolveGroupGuardMessages()

  ctx.command('查询入群认证 <qqID>', renderMessageTemplate(messages.admissionQueryCommandDescription))
    .action(({ session }, qqID) => runAdmissionCommand(deps, async () => {
      const command = await resolveAdmissionCommandContext(session, qqID, deps)
      if (typeof command === 'string') return command
      const admission = await deps.platform.getAdmissionSessionByMember(admissionSubject(command))
      return formatAdmissionSessionSummary(admission, await getGroupGuardMessages(deps.messageProvider))
    }))

  ctx.command('重发认证链接 <qqID>', renderMessageTemplate(messages.admissionResendCommandDescription))
    .action(({ session }, qqID) => runAdmissionCommand(deps, async () => {
      const command = await resolveAdmissionCommandContext(session, qqID, deps)
      if (typeof command === 'string') return command
      const dedupeKey = admissionCommandDedupeKey('resend', command)
      if (!commandDeduper.claim(dedupeKey)) return
      try {
        const admission = await deps.platform.resendAdmissionSessionLink(admissionSubject(command))
        return sendAdmissionReminderForCommand(command, deps, admission)
      } catch (error) {
        commandDeduper.forget(dedupeKey)
        throw error
      }
    }))

  ctx.command('重新生成认证链接 <qqID>', renderMessageTemplate(messages.admissionRegenerateCommandDescription))
    .action(({ session }, qqID) => runAdmissionCommand(deps, async () => {
      const command = await resolveAdmissionCommandContext(session, qqID, deps)
      if (typeof command === 'string') return command
      const dedupeKey = admissionCommandDedupeKey('regenerate', command)
      if (!commandDeduper.claim(dedupeKey)) return
      try {
        const created = await deps.platform.regenerateAdmissionSessionLink({
          platform: command.platform,
          guildID: command.guildID,
          channelID: command.channelID,
          qqID: command.qqID,
          botSelfID: command.botSelfID,
        })
        if (created.session.status === 'verified') {
          return releaseVerifiedAdmissionForCommand(command, deps, created)
        }
        const currentMessages = await getGroupGuardMessages(deps.messageProvider)
        await resetMemberMute(command.session, created.session, currentMessages)
        const synced = await updateLocalAdmissionRecord(command, deps.guardStore, created)
        if (synced === false) {
          return groupGuardMessage(currentMessages, 'admissionCommandStaleRecord')
        }
        return sendAdmissionReminderForCommand(command, deps, created.session)
      } catch (error) {
        commandDeduper.forget(dedupeKey)
        throw error
      }
    }))

  ctx.command('跳过入群认证 <qqID>', renderMessageTemplate(messages.admissionSkipCommandDescription))
    .action(({ session }, qqID) => runAdmissionCommand(deps, async () => {
      const command = await resolveAdmissionCommandContext(session, qqID, deps)
      if (typeof command === 'string') return command
      const dedupeKey = admissionCommandDedupeKey('skip', command)
      if (!commandDeduper.claim(dedupeKey)) return
      const ref = admissionCommandSubjectRef(command)
      deps.admissionSubjectCoordinator?.cancelSubject(ref)
      try {
        const skipped = await deps.platform.skipAdmissionSessionForMember(admissionOperatorSubject(command))
        const released = await runAdmissionCommandSubjectExclusive(ref, deps, skipped.id, async () => {
          await releaseMemberMuteForCommand(command)
          return markLocalAdmissionSkipped(command, deps.guardStore, skipped)
        })
        const currentMessages = await getGroupGuardMessages(deps.messageProvider)
        if (released === false) {
          return groupGuardMessage(currentMessages, 'admissionCommandStaleRecord')
        }
        return groupGuardMessage(currentMessages, 'admissionSkipSuccess', {
          at: h.at(command.qqID),
          qqID: command.qqID,
        })
      } catch (error) {
        deps.admissionSubjectCoordinator?.clearSubjectCancellation(ref)
        commandDeduper.forget(dedupeKey)
        throw error
      }
    }))

  ctx.command('清空入群未认证次数 <qqID>', renderMessageTemplate(messages.admissionResetFailureCountCommandDescription))
    .action(({ session }, qqID) => runAdmissionCommand(deps, async () => {
      const command = await resolveAdmissionCommandContext(session, qqID, deps)
      if (typeof command === 'string') return command
      const result = await deps.platform.resetAdmissionFailureCount(admissionOperatorSubject(command))
      return groupGuardMessage(await getGroupGuardMessages(deps.messageProvider), 'admissionResetFailureCountSuccess', {
        qqID: result.qqID,
        previousFailureCount: result.previousFailureCount,
      })
    }))

  ctx.command('解除入群拉黑 <qqID>', renderMessageTemplate(messages.admissionReleaseBlacklistCommandDescription))
    .action(({ session }, qqID) => runAdmissionCommand(deps, async () => {
      const command = await resolveAdmissionCommandContext(session, qqID, deps)
      if (typeof command === 'string') return command
      try {
        await deps.platform.releaseMemberBlacklistBySubject({
          platform: command.platform,
          subjectType: 'qq_user',
          subjectID: command.qqID,
          scopeType: 'guild',
          guildID: command.guildID,
          releaseReasonCode: 'release_only',
          releaseReason: 'released by admission admin command',
          operatorQQID: command.session.userId,
        })
      } catch (error) {
        if (error instanceof PlatformAPIError && error.status === 404) {
          return groupGuardMessage(await getGroupGuardMessages(deps.messageProvider), 'admissionReleaseBlacklistNotFound', { qqID: command.qqID })
        }
        throw error
      }
      return groupGuardMessage(await getGroupGuardMessages(deps.messageProvider), 'admissionReleaseBlacklistSuccess', { qqID: command.qqID })
    }))
}

class AdmissionAdminCommandDeduper {
  private readonly claimedAtByKey = new Map<string, number>()

  claim(key: string, now = new Date()) {
    const current = now.getTime()
    const claimedAt = this.claimedAtByKey.get(key)
    if (claimedAt !== undefined && current - claimedAt <= DUPLICATE_COMMAND_SUPPRESS_MS) {
      return false
    }
    this.claimedAtByKey.set(key, current)
    this.prune(current)
    return true
  }

  forget(key: string) {
    this.claimedAtByKey.delete(key)
  }

  private prune(nowMs: number) {
    for (const [key, claimedAt] of this.claimedAtByKey) {
      if (nowMs - claimedAt > DUPLICATE_COMMAND_SUPPRESS_MS) {
        this.claimedAtByKey.delete(key)
      }
    }
  }
}

function admissionCommandDedupeKey(action: 'resend' | 'regenerate' | 'skip', command: AdmissionCommandContext) {
  return JSON.stringify([
    action,
    command.platform,
    command.botSelfID,
    command.guildID,
    command.qqID,
  ])
}

async function resolveAdmissionCommandContext(
  session: Session | undefined,
  qqID: string | undefined,
  deps: AdmissionAdminCommandDeps,
): Promise<AdmissionCommandContext | string> {
  const messages = await getGroupGuardMessages(deps.messageProvider)
  if (!session) {
    return groupGuardMessage(messages, 'admissionCommandGroupOnly')
  }
  if (deps.runtimeSettings && !await deps.runtimeSettings.isAdmissionCommandsEnabled()) {
    return groupGuardMessage(messages, 'admissionCommandsDisabled')
  }
  const guildID = session.guildId || session.channelId
  if (!guildID) {
    return groupGuardMessage(messages, 'admissionCommandGroupOnly')
  }
  const targetQQID = qqID?.trim()
  if (!targetQQID) {
    return groupGuardMessage(messages, 'admissionCommandMissingQQ')
  }
  if (!session.userId) {
    return groupGuardMessage(messages, 'admissionCommandMissingOperator')
  }
  const accessDenied = await ensureAdmissionCommandAccess(session, guildID, deps)
  if (accessDenied) {
    return accessDenied
  }
  const platform = resolveSessionAdmissionPlatform(session)
  if (!platform) {
    return groupGuardMessage(messages, 'admissionCommandUnsupportedPlatform')
  }
  const policy = await deps.policyStore.resolvePolicy(platform, guildID)
  if (!policy) {
    return groupGuardMessage(messages, 'admissionCommandPolicyDisabled')
  }
  return {
    session,
    platform,
    guildID,
    channelID: session.channelId || guildID,
    qqID: targetQQID,
    botSelfID: session.selfId,
  }
}

function resolveSessionAdmissionPlatform(session: Session) {
  return resolveAdmissionSubjectPlatform(session.platform) ||
    resolveAdmissionSubjectPlatform((session.bot as { platform?: string }).platform)
}

async function ensureAdmissionCommandAccess(
  session: Session,
  guildID: string,
  deps: AdmissionAdminCommandDeps,
) {
  const [policy, memberRoles] = await Promise.all([
    deps.moderationStore.getCommandPolicy(COMMAND_POLICY_IDS.admissionAdmin),
    deps.moderationStore.getMemberRoles(guildID, session.userId),
  ])
  const allowed = canExecuteCommand({
    authority: resolveAuthority(session),
    memberRoles,
    policy: policy ?? DEFAULT_ADMISSION_COMMAND_POLICY,
  })
  if (allowed) {
    return
  }
  return groupGuardMessage(await getGroupGuardMessages(deps.messageProvider), 'commandAccessDenied')
}

function resolveAuthority(session: Session) {
  return (session as { user?: { authority?: number } }).user?.authority ?? 0
}

function admissionSubject(command: AdmissionCommandContext) {
  return {
    platform: command.platform,
    guildID: command.guildID,
    qqID: command.qqID,
  }
}

function admissionOperatorSubject(command: AdmissionCommandContext) {
  return {
    ...admissionSubject(command),
    operatorQQID: command.session.userId,
  }
}

async function runAdmissionCommandSubjectExclusive<T>(
  ref: ReturnType<typeof admissionCommandSubjectRef>,
  deps: AdmissionAdminCommandDeps,
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

function admissionCommandSubjectRef(command: AdmissionCommandContext) {
  return {
    platform: command.platform,
    botSelfId: command.botSelfID,
    guildId: command.guildID,
    memberId: command.qqID,
  }
}

async function runAdmissionCommand(
  deps: AdmissionAdminCommandDeps,
  run: () => Promise<string | void>,
) {
  try {
    return await run()
  } catch (error) {
    return formatAdmissionCommandError(error, await getGroupGuardMessages(deps.messageProvider))
  }
}

function formatAdmissionCommandError(error: unknown, messages: GroupGuardMessages) {
  if (error instanceof PlatformAPIError) {
    if (error.status === 404) {
      return groupGuardMessage(messages, 'admissionCommandNotFound')
    }
    if (error.status === 409) {
      return groupGuardMessage(messages, 'admissionCommandInvalidState')
    }
    if (error.status === 401 || error.status === 403) {
      return groupGuardMessage(messages, 'admissionCommandUnauthorized')
    }
    return groupGuardMessage(messages, 'admissionCommandPlatformError', {
      status: error.status,
      message: error.message,
    })
  }
  return groupGuardMessage(messages, 'admissionCommandFailed', {
    error: error instanceof Error ? error.message : '',
  })
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

function formatAdmissionReminderForSession(
  session: AdmissionSession,
  messages: GroupGuardMessages,
) {
  if (!session.authURL) {
    return groupGuardMessage(messages, 'admissionCommandMissingResendURL')
  }
  return formatAdmissionReminder({
    memberId: session.qqID,
    authURL: session.authURL,
    deadlineAt: reminderDeadline(session),
    messages,
  })
}

async function sendAdmissionReminderForCommand(
  command: AdmissionCommandContext,
  deps: AdmissionAdminCommandDeps,
  admission: AdmissionSession,
) {
  const messages = await getGroupGuardMessages(deps.messageProvider)
  const message = formatAdmissionReminderForSession(admission, messages)
  if (!admission.authURL) {
    return message
  }
  const delivery = await deps.runtimeSettings?.getAdmissionReminderDeliveryConfig()
  deps.reminderDeduper?.remember(admission.id)
  let messageID: string | undefined
  try {
    const result = await sendAdmissionReminderMessage({
      bot: command.session.bot,
      guildId: command.guildID,
      channelId: command.channelID,
      memberId: command.qqID,
      content: message,
      delivery,
      messages,
      sendGroupMessage: (content) => command.session.send(content),
    })
    messageID = result.messageID
  } catch (error) {
    deps.reminderDeduper?.forget(admission.id)
    throw error
  }
  const marked = await markLocalReminderSent(command, deps.guardStore)
  if (marked === false) {
    return groupGuardMessage(messages, 'admissionCommandStaleRecord')
  }
  await deps.platform.recordAdmissionEvent(admission.id, {
    action: 'remind',
    success: true,
    ...(messageID ? { messageID } : {}),
  })
}

async function resetMemberMute(
  session: Session,
  admission: AdmissionSession,
  messages: GroupGuardMessages,
) {
  const muteDuration = new Date(admission.initialMuteUntil).getTime() - Date.now()
  if (!Number.isFinite(muteDuration) || muteDuration <= 0) {
    throw new Error(groupGuardMessage(messages, 'admissionCommandInvalidMuteDeadline'))
  }
  await session.bot.muteGuildMember(admission.guildID, admission.qqID, muteDuration)
}

async function releaseMemberMuteForCommand(command: AdmissionCommandContext) {
  await command.session.bot.muteGuildMember(command.guildID, command.qqID, 0)
}

async function releaseVerifiedAdmissionForCommand(
  command: AdmissionCommandContext,
  deps: AdmissionAdminCommandDeps,
  created: AdmissionSessionCreateResult,
) {
  const messages = await getGroupGuardMessages(deps.messageProvider)
  await command.session.bot.muteGuildMember(created.session.guildID, created.session.qqID, 0)
  const record = await deps.guardStore.findActiveBySubject({
    platform: command.platform,
    botSelfId: command.botSelfID,
    guildId: command.guildID,
    memberId: command.qqID,
  })
  if (record) {
    const synced = await deps.guardStore.markBackendSynced(record.id, backendSyncUpdate(created))
    if (synced === false) {
      return groupGuardMessage(messages, 'admissionCommandStaleRecord')
    }
    const released = await deps.guardStore.markReleased(record.id, new Date())
    if (released === false) {
      return groupGuardMessage(messages, 'admissionCommandStaleRecord')
    }
  }
  await deps.platform.recordAdmissionEvent(created.session.id, {
    action: 'release',
    success: true,
  })
  return groupGuardMessage(messages, 'admissionAlreadyVerifiedRegenerate', {
    at: h.at(command.qqID),
    qqID: command.qqID,
  })
}

async function markLocalAdmissionSkipped(
  command: AdmissionCommandContext,
  guardStore: GuardMemberStore,
  admission: AdmissionSession,
) {
  const record = await guardStore.findActiveBySubject({
    platform: command.platform,
    botSelfId: command.botSelfID,
    guildId: command.guildID,
    memberId: command.qqID,
  })
  if (!record) return true
  const synced = await guardStore.markBackendSynced(record.id, {
    admissionSessionID: admission.id,
    backendSyncPending: false,
    deadlineAt: new Date(admission.linkWaitDeadlineAt),
    nextReminderAt: null,
    manualReviewDeadlineAt: admission.manualReviewDeadlineAt ? new Date(admission.manualReviewDeadlineAt) : null,
  })
  if (synced === false) return false
  const released = await guardStore.markReleased(record.id, new Date())
  return released !== false
}

async function updateLocalAdmissionRecord(
  command: AdmissionCommandContext,
  guardStore: GuardMemberStore,
  created: AdmissionSessionCreateResult,
) {
  const record = await guardStore.findActiveBySubject({
    platform: command.platform,
    botSelfId: command.botSelfID,
    guildId: command.guildID,
    memberId: command.qqID,
  })
  if (!record) return true
  const synced = await guardStore.markBackendSynced(record.id, backendSyncUpdate(created))
  return synced !== false
}

async function markLocalReminderSent(
  command: AdmissionCommandContext,
  guardStore: GuardMemberStore,
) {
  const record = await guardStore.findActiveBySubject({
    platform: command.platform,
    botSelfId: command.botSelfID,
    guildId: command.guildID,
    memberId: command.qqID,
  })
  if (!record) return true
  const marked = await guardStore.markReminderSent(record.id, new Date())
  return marked !== false
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
