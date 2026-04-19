import { randomUUID } from 'node:crypto'

import { Context } from 'koishi'

import { GUARD_MEMBER_TABLE } from '@stuhelper/koishi-shared'

import {
  MODERATION_COMMAND_POLICY_TABLE,
  MODERATION_EVENT_TABLE,
  MODERATION_FUN_PROFILE_TABLE,
  MODERATION_KEYWORD_RULE_TABLE,
  MODERATION_MEMBER_ROLE_TABLE,
  MODERATION_MESSAGE_LEDGER_TABLE,
  MODERATION_REPORT_TABLE,
  MODERATION_REVIEW_TABLE,
  MODERATION_WARNING_TABLE,
} from './constants'
import type {
  CommandPolicyRecord,
  FunProfileRecord,
  KeywordRuleRecord,
  MemberRoleRecord,
  MessageLedgerRecord,
  ModerationEventRecord,
  ModerationOverview,
  ModerationReportRecord,
  ReviewQueueRecord,
  ReviewStatus,
  WarningCounterRecord,
} from './types'

const REVIEW_STATUS_PENDING: ReviewStatus = 'pending'

export class ModerationStore {
  constructor(private readonly ctx: Context) {}

  async appendEvent(input: Omit<ModerationEventRecord, 'id' | 'createdAt' | 'updatedAt'>) {
    const now = new Date()
    const record: ModerationEventRecord = { id: randomUUID(), createdAt: now, updatedAt: now, ...input }
    await this.ctx.database.create(MODERATION_EVENT_TABLE, record)
    return record
  }

  async listRecentEvents(limit = 20) {
    const records = await this.ctx.database.get(MODERATION_EVENT_TABLE, {})
    return records.sort(sortByCreatedDesc).slice(0, limit)
  }

  async saveMessage(record: MessageLedgerRecord) {
    const [existing] = await this.ctx.database.get(MODERATION_MESSAGE_LEDGER_TABLE, { messageId: record.messageId })
    if (existing) {
      await this.ctx.database.set(MODERATION_MESSAGE_LEDGER_TABLE, { messageId: record.messageId }, record)
      return
    }
    await this.ctx.database.create(MODERATION_MESSAGE_LEDGER_TABLE, record)
  }

  async listRecentMessages(guildId: string, limit: number) {
    const records = await this.ctx.database.get(MODERATION_MESSAGE_LEDGER_TABLE, { guildId })
    return records.sort(sortByCreatedDesc).slice(0, limit)
  }

  async markMessageDeleted(messageId: string, deletedAt: Date) {
    await this.ctx.database.set(MODERATION_MESSAGE_LEDGER_TABLE, { messageId }, { deletedAt })
  }

  async getMessage(messageId: string) {
    const [record] = await this.ctx.database.get(MODERATION_MESSAGE_LEDGER_TABLE, { messageId })
    return record as MessageLedgerRecord | undefined
  }

  async incrementWarning(guildId: string, memberId: string, reason: string, now: Date) {
    const id = createGuildScopedID(guildId, memberId)
    const [record] = await this.ctx.database.get(MODERATION_WARNING_TABLE, { id })
    if (!record) {
      const created: WarningCounterRecord = {
        id,
        guildId,
        memberId,
        total: 1,
        lastReason: reason,
        lastAt: now,
        createdAt: now,
        updatedAt: now,
      }
      await this.ctx.database.create(MODERATION_WARNING_TABLE, created)
      return created
    }
    const total = record.total + 1
    await this.ctx.database.set(MODERATION_WARNING_TABLE, { id }, {
      total,
      lastReason: reason,
      lastAt: now,
      updatedAt: now,
    })
    return { ...record, total, lastReason: reason, lastAt: now, updatedAt: now }
  }

  async getWarningCounter(guildId: string, memberId: string) {
    const id = createGuildScopedID(guildId, memberId)
    const [record] = await this.ctx.database.get(MODERATION_WARNING_TABLE, { id })
    return record as WarningCounterRecord | undefined
  }

  async listWarningCounters(guildId?: string) {
    const query = guildId ? { guildId } : {}
    return this.ctx.database.get(MODERATION_WARNING_TABLE, query)
  }

  async upsertKeywordRule(record: KeywordRuleRecord) {
    const [existing] = await this.ctx.database.get(MODERATION_KEYWORD_RULE_TABLE, { id: record.id })
    if (existing) {
      await this.ctx.database.set(MODERATION_KEYWORD_RULE_TABLE, { id: record.id }, record)
      return
    }
    await this.ctx.database.create(MODERATION_KEYWORD_RULE_TABLE, record)
  }

  async listKeywordRules(guildId: string) {
    const records = await this.ctx.database.get(MODERATION_KEYWORD_RULE_TABLE, {})
    return records.filter((record) => record.guildId === guildId || record.guildId === '*')
  }

  async listAllKeywordRules() {
    return this.ctx.database.get(MODERATION_KEYWORD_RULE_TABLE, {})
  }

