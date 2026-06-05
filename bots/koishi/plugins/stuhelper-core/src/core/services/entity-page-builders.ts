import type {
  ModerationEventRecord,
  ModerationReportRecord,
  ReviewQueueRecord,
} from '@stuhelper/koishi-moderation-core'
import type { GuardMemberRecord } from '@stuhelper/koishi-shared'

import type {
  EntityBlacklistFact,
  EntityReportFact,
  EntityRestrictedFact,
  EntityReviewFact,
  EntityWarnFact,
  GuildEntityProfile,
} from './page-types'
import type {
  EntityPageServiceDeps,
  EntityProfileScope,
  GuildNameLookup,
  RawBlacklistEntry,
  RawWarnEntry,
} from './entity-page.service'
import {
  byCreatedDesc,
  byJoinedDesc,
  toEventFact,
  toReportFact,
  toRestrictedFact,
  toReviewFact,
} from './entity-page-converters'

export const RECENT_LIMIT = 10

interface GuildProfileInput {
  readonly guildId: string
  readonly deps: EntityPageServiceDeps
  readonly warnsByGuild: Record<string, Record<string, RawWarnEntry>>
  readonly guardRecords: GuardMemberRecord[]
  readonly reviews: ReviewQueueRecord[]
  readonly reports: ModerationReportRecord[]
  readonly events: ModerationEventRecord[]
  readonly configured: boolean
}

export function assertScopeGuildAccess(
  scope: EntityProfileScope,
  guildId: string | undefined,
  resource: string,
): void {
  if (scope.guildIds === null || !guildId || scope.guildIds.has(guildId)) return
  throw new Error(`${resource} is outside of the current console guild scope: ${guildId}`)
}

export function buildGuildProfile(input: GuildProfileInput): GuildEntityProfile {
  const meta = input.deps.resolveGuildName(input.guildId)
  const guildWarnsRecord = input.warnsByGuild[input.guildId] ?? {}
  const guildGuardRecords = input.guardRecords.filter((record) => record.guildId === input.guildId)
  const warns = collectGuildWarns(input.guildId, guildWarnsRecord, meta?.name ?? null)
  const restricted = collectGuildRestricted(guildGuardRecords, input.deps.resolveGuildName)
  const reviews = collectGuildReviews(input.reviews, input.guildId, input.deps.resolveGuildName)
  const reports = collectGuildReports(input.reports, input.guildId, input.deps.resolveGuildName)
  const recentEvents = collectGuildEvents(input.events, input.guildId, input.deps.resolveGuildName)

  return {
    kind: 'guild',
    generatedAt: new Date().toISOString(),
    id: input.guildId,
    name: meta?.name ?? null,
    avatar: meta?.avatar ?? null,
    summary: {
      configured: input.configured,
      pendingMembers: countPendingMembers(guildGuardRecords),
      warnedUsers: countWarnedUsers(guildWarnsRecord),
      pendingReviews: reviews.filter((review) => isPendingReviewStatus(review.status)).length,
      openReports: reports.length,
    },
    warns,
    restricted,
    reviews,
    reports,
    recentEvents,
  }
}

export function collectUserWarns(
  warnsByGuild: Record<string, Record<string, RawWarnEntry>>,
  userId: string,
  resolveGuildName: GuildNameLookup,
): EntityWarnFact[] {
  const result: EntityWarnFact[] = []
  for (const [guildId, byUser] of Object.entries(warnsByGuild)) {
    const entry = byUser[userId]
    if (!entry || entry.count <= 0) continue
    const meta = resolveGuildName(guildId)
    result.push({ guildId, guildName: meta?.name ?? null, userId, count: entry.count })
  }
  result.sort((left, right) => right.count - left.count)
  return result.slice(0, RECENT_LIMIT)
}

export function collectUserRestricted(
  records: readonly GuardMemberRecord[],
  userId: string,
  resolveGuildName: GuildNameLookup,
) {
  return records
    .filter((record) => record.memberId === userId)
    .map((record) => toRestrictedFact(record, resolveGuildName))
    .sort(byJoinedDesc)
    .slice(0, RECENT_LIMIT)
}

export function collectUserReviews(
  reviews: readonly ReviewQueueRecord[],
  userId: string,
  resolveGuildName: GuildNameLookup,
) {
  return reviews
    .filter((review) => review.memberId === userId)
    .map((review) => toReviewFact(review, resolveGuildName))
    .sort(byCreatedDesc)
    .slice(0, RECENT_LIMIT)
}

