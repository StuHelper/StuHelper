import { randomUUID } from 'node:crypto'

import { Context } from 'koishi'

import { GUARD_MEMBER_TABLE } from '@stuhelper/koishi-shared'

import { KeyedMutex } from './keyed-mutex'

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
import type {
  ClaimPendingReviewInput,
  FinalizeClaimedReviewInput,
  IncrementWarningInput,
  PruneExpiredInput,
  RollbackClaimedReviewInput,
  ResolveReviewInput,
  UpdatePendingReviewInput,
  UpdateReportAIResultInput,
} from './store-inputs'

const DAY_IN_MS = 24 * 60 * 60 * 1000

const REVIEW_STATUS_PENDING: ReviewStatus = 'pending'
const REVIEW_STATUS_APPROVED: ReviewStatus = 'approved'
const REVIEW_STATUS_EXECUTED: ReviewStatus = 'executed'
const REVIEW_STATUS_STUCK_MANUAL: ReviewStatus = 'stuck_manual'
const ACTIONABLE_REVIEW_STATUSES = new Set<ReviewStatus>([
  REVIEW_STATUS_PENDING,
  REVIEW_STATUS_STUCK_MANUAL,
])

// 模块级互斥锁：同一进程内所有 ModerationStore 实例（core/group-guard/admin
// 插件各持一份）共享，保证同一计数器的并发自增不会互相覆盖。
const warningCounterMutex = new KeyedMutex()

export class ModerationStore {
  constructor(private readonly ctx: Pick<Context, 'database'>) {}

  async appendEvent(input: Omit<ModerationEventRecord, 'id' | 'createdAt' | 'updatedAt'>) {
    const now = new Date()
    const record: ModerationEventRecord = { id: randomUUID(), createdAt: now, updatedAt: now, ...input }
    await this.ctx.database.create(MODERATION_EVENT_TABLE, record)
    return record
  }

  async listRecentEvents(limit = 20) {
    const records = await this.ctx.database.get(MODERATION_EVENT_TABLE, {}, {
      sort: { createdAt: 'desc' },
      limit,
    })
    return records as ModerationEventRecord[]
  }

  async saveMessage(record: MessageLedgerRecord) {
    const [existing] = await this.ctx.database.get(MODERATION_MESSAGE_LEDGER_TABLE, { messageId: record.messageId })
    if (existing) {
      await this.ctx.database.set(
        MODERATION_MESSAGE_LEDGER_TABLE,
        { messageId: record.messageId },
        {
          content: record.content,
          normalizedContent: record.normalizedContent,
          quoteMessageId: record.quoteMessageId,
          deletedAt: record.deletedAt,
        },
      )
      return
    }
    await this.ctx.database.create(MODERATION_MESSAGE_LEDGER_TABLE, record)
  }

  async listRecentMessages(guildId: string, limit: number) {
    const records = await this.ctx.database.get(MODERATION_MESSAGE_LEDGER_TABLE, { guildId }, {
      sort: { createdAt: 'desc' },
      limit,
    })
    return records as MessageLedgerRecord[]
  }

  /**
   * 按保留期修剪消息台账与事件日志（F004）。
   * 两张表都是 append-only 热表，不修剪会无限增长拖慢全部查询。
   */
  async pruneExpired(input: PruneExpiredInput) {
    const now = input.now ?? new Date()
    const messageCutoff = new Date(now.getTime() - input.messageRetentionDays * DAY_IN_MS)
    const eventCutoff = new Date(now.getTime() - input.eventRetentionDays * DAY_IN_MS)
    const [messages, events] = await Promise.all([
      this.ctx.database.remove(MODERATION_MESSAGE_LEDGER_TABLE, { createdAt: { $lt: messageCutoff } }),
      this.ctx.database.remove(MODERATION_EVENT_TABLE, { createdAt: { $lt: eventCutoff } }),
    ])
    return {
      removedMessages: messages.removed ?? 0,
      removedEvents: events.removed ?? 0,
    }
  }

