import { h, type Universal } from 'koishi'

import {
  renderMessageTemplate,
  resolveGroupGuardMessages,
  type GuardMemberRecord,
  type StuhelperAdmissionReminderDeliveryConfig,
  type StuhelperGroupGuardMessageConfig,
} from '@stuhelper/koishi-shared'

import { formatAdmissionReminder, type AdmissionReminderInput } from './admission-format'
import { sendAdmissionReminderMessage } from './admission-reminder-delivery'

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

type ReminderSendGuard = () => boolean | Promise<boolean>

export async function sendAdmissionReminder(
  bot: Universal.Methods,
  record: GuardMemberRecord,
  authURL: string,
  context: ReminderContext = {},
  messages?: Partial<StuhelperGroupGuardMessageConfig>,
  delivery?: Partial<StuhelperAdmissionReminderDeliveryConfig>,
  shouldSend?: ReminderSendGuard,
) {
  const result = await sendAdmissionReminderMessage({
    bot,
    guildId: record.guildId,
    channelId: record.channelId,
    memberId: record.memberId,
    content: formatAdmissionReminder({
      memberId: record.memberId,
      authURL,
      deadlineAt: record.deadlineAt,
      failureCount: context.failureCount,
      remainingRetryCount: context.remainingRetryCount,
      willBlacklistOnTimeout: context.willBlacklistOnTimeout,
      messages,
    }),
    delivery,
    messages,
    shouldSend,
  })
  if (result.cancelled) return false
  return result.messageID
}

export async function sendBackendPendingReminder(
  bot: Universal.Methods,
  record: GuardMemberRecord,
  reminderTemplate: string,
  messages?: Partial<StuhelperGroupGuardMessageConfig>,
  delivery?: Partial<StuhelperAdmissionReminderDeliveryConfig>,
) {
  const message = renderMessageTemplate(resolveGroupGuardMessages(messages).backendPendingReminder, {
    at: h.at(record.memberId),
    memberId: record.memberId,
    reminderTemplate,
  })
  if (message) {
    await sendAdmissionReminderMessage({
      bot,
      guildId: record.guildId,
      channelId: record.channelId,
      memberId: record.memberId,
      content: message,
      delivery,
      messages,
    })
  }
}
