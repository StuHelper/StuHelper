import type {
  ModerationEventRecord,
  ModerationReportRecord,
  ReviewQueueRecord,
} from '@stuhelper/koishi-moderation-core'
import type { GuardMemberRecord } from '@stuhelper/koishi-shared'

import type {
  EntityBlacklistFact,
  EntityEventFact,
  EntityProfile,
  EntityProfileQuery,
  EntityReportFact,
  EntityRestrictedFact,
  EntityReviewFact,
  EntityWarnFact,
  GuildEntityProfile,
  UserEntityProfile,
} from './page-types'

export interface EntityNameMeta {
  readonly name: string | null
  readonly avatar: string | null
}

export type GuildNameLookup = (guildId: string) => EntityNameMeta | undefined
export type UserNameLookup = (userId: string) => EntityNameMeta | undefined

export interface RawWarnEntry {
  count: number
  timestamp: number
}

export interface RawBlacklistEntry {
  userId: string
  timestamp: number
  reason?: string
}

export interface EntityPageServiceDeps {
  loadWarns(): Promise<Record<string, Record<string, RawWarnEntry>>>
  loadBlacklist(): Promise<Record<string, RawBlacklistEntry>>
  loadGuardRecords(): Promise<GuardMemberRecord[]>
  loadReviews(): Promise<ReviewQueueRecord[]>
  loadReports(): Promise<ModerationReportRecord[]>
  loadEvents(limit: number): Promise<ModerationEventRecord[]>
  hasGuildConfig(guildId: string): Promise<boolean>
  resolveGuildName: GuildNameLookup
  resolveUserName: UserNameLookup
}

const RECENT_LIMIT = 10
const EVENT_FETCH_LIMIT = 200

export class EntityPageService {
  constructor(private readonly deps: EntityPageServiceDeps) {}

  async getProfile(query: EntityProfileQuery): Promise<EntityProfile> {
    const id = query.id?.trim()
    if (!id) {
      throw new Error('entity profile: id is required')
    }
    if (query.kind === 'user') {
      return this.getUserProfile(id, query.guildId?.trim() || undefined)
    }
    return this.getGuildProfile(id)
  }

  private async getUserProfile(userId: string, _contextGuildId?: string): Promise<UserEntityProfile> {
    const [warnsByGuild, blacklistMap, guardRecords, reviews, reports, events] = await Promise.all([
      this.deps.loadWarns(),
      this.deps.loadBlacklist(),
      this.deps.loadGuardRecords(),
      this.deps.loadReviews(),
      this.deps.loadReports(),
      this.deps.loadEvents(EVENT_FETCH_LIMIT),
    ])

    const warns = collectUserWarns(warnsByGuild, userId, this.deps.resolveGuildName)
    const blacklist = mapBlacklist(blacklistMap[userId])
    const restricted = guardRecords
      .filter((record) => record.memberId === userId)
      .map((record) => toRestrictedFact(record, this.deps.resolveGuildName))
      .sort(byJoinedDesc)
      .slice(0, RECENT_LIMIT)
    const userReviews = reviews
      .filter((review) => review.memberId === userId)
      .map((review) => toReviewFact(review, this.deps.resolveGuildName))
      .sort(byCreatedDesc)
      .slice(0, RECENT_LIMIT)
    const userReports = reports
      .filter((report) => report.targetMemberId === userId || report.reporterMemberId === userId)
      .map((report) => toReportFact(report, this.deps.resolveGuildName))
      .sort(byCreatedDesc)
      .slice(0, RECENT_LIMIT)
    const userEvents = events
      .filter((event) => event.memberId === userId)
      .map((event) => toEventFact(event, this.deps.resolveGuildName))
      .sort(byCreatedDesc)
      .slice(0, RECENT_LIMIT)

    const userMeta = this.deps.resolveUserName(userId)

    return {
      kind: 'user',
      generatedAt: new Date().toISOString(),
      id: userId,
      displayName: userMeta?.name ?? null,
      avatar: userMeta?.avatar ?? null,
      summary: {
        activeWarnGuilds: warns.length,
        totalWarns: warns.reduce((acc, fact) => acc + fact.count, 0),
        blacklisted: blacklist !== null,
        pendingReviews: userReviews.filter((review) => isPendingReviewStatus(review.status)).length,
        openReports: userReports.length,
        restrictedGuilds: restricted.filter((record) => record.status === 'pending').length,
      },
      warns,
      blacklist,
      restricted,
      reviews: userReviews,
      reports: userReports,
      recentEvents: userEvents,
    }
  }

