import type { Context, Universal } from 'koishi'

import { GUARD_MEMBER_TABLE, type GuardMemberRecord } from '@stuhelper/koishi-shared'
import {
  MODERATION_REPORT_TABLE,
  MODERATION_REVIEW_TABLE,
  type ModerationReportRecord,
  type ReviewQueueRecord,
} from '@stuhelper/koishi-moderation-core'

type ManagedBot = Universal.Methods & { platform?: string; selfId: string }

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
  const bot = ctx.bots.find((item) => item.platform === platform && item.selfId === botSelfId)
  if (!bot) {
    throw new Error(`console bot not found: ${platform}:${botSelfId}`)
  }
  return bot as ManagedBot
}

export function createReviewResolvedEvent(review: ReviewQueueRecord, level: 'info' | 'high', note?: string) {
  return {
    platform: review.platform,
    botSelfId: review.botSelfId,
    guildId: review.guildId,
    channelId: review.channelId,
    memberId: review.memberId,
    type: 'review_resolved',
    level,
    summary: level === 'info' ? `复核已驳回：${review.memberId}` : `复核已执行：${review.memberId}`,
    payload: { reviewId: review.id, note: note || null, actionType: review.actionType },
  }
}

export function createAdmissionEvent(
  record: GuardMemberRecord,
  type: 'join_released' | 'action_executed',
  level: 'info' | 'medium' | 'high',
  summaryPrefix: string,
  note?: string,
  action?: string,
  deadlineAt?: string,
) {
  return {
    platform: record.platform,
    botSelfId: record.botSelfId,
    guildId: record.guildId,
    channelId: record.channelId,
    memberId: record.memberId,
    type,
    level,
    summary: `${summaryPrefix} ${record.memberId}`,
    payload: { action: action || null, deadlineAt: deadlineAt || null, note: note || null, source: 'console' },
  }
}

export function createReportActionEvent(
  report: ModerationReportRecord,
  summaryPrefix: string,
  level: 'info' | 'high',
  action: string,
  note?: string,
) {
  return {
    platform: report.platform,
    botSelfId: report.botSelfId,
    guildId: report.guildId,
    channelId: report.channelId,
    memberId: report.targetMemberId,
    type: 'action_executed',
    level,
    summary: `${summaryPrefix}：${report.targetMemberId}`,
    payload: { action, note: note || null, reportId: report.id, source: 'console' },
  }
}
