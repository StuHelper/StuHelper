import type { Logger, Session, Universal } from 'koishi'

import {
  PlatformAPIError,
  renderMessageTemplate,
  resolveGroupGuardMessages,
  type GuardMemberRecord,
  type GuardMemberStore,
  type StuhelperGroupGuardMessageConfig,
} from '@stuhelper/koishi-shared'
import type { ModerationStore } from '@stuhelper/koishi-moderation-core'

import { requireAdmissionSubjectPlatform } from './admission-subject-platform'
import { formatAdmissionActionError } from './admission-actions'
import { requireMemberID, resolveGuildID } from './member-records'

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
    messages: input.messages,
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
    messages: input.messages,
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
    summary: groupGuardMessage(input.messages, 'admissionBlacklistEventSummary', {
      memberId: input.memberId,
    }),
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
  readonly messages?: Partial<StuhelperGroupGuardMessageConfig>
  readonly error: unknown
}

interface BlacklistedPendingInput {
  readonly bot: Universal.Methods
  readonly record: GuardMemberRecord
  readonly guardStore: GuardMemberStore
  readonly moderationStore: ModerationStore
  readonly logger: Logger
  readonly messages?: Partial<StuhelperGroupGuardMessageConfig>
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
  readonly messages?: Partial<StuhelperGroupGuardMessageConfig>
  readonly error: unknown
  readonly guardRecordID?: string
}

function groupGuardMessage(
  messages: Partial<StuhelperGroupGuardMessageConfig> | undefined,
  key: keyof ReturnType<typeof resolveGroupGuardMessages>,
  variables: Record<string, unknown> = {},
) {
  return renderMessageTemplate(resolveGroupGuardMessages(messages)[key], variables)
}
