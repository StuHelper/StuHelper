import type { Context, Universal } from 'koishi'

import { GUARD_MEMBER_TABLE, type GuardMemberRecord } from '@stuhelper/koishi-shared'
import {
  MODERATION_REPORT_TABLE,
  MODERATION_REVIEW_TABLE,
  type ModerationReportRecord,
  type ReviewQueueRecord,
} from '@stuhelper/koishi-moderation-core'

import type { WorkItemActionActor } from './review-action-types'

type ManagedBot = Universal.Methods & { platform?: string; selfId: string }
const QQ_COMPATIBLE_RUNTIME_PLATFORMS = new Set(['qq', 'onebot', 'red'])

interface ReviewResolvedEventInput {
  readonly review: ReviewQueueRecord
  readonly level: 'info' | 'high'
  readonly actor: WorkItemActionActor
  readonly note?: string
}

interface AdmissionEventInput {
  readonly record: GuardMemberRecord
  readonly type: 'join_released' | 'action_executed'
  readonly level: 'info' | 'medium' | 'high'
  readonly summaryPrefix: string
  readonly actor: WorkItemActionActor
  readonly note?: string
  readonly action?: string
  readonly deadlineAt?: string
}

interface ReportActionEventInput {
  readonly report: ModerationReportRecord
  readonly summaryPrefix: string
  readonly level: 'info' | 'high'
  readonly action: string
  readonly actor: WorkItemActionActor
  readonly note?: string
}

export async function requireReviewRecord(ctx: Context, reviewId: string) {
  const [review] = await ctx.database.get(MODERATION_REVIEW_TABLE, { id: reviewId }) as ReviewQueueRecord[]
  if (!review) {
    throw new Error(`review not found: ${reviewId}`)
  }
  return review
}

export async function requireGuardRecord(ctx: Context, guardId: string) {
  const [record] = await ctx.database.get(GUARD_MEMBER_TABLE, { id: guardId }) as GuardMemberRecord[]
  if (!record) {
    throw new Error(`guard member not found: ${guardId}`)
  }
  return record
}

export async function requireReportRecord(ctx: Context, reportId: string) {
  const [report] = await ctx.database.get(MODERATION_REPORT_TABLE, { id: reportId }) as ModerationReportRecord[]
  if (!report) {
    throw new Error(`report not found: ${reportId}`)
  }
  return report
}

export function resolveManagedBot(ctx: Context, platform: string, botSelfId: string) {
  const bot = ctx.bots.find((item) => item.platform === platform && item.selfId === botSelfId) ||
    ctx.bots.find((item) => isCompatibleRuntimeBot(platform, item.platform) && item.selfId === botSelfId)
  if (!bot) {
    throw new Error(`console bot not found: ${platform}:${botSelfId}`)
  }
  return bot as ManagedBot
}

function isCompatibleRuntimeBot(recordPlatform: string, runtimePlatform: string | undefined) {
  return recordPlatform === 'qq' &&
    Boolean(runtimePlatform && QQ_COMPATIBLE_RUNTIME_PLATFORMS.has(runtimePlatform))
}

export function createReviewResolvedEvent(input: ReviewResolvedEventInput) {
  const { review, level, actor, note } = input
  return {
    platform: review.platform,
    botSelfId: review.botSelfId,
    guildId: review.guildId,
    channelId: review.channelId,
    memberId: review.memberId,
    type: 'review_resolved',
    level,
    summary: level === 'info' ? `复核已驳回：${review.memberId}` : `复核已执行：${review.memberId}`,
    payload: {
      reviewId: review.id,
      note: note || null,
      actionType: review.actionType,
      operatorMemberId: actor.memberId,
      operatorName: actor.displayName,
      source: 'console',
    },
  }
}

export function createAdmissionEvent(input: AdmissionEventInput) {
  const { record, type, level, summaryPrefix, actor, note, action, deadlineAt } = input
  return {
    platform: record.platform,
    botSelfId: record.botSelfId,
    guildId: record.guildId,
    channelId: record.channelId,
    memberId: record.memberId,
    type,
    level,
    summary: `${summaryPrefix} ${record.memberId}`,
    payload: {
      action: action || null,
      deadlineAt: deadlineAt || null,
      note: note || null,
      source: 'console',
      operatorMemberId: actor.memberId,
      operatorName: actor.displayName,
    },
  }
}

export function createReportActionEvent(input: ReportActionEventInput) {
  const { report, summaryPrefix, level, action, actor, note } = input
  return {
    platform: report.platform,
    botSelfId: report.botSelfId,
    guildId: report.guildId,
    channelId: report.channelId,
    memberId: report.targetMemberId,
    type: 'action_executed',
    level,
    summary: `${summaryPrefix}：${report.targetMemberId}`,
    payload: {
      action,
      note: note || null,
      reportId: report.id,
      source: 'console',
      operatorMemberId: actor.memberId,
      operatorName: actor.displayName,
    },
  }
}
