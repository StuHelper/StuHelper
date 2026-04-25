import {
  buildDashboardPageData,
  type DashboardPageBuilderInput,
} from '../services/dashboard-page.service'
import {
  buildIdentityPageData,
  type IdentityPageBuilderInput,
} from '../services/identity-page.service'
import {
  buildReviewPageData,
  type ReviewPageBuilderInput,
} from '../services/review-page.service'
import {
  buildConfigGovernanceData,
  type ConfigGovernanceBuilderInput,
} from '../services/config-governance.service'

import {
  hasConsoleGuildAccess,
  type ConsoleGuildScope,
} from './console-guild-scope'

export function buildScopedDashboardPageData(
  input: DashboardPageBuilderInput,
  scope: ConsoleGuildScope,
) {
  if (scope.kind === 'all') {
    return buildDashboardPageData(input)
  }

  return buildDashboardPageData({
    ...input,
    pendingMembers: filterGuildRecords(input.pendingMembers, scope),
    pendingReviews: filterGuildRecords(input.pendingReviews, scope),
    recentEvents: filterGuildRecords(input.recentEvents, scope),
    recentReports: filterGuildRecords(input.recentReports, scope),
    guardBindings: filterGuildRecords(input.guardBindings, scope),
  })
}

export function buildScopedIdentityPageData(
  input: IdentityPageBuilderInput,
  scope: ConsoleGuildScope,
) {
  if (scope.kind === 'all') {
    return buildIdentityPageData(input)
  }

  const guardRecords = filterGuildRecords(input.guardRecords, scope)
  const memberIds = new Set(guardRecords.map((record) => record.memberId))
  return buildIdentityPageData({
    ...input,
    guardRecords,
    verificationProfiles: input.verificationProfiles.filter((profile) => memberIds.has(profile.qqID)),
    lookupErrors: input.lookupErrors.filter((error) => memberIds.has(error.memberId)),
  })
}

export function buildScopedReviewPageData(
  input: ReviewPageBuilderInput,
  scope: ConsoleGuildScope,
) {
  if (scope.kind === 'all') {
    return buildReviewPageData(input)
  }

  return buildReviewPageData({
    ...input,
    pendingReviews: filterGuildRecords(input.pendingReviews, scope),
    pendingMembers: filterGuildRecords(input.pendingMembers, scope),
    reports: filterGuildRecords(input.reports, scope),
    events: filterGuildRecords(input.events, scope),
  })
}

export function buildScopedConfigGovernancePageData(
  input: ConfigGovernanceBuilderInput,
  scope: ConsoleGuildScope,
) {
  if (scope.kind === 'all') {
    return buildConfigGovernanceData(input)
  }

  const groupConfigs = Object.fromEntries(
    Object.entries(input.groupConfigs).filter(([guildId]) => hasConsoleGuildAccess(scope, guildId)),
  )
  const guildNames = Object.fromEntries(
    Object.entries(input.guildNames).filter(([guildId]) => hasConsoleGuildAccess(scope, guildId)),
  )

  return buildConfigGovernanceData({
    ...input,
    groupConfigs,
    guildNames,
    bindings: filterGuildRecords(input.bindings, scope),
  })
}

function filterGuildRecords<T extends { guildId?: string | null }>(
  items: readonly T[],
  scope: ConsoleGuildScope,
) {
  return items.filter((item) => hasConsoleGuildAccess(scope, item.guildId || undefined))
}