export function collectUserReports(
  reports: readonly ModerationReportRecord[],
  userId: string,
  resolveGuildName: GuildNameLookup,
) {
  return reports
    .filter((report) => report.targetMemberId === userId || report.reporterMemberId === userId)
    .map((report) => toReportFact(report, resolveGuildName))
    .sort(byCreatedDesc)
    .slice(0, RECENT_LIMIT)
}

export function collectUserEvents(
  events: readonly ModerationEventRecord[],
  userId: string,
  resolveGuildName: GuildNameLookup,
) {
  return events
    .filter((event) => event.memberId === userId)
    .map((event) => toEventFact(event, resolveGuildName))
    .sort(byCreatedDesc)
    .slice(0, RECENT_LIMIT)
}

export function hasUserFacts(input: {
  readonly warns: readonly EntityWarnFact[]
  readonly blacklist: EntityBlacklistFact | null
  readonly restricted: readonly EntityRestrictedFact[]
  readonly reviews: readonly EntityReviewFact[]
  readonly reports: readonly EntityReportFact[]
  readonly events: readonly import('./page-types').EntityEventFact[]
}): boolean {
  return Boolean(
    input.warns.length ||
    input.blacklist ||
    input.restricted.length ||
    input.reviews.length ||
    input.reports.length ||
    input.events.length
  )
}

export function mapBlacklist(
  entry: RawBlacklistEntry | undefined,
  scope: EntityProfileScope,
): EntityBlacklistFact | null {
  if (!entry || !hasBlacklistScopeAccess(entry, scope)) return null
  return {
    userId: entry.userId,
    reason: entry.reason ?? null,
    addedAt: entry.timestamp ? new Date(entry.timestamp).toISOString() : null,
  }
}

export function filterWarnsByScope(
  warnsByGuild: Record<string, Record<string, RawWarnEntry>>,
  scope: EntityProfileScope,
) {
  const guildIds = scope.guildIds
  if (guildIds === null) return warnsByGuild
  return Object.fromEntries(Object.entries(warnsByGuild).filter(([guildId]) => guildIds.has(guildId)))
}

export function filterGuildRecords<T extends { guildId?: string | null }>(
  records: readonly T[],
  scope: EntityProfileScope,
): T[] {
  const guildIds = scope.guildIds
  if (guildIds === null) return [...records]
  return records.filter((record) => typeof record.guildId === 'string' && guildIds.has(record.guildId))
}

export function isPendingReviewStatus(status: string): boolean {
  return status === 'pending' || status === 'awaiting' || status === 'in_progress'
}

function collectGuildWarns(guildId: string, warns: Record<string, RawWarnEntry>, guildName: string | null) {
  return Object.entries(warns)
    .filter(([, info]) => info.count > 0)
    .map(([userId, info]) => ({ guildId, guildName, userId, count: info.count }))
    .sort((left, right) => right.count - left.count)
    .slice(0, RECENT_LIMIT)
}

function collectGuildRestricted(records: GuardMemberRecord[], resolveGuildName: GuildNameLookup) {
  return records
    .map((record) => toRestrictedFact(record, resolveGuildName))
    .sort(byJoinedDesc)
    .slice(0, RECENT_LIMIT)
}

function collectGuildReviews(reviews: ReviewQueueRecord[], guildId: string, resolveGuildName: GuildNameLookup) {
  return reviews
    .filter((review) => review.guildId === guildId)
    .map((review) => toReviewFact(review, resolveGuildName))
    .sort(byCreatedDesc)
    .slice(0, RECENT_LIMIT)
}

function collectGuildReports(reports: ModerationReportRecord[], guildId: string, resolveGuildName: GuildNameLookup) {
  return reports
    .filter((report) => report.guildId === guildId)
    .map((report) => toReportFact(report, resolveGuildName))
    .sort(byCreatedDesc)
    .slice(0, RECENT_LIMIT)
}

function collectGuildEvents(events: ModerationEventRecord[], guildId: string, resolveGuildName: GuildNameLookup) {
  return events
    .filter((event) => event.guildId === guildId)
    .map((event) => toEventFact(event, resolveGuildName))
    .sort(byCreatedDesc)
    .slice(0, RECENT_LIMIT)
}

function hasBlacklistScopeAccess(entry: RawBlacklistEntry, scope: EntityProfileScope): boolean {
  if (scope.guildIds === null) return true
  if (entry.scopeType === 'global') return true
  return typeof entry.guildId === 'string' && scope.guildIds.has(entry.guildId)
}

function countWarnedUsers(warns: Record<string, RawWarnEntry>) {
  return Object.values(warns).filter((info) => info.count > 0).length
}

function countPendingMembers(records: readonly GuardMemberRecord[]) {
  return records.filter((record) => !record.releasedAt && !record.kickedAt).length
}
