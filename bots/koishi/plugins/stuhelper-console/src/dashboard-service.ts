import { DataService } from '@koishijs/console'
import { Context } from 'koishi'

import {
  ModerationStore,
  SUPPORTED_COMMAND_POLICY_IDS,
} from '@stuhelper/koishi-moderation-core'
import {
  GUARD_MEMBER_TABLE,
  GuardPolicyStore,
  type GuardMemberRecord,
} from '@stuhelper/koishi-shared'

import { STUHELPER_CONSOLE_SERVICE } from './constants'
import type {
  StuhelperConsoleData,
  StuhelperConsoleGuardMember,
  StuhelperConsoleServiceConfig,
} from './console-types'

declare module '@koishijs/console' {
  namespace Console {
    interface Services {
      stuhelperConsole: StuhelperConsoleDataService
    }
  }
}

export class StuhelperConsoleDataService extends DataService<StuhelperConsoleData> {
  private readonly moderationStore: ModerationStore
  private readonly guardPolicyStore: GuardPolicyStore
  private readonly title: string

  constructor(ctx: Context, config: StuhelperConsoleServiceConfig) {
    super(ctx, STUHELPER_CONSOLE_SERVICE, { authority: 4 })
    this.moderationStore = new ModerationStore(ctx)
    this.guardPolicyStore = new GuardPolicyStore(ctx)
    this.title = config.title
  }

  async get() {
    const [overview, pendingMembers, pendingReviews, keywordRules, commandPolicies, memberRoles, guardTemplates, guardBindings, recentReports] = await Promise.all([
      this.moderationStore.getOverview(),
      listActiveGuardMembers(this.ctx),
      this.moderationStore.listPendingReviews(),
      this.moderationStore.listAllKeywordRules(),
      this.moderationStore.listCommandPolicies(),
      this.moderationStore.listMemberRoles(),
      this.guardPolicyStore.listTemplates(),
      this.guardPolicyStore.listBindings(),
      this.moderationStore.listOpenReports(),
    ])
    return {
      title: this.title,
      generatedAt: new Date().toISOString(),
      supportedCommandIds: [...SUPPORTED_COMMAND_POLICY_IDS],
      overview: {
        pendingReviews: overview.pendingReviews,
        openReports: overview.openReports,
        warningMembers: overview.warningMembers,
        highRiskEvents: overview.highRiskEvents,
      },
      pendingMembers: pendingMembers.map(serializeGuardMember),
      pendingReviews: pendingReviews
        .sort((left, right) => right.createdAt.getTime() - left.createdAt.getTime())
        .map(serializeRecord),
      keywordRules: keywordRules
        .sort((left, right) => left.guildId.localeCompare(right.guildId) || left.id.localeCompare(right.id))
        .map(serializeRecord),
      commandPolicies: commandPolicies
        .sort((left, right) => left.commandId.localeCompare(right.commandId))
        .map(serializeRecord),
      memberRoles: memberRoles
        .sort((left, right) => left.guildId.localeCompare(right.guildId) || left.memberId.localeCompare(right.memberId))
        .map(serializeRecord),
      guardTemplates: guardTemplates
        .sort((left, right) => left.name.localeCompare(right.name) || left.id.localeCompare(right.id))
        .map(serializeRecord),
      guardBindings: guardBindings
        .sort((left, right) => left.platform.localeCompare(right.platform) || left.guildId.localeCompare(right.guildId))
        .map(serializeRecord),
      recentEvents: overview.recentEvents.map(serializeRecord),
      recentReports: recentReports
        .sort((left, right) => right.createdAt.getTime() - left.createdAt.getTime())
        .slice(0, 20)
        .map(serializeRecord),
    }
  }
}

async function listActiveGuardMembers(ctx: Context) {
  const records = await ctx.database.get(GUARD_MEMBER_TABLE, {}) as GuardMemberRecord[]
  return records.filter((record) => !record.releasedAt && !record.kickedAt)
}

function serializeGuardMember(record: GuardMemberRecord): StuhelperConsoleGuardMember {
  return {
    ...record,
    joinedAt: record.joinedAt.toISOString(),
    deadlineAt: record.deadlineAt.toISOString(),
    mutedAt: serializeNullableDate(record.mutedAt),
    reminderSentAt: serializeNullableDate(record.reminderSentAt),
    releasedAt: serializeNullableDate(record.releasedAt),
    kickedAt: serializeNullableDate(record.kickedAt),
    createdAt: record.createdAt.toISOString(),
    updatedAt: record.updatedAt.toISOString(),
  }
}

function serializeNullableDate(value: Date | null) {
  return value ? value.toISOString() : null
}

function serializeRecord<T extends { createdAt: Date, updatedAt: Date }>(record: T) {
  return {
    ...record,
    createdAt: record.createdAt.toISOString(),
    updatedAt: record.updatedAt.toISOString(),
  }
}
