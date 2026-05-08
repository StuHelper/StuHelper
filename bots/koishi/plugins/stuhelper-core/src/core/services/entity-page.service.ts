import type {
  ModerationEventRecord,
  ModerationReportRecord,
  ReviewQueueRecord,
} from '@stuhelper/koishi-moderation-core'
import type { GuardMemberRecord } from '@stuhelper/koishi-shared'

import type {
  EntityProfile,
  EntityProfileQuery,
  GuildEntityProfile,
  UserEntityProfile,
} from './page-types'
import {
  assertScopeGuildAccess,
  buildGuildProfile,
  collectUserEvents,
  collectUserReports,
  collectUserRestricted,
  collectUserReviews,
  collectUserWarns,
  filterGuildRecords,
  filterWarnsByScope,
  hasUserFacts,
  isPendingReviewStatus,
  mapBlacklist,
} from './entity-page-builders'

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
  guildId?: string
  scopeType?: 'guild' | 'global'
  timestamp: number
  reason?: string
}

export interface EntityProfileScope {
  readonly guildIds: ReadonlySet<string> | null
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

interface ScopedEntityFacts {
  readonly warnsByGuild: Record<string, Record<string, RawWarnEntry>>
  readonly blacklistMap: Record<string, RawBlacklistEntry>
  readonly guardRecords: GuardMemberRecord[]
  readonly reviews: ReviewQueueRecord[]
  readonly reports: ModerationReportRecord[]
  readonly events: ModerationEventRecord[]
}

const EVENT_FETCH_LIMIT = 200
const GLOBAL_ENTITY_SCOPE: EntityProfileScope = { guildIds: null }

export class EntityPageService {
  constructor(private readonly deps: EntityPageServiceDeps) {}

  async getProfile(
    query: EntityProfileQuery,
    scope: EntityProfileScope = GLOBAL_ENTITY_SCOPE,
  ): Promise<EntityProfile> {
    const id = query.id?.trim()
    if (!id) throw new Error('entity profile: id is required')

    if (query.kind === 'user') {
      const contextGuildId = query.guildId?.trim() || undefined
      assertScopeGuildAccess(scope, contextGuildId, 'entity profile guild context')
      return this.getUserProfile(id, scope)
    }

    assertScopeGuildAccess(scope, id, 'entity profile guild')
    return this.getGuildProfile(id)
  }

  private async getUserProfile(userId: string, scope: EntityProfileScope): Promise<UserEntityProfile> {
    const facts = await this.loadScopedFacts(scope)
    const warns = collectUserWarns(facts.warnsByGuild, userId, this.deps.resolveGuildName)
    const blacklist = mapBlacklist(facts.blacklistMap[userId], scope)
    const restricted = collectUserRestricted(facts.guardRecords, userId, this.deps.resolveGuildName)
    const reviews = collectUserReviews(facts.reviews, userId, this.deps.resolveGuildName)
    const reports = collectUserReports(facts.reports, userId, this.deps.resolveGuildName)
    const events = collectUserEvents(facts.events, userId, this.deps.resolveGuildName)
    const userMeta = this.resolveVisibleUserMeta(userId, scope, { warns, blacklist, restricted, reviews, reports, events })

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
        pendingReviews: reviews.filter((review) => isPendingReviewStatus(review.status)).length,
        openReports: reports.length,
        restrictedGuilds: restricted.filter((record) => record.status === 'pending').length,
      },
      warns,
      blacklist,
      restricted,
      reviews,
      reports,
      recentEvents: events,
    }
  }

  private async loadScopedFacts(scope: EntityProfileScope): Promise<ScopedEntityFacts> {
    const [warnsByGuild, blacklistMap, guardRecords, reviews, reports, events] = await Promise.all([
      this.deps.loadWarns(),
      this.deps.loadBlacklist(),
      this.deps.loadGuardRecords(),
      this.deps.loadReviews(),
      this.deps.loadReports(),
      this.deps.loadEvents(EVENT_FETCH_LIMIT),
    ])

    return {
      warnsByGuild: filterWarnsByScope(warnsByGuild, scope),
      blacklistMap,
      guardRecords: filterGuildRecords(guardRecords, scope),
      reviews: filterGuildRecords(reviews, scope),
      reports: filterGuildRecords(reports, scope),
      events: filterGuildRecords(events, scope),
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

    return buildGuildProfile({
      guildId,
      deps: this.deps,
      warnsByGuild,
      guardRecords,
      reviews,
      reports,
      events,
      configured,
    })
  }

  private resolveVisibleUserMeta(
    userId: string,
    scope: EntityProfileScope,
    facts: Parameters<typeof hasUserFacts>[0],
  ) {
    return hasUserFacts(facts) || scope.guildIds === null
      ? this.deps.resolveUserName(userId)
      : undefined
  }
}