  async createReview(input: Omit<ReviewQueueRecord, 'id' | 'createdAt' | 'updatedAt'>) {
    const now = new Date()
    const record: ReviewQueueRecord = { id: randomUUID(), createdAt: now, updatedAt: now, ...input }
    await this.ctx.database.create(MODERATION_REVIEW_TABLE, record)
    return record
  }

  async listPendingReviews(guildId?: string) {
    const query = guildId
      ? { guildId, status: REVIEW_STATUS_PENDING }
      : { status: REVIEW_STATUS_PENDING }
    return this.ctx.database.get(MODERATION_REVIEW_TABLE, query)
  }

  async resolveReview(id: string, status: ReviewStatus, operatorMemberId: string, resolutionNote: string | null) {
    const updatedAt = new Date()
    await this.ctx.database.set(MODERATION_REVIEW_TABLE, { id }, {
      status,
      operatorMemberId,
      resolutionNote,
      updatedAt,
    })
  }

  async setMemberRoles(guildId: string, memberId: string, roles: string[]) {
    const now = new Date()
    const id = createGuildScopedID(guildId, memberId)
    const record: MemberRoleRecord = { id, guildId, memberId, roles, createdAt: now, updatedAt: now }
    const [existing] = await this.ctx.database.get(MODERATION_MEMBER_ROLE_TABLE, { id })
    if (existing) {
      await this.ctx.database.set(MODERATION_MEMBER_ROLE_TABLE, { id }, { roles, updatedAt: now })
      return
    }
    await this.ctx.database.create(MODERATION_MEMBER_ROLE_TABLE, record)
  }

  async getMemberRoles(guildId: string, memberId: string) {
    const id = createGuildScopedID(guildId, memberId)
    const [record] = await this.ctx.database.get(MODERATION_MEMBER_ROLE_TABLE, { id })
    return record?.roles || []
  }

  async upsertCommandPolicy(record: CommandPolicyRecord) {
    const [existing] = await this.ctx.database.get(MODERATION_COMMAND_POLICY_TABLE, { commandId: record.commandId })
    if (existing) {
      await this.ctx.database.set(MODERATION_COMMAND_POLICY_TABLE, { commandId: record.commandId }, record)
      return
    }
    await this.ctx.database.create(MODERATION_COMMAND_POLICY_TABLE, record)
  }

  async getCommandPolicy(commandId: string) {
    const [record] = await this.ctx.database.get(MODERATION_COMMAND_POLICY_TABLE, { commandId })
    return record as CommandPolicyRecord | undefined
  }

  async listCommandPolicies() {
    return this.ctx.database.get(MODERATION_COMMAND_POLICY_TABLE, {})
  }

  async listMemberRoles(guildId?: string) {
    const query = guildId ? { guildId } : {}
    return this.ctx.database.get(MODERATION_MEMBER_ROLE_TABLE, query)
  }

  async getFunProfile(memberId: string) {
    const [record] = await this.ctx.database.get(MODERATION_FUN_PROFILE_TABLE, { memberId })
    return record as FunProfileRecord | undefined
  }

  async saveFunProfile(record: FunProfileRecord) {
    const [existing] = await this.ctx.database.get(MODERATION_FUN_PROFILE_TABLE, { memberId: record.memberId })
    if (existing) {
      await this.ctx.database.set(MODERATION_FUN_PROFILE_TABLE, { memberId: record.memberId }, record)
      return
    }
    await this.ctx.database.create(MODERATION_FUN_PROFILE_TABLE, record)
  }

  async createReport(input: Omit<ModerationReportRecord, 'id' | 'createdAt' | 'updatedAt'>) {
    const now = new Date()
    const record: ModerationReportRecord = { id: randomUUID(), createdAt: now, updatedAt: now, ...input }
    await this.ctx.database.create(MODERATION_REPORT_TABLE, record)
    return record
  }

  async listOpenReports(guildId?: string) {
    const query = guildId ? { guildId } : {}
    return this.ctx.database.get(MODERATION_REPORT_TABLE, query)
  }

  async updateReportAIResult(id: string, aiStatus: ModerationReportRecord['aiStatus'], aiSeverity: ModerationReportRecord['aiSeverity'], aiSummary: string | null) {
    await this.ctx.database.set(MODERATION_REPORT_TABLE, { id }, {
      aiStatus,
      aiSeverity,
      aiSummary,
      updatedAt: new Date(),
    })
  }

  async getOverview(): Promise<ModerationOverview> {
    const [events, reviews, reports, warnings, guards] = await Promise.all([
      this.listRecentEvents(20),
      this.listPendingReviews(),
      this.listOpenReports(),
      this.listWarningCounters(),
      this.ctx.database.get(GUARD_MEMBER_TABLE, {}),
    ])
    return {
      pendingReviews: reviews.length,
      openReports: reports.length,
      warningMembers: warnings.filter((item) => item.total > 0).length,
      highRiskEvents: events.filter((item) => item.level === 'high' || item.level === 'critical').length,
      recentEvents: events,
    }
  }
}

function createGuildScopedID(guildId: string, memberId: string) {
  return `${guildId}:${memberId}`
}

function sortByCreatedDesc<T extends { createdAt: Date }>(left: T, right: T) {
  return right.createdAt.getTime() - left.createdAt.getTime()
}