  async markMessageDeleted(messageId: string, deletedAt: Date) {
    await this.ctx.database.set(MODERATION_MESSAGE_LEDGER_TABLE, { messageId }, { deletedAt })
  }

  async getMessage(messageId: string) {
    const [record] = await this.ctx.database.get(MODERATION_MESSAGE_LEDGER_TABLE, { messageId })
    return record as MessageLedgerRecord | undefined
  }

  async incrementWarning(input: IncrementWarningInput) {
    const id = createGuildScopedID(input.guildId, input.memberId)
    return warningCounterMutex.run(id, () => this.applyWarningIncrement(id, input))
  }

  private async applyWarningIncrement(id: string, input: IncrementWarningInput): Promise<WarningCounterRecord> {
    const [record] = await this.ctx.database.get(MODERATION_WARNING_TABLE, { id })
    if (!record) {
      const created: WarningCounterRecord = {
        id,
        guildId: input.guildId,
        memberId: input.memberId,
        total: 1,
        lastReason: input.reason,
        lastAt: input.now,
        createdAt: input.now,
        updatedAt: input.now,
      }
      await this.ctx.database.create(MODERATION_WARNING_TABLE, created)
      return created
    }
    const total = record.total + 1
    await this.ctx.database.set(MODERATION_WARNING_TABLE, { id }, {
      total,
      lastReason: input.reason,
      lastAt: input.now,
      updatedAt: input.now,
    })
    return { ...record, total, lastReason: input.reason, lastAt: input.now, updatedAt: input.now }
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
      await this.ctx.database.set(
        MODERATION_KEYWORD_RULE_TABLE,
        { id: record.id },
        omitFields(record, ['id', 'createdAt']),
      )
      return
    }
    await this.ctx.database.create(MODERATION_KEYWORD_RULE_TABLE, record)
  }

  async getKeywordRule(id: string) {
    const [record] = await this.ctx.database.get(MODERATION_KEYWORD_RULE_TABLE, { id })
    return record as KeywordRuleRecord | undefined
  }

  async listKeywordRules(guildId: string) {
    const records = await this.ctx.database.get(MODERATION_KEYWORD_RULE_TABLE, {
      guildId: { $in: [guildId, '*'] },
    })
    return records as KeywordRuleRecord[]
  }

  async listAllKeywordRules() {
    return this.ctx.database.get(MODERATION_KEYWORD_RULE_TABLE, {})
  }

  async deleteKeywordRule(id: string) {
    return this.ctx.database.remove(MODERATION_KEYWORD_RULE_TABLE, { id })
  }

  async createReview(input: Omit<ReviewQueueRecord, 'id' | 'createdAt' | 'updatedAt'>) {
    const now = new Date()
    const record: ReviewQueueRecord = { id: randomUUID(), createdAt: now, updatedAt: now, ...input }
    await this.ctx.database.create(MODERATION_REVIEW_TABLE, record)
    return record
  }

  async listPendingReviews(guildId: string) {
    const normalizedGuildId = guildId.trim()
    if (!normalizedGuildId) {
      return []
    }
    const records = await this.ctx.database.get(MODERATION_REVIEW_TABLE, { guildId: normalizedGuildId })
    return records.filter((record) => ACTIONABLE_REVIEW_STATUSES.has(record.status))
  }

  async listAllPendingReviews() {
    const records = await this.ctx.database.get(MODERATION_REVIEW_TABLE, {})
    return records.filter((record) => ACTIONABLE_REVIEW_STATUSES.has(record.status))
  }

  async listApprovedReviews() {
    return this.ctx.database.get(MODERATION_REVIEW_TABLE, { status: REVIEW_STATUS_APPROVED })
  }

  async resolveReview(input: ResolveReviewInput) {
    const updatedAt = new Date()
    await this.ctx.database.set(MODERATION_REVIEW_TABLE, { id: input.id }, {
      status: input.status,
      operatorMemberId: input.operatorMemberId,
      resolutionNote: input.resolutionNote,
      updatedAt,
    })
  }

  async tryUpdatePendingReview(input: UpdatePendingReviewInput) {
    const result = await this.ctx.database.set(MODERATION_REVIEW_TABLE, {
      id: input.review.id,
      status: REVIEW_STATUS_PENDING,
      updatedAt: input.review.updatedAt,
    }, {
      status: input.status,
      operatorMemberId: input.operatorMemberId,
      resolutionNote: input.resolutionNote,
      updatedAt: input.updatedAt,
    })
    return result.matched === 1
  }

  async tryClaimPendingReview(input: ClaimPendingReviewInput) {
    const result = await this.ctx.database.set(MODERATION_REVIEW_TABLE, {
      id: input.review.id,
      status: REVIEW_STATUS_PENDING,
      updatedAt: input.review.updatedAt,
    }, {
      status: REVIEW_STATUS_APPROVED,
      operatorMemberId: input.operatorMemberId,
      resolutionNote: input.resolutionNote,
      updatedAt: input.claimedAt,
    })
    return result.matched === 1
  }

  async tryFinalizeClaimedReview(input: FinalizeClaimedReviewInput) {
    const result = await this.ctx.database.set(MODERATION_REVIEW_TABLE, {
      id: input.reviewId,
      status: REVIEW_STATUS_APPROVED,
      updatedAt: input.claimedAt,
    }, {
      status: REVIEW_STATUS_EXECUTED,
      operatorMemberId: input.operatorMemberId,
      resolutionNote: input.resolutionNote,
      updatedAt: input.executedAt,
    })
    return result.matched === 1
  }

  async rollbackClaimedReview(input: RollbackClaimedReviewInput) {
    await this.ctx.database.set(MODERATION_REVIEW_TABLE, {
      id: input.reviewId,
      status: REVIEW_STATUS_APPROVED,
      updatedAt: input.claimedAt,
    }, {
      status: REVIEW_STATUS_PENDING,
      operatorMemberId: null,
      resolutionNote: null,
      updatedAt: input.rolledBackAt,
    })
  }

  async markReviewStuck(review: Pick<ReviewQueueRecord, 'id' | 'updatedAt'>, resolutionNote: string | null) {
    const updatedAt = new Date()
    const result = await this.ctx.database.set(MODERATION_REVIEW_TABLE, {
      id: review.id,
      status: REVIEW_STATUS_APPROVED,
      updatedAt: review.updatedAt,
    }, {
      status: REVIEW_STATUS_STUCK_MANUAL,
      resolutionNote,
      updatedAt,
    })
    return result.matched === 1
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
      await this.ctx.database.set(
        MODERATION_COMMAND_POLICY_TABLE,
        { commandId: record.commandId },
        omitFields(record, ['commandId', 'createdAt']),
      )
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
      await this.ctx.database.set(
        MODERATION_FUN_PROFILE_TABLE,
        { memberId: record.memberId },
        omitFields(record, ['memberId', 'createdAt']),
      )
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

  async removeReport(id: string) {
    return this.ctx.database.remove(MODERATION_REPORT_TABLE, { id })
  }

  async updateReportAIResult(input: UpdateReportAIResultInput) {
    await this.ctx.database.set(MODERATION_REPORT_TABLE, { id: input.id }, {
      aiStatus: input.aiStatus,
      aiSeverity: input.aiSeverity,
      aiSummary: input.aiSummary,
      updatedAt: new Date(),
    })
  }

  async transact<T>(callback: (store: ModerationStore) => Promise<T>) {
    return this.ctx.database.transact(async (database) => {
      return callback(new ModerationStore({ database }))
    })
  }

  async getOverview(): Promise<ModerationOverview> {
    const [events, reviews, reports, warnings, guards] = await Promise.all([
      this.listRecentEvents(20),
      this.listAllPendingReviews(),
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

function omitFields<T extends object, K extends keyof T>(record: T, keys: readonly K[]): Omit<T, K> {
  const patch = { ...record }
  for (const key of keys) {
    delete (patch as Partial<T>)[key]
  }
  return patch
}
