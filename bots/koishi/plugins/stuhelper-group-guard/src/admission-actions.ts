import { h, type Universal } from 'koishi'

import {
  PlatformAPIError,
  type AdmissionBotEventRequest,
  type AdmissionPendingAction,
  type GuardMemberRecord,
} from '@stuhelper/koishi-shared'

import { formatAdmissionReminder } from './admission-format'

type ActionMark = 'reminder' | 'released' | 'kicked'

interface ActionTarget {
  readonly guildID: string
  readonly channelID: string
  readonly qqID: string
  readonly deadlineAt: Date
}

interface ActionResult {
  readonly event: AdmissionBotEventRequest
  readonly mark: ActionMark
}

export async function executeAdmissionAction(
  bot: Universal.Methods,
  action: AdmissionPendingAction,
  record: GuardMemberRecord | null,
): Promise<ActionResult> {
  const target = resolveActionTarget(action, record)
  switch (action.action) {
    case 'remind':
      return executeReminder(bot, action, target)
    case 'release':
      return executeRelease(bot, action, target)
    case 'kick':
      return executeKick(bot, action, target)
    case 'blacklist':
      return executeBlacklist(bot, action, target)
    default:
      throw new Error(`unknown admission action: ${action.action}`)
  }
}

export function formatAdmissionActionError(error: unknown) {
  if (error instanceof PlatformAPIError) {
    return `${error.status}:${error.message}`
  }
  return error instanceof Error ? error.message : String(error)
}

async function executeReminder(
  bot: Universal.Methods,
  action: AdmissionPendingAction,
  target: ActionTarget,
): Promise<ActionResult> {
  if (!action.authURL) {
    throw new Error(`admission remind action ${action.sessionID} missing authURL`)
  }
  const messageID = await sendActionMessage(bot, target.channelID, formatAdmissionReminder({
    memberId: target.qqID,
    authURL: action.authURL,
    deadlineAt: target.deadlineAt,
    failureCount: action.failureCount,
    remainingRetryCount: action.remainingRetryCount,
    willBlacklistOnTimeout: action.willBlacklistOnTimeout,
  }))
  return successResult(action, 'reminder', messageID)
}

async function executeRelease(
  bot: Universal.Methods,
  action: AdmissionPendingAction,
  target: ActionTarget,
): Promise<ActionResult> {
  await bot.muteGuildMember(target.guildID, target.qqID, 0)
  const messageID = await sendActionMessage(
    bot,
    target.channelID,
    `${h.at(target.qqID)} 已检测到你完成 StuHelper 学生认证，已自动解除禁言。`,
  )
  return successResult(action, 'released', messageID)
}

async function executeKick(
  bot: Universal.Methods,
  action: AdmissionPendingAction,
  target: ActionTarget,
): Promise<ActionResult> {
  const messageID = await sendActionMessage(
    bot,
    target.channelID,
    `${h.at(target.qqID)} 认证超时，机器人将自动移出群聊。`,
  )
  await bot.kickGuildMember(target.guildID, target.qqID)
  return successResult(action, 'kicked', messageID)
}

async function executeBlacklist(
  bot: Universal.Methods,
  action: AdmissionPendingAction,
  target: ActionTarget,
): Promise<ActionResult> {
  const messageID = await sendActionMessage(
    bot,
    target.channelID,
    `${h.at(target.qqID)} 认证失败次数已达上限，已加入入群黑名单，机器人将移出群聊。`,
  )
  await bot.kickGuildMember(target.guildID, target.qqID, true)
  return successResult(action, 'kicked', messageID)
}

function resolveActionTarget(action: AdmissionPendingAction, record: GuardMemberRecord | null): ActionTarget {
  return {
    guildID: requiredTargetValue(action.guildID ?? record?.guildId, action, 'guildID'),
    channelID: requiredTargetValue(action.channelID ?? record?.channelId, action, 'channelID'),
    qqID: requiredTargetValue(action.qqID ?? record?.memberId, action, 'qqID'),
    deadlineAt: resolveDeadline(action, record),
  }
}

function requiredTargetValue(value: string | undefined, action: AdmissionPendingAction, field: string) {
  if (!value) {
    throw new Error(`admission action ${action.sessionID} missing ${field}`)
  }
  return value
}

function resolveDeadline(action: AdmissionPendingAction, record: GuardMemberRecord | null) {
  const deadlineAt = action.deadlineAt ? new Date(action.deadlineAt) : record?.deadlineAt
  if (!deadlineAt || Number.isNaN(deadlineAt.getTime())) {
    throw new Error(`admission action ${action.sessionID} missing valid deadlineAt`)
  }
  return deadlineAt
}

async function sendActionMessage(bot: Universal.Methods, channelID: string, content: string) {
  const result = await bot.sendMessage(channelID, content)
  if (Array.isArray(result)) return result[0]
  return typeof result === 'string' ? result : undefined
}

function successResult(action: AdmissionPendingAction, mark: ActionMark, messageID?: string): ActionResult {
  const event = { action: action.action, success: true }
  if (!messageID) return { event, mark }
  return { event: { ...event, messageID }, mark }
}
