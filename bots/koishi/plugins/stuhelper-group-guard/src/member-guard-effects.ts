import { h, type Universal } from 'koishi'

import type { GuardMemberRecord } from '@stuhelper/koishi-shared'

import { formatAdmissionReminder } from './admission-format'

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
) {
  await bot.sendMessage(record.channelId, formatAdmissionReminder({
    memberId: record.memberId,
    authURL,
    deadlineAt: record.deadlineAt,
  }))
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
