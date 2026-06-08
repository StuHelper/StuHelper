import { h, type Universal } from 'koishi'

import {
  renderMessageTemplate,
  resolveGroupGuardMessages,
  type GuardMemberRecord,
  type StuhelperGroupGuardMessageConfig,
} from '@stuhelper/koishi-shared'

import { formatAdmissionReminder, type AdmissionReminderInput } from './admission-format'

export async function muteGuardedMember(input: {
  readonly bot: Universal.Methods
  readonly guildId: string
  readonly memberId: string
  readonly muteDurationMs: number
}) {
  const { bot, guildId, memberId, muteDurationMs } = input
  await bot.muteGuildMember(guildId, memberId, muteDurationMs)
}

type ReminderContext = Pick<
  AdmissionReminderInput,
  'failureCount' | 'remainingRetryCount' | 'willBlacklistOnTimeout'
>

export async function sendAdmissionReminder(
  bot: Universal.Methods,
  record: GuardMemberRecord,
  authURL: string,
  context: ReminderContext = {},
  messages?: Partial<StuhelperGroupGuardMessageConfig>,
) {
  const result = await bot.sendMessage(record.channelId, formatAdmissionReminder({
    memberId: record.memberId,
    authURL,
    deadlineAt: record.deadlineAt,
    failureCount: context.failureCount,
    remainingRetryCount: context.remainingRetryCount,
    willBlacklistOnTimeout: context.willBlacklistOnTimeout,
    messages,
  }))
  return firstMessageID(result)
}

export async function sendBackendPendingReminder(
  bot: Universal.Methods,
  record: GuardMemberRecord,
  reminderTemplate: string,
  messages?: Partial<StuhelperGroupGuardMessageConfig>,
) {
  const message = renderMessageTemplate(resolveGroupGuardMessages(messages).backendPendingReminder, {
    at: h.at(record.memberId),
    memberId: record.memberId,
    reminderTemplate,
  })
  if (message) {
    await bot.sendMessage(record.channelId, message)
  }
}

function firstMessageID(result: unknown): string | undefined {
  if (Array.isArray(result)) {
    return typeof result[0] === 'string' ? result[0] : undefined
  }
  return typeof result === 'string' ? result : undefined
}
