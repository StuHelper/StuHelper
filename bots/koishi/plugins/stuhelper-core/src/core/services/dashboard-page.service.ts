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
import type { DashboardOverview, DashboardPageData, OverviewPageData } from './page-types'

export interface DashboardPageBuilderInput {
  generatedAt: string
  pendingMembers: GuardMemberRecord[]
  pendingReviews: ReviewQueueRecord[]
  recentEvents: ModerationEventRecord[]
  recentReports: ModerationReportRecord[]
  commandPolicies: CommandPolicyRecord[]
  guardTemplates: GuardTemplateRecord[]
  guardBindings: GuardGroupBindingRecord[]
}

/**
 * 总览计数的唯一来源：dashboard 全量页与脉冲轻量端点都经此计算，保证两侧数字永远一致。
 * 注意 highRiskEvents 是「最近窗口内」的高危计数（与 recentEvents 同源），不是全表计数，
 * 因此脉冲端点不能改用全表 count 查询，否则会与 dashboard 显示的数字漂移。
 */
export function buildDashboardOverview(input: DashboardPageBuilderInput): DashboardOverview {
  return {
    pendingReviews: input.pendingReviews.length,
    pendingAdmissions: input.pendingMembers.length,
    openReports: input.recentReports.length,
    highRiskEvents: input.recentEvents.filter((item) => item.level === 'high' || item.level === 'critical').length,
    policyItems: input.commandPolicies.length + input.guardTemplates.length + input.guardBindings.length,
  }
}

export function buildDashboardPageData(input: DashboardPageBuilderInput): DashboardPageData {
  return {
    generatedAt: input.generatedAt,
    overview: buildDashboardOverview(input),
    pendingMembers: sortByUpdatedDesc(input.pendingMembers).map(serializeGuardMember),
    pendingReviews: sortByCreatedDesc(input.pendingReviews).map(serializeReview),
    recentEvents: sortByCreatedDesc(input.recentEvents).slice(0, 12).map(serializeEvent),
    recentReports: sortByCreatedDesc(input.recentReports).slice(0, 12).map(serializeReport),
    commandPolicies: [...input.commandPolicies].sort((left, right) => left.commandId.localeCompare(right.commandId)).map(serializeCommandPolicy),
    guardTemplates: [...input.guardTemplates].sort((left, right) => left.name.localeCompare(right.name)).map(serializeGuardTemplate),
    guardBindings: [...input.guardBindings].sort((left, right) => left.guildId.localeCompare(right.guildId)).map(serializeGuardBinding),
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
    ] = await Promise.all([
      this.deps.loadPendingMembers(),
      this.deps.loadPendingReviews(),
      this.deps.loadRecentEvents(),
      this.deps.loadRecentReports(),
      this.deps.loadCommandPolicies(),
      this.deps.loadGuardTemplates(),
      this.deps.loadGuardBindings(),
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
    })
  }

  /**
   * 脉冲轻量总览：与 getPageData 共用同一组加载器与 buildDashboardOverview，
   * 保证计数与 dashboard 完全一致，但不序列化任何明细列表、不返回大体积负载，
   * 适合 AppShell 每 30s 的全局轮询。
   */
  async getOverviewData(): Promise<OverviewPageData> {
    const [
      pendingMembers,
      pendingReviews,
      recentEvents,
      recentReports,
      commandPolicies,
      guardTemplates,
      guardBindings,
    ] = await Promise.all([
      this.deps.loadPendingMembers(),
      this.deps.loadPendingReviews(),
      this.deps.loadRecentEvents(),
      this.deps.loadRecentReports(),
      this.deps.loadCommandPolicies(),
      this.deps.loadGuardTemplates(),
      this.deps.loadGuardBindings(),
    ])

    return {
      generatedAt: new Date().toISOString(),
      overview: buildDashboardOverview({
        generatedAt: '',
        pendingMembers,
        pendingReviews,
        recentEvents,
        recentReports,
        commandPolicies,
        guardTemplates,
        guardBindings,
      }),
    }
  }
}

function sortByCreatedDesc<T extends { createdAt: Date }>(items: readonly T[]) {
  return [...items].sort((left, right) => right.createdAt.getTime() - left.createdAt.getTime())
}

function sortByUpdatedDesc<T extends { updatedAt: Date }>(items: readonly T[]) {
  return [...items].sort((left, right) => right.updatedAt.getTime() - left.updatedAt.getTime())
}
