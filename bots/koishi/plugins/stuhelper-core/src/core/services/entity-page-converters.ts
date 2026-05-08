import type {
  ModerationEventRecord,
  ModerationReportRecord,
  ReviewQueueRecord,
} from '@stuhelper/koishi-moderation-core'
import type { GuardMemberRecord } from '@stuhelper/koishi-shared'

import type {
  EntityEventFact,
  EntityReportFact,
  EntityRestrictedFact,
  EntityReviewFact,
} from './page-types'
import type { GuildNameLookup } from './entity-page.service'

export function toRestrictedFact(record: GuardMemberRecord, resolveGuildName: GuildNameLookup): EntityRestrictedFact {
  const status: 'pending' | 'released' | 'kicked' = record.kickedAt
    ? 'kicked'
    : record.releasedAt ? 'released' : 'pending'
  const meta = resolveGuildName(record.guildId)
  return {
    guildId: record.guildId,
    guildName: meta?.name ?? null,
    memberId: record.memberId,
    status,
    joinedAt: dateToIso(record.joinedAt),
    deadlineAt: dateToIso(record.deadlineAt),
  }
}

export function toReviewFact(review: ReviewQueueRecord, resolveGuildName: GuildNameLookup): EntityReviewFact {
  const meta = resolveGuildName(review.guildId)
  return {
    id: review.id,
    guildId: review.guildId,
    guildName: meta?.name ?? null,
    memberId: review.memberId ?? null,
    status: review.status,
    actionType: review.actionType,
    reason: review.reason ?? '',
    createdAt: dateToIso(review.createdAt),
  }
}

export function toReportFact(report: ModerationReportRecord, resolveGuildName: GuildNameLookup): EntityReportFact {
  const meta = resolveGuildName(report.guildId)
  return {
    id: report.id,
    guildId: report.guildId,
    guildName: meta?.name ?? null,
    targetMemberId: report.targetMemberId,
    reporterMemberId: report.reporterMemberId,
    reason: report.reason ?? '',
    status: report.aiStatus,
    createdAt: dateToIso(report.createdAt),
  }
}

export function toEventFact(event: ModerationEventRecord, resolveGuildName: GuildNameLookup): EntityEventFact {
  const meta = resolveGuildName(event.guildId)
  return {
    id: event.id,
    guildId: event.guildId,
    guildName: meta?.name ?? null,
    memberId: event.memberId ?? null,
    kind: event.type,
    severity: mapSeverity(event.level),
    message: event.summary ?? '',
    createdAt: dateToIso(event.createdAt),
  }
}

export function byJoinedDesc<T extends { joinedAt: string }>(left: T, right: T): number {
  return new Date(right.joinedAt).getTime() - new Date(left.joinedAt).getTime()
}

export function byCreatedDesc<T extends { createdAt: string }>(left: T, right: T): number {
  return new Date(right.createdAt).getTime() - new Date(left.createdAt).getTime()
}

function mapSeverity(level: string | undefined): EntityEventFact['severity'] {
  switch (level) {
    case 'critical':
      return 'critical'
    case 'high':
      return 'high'
    case 'medium':
    case 'low':
      return 'warning'
    default:
      return 'info'
  }
}

function dateToIso(value: Date | string | number): string {
  if (value instanceof Date) return value.toISOString()
  return new Date(value).toISOString()
}
