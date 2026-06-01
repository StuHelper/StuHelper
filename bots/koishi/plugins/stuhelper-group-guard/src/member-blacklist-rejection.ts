import type { Logger, Session, Universal } from 'koishi'

import { PlatformAPIError } from '@stuhelper/koishi-shared'
import type { ModerationStore } from '@stuhelper/koishi-moderation-core'

import { requireAdmissionSubjectPlatform } from './admission-subject-platform'
import { formatAdmissionActionError } from './admission-actions'
import { requireMemberID, resolveGuildID } from './member-records'
import type { GuardMemberRecord } from './model'
import type { GuardMemberStore } from './store'

const MEMBER_BLACKLISTED_ERROR_CODE = 'admission.member_blacklisted'

export function isMemberBlacklistedError(error: unknown) {
  if (error instanceof PlatformAPIError) {
    return error.code === MEMBER_BLACKLISTED_ERROR_CODE
  }
  if (typeof error !== 'object' || error === null) return false
  return (error as { code?: unknown }).code === MEMBER_BLACKLISTED_ERROR_CODE
}

export async function kickBlacklistedJoin(input: BlacklistedJoinInput) {
  const guildId = requireBlacklistedGuildID(input.session)
  const memberId = requireMemberID(input.session)
  await input.session.bot.kickGuildMember(guildId, memberId, false)
  await reportBlacklistedMember({
    platform: requireAdmissionSubjectPlatform(input.session.platform),
    botSelfId: input.session.selfId,
    guildId,
    channelId: input.session.channelId || guildId,
    memberId,
    moderationStore: input.moderationStore,
    logger: input.logger,
    error: input.error,
  })
}

export async function kickBlacklistedPendingMember(input: BlacklistedPendingInput) {
  await input.bot.kickGuildMember(input.record.guildId, input.record.memberId, false)
  await input.guardStore.markKicked(input.record.id, input.now)
  await reportBlacklistedMember({
    platform: input.record.platform,
    botSelfId: input.record.botSelfId,
    guildId: input.record.guildId,
    channelId: input.record.channelId,
    memberId: input.record.memberId,
    guardRecordID: input.record.id,
    moderationStore: input.moderationStore,
    logger: input.logger,
    error: input.error,
  })
}

async function reportBlacklistedMember(input: BlacklistedMemberEventInput) {
  const message = formatAdmissionActionError(input.error)
  await input.moderationStore.appendEvent({
    platform: input.platform,
    botSelfId: input.botSelfId,
    guildId: input.guildId,
    channelId: input.channelId,
    memberId: input.memberId,
    type: 'join_guarded',
    level: 'high',
    summary: `成员 ${input.memberId} 命中入群黑名单，已移出群聊`,
    payload: { memberBlacklisted: true, error: message },
  })
  input.logger.warn('group guard rejected blacklisted member', {
    guardRecordID: input.guardRecordID,
    error: message,
  })
}

function requireBlacklistedGuildID(session: Session) {
  const guildId = resolveGuildID(session)
  if (!guildId) {
    throw new Error('group guard blacklist kick requires guildId or channelId')
  }
  return guildId
}

interface BlacklistedJoinInput {
  readonly session: Session
  readonly moderationStore: ModerationStore
  readonly logger: Logger
  readonly error: unknown
}

interface BlacklistedPendingInput {
  readonly bot: Universal.Methods
  readonly record: GuardMemberRecord
  readonly guardStore: GuardMemberStore
  readonly moderationStore: ModerationStore
  readonly logger: Logger
  readonly error: unknown
  readonly now: Date
}

interface BlacklistedMemberEventInput {
  readonly platform: string
  readonly botSelfId: string
  readonly guildId: string
  readonly channelId: string
  readonly memberId: string
  readonly moderationStore: ModerationStore
  readonly logger: Logger
  readonly error: unknown
  readonly guardRecordID?: string
}
