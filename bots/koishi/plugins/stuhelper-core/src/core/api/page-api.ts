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
  type StuhelperGuardConfig,
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
import type { EntityProfileQuery, EntityProfileScope } from '../services'
import { IdentityProfileLookup } from './identity-profile-lookup'
import {
  buildScopedConfigGovernancePageData,
  buildScopedDashboardPageData,
  buildScopedIdentityPageData,
  buildScopedReviewPageData,
} from './page-scope'
import { resolveRequiredConsoleGuildScope } from './console-guild-scope'

interface PageApiOptions {
  service: StuhelperGroupCenterService
  platform: StuhelperPlatformConfig
  guard: StuhelperGuardConfig
}

export function registerPageAPI(ctx: Context, options: PageApiOptions) {
  if (!ctx.console) {
    ctx.logger('stuhelperGroupCenter').warn('console 服务未启用，页面域 API 跳过注册')
    return
  }

  const moderationStore = new ModerationStore(ctx)
  const guardPolicyStore = new GuardPolicyStore(ctx, options.guard)
  const platform = createPlatformClient(options.platform)
  const identityProfileLookup = new IdentityProfileLookup({
    getQQVerificationStatus: (memberId) => platform.getQQVerificationStatus(memberId),
  })

  const dashboardDeps = {
    loadPendingMembers: () => listActiveGuardMembers(ctx),
    loadPendingReviews: () => moderationStore.listPendingReviews(),
    loadRecentEvents: () => moderationStore.listRecentEvents(20),
    loadRecentReports: () => moderationStore.listOpenReports(),
    loadCommandPolicies: () => moderationStore.listCommandPolicies(),
    loadGuardTemplates: () => guardPolicyStore.listTemplates(),
    loadGuardBindings: () => guardPolicyStore.listBindings(),
    loadModuleStates: () => options.service.getAllModules().map((module) => ({
      name: module.meta.name,
      description: module.meta.description,
      state: module.state,
      error: module.error?.message,
    })),
  }
  const dashboardPage = new DashboardPageService(dashboardDeps)

  const identityDeps = {
    loadGuardRecords: () => listGuardRecords(ctx),
    lookupVerificationProfiles: async (memberIds) => identityProfileLookup.lookup(memberIds),
  }
  const identityPage = new IdentityPageService(identityDeps)

  const reviewDeps = {
    loadPendingReviews: () => moderationStore.listPendingReviews(),
    loadPendingMembers: () => listActiveGuardMembers(ctx),
    loadReports: () => moderationStore.listOpenReports(),
    loadEvents: () => moderationStore.listRecentEvents(50),
  }
  const reviewPage = new ReviewPageService(reviewDeps)

  const configDeps = {
    loadGroupConfigs: async () => options.service.data.groupConfig.getAll() as Record<string, Record<string, unknown>>,
    loadGuildNames: async () => {
      const cacheData = options.service.cache.getCachedData()
      return Object.fromEntries(
        Object.entries(cacheData.guilds).map(([guildId, value]) => [guildId, {
          name: value.name,
          avatar: value.avatar,
        }]),
      )
    },
    loadTemplates: () => guardPolicyStore.listTemplates(),
    loadBindings: () => guardPolicyStore.listBindings(),
    loadCommandPolicies: () => moderationStore.listCommandPolicies(),
    loadSupportedCommandIds: () => [...SUPPORTED_COMMAND_POLICY_IDS],
  }
  const configPage = new ConfigGovernanceService(configDeps)

  const entityDeps = {
    loadWarns: async () => {
      const all = await options.service.data.warns.getAll()
      return all as Record<string, Record<string, { count: number; timestamp: number }>>
    },
    loadBlacklist: async () => {
      const all = await options.service.data.blacklist.getAll()
      return all as Record<string, { userId: string; timestamp: number; reason?: string }>
    },
    loadGuardRecords: () => listGuardRecords(ctx),
    loadReviews: () => moderationStore.listPendingReviews(),
    loadReports: () => moderationStore.listOpenReports(),
    loadEvents: (limit: number) => moderationStore.listRecentEvents(limit),
    hasGuildConfig: async (guildId: string) => {
      const all = await options.service.data.groupConfig.getAll()
      return Boolean((all as Record<string, unknown>)[guildId])
    },
    resolveGuildName: (guildId: string) => {
      const cache = options.service.cache.getCachedData()
      const entry = cache.guilds[guildId]
      if (!entry) return undefined
      return { name: entry.name ?? null, avatar: entry.avatar ?? null }
    },
    resolveUserName: (userId: string) => {
      const cache = options.service.cache.getCachedData()
      const entry = cache.users[userId]
      if (!entry) return undefined
      return { name: entry.name ?? null, avatar: entry.avatar ?? null }
    },
  }
  const entityPage = new EntityPageService(entityDeps)

  ctx.console.addListener('stuhelperGroupCenter/page/dashboard', async function () {
    const scope = await resolveRequiredConsoleGuildScope(this, createScopeDeps(ctx, options.service))
    if (scope.kind === 'all') {
      return dashboardPage.getPageData()
    }
    const [
      pendingMembers,
      pendingReviews,
      recentEvents,
      recentReports,
      commandPolicies,
      guardTemplates,
      guardBindings,
      moduleStates,
    ] = await Promise.all([
      dashboardDeps.loadPendingMembers(),
      dashboardDeps.loadPendingReviews(),
      dashboardDeps.loadRecentEvents(),
      dashboardDeps.loadRecentReports(),
      dashboardDeps.loadCommandPolicies(),
      dashboardDeps.loadGuardTemplates(),
      dashboardDeps.loadGuardBindings(),
      dashboardDeps.loadModuleStates(),
    ])
    return buildScopedDashboardPageData({
      generatedAt: new Date().toISOString(),
      pendingMembers,
      pendingReviews,
      recentEvents,
      recentReports,
      commandPolicies,
      guardTemplates,
      guardBindings,
      moduleStates,
    }, scope)
  }, { authority: 4 })

  ctx.console.addListener('stuhelperGroupCenter/page/identity', async function () {
    const scope = await resolveRequiredConsoleGuildScope(this, createScopeDeps(ctx, options.service))
    if (scope.kind === 'all') {
      return identityPage.getPageData()
    }
    const guardRecords = await identityDeps.loadGuardRecords()
    const scopedRecords = guardRecords.filter((record) => scope.guildIds.has(record.guildId))
    const memberIds = [...new Set(scopedRecords.map((record) => record.memberId))]
    const { profiles, errors } = await identityDeps.lookupVerificationProfiles(memberIds)
    return buildScopedIdentityPageData({
      generatedAt: new Date().toISOString(),
      guardRecords: scopedRecords,
      verificationProfiles: profiles,
      lookupErrors: errors,
    }, scope)
  }, { authority: 4 })

  ctx.console.addListener('stuhelperGroupCenter/page/review', async function () {
    const scope = await resolveRequiredConsoleGuildScope(this, createScopeDeps(ctx, options.service))
    if (scope.kind === 'all') {
      return reviewPage.getPageData()
    }
    const [pendingReviews, pendingMembers, reports, events] = await Promise.all([
      reviewDeps.loadPendingReviews(),
      reviewDeps.loadPendingMembers(),
      reviewDeps.loadReports(),
      reviewDeps.loadEvents(),
    ])
    return buildScopedReviewPageData({
      generatedAt: new Date().toISOString(),
      pendingReviews,
      pendingMembers,
      reports,
      events,
    }, scope)
  }, { authority: 4 })

  ctx.console.addListener('stuhelperGroupCenter/page/config-governance', async function () {
    const scope = await resolveRequiredConsoleGuildScope(this, createScopeDeps(ctx, options.service))
    if (scope.kind === 'all') {
      return configPage.getPageData()
    }
    const [groupConfigs, guildNames, templates, bindings, commandPolicies, supportedCommandIds] = await Promise.all([
      configDeps.loadGroupConfigs(),
      configDeps.loadGuildNames(),
      configDeps.loadTemplates(),
      configDeps.loadBindings(),
      configDeps.loadCommandPolicies(),
      configDeps.loadSupportedCommandIds(),
    ])
    return buildScopedConfigGovernancePageData({
      generatedAt: new Date().toISOString(),
      groupConfigs,
      guildNames,
      templates,
      bindings,
      commandPolicies,
      supportedCommandIds: [...supportedCommandIds],
    }, scope)
  }, { authority: 4 })

  ctx.console.addListener('stuhelperGroupCenter/page/entity-profile', async function (query: EntityProfileQuery) {
    const scope = await resolveRequiredConsoleGuildScope(this, createScopeDeps(ctx, options.service))
    if (!query || (query.kind !== 'user' && query.kind !== 'guild')) {
      throw new Error('entity profile: kind must be user or guild')
    }
    const trimmedId = typeof query.id === 'string' ? query.id.trim() : ''
    if (!trimmedId) {
      throw new Error('entity profile: id is required')
    }
    return entityPage.getProfile({
      kind: query.kind,
      id: trimmedId,
      guildId: typeof query.guildId === 'string' ? query.guildId.trim() || undefined : undefined,
    }, toEntityProfileScope(scope))
  }, { authority: 4 })
}

function toEntityProfileScope(
  scope: Awaited<ReturnType<typeof resolveRequiredConsoleGuildScope>>,
): EntityProfileScope {
  if (scope.kind === 'all') {
    return { guildIds: null }
  }
  return { guildIds: scope.guildIds }
}

async function listGuardRecords(ctx: Context) {
  return ctx.database.get(GUARD_MEMBER_TABLE, {}) as Promise<GuardMemberRecord[]>
}

async function listActiveGuardMembers(ctx: Context) {
  const records = await listGuardRecords(ctx)
  return records.filter((record) => !record.releasedAt && !record.kickedAt)
}

function createScopeDeps(ctx: Context, service: StuhelperGroupCenterService) {
  return {
    roles: service.auth.getRoles(),
    getUserRoleIds: (userId: string) => service.auth.getUserRoleIds(userId),
    listBindingsByAuthId: (authId: number) => ctx.database.get('binding', { aid: authId }),
  }
}
