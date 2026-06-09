import { h, type Universal } from 'koishi'

import type {
  GuardMemberRecord,
  StuhelperAdmissionReminderDeliveryConfig,
  StuhelperGroupGuardMessageConfig,
} from '@stuhelper/koishi-shared'
import {
  renderMessageTemplate,
  resolveGroupGuardMessages,
} from '@stuhelper/koishi-shared'

import { formatAdmissionReminder } from './admission-format'
import { sendAdmissionReminderMessage } from './admission-reminder-delivery'

const POSITIVE_MUTE_DURATION_REQUIRED = 'admission session initialMuteUntil must be in the future'

export function muteDurationMs(initialMuteUntil: Date) {
  const duration = initialMuteUntil.getTime() - Date.now()
  if (!Number.isFinite(duration) || duration <= 0) {
    throw new Error(POSITIVE_MUTE_DURATION_REQUIRED)
  }
  return duration
}

export async function muteGuardedMember(
  bot: Universal.Methods,
  guildId: string,
  memberId: string,
  muteDurationMs: number,
) {
  await bot.muteGuildMember(guildId, memberId, muteDurationMs)
}

export async function sendAdmissionReminder(
  bot: Universal.Methods,
  record: GuardMemberRecord,
  authURL: string,
  messages?: Partial<StuhelperGroupGuardMessageConfig>,
  delivery?: Partial<StuhelperAdmissionReminderDeliveryConfig>,
) {
  await sendAdmissionReminderMessage({
    bot,
    guildId: record.guildId,
    channelId: record.channelId,
    memberId: record.memberId,
    content: formatAdmissionReminder({
      memberId: record.memberId,
      authURL,
      deadlineAt: record.deadlineAt,
      messages,
    }),
    delivery,
    messages,
  })
}

export async function sendBackendPendingReminder(
  bot: Universal.Methods,
  record: GuardMemberRecord,
  reminderTemplate: string,
  messages?: Partial<StuhelperGroupGuardMessageConfig>,
  delivery?: Partial<StuhelperAdmissionReminderDeliveryConfig>,
) {
  await sendAdmissionReminderMessage({
    bot,
    guildId: record.guildId,
    channelId: record.channelId,
    memberId: record.memberId,
    content: renderMessageTemplate(resolveGroupGuardMessages(messages).backendPendingReminder, {
      at: h.at(record.memberId),
      memberId: record.memberId,
      reminderTemplate,
    }),
    delivery,
    messages,
  })
}