  private async getGuildProfile(guildId: string): Promise<GuildEntityProfile> {
    const [warnsByGuild, guardRecords, reviews, reports, events, configured] = await Promise.all([
      this.deps.loadWarns(),
      this.deps.loadGuardRecords(),
      this.deps.loadReviews(),
      this.deps.loadReports(),
      this.deps.loadEvents(EVENT_FETCH_LIMIT),
      this.deps.hasGuildConfig(guildId),
    ])

    const meta = this.deps.resolveGuildName(guildId)
    const guildWarnsRecord = warnsByGuild[guildId] ?? {}
    const warns: EntityWarnFact[] = Object.entries(guildWarnsRecord)
      .filter(([, info]) => info.count > 0)
      .map(([userId, info]) => ({
        guildId,
        guildName: meta?.name ?? null,
        userId,
        count: info.count,
      }))
      .sort((left, right) => right.count - left.count)
      .slice(0, RECENT_LIMIT)

    const guildGuardRecords = guardRecords.filter((record) => record.guildId === guildId)
    const restricted = guildGuardRecords
      .map((record) => toRestrictedFact(record, this.deps.resolveGuildName))
      .sort(byJoinedDesc)
      .slice(0, RECENT_LIMIT)

    const guildReviews = reviews
      .filter((review) => review.guildId === guildId)
      .map((review) => toReviewFact(review, this.deps.resolveGuildName))
      .sort(byCreatedDesc)
      .slice(0, RECENT_LIMIT)

    const guildReports = reports
      .filter((report) => report.guildId === guildId)
      .map((report) => toReportFact(report, this.deps.resolveGuildName))
      .sort(byCreatedDesc)
      .slice(0, RECENT_LIMIT)

    const guildEvents = events
      .filter((event) => event.guildId === guildId)
      .map((event) => toEventFact(event, this.deps.resolveGuildName))
      .sort(byCreatedDesc)
      .slice(0, RECENT_LIMIT)

    const warnedUsers = Object.values(guildWarnsRecord).filter((info) => info.count > 0).length
    const pendingMembers = guildGuardRecords.filter(
      (record) => !record.releasedAt && !record.kickedAt,
    ).length

    return {
      kind: 'guild',
      generatedAt: new Date().toISOString(),
      id: guildId,
      name: meta?.name ?? null,
      avatar: meta?.avatar ?? null,
      summary: {
        configured,
        pendingMembers,
        warnedUsers,
        pendingReviews: guildReviews.filter((review) => isPendingReviewStatus(review.status)).length,
        openReports: guildReports.length,
      },
      warns,
      restricted,
      reviews: guildReviews,
      reports: guildReports,
      recentEvents: guildEvents,
    }
  }
}

function collectUserWarns(
  warnsByGuild: Record<string, Record<string, RawWarnEntry>>,
  userId: string,
  resolveGuildName: GuildNameLookup,
): EntityWarnFact[] {
  const result: EntityWarnFact[] = []
  for (const [guildId, byUser] of Object.entries(warnsByGuild)) {
    const entry = byUser[userId]
    if (!entry || entry.count <= 0) continue
    const meta = resolveGuildName(guildId)
    result.push({
      guildId,
      guildName: meta?.name ?? null,
      userId,
      count: entry.count,
    })
  }
  result.sort((left, right) => right.count - left.count)
  return result.slice(0, RECENT_LIMIT)
}

function mapBlacklist(entry: RawBlacklistEntry | undefined): EntityBlacklistFact | null {
  if (!entry) return null
  return {
    userId: entry.userId,
    reason: entry.reason ?? null,
    addedAt: entry.timestamp ? new Date(entry.timestamp).toISOString() : null,
  }
}

function toRestrictedFact(record: GuardMemberRecord, resolveGuildName: GuildNameLookup): EntityRestrictedFact {
  const status: 'pending' | 'released' | 'kicked' = record.kickedAt
    ? 'kicked'
    : record.releasedAt
      ? 'released'
      : 'pending'
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

function toReviewFact(review: ReviewQueueRecord, resolveGuildName: GuildNameLookup): EntityReviewFact {
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

function toReportFact(report: ModerationReportRecord, resolveGuildName: GuildNameLookup): EntityReportFact {
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

function toEventFact(event: ModerationEventRecord, resolveGuildName: GuildNameLookup): EntityEventFact {
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

function isPendingReviewStatus(status: string): boolean {
  return status === 'pending' || status === 'awaiting' || status === 'in_progress'
}

function byJoinedDesc<T extends { joinedAt: string }>(left: T, right: T): number {
  return new Date(right.joinedAt).getTime() - new Date(left.joinedAt).getTime()
}

function byCreatedDesc<T extends { createdAt: string }>(left: T, right: T): number {
  return new Date(right.createdAt).getTime() - new Date(left.createdAt).getTime()
}
