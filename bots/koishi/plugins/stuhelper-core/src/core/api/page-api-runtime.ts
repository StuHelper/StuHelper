import type { Context } from 'koishi'
import {
  SUPPORTED_COMMAND_POLICY_IDS,
  ModerationStore,
} from '@stuhelper/koishi-moderation-core'
import {
  GUARD_MEMBER_TABLE,
  GuardPolicyStore,
  createPlatformClient,
  type GuardMemberRecord,
  type MemberBlacklistEntry,
  type StuhelperPlatformConfig,
} from '@stuhelper/koishi-shared'

import type { StuhelperGroupCenterService } from '../services'
import {
  ConfigGovernanceService,
  DashboardPageService,
  EntityPageService,
  IdentityPageService,
  ReviewPageService,
} from '../services'
import { IdentityProfileLookup } from './identity-profile-lookup'
import { DEFAULT_MEMBER_BLACKLIST_PLATFORM } from './member-blacklist-defaults'
import { MEMBER_BLACKLIST_PAGE_SIZE } from '../member-blacklist-pages'

const RECENT_DASHBOARD_EVENT_LIMIT = 20
const RECENT_REVIEW_EVENT_LIMIT = 50

export interface PageApiOptions {
  service: StuhelperGroupCenterService
  platform: StuhelperPlatformConfig
}

export interface PageApiRuntime {
  readonly ctx: Context
  readonly service: StuhelperGroupCenterService
  readonly dashboardDeps: ReturnType<typeof createDashboardDeps>
  readonly dashboardPage: DashboardPageService
  readonly identityDeps: ReturnType<typeof createIdentityDeps>
  readonly identityPage: IdentityPageService
  readonly reviewDeps: ReturnType<typeof createReviewDeps>
  readonly reviewPage: ReviewPageService
  readonly configDeps: ReturnType<typeof createConfigDeps>
  readonly configPage: ConfigGovernanceService
  readonly entityPage: EntityPageService
}

export function createPageApiRuntime(ctx: Context, options: PageApiOptions): PageApiRuntime {
  const moderationStore = new ModerationStore(ctx)
  const guardPolicyStore = new GuardPolicyStore(ctx)
  const platform = createPlatformClient(options.platform)
  const identityProfileLookup = new IdentityProfileLookup({
    getQQVerificationStatus: (memberId) => platform.getQQVerificationStatus(memberId),
  })
  const dashboardDeps = createDashboardDeps({
    ctx,
    service: options.service,
    moderationStore,
    guardPolicyStore,
  })
  const identityDeps = createIdentityDeps(ctx, identityProfileLookup)
  const reviewDeps = createReviewDeps(ctx, moderationStore)
  const configDeps = createConfigDeps(options.service, moderationStore, guardPolicyStore)

  return {
    ctx,
    service: options.service,
    dashboardDeps,
    dashboardPage: new DashboardPageService(dashboardDeps),
    identityDeps,
    identityPage: new IdentityPageService(identityDeps),
    reviewDeps,
    reviewPage: new ReviewPageService(reviewDeps),
    configDeps,
    configPage: new ConfigGovernanceService(configDeps),
    entityPage: new EntityPageService(createEntityDeps({
      ctx,
      service: options.service,
      moderationStore,
      platform,
      guardPolicyStore,
    })),
  }
}

export function createScopeDeps(ctx: Context, service: StuhelperGroupCenterService) {
  return {
    roles: service.auth.getRoles(),
    getUserRoleIds: (userId: string) => service.auth.getUserRoleIds(userId),
    listBindingsByAuthId: (authId: number) => ctx.database.get('binding', { aid: authId }),
  }
}

function createDashboardDeps(input: {
  readonly ctx: Context
  readonly service: StuhelperGroupCenterService
  readonly moderationStore: ModerationStore
  readonly guardPolicyStore: GuardPolicyStore
}) {
  const { ctx, service, moderationStore, guardPolicyStore } = input
  return {
    loadPendingMembers: () => listActiveGuardMembers(ctx),
    loadPendingReviews: () => moderationStore.listAllPendingReviews(),
    loadRecentEvents: () => moderationStore.listRecentEvents(RECENT_DASHBOARD_EVENT_LIMIT),
    loadRecentReports: () => moderationStore.listOpenReports(),
    loadCommandPolicies: () => moderationStore.listCommandPolicies(),
    loadGuardTemplates: () => guardPolicyStore.listTemplates(),
    loadGuardBindings: () => guardPolicyStore.listBindings(),
  }
}

function createIdentityDeps(ctx: Context, identityProfileLookup: IdentityProfileLookup) {
  return {
    loadGuardRecords: () => listGuardRecords(ctx),
    lookupVerificationProfiles: async (memberIds: string[]) => identityProfileLookup.lookup(memberIds),
  }
}

