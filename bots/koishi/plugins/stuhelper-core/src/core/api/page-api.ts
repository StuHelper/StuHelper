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
  IdentityPageService,
  ReviewPageService,
} from '../services'
import { IdentityProfileLookup } from './identity-profile-lookup'

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

  const dashboardPage = new DashboardPageService({
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
  })

  const identityPage = new IdentityPageService({
    loadGuardRecords: () => listGuardRecords(ctx),
    lookupVerificationProfiles: async (memberIds) => identityProfileLookup.lookup(memberIds),
  })

  const reviewPage = new ReviewPageService({
    loadPendingReviews: () => moderationStore.listPendingReviews(),
    loadPendingMembers: () => listActiveGuardMembers(ctx),
    loadReports: () => moderationStore.listOpenReports(),
    loadEvents: () => moderationStore.listRecentEvents(50),
  })

  const configPage = new ConfigGovernanceService({
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
  })

  ctx.console.addListener('stuhelperGroupCenter/page/dashboard', async () => {
    return dashboardPage.getPageData()
  }, { authority: 4 })

  ctx.console.addListener('stuhelperGroupCenter/page/identity', async () => {
    return identityPage.getPageData()
  }, { authority: 4 })

  ctx.console.addListener('stuhelperGroupCenter/page/review', async () => {
    return reviewPage.getPageData()
  }, { authority: 4 })

  ctx.console.addListener('stuhelperGroupCenter/page/config-governance', async () => {
    return configPage.getPageData()
  }, { authority: 4 })
}

async function listGuardRecords(ctx: Context) {
  return ctx.database.get(GUARD_MEMBER_TABLE, {}) as Promise<GuardMemberRecord[]>
}

async function listActiveGuardMembers(ctx: Context) {
  const records = await listGuardRecords(ctx)
  return records.filter((record) => !record.releasedAt && !record.kickedAt)
}
