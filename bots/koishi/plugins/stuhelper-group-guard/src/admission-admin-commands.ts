import { h, type Context, type Session } from 'koishi'

import {
  COMMAND_POLICY_IDS,
  canExecuteCommand,
  type CommandPolicyRecord,
  type ModerationStore,
} from '@stuhelper/koishi-moderation-core'
import {
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
import { formatAdmissionActionError } from './admission-actions'
import {
  asPlatformAPIError,
  formatAdmissionSessionSummary,
  reminderDeadline,
  skipAdmissionSessionOrUseCancelled,
} from './admission-summary'
import { resolveAdmissionSubjectPlatform } from './admission-subject-platform'
import type { AdmissionSubjectCoordinator } from './admission-subject-coordinator'
import {
  ADMISSION_DEDUPE_WINDOW_MS,
  type AdmissionReminderDeduper,
} from './admission-reminder-deduper'
import { backendSyncUpdate } from './member-records'
import {
  getGroupGuardMessages,
  groupGuardMessage,
  type GroupGuardMessageProvider,
  type GroupGuardMessages,
} from './group-guard-message-provider'

const DEFAULT_ADMISSION_COMMAND_AUTHORITY = 4
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
        const skipped = await skipAdmissionSessionOrUseCancelled(
          deps.platform,
          admissionSubject(command),
          command.session.userId,
        )
        let unmuteError: unknown
        const released = await runAdmissionCommandSubjectExclusive(ref, deps, skipped.id, async () => {
          const marked = await markLocalAdmissionSkipped(command, deps.guardStore, skipped)
          if (marked === false) return false
          try {
            await releaseMemberMuteForCommand(command)
          } catch (error) {
            unmuteError = error
          }
          return true
        })
        const currentMessages = await getGroupGuardMessages(deps.messageProvider)
        if (released === false) {
          return groupGuardMessage(currentMessages, 'admissionCommandStaleRecord')
        }
        if (unmuteError) {
          return groupGuardMessage(currentMessages, 'admissionSkipUnmuteFailed', {
            at: h.at(command.qqID),
            qqID: command.qqID,
            error: formatAdmissionActionError(unmuteError),
          })
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
        if (asPlatformAPIError(error)?.status === 404) {
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
    if (claimedAt !== undefined && current - claimedAt <= ADMISSION_DEDUPE_WINDOW_MS) {
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
      if (nowMs - claimedAt > ADMISSION_DEDUPE_WINDOW_MS) {
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
  const platformError = asPlatformAPIError(error)
  if (platformError) {
    if (platformError.status === 404) {
      return groupGuardMessage(messages, 'admissionCommandNotFound')
    }
    if (platformError.status === 409) {
      return groupGuardMessage(messages, 'admissionCommandInvalidState')
    }
    if (platformError.status === 401 || platformError.status === 403) {
      return groupGuardMessage(messages, 'admissionCommandUnauthorized')
    }
    return groupGuardMessage(messages, 'admissionCommandPlatformError', {
      status: platformError.status,
      message: platformError.message,
    })
  }
  return groupGuardMessage(messages, 'admissionCommandFailed', {
    error: error instanceof Error ? error.message : '',
  })
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
