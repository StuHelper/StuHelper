import type { Context } from 'koishi'

import type { EntityProfileQuery, EntityProfileScope } from '../services'
import {
  resolveRequiredConsoleGuildScope,
  type ScopedConsoleClient,
} from './console-guild-scope'
import {
  buildScopedConfigGovernancePageData,
  buildScopedDashboardPageData,
  buildScopedIdentityPageData,
  buildScopedReviewPageData,
} from './page-scope'
import {
  createPageApiRuntime,
  createScopeDeps,
  type PageApiOptions,
  type PageApiRuntime,
} from './page-api-runtime'

const CONSOLE_AUTHORITY = 4

export function registerPageAPI(ctx: Context, options: PageApiOptions) {
  if (!ctx.console) {
    ctx.logger('stuhelperGroupCenter').warn('console 服务未启用，页面域 API 跳过注册')
    return
  }

  const runtime = createPageApiRuntime(ctx, options)
  ctx.console.addListener('stuhelperGroupCenter/page/dashboard', function () {
    return handleDashboardPage(runtime, this)
  }, { authority: CONSOLE_AUTHORITY })
  ctx.console.addListener('stuhelperGroupCenter/page/identity', function () {
    return handleIdentityPage(runtime, this)
  }, { authority: CONSOLE_AUTHORITY })
  ctx.console.addListener('stuhelperGroupCenter/page/review', function () {
    return handleReviewPage(runtime, this)
  }, { authority: CONSOLE_AUTHORITY })
  ctx.console.addListener('stuhelperGroupCenter/page/config-governance', function () {
    return handleConfigGovernancePage(runtime, this)
  }, { authority: CONSOLE_AUTHORITY })
  ctx.console.addListener('stuhelperGroupCenter/page/entity-profile', function (query: EntityProfileQuery) {
    return handleEntityProfilePage(runtime, this, query)
  }, { authority: CONSOLE_AUTHORITY })
}

async function handleDashboardPage(runtime: PageApiRuntime, client: ScopedConsoleClient) {
  const scope = await resolvePageScope(runtime, client)
  if (scope.kind === 'all') return runtime.dashboardPage.getPageData()
  const data = await loadDashboardData(runtime.dashboardDeps)
  return buildScopedDashboardPageData({ generatedAt: new Date().toISOString(), ...data }, scope)
}

async function handleIdentityPage(runtime: PageApiRuntime, client: ScopedConsoleClient) {
  const scope = await resolvePageScope(runtime, client)
  if (scope.kind === 'all') return runtime.identityPage.getPageData()
  const guardRecords = await runtime.identityDeps.loadGuardRecords()
  const scopedRecords = guardRecords.filter((record) => scope.guildIds.has(record.guildId))
  const memberIds = [...new Set(scopedRecords.map((record) => record.memberId))]
  const { profiles, errors } = await runtime.identityDeps.lookupVerificationProfiles(memberIds)
  return buildScopedIdentityPageData({
    generatedAt: new Date().toISOString(),
    guardRecords: scopedRecords,
    verificationProfiles: profiles,
    lookupErrors: errors,
  }, scope)
}

async function handleReviewPage(runtime: PageApiRuntime, client: ScopedConsoleClient) {
  const scope = await resolvePageScope(runtime, client)
  if (scope.kind === 'all') return runtime.reviewPage.getPageData()
  const [pendingReviews, pendingMembers, reports, events] = await Promise.all([
    runtime.reviewDeps.loadPendingReviews(),
    runtime.reviewDeps.loadPendingMembers(),
    runtime.reviewDeps.loadReports(),
    runtime.reviewDeps.loadEvents(),
  ])
  return buildScopedReviewPageData({
    generatedAt: new Date().toISOString(),
    pendingReviews,
    pendingMembers,
    reports,
    events,
  }, scope)
}

async function handleConfigGovernancePage(runtime: PageApiRuntime, client: ScopedConsoleClient) {
  const scope = await resolvePageScope(runtime, client)
  if (scope.kind === 'all') return runtime.configPage.getPageData()
  const [groupConfigs, guildNames, templates, bindings, commandPolicies, supportedCommandIds] = await Promise.all([
    runtime.configDeps.loadGroupConfigs(),
    runtime.configDeps.loadGuildNames(),
    runtime.configDeps.loadTemplates(),
    runtime.configDeps.loadBindings(),
    runtime.configDeps.loadCommandPolicies(),
    runtime.configDeps.loadSupportedCommandIds(),
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
}

async function handleEntityProfilePage(
  runtime: PageApiRuntime,
  client: ScopedConsoleClient,
  query: EntityProfileQuery,
) {
  const scope = await resolvePageScope(runtime, client)
  return runtime.entityPage.getProfile(normalizeEntityProfileQuery(query), toEntityProfileScope(scope))
}

async function loadDashboardData(deps: PageApiRuntime['dashboardDeps']) {
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
    deps.loadPendingMembers(),
    deps.loadPendingReviews(),
    deps.loadRecentEvents(),
    deps.loadRecentReports(),
    deps.loadCommandPolicies(),
    deps.loadGuardTemplates(),
    deps.loadGuardBindings(),
    deps.loadModuleStates(),
  ])
  return { pendingMembers, pendingReviews, recentEvents, recentReports, commandPolicies, guardTemplates, guardBindings, moduleStates }
}

function normalizeEntityProfileQuery(query: EntityProfileQuery): EntityProfileQuery {
  if (!query || (query.kind !== 'user' && query.kind !== 'guild')) {
    throw new Error('entity profile: kind must be user or guild')
  }
  const trimmedId = typeof query.id === 'string' ? query.id.trim() : ''
  if (!trimmedId) {
    throw new Error('entity profile: id is required')
  }
  return {
    kind: query.kind,
    id: trimmedId,
    guildId: typeof query.guildId === 'string' ? query.guildId.trim() || undefined : undefined,
  }
}

function toEntityProfileScope(
  scope: Awaited<ReturnType<typeof resolveRequiredConsoleGuildScope>>,
): EntityProfileScope {
  if (scope.kind === 'all') {
    return { guildIds: null }
  }
  return { guildIds: scope.guildIds }
}

function resolvePageScope(runtime: PageApiRuntime, client: ScopedConsoleClient) {
  return resolveRequiredConsoleGuildScope(client, createScopeDeps(runtime.ctx, runtime.service))
}
