import type { Context, Session } from 'koishi'

import {
  PlatformAPIError,
  type AdmissionSession,
  type AdmissionSessionCreateResult,
  type GuardPolicyStore,
  type PlatformClient,
  type StuhelperGroupGuardPluginConfig,
} from '@stuhelper/koishi-shared'

import { formatAdmissionReminder } from './admission-format'
import { resolveAdmissionSubjectPlatform } from './admission-subject-platform'
import type { AdmissionReminderDeduper } from './admission-reminder-deduper'
import { backendSyncUpdate } from './member-records'
import type { GuardMemberStore } from './store'

const DEFAULT_ADMISSION_COMMAND_AUTHORITY = 4
const DUPLICATE_COMMAND_SUPPRESS_MS = 30_000

interface AdmissionAdminCommandDeps {
  readonly platform: PlatformClient
  readonly guardStore: GuardMemberStore
  readonly policyStore: GuardPolicyStore
  readonly config: StuhelperGroupGuardPluginConfig
  readonly reminderDeduper?: AdmissionReminderDeduper
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

  ctx.command('查询入群认证 <qqID>', '查询指定 QQ 的入群认证状态')
    .action(({ session }, qqID) => runAdmissionCommand(async () => {
      const command = await resolveAdmissionCommandContext(session, qqID, deps)
      if (typeof command === 'string') return command
      const admission = await deps.platform.getAdmissionSessionByMember(admissionSubject(command))
      return formatAdmissionSessionSummary(admission)
    }))

