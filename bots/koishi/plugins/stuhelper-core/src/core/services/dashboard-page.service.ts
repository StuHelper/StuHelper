import type {
  CommandPolicyRecord,
  ModerationEventRecord,
  ModerationReportRecord,
  ReviewQueueRecord,
} from '@stuhelper/koishi-moderation-core'
import type {
  GuardGroupBindingRecord,
  GuardMemberRecord,
  GuardTemplateRecord,
} from '@stuhelper/koishi-shared'

import {
  serializeCommandPolicy,
  serializeEvent,
  serializeGuardBinding,
  serializeGuardMember,
  serializeGuardTemplate,
  serializeReport,
  serializeReview,
} from './page-serializers'
import type { DashboardPageData, ModuleStateSnapshot } from './page-types'

export interface DashboardPageBuilderInput {
  generatedAt: string
  pendingMembers: GuardMemberRecord[]
  pendingReviews: ReviewQueueRecord[]
  recentEvents: ModerationEventRecord[]
  recentReports: ModerationReportRecord[]
  commandPolicies: CommandPolicyRecord[]
  guardTemplates: GuardTemplateRecord[]
  guardBindings: GuardGroupBindingRecord[]
  moduleStates: ModuleStateSnapshot[]
}

export function buildDashboardPageData(input: DashboardPageBuilderInput): DashboardPageData {
  return {
    generatedAt: input.generatedAt,
    overview: {
      pendingReviews: input.pendingReviews.length,
      pendingAdmissions: input.pendingMembers.length,
      openReports: input.recentReports.length,
      highRiskEvents: input.recentEvents.filter((item) => item.level === 'high' || item.level === 'critical').length,
      policyItems: input.commandPolicies.length + input.guardTemplates.length + input.guardBindings.length,
    },
    pendingMembers: sortByUpdatedDesc(input.pendingMembers).map(serializeGuardMember),
    pendingReviews: sortByCreatedDesc(input.pendingReviews).map(serializeReview),
    recentEvents: sortByCreatedDesc(input.recentEvents).slice(0, 12).map(serializeEvent),
    recentReports: sortByCreatedDesc(input.recentReports).slice(0, 12).map(serializeReport),
    commandPolicies: [...input.commandPolicies].sort((left, right) => left.commandId.localeCompare(right.commandId)).map(serializeCommandPolicy),
    guardTemplates: [...input.guardTemplates].sort((left, right) => left.name.localeCompare(right.name)).map(serializeGuardTemplate),
    guardBindings: [...input.guardBindings].sort((left, right) => left.guildId.localeCompare(right.guildId)).map(serializeGuardBinding),
    systemStatus: [...input.moduleStates],
  }
}

export interface DashboardPageServiceDeps {
  loadPendingMembers: () => Promise<GuardMemberRecord[]>
  loadPendingReviews: () => Promise<ReviewQueueRecord[]>
  loadRecentEvents: () => Promise<ModerationEventRecord[]>
  loadRecentReports: () => Promise<ModerationReportRecord[]>
  loadCommandPolicies: () => Promise<CommandPolicyRecord[]>
  loadGuardTemplates: () => Promise<GuardTemplateRecord[]>
  loadGuardBindings: () => Promise<GuardGroupBindingRecord[]>
  loadModuleStates: () => Promise<ModuleStateSnapshot[]> | ModuleStateSnapshot[]
}

export class DashboardPageService {
  constructor(private readonly deps: DashboardPageServiceDeps) {}

  async getPageData() {
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
      this.deps.loadPendingMembers(),
      this.deps.loadPendingReviews(),
      this.deps.loadRecentEvents(),
      this.deps.loadRecentReports(),
      this.deps.loadCommandPolicies(),
      this.deps.loadGuardTemplates(),
      this.deps.loadGuardBindings(),
      this.deps.loadModuleStates(),
    ])

    return buildDashboardPageData({
      generatedAt: new Date().toISOString(),
      pendingMembers,
      pendingReviews,
      recentEvents,
      recentReports,
      commandPolicies,
      guardTemplates,
      guardBindings,
      moduleStates: [...moduleStates],
    })
  }
}

function sortByCreatedDesc<T extends { createdAt: Date }>(items: readonly T[]) {
  return [...items].sort((left, right) => right.createdAt.getTime() - left.createdAt.getTime())
}

function sortByUpdatedDesc<T extends { updatedAt: Date }>(items: readonly T[]) {
  return [...items].sort((left, right) => right.updatedAt.getTime() - left.updatedAt.getTime())
}
