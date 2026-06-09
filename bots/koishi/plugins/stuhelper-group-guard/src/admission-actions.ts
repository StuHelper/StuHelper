import { h, type Universal } from 'koishi'

import {
  PlatformAPIError,
  renderMessageTemplate,
  resolveGroupGuardMessages,
  type AdmissionBotEventRequest,
  type AdmissionPendingAction,
  type GuardMemberRecord,
  type StuhelperAdmissionReminderDeliveryConfig,
  type StuhelperGroupGuardMessageConfig,
} from '@stuhelper/koishi-shared'

import { formatAdmissionReminder } from './admission-format'
import { sendAdmissionReminderMessage } from './admission-reminder-delivery'

type ActionMark = 'reminder' | 'released' | 'kicked' | 'none'
type ReminderSendGuard = () => boolean | Promise<boolean>

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
  messages?: Partial<StuhelperGroupGuardMessageConfig>,
  delivery?: Partial<StuhelperAdmissionReminderDeliveryConfig>,
  shouldSendReminder?: ReminderSendGuard,
): Promise<ActionResult> {
  const target = resolveActionTarget(action, record)
  const resolvedMessages = resolveGroupGuardMessages(messages)
  switch (action.action) {
    case 'remind':
      return executeReminder(bot, action, target, resolvedMessages, delivery, shouldSendReminder)
    case 'release':
      return executeRelease(bot, action, target, resolvedMessages)
    case 'kick':
      return executeKick(bot, action, target, resolvedMessages)
    case 'blacklist':
      return executeBlacklist(bot, action, target, resolvedMessages)
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
  messages: ReturnType<typeof resolveGroupGuardMessages>,
  delivery?: Partial<StuhelperAdmissionReminderDeliveryConfig>,
  shouldSend?: ReminderSendGuard,
): Promise<ActionResult> {
  if (!action.authURL) {
    throw new Error(`admission remind action ${action.sessionID} missing authURL`)
  }
  const result = await sendAdmissionReminderMessage({
    bot,
    guildId: target.guildID,
    channelId: target.channelID,
    memberId: target.qqID,
    content: formatAdmissionReminder({
      memberId: target.qqID,
      authURL: action.authURL,
      deadlineAt: target.deadlineAt,
      failureCount: action.failureCount,
      remainingRetryCount: action.remainingRetryCount,
      willBlacklistOnTimeout: action.willBlacklistOnTimeout,
      messages,
    }),
    delivery,
    messages,
    shouldSend,
  })
  if (result.cancelled) {
    return successResult(action, 'none')
  }
  return successResult(action, 'reminder', result.messageID)
}

async function executeRelease(
  bot: Universal.Methods,
  action: AdmissionPendingAction,
  target: ActionTarget,
  messages: ReturnType<typeof resolveGroupGuardMessages>,
): Promise<ActionResult> {
  await bot.muteGuildMember(target.guildID, target.qqID, 0)
  const content = renderMessageTemplate(messages.admissionReleaseCompleted, {
    at: h.at(target.qqID),
    memberId: target.qqID,
  })
  const messageID = content ? await sendActionMessage(bot, target.channelID, content) : undefined
  return successResult(action, 'released', messageID)
}

async function executeKick(
  bot: Universal.Methods,
  action: AdmissionPendingAction,
  target: ActionTarget,
  messages: ReturnType<typeof resolveGroupGuardMessages>,
): Promise<ActionResult> {
  const content = renderMessageTemplate(messages.admissionKickTimeout, {
    at: h.at(target.qqID),
    memberId: target.qqID,
  })
  const messageID = content ? await sendActionMessage(bot, target.channelID, content) : undefined
  await bot.kickGuildMember(target.guildID, target.qqID)
  return successResult(action, 'kicked', messageID)
}

async function executeBlacklist(
  bot: Universal.Methods,
  action: AdmissionPendingAction,
  target: ActionTarget,
  messages: ReturnType<typeof resolveGroupGuardMessages>,
): Promise<ActionResult> {
  const content = renderMessageTemplate(messages.admissionBlacklistKick, {
    at: h.at(target.qqID),
    memberId: target.qqID,
  })
  const messageID = content ? await sendActionMessage(bot, target.channelID, content) : undefined
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