function createReviewDeps(ctx: Context, moderationStore: ModerationStore) {
  return {
    loadPendingReviews: () => moderationStore.listAllPendingReviews(),
    loadPendingMembers: () => listActiveGuardMembers(ctx),
    loadReports: () => moderationStore.listOpenReports(),
    loadEvents: () => moderationStore.listRecentEvents(RECENT_REVIEW_EVENT_LIMIT),
  }
}

function createConfigDeps(
  service: StuhelperGroupCenterService,
  moderationStore: ModerationStore,
  guardPolicyStore: GuardPolicyStore,
) {
  return {
    loadGroupConfigs: async () => service.data.groupConfig.getAll() as Record<string, Record<string, unknown>>,
    loadGuildNames: async () => readCachedGuildNames(service),
    loadTemplates: () => guardPolicyStore.listTemplates(),
    loadBindings: () => guardPolicyStore.listBindings(),
    loadCommandPolicies: () => moderationStore.listCommandPolicies(),
    loadSupportedCommandIds: () => [...SUPPORTED_COMMAND_POLICY_IDS],
  }
}

function createEntityDeps(input: {
  readonly ctx: Context
  readonly service: StuhelperGroupCenterService
  readonly moderationStore: ModerationStore
  readonly platform: ReturnType<typeof createPlatformClient>
  readonly guardPolicyStore: GuardPolicyStore
}) {
  const { ctx, service, moderationStore, platform, guardPolicyStore } = input
  return {
    loadWarns: async () => service.data.warns.getAll() as Record<string, Record<string, { count: number; timestamp: number }>>,
    loadBlacklist: async () => readMemberBlacklistMap(platform),
    loadGuardRecords: () => listGuardRecords(ctx),
    loadReviews: () => moderationStore.listAllPendingReviews(),
    loadReports: () => moderationStore.listOpenReports(),
    loadEvents: (limit: number) => moderationStore.listRecentEvents(limit),
    hasGuildConfig: async (guildId: string) => Boolean((await service.data.groupConfig.getAll() as Record<string, unknown>)[guildId]),
    hasAdmissionPolicy: async (guildId: string) => {
      const normalizedGuildId = guildId.trim()
      if (!normalizedGuildId) return false
      const bindings = await guardPolicyStore.listBindings()
      return bindings.some((binding) => binding.guildId === normalizedGuildId && binding.enabled)
    },
    resolveGuildName: (guildId: string) => readCachedGuildName(service, guildId),
    resolveUserName: (userId: string) => readCachedUserName(service, userId),
  }
}

async function listGuardRecords(ctx: Context) {
  return ctx.database.get(GUARD_MEMBER_TABLE, {}) as Promise<GuardMemberRecord[]>
}

async function listActiveGuardMembers(ctx: Context) {
  const records = await listGuardRecords(ctx)
  return records.filter((record) => !record.releasedAt && !record.kickedAt)
}

function readCachedGuildNames(service: StuhelperGroupCenterService) {
  const cacheData = service.cache.getCachedData()
  return Object.fromEntries(
    Object.entries(cacheData.guilds).map(([guildId, value]) => [guildId, {
      name: value.name,
      avatar: value.avatar,
    }]),
  )
}

function readCachedGuildName(service: StuhelperGroupCenterService, guildId: string) {
  const entry = service.cache.getCachedData().guilds[guildId]
  if (!entry) return undefined
  return { name: entry.name ?? null, avatar: entry.avatar ?? null }
}

function readCachedUserName(service: StuhelperGroupCenterService, userId: string) {
  const entry = service.cache.getCachedData().users[userId]
  if (!entry) return undefined
  return { name: entry.name ?? null, avatar: entry.avatar ?? null }
}

async function readMemberBlacklistMap(platform: ReturnType<typeof createPlatformClient>) {
  const entries = await loadActiveMemberBlacklists(platform)
  return Object.fromEntries(entries.map((entry) => [entry.subjectID, {
    userId: entry.subjectID,
    guildId: entry.guildID,
    scopeType: entry.scopeType,
    timestamp: Date.parse(entry.createdAt),
    reason: entry.reasonText || entry.reasonCode,
  }]))
}

async function loadActiveMemberBlacklists(platform: ReturnType<typeof createPlatformClient>) {
  const entries: MemberBlacklistEntry[] = []
  for (let page = 1; ; page++) {
    const result = await platform.listMemberBlacklist({
      platform: DEFAULT_MEMBER_BLACKLIST_PLATFORM,
      status: 'active',
      page,
      pageSize: MEMBER_BLACKLIST_PAGE_SIZE,
    })
    entries.push(...result.list)
    if (entries.length >= result.total) return entries
    if (result.list.length === 0) {
      throw new Error(`member blacklist pagination ended before total was reached: ${entries.length}/${result.total}`)
    }
  }
}
