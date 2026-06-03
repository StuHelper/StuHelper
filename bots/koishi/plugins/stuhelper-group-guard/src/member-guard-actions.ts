import { h, type Universal } from 'koishi'

import { formatAdmissionReminder, type AdmissionReminderInput } from './admission-format'
import type { GuardMemberRecord } from './model'

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
) {
  const result = await bot.sendMessage(record.channelId, formatAdmissionReminder({
    memberId: record.memberId,
    authURL,
    deadlineAt: record.deadlineAt,
    failureCount: context.failureCount,
    remainingRetryCount: context.remainingRetryCount,
    willBlacklistOnTimeout: context.willBlacklistOnTimeout,
  }))
  return firstMessageID(result)
}

export async function sendBackendPendingReminder(
  bot: Universal.Methods,
  record: GuardMemberRecord,
  reminderTemplate: string,
) {
  await bot.sendMessage(record.channelId, [
    `${h.at(record.memberId)} ${reminderTemplate}`,
    '认证链接暂时无法创建，机器人会自动重试。',
  ].join('\n'))
}

function firstMessageID(result: unknown): string | undefined {
  if (Array.isArray(result)) {
    return typeof result[0] === 'string' ? result[0] : undefined
  }
  return typeof result === 'string' ? result : undefined
}
