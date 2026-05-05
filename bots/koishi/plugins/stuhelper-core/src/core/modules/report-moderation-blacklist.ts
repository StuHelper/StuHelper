import type { PlatformClient } from '@stuhelper/koishi-shared'

import type { ReportModule } from './report.module'
import type { ViolationInfo } from './report-types'

const BLACKLIST_REASON_MAX_LENGTH = 50

interface ModerationBlacklistInput {
  readonly host: ReportModule
  readonly session: any
  readonly userId: string
  readonly violation: ViolationInfo
}

export async function kickUserWithModerationBlacklist(
  input: ModerationBlacklistInput,
): Promise<void> {
  const guildID = requireReportGuildID(input.session)
  await input.session.bot.kickGuildMember(guildID, input.userId, true)
  await requireMemberBlacklistBackend(input.host).createMemberBlacklist({
    platform: requireReportPlatform(input.session),
    subjectType: 'qq_user',
    subjectID: input.userId,
    scopeType: 'guild',
    guildID,
    source: 'moderation_action',
    reasonCode: 'violation_review_blacklist',
    reasonText: moderationBlacklistReason(input),
    metadata: moderationBlacklistMetadata(input, guildID),
  })
}

function requireMemberBlacklistBackend(
  host: ReportModule,
): Pick<PlatformClient, 'createMemberBlacklist'> {
  if (!host.memberBlacklistBackend) {
    throw new Error('member blacklist backend client is required for moderation blacklist action')
  }
  return host.memberBlacklistBackend
}

function moderationBlacklistMetadata(
  input: ModerationBlacklistInput,
  guildID: string,
): Record<string, unknown> {
  const workItemID = moderationWorkItemID(input.session, input.userId)
  return { reviewID: workItemID, workItemID, targetGuildID: guildID }
}

function moderationWorkItemID(session: any, userId: string): string {
  const messageID = String(session.messageId || 'unknown-message')
  return `moderation:${session.platform}:${session.guildId}:${messageID}:${userId}`
}

function moderationBlacklistReason(input: ModerationBlacklistInput): string {
  return `moderation review blacklist: ${shorten(input.violation.reason, BLACKLIST_REASON_MAX_LENGTH)}`
}

function requireReportGuildID(session: any): string {
  if (!session.guildId) {
    throw new Error('moderation blacklist action requires guildId')
  }
  return session.guildId
}

function requireReportPlatform(session: any): string {
  if (!session.platform) {
    throw new Error('moderation blacklist action requires platform')
  }
  return session.platform
}

function shorten(value: string, maxLength: number): string {
  return value.length > maxLength ? `${value.substring(0, maxLength)}...` : value
}