  ctx.command('重发认证链接 <qqID>', '重发当前仍可继续的入群认证链接')
    .action(({ session }, qqID) => runAdmissionCommand(async () => {
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

  ctx.command('重新生成认证链接 <qqID>', '取消旧会话并重新生成入群认证链接')
    .action(({ session }, qqID) => runAdmissionCommand(async () => {
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
        await resetMemberMute(command.session, created.session)
        await updateLocalAdmissionRecord(command, deps.guardStore, created)
        return sendAdmissionReminderForCommand(command, deps, created.session)
      } catch (error) {
        commandDeduper.forget(dedupeKey)
        throw error
      }
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

function admissionCommandDedupeKey(action: 'resend' | 'regenerate', command: AdmissionCommandContext) {
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
  if (!session) {
    return '入群认证命令只能在群聊中使用。'
  }
  const accessDenied = ensureAdmissionCommandAccess(session, deps.config)
  if (accessDenied) {
    return accessDenied
  }
  const guildID = session.guildId || session.channelId
  if (!guildID) {
    return '入群认证命令只能在群聊中使用。'
  }
  const targetQQID = qqID?.trim()
  if (!targetQQID) {
    return '请提供要操作的 QQ 号。'
  }
  const platform = resolveSessionAdmissionPlatform(session)
  if (!platform) {
    return '当前机器人平台不支持入群认证。'
  }
  const policy = await deps.policyStore.resolvePolicy(platform, guildID)
  if (!policy) {
    return '当前群未启用 StuHelper 入群认证。'
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

function ensureAdmissionCommandAccess(
  session: Session,
  config: StuhelperGroupGuardPluginConfig,
) {
  const commandConfig = config.admissionCommands
  if (commandConfig?.operatorQQIDs?.includes(session.userId)) {
    return
  }
  const minAuthority = commandConfig?.minAuthority ?? DEFAULT_ADMISSION_COMMAND_AUTHORITY
  const authority = (session as { user?: { authority?: number } }).user?.authority ?? 0
  if (authority >= minAuthority) {
    return
  }
  return '命令权限不足。'
}

function admissionSubject(command: AdmissionCommandContext) {
  return {
    platform: command.platform,
    guildID: command.guildID,
    qqID: command.qqID,
  }
}

async function runAdmissionCommand(run: () => Promise<string | void>) {
  try {
    return await run()
  } catch (error) {
    return formatAdmissionCommandError(error)
  }
}

function formatAdmissionCommandError(error: unknown) {
  if (error instanceof PlatformAPIError) {
    if (error.status === 404) {
      return '未找到该 QQ 的入群认证记录。'
    }
    if (error.status === 409) {
      return '当前入群认证状态不允许该操作。'
    }
    if (error.status === 401 || error.status === 403) {
      return '机器人服务凭据无权访问入群认证接口。'
    }
    return `StuHelper 平台接口异常：${error.status} ${error.message}`
  }
  return error instanceof Error ? `入群认证命令执行失败：${error.message}` : '入群认证命令执行失败。'
}

function formatAdmissionSessionSummary(session: AdmissionSession) {
  const rows = [
    `入群认证状态：${session.qqID}`,
    `状态：${statusLabel(session.status)}`,
    `会话：${session.id}`,
    `QQ 绑定：${isQQLinked(session) ? '已完成' : '未完成'}`,
    `学生认证：${studentVerificationLabel(session)}`,
  ]
  const deadline = describeDeadline(session)
  if (deadline) {
    rows.push(deadline)
  }
  rows.push(`下一步：${nextAdmissionStep(session)}`)
  if (session.lastBotError) {
    rows.push(`最近机器人错误：${session.lastBotError}`)
  }
  return rows.join('\n')
}

function formatAdmissionReminderForSession(session: AdmissionSession) {
  if (!session.authURL) {
    return '后端没有返回可重发的认证链接。'
  }
  return formatAdmissionReminder({
    memberId: session.qqID,
    authURL: session.authURL,
    deadlineAt: reminderDeadline(session),
  })
}

async function sendAdmissionReminderForCommand(
  command: AdmissionCommandContext,
  deps: AdmissionAdminCommandDeps,
  admission: AdmissionSession,
) {
  const message = formatAdmissionReminderForSession(admission)
  if (!admission.authURL) {
    return message
  }
  deps.reminderDeduper?.remember(admission.id)
  let messageID: string | undefined
  try {
    messageID = await sendCommandMessage(command.session, message)
  } catch (error) {
    deps.reminderDeduper?.forget(admission.id)
    throw error
  }
  await markLocalReminderSent(command, deps.guardStore)
  await deps.platform.recordAdmissionEvent(admission.id, {
    action: 'remind',
    success: true,
    ...(messageID ? { messageID } : {}),
  })
}

async function sendCommandMessage(session: Session, message: string) {
  const result = await session.send(message)
  if (Array.isArray(result)) return typeof result[0] === 'string' ? result[0] : undefined
  return typeof result === 'string' ? result : undefined
}

async function resetMemberMute(session: Session, admission: AdmissionSession) {
  const muteDuration = new Date(admission.initialMuteUntil).getTime() - Date.now()
  if (!Number.isFinite(muteDuration) || muteDuration <= 0) {
    throw new Error('入群认证禁言期限无效，无法重置禁言。')
  }
  await session.bot.muteGuildMember(admission.guildID, admission.qqID, muteDuration)
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
  if (!record) return
  await guardStore.markBackendSynced(record.id, backendSyncUpdate(created))
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
  if (!record) return
  await guardStore.markReminderSent(record.id, new Date())
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

function describeDeadline(session: AdmissionSession) {
  switch (session.status) {
    case 'joined_muted':
      return `链接有效期：${session.linkWaitDeadlineAt}`
    case 'linked':
      return `学生认证截止：${session.submissionWaitDeadlineAt}`
    case 'material_submitted':
      return `人工审核截止：${session.manualReviewDeadlineAt || '未设置'}`
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

function studentVerificationLabel(session: AdmissionSession) {
  switch (session.status) {
    case 'verified':
      return '已通过'
    case 'material_submitted':
      return '新生材料待审核'
    default:
      return '未通过'
  }
}

function nextAdmissionStep(session: AdmissionSession) {
  switch (session.status) {
    case 'joined_muted':
      return '让成员打开认证链接登录并完成 QQ 绑定与学生认证；链接找不到时可使用“重发认证链接”。'
    case 'linked':
      return 'QQ 已绑定，但还没有学生认证凭据；请继续完成老生邮箱/学校 SSO 认证，或提交新生材料。'
    case 'material_submitted':
      return '等待具备审核权限的管理员审核新生材料；审核通过后机器人会解除禁言。'
    case 'verified':
      return session.lastBotError
        ? '后端已通过学生认证，但机器人执行失败；请检查最近机器人错误和 Koishi 日志。'
        : '后端已通过学生认证，机器人会解除禁言；若仍被禁言，请查询 Koishi release 日志。'
    case 'expired_kicked':
      return hasLinkedUser(session)
        ? '旧会话已超时；该 QQ 曾绑定账号但未完成学生认证，请使用“重新生成认证链接”后重新认证。'
        : '旧会话已超时；请使用“重新生成认证链接”后让成员重新认证。'
    case 'cancelled':
      return '当前会话已取消；请使用“重新生成认证链接”创建新链接。'
    default:
      return '按当前状态继续处理。'
  }
}

function statusLabel(status: AdmissionSession['status']) {
  switch (status) {
    case 'joined_muted':
      return '等待打开链接并绑定 QQ'
    case 'linked':
      return '已绑定 QQ，等待学生认证'
    case 'material_submitted':
      return '新生材料待审核'
    case 'verified':
      return '学生认证已通过，等待或已完成解除禁言'
    case 'expired_kicked':
      return '已超时移出或等待移出'
    case 'cancelled':
      return '已取消'
    default:
      return status
  }
}
